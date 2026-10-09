#!/usr/bin/env python3
"""
Bounded integration checks for the native nginx upstream healthcheck module.

The command runs local DNS and backend fixtures around the supplied nginx
binary, then exits nonzero on the first failed runtime assertion.
"""

import argparse
import http.client
import json
import math
import os
import re
import signal
import socket
import ssl
import struct
import subprocess
import sys
import tempfile
import threading
import time
from collections import Counter
from pathlib import Path
from typing import Any, Callable, Dict, Iterable, List, Optional, Tuple, Union


MetricSample = Tuple[str, Dict[str, str], float]
SRVRecord = Tuple[int, int, str, int]


class VerificationFailure(Exception):
    """
    Signal a failed assertion in the runtime verifier.
    """
    pass


def require(condition: bool, message: str) -> None:
    """
    Print a successful check or raise a verification failure.
    """
    if not condition:
        raise VerificationFailure(message)
    print(f"PASS {message}", flush=True)


def free_port(address: str = "127.0.0.1", family: int = socket.AF_INET, exclude: Optional[Iterable[int]] = None) -> int:
    """
    Find an available TCP port bound to a requested local address.
    """
    excluded = set(exclude or ())
    for _ in range(64):
        with socket.socket(family, socket.SOCK_STREAM) as sock:
            sock.bind((address, 0))
            port = sock.getsockname()[1]
        if port not in excluded:
            return port
    raise VerificationFailure("could not find a loopback port outside the excluded set")


def dns_name_wire(name: str) -> bytes:
    """
    Encode a DNS hostname as uncompressed wire-format labels.
    """
    return b"".join(bytes((len(part),)) + part.encode("ascii") for part in name.rstrip(".").split(".")) + b"\0"


def decode_dns_name(packet: bytes, offset: int) -> Tuple[str, int]:
    """
    Decode a DNS question name and return its next packet offset.
    """
    labels = []
    while offset < len(packet):
        length = packet[offset]
        offset += 1
        if length & 0xC0 == 0xC0:
            offset += 1
            break
        if length == 0:
            break
        labels.append(packet[offset:offset + length].decode("ascii"))
        offset += length
    return ".".join(labels).lower(), offset


class AuthoritativeDNS:
    """
    Serve mutable A, AAAA, and SRV answers on a loopback UDP socket.
    """
    def __init__(self) -> None:
        """
        Bind a local DNS socket and start its response thread.
        """
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.socket.bind(("127.0.0.1", 0))
        self.port = self.socket.getsockname()[1]
        self.answers = {}
        self.srv = {}
        self.queries = Counter()
        self.lock = threading.Lock()
        self.stopping = threading.Event()
        self.thread = threading.Thread(target=self._serve, daemon=True)
        self.thread.start()

    def set_addresses(self, name: str, addresses: Iterable[str]) -> None:
        """
        Replace the IPv4 answer list for one fixture name.
        """
        with self.lock:
            self.answers[(name.lower(), 1)] = list(addresses)

    def set_ipv6(self, name: str, addresses: Iterable[str]) -> None:
        """
        Replace the IPv6 answer list for one fixture name.
        """
        with self.lock:
            self.answers[(name.lower(), 28)] = list(addresses)

    def set_srv(self, name: str, records: Iterable[SRVRecord]) -> None:
        """
        Replace the SRV records for one fixture service.
        """
        with self.lock:
            self.srv[name.lower()] = [(priority, weight, target.lower(), port) for priority, weight, target, port in records]

    def query_count(self, name: str, qtype: int) -> int:
        """
        Return the synchronized count of one DNS question type.
        """
        with self.lock:
            return self.queries[(name.lower(), qtype)]

    def _serve(self) -> None:
        self.socket.settimeout(0.2)
        while not self.stopping.is_set():
            try:
                packet, address = self.socket.recvfrom(4096)
            except socket.timeout:
                continue
            except OSError:
                return
            try:
                self._answer(packet, address)
            except (IndexError, UnicodeDecodeError, OSError, struct.error):
                continue

    def _answer(self, packet: bytes, address: Tuple[str, int]) -> None:
        if len(packet) < 17:
            return
        name, offset = decode_dns_name(packet, 12)
        qtype, qclass = struct.unpack("!HH", packet[offset:offset + 4])
        question_end = offset + 4
        with self.lock:
            self.queries[(name, qtype)] += 1
            addresses = list(self.answers.get((name, qtype), []))
            srv = list(self.srv.get(name, []))
        answers = []
        if qclass == 1 and qtype == 1:
            for value in addresses:
                answers.append((b"\xc0\x0c" + struct.pack("!HHI", 1, 1, 1), socket.inet_aton(value)))
        elif qclass == 1 and qtype == 28:
            for value in addresses:
                answers.append((b"\xc0\x0c" + struct.pack("!HHI", 28, 1, 1), socket.inet_pton(socket.AF_INET6, value)))
        elif qclass == 1 and qtype == 33:
            for priority, weight, target, port in srv:
                record = struct.pack("!HHH", priority, weight, port) + dns_name_wire(target)
                answers.append((b"\xc0\x0c" + struct.pack("!HHI", 33, 1, 1), record))
        flags = 0x8580
        header = packet[:2] + struct.pack("!HHHHH", flags, 1, len(answers), 0, 0)
        response = header + packet[12:question_end]
        for record, data in answers:
            response += record[:10] + struct.pack("!H", len(data)) + data
        self.socket.sendto(response, address)

    def close(self) -> None:
        """
        Stop the DNS responder and join its thread.
        """
        self.stopping.set()
        self.socket.close()
        self.thread.join(timeout=1)
        if self.thread.is_alive():
            raise VerificationFailure("DNS fixture responder did not stop")


class BackendState:
    """
    Track fixture request state and control readiness responses.
    """
    def __init__(self, identity: str, status: int = 200) -> None:
        """
        Create a backend identity with its initial health response code.
        """
        self.identity = identity
        self.ready_status = status
        self.ready_delay = 0.0
        self.malformed = False
        self.hosts = []
        self.paths = Counter()
        self.sni = []
        self.lock = threading.Lock()
        self.block_next = None
        self.block_started = threading.Event()
        self.block_write_succeeded = threading.Event()

    def record(self, path: str, host: str) -> Tuple[int, float, bool, bool]:
        """
        Record a request and return the response behavior to its handler.
        """
        with self.lock:
            self.paths[path] += 1
            self.hosts.append((path, host))
            block = self.block_next if path == "/readyz" else None
            if block is not None:
                self.block_next = None
            status = self.ready_status
            delay = self.ready_delay
            malformed = self.malformed
        if block is not None:
            release, status = block
            self.block_started.set()
            release.wait(timeout=40)
            return status, 0.0, False, True
        return status, delay, malformed, False

    def count(self, path: str = "/readyz") -> int:
        """
        Return the number of observed requests for one path.
        """
        with self.lock:
            return self.paths[path]

    def hosts_for(self, path: str = "/readyz") -> List[str]:
        """
        Return Host headers observed for one request path.
        """
        with self.lock:
            return [host for seen_path, host in self.hosts if seen_path == path]


class FixtureHTTPServer:
    """
    Run a loopback HTTP or HTTPS readiness and traffic fixture.
    """
    def __init__(self, address: str, port: int, state: BackendState, certificate: Optional[Path] = None, private_key: Optional[Path] = None) -> None:
        """
        Bind a backend endpoint and optionally wrap it in TLS.
        """
        from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

        class Handler(BaseHTTPRequestHandler):
            """
            Handle fixture health probes and client traffic requests.
            """
            def do_GET(self) -> None:
                """
                Return configured health responses and stable traffic identity.
                """
                status, delay, malformed, blocked = self.server.fixture_state.record(self.path, self.headers.get("Host", ""))
                if delay:
                    time.sleep(delay)
                if malformed:
                    try:
                        self.connection.sendall(b"NOT HTTP\r\n\r\n")
                    except OSError:
                        pass
                    self.close_connection = True
                    return
                try:
                    if self.path == "/readyz":
                        self.send_response(status)
                        self.end_headers()
                        if status != 204:
                            self.wfile.write(b"ready\n" if status == 200 else b"unready\n")
                        if blocked:
                            self.server.fixture_state.block_write_succeeded.set()
                    else:
                        self.send_response(200)
                        self.end_headers()
                        self.wfile.write((self.server.fixture_state.identity + "\n").encode())
                except OSError:
                    return

            def log_message(self, format_string: str, *args: Any) -> None:
                """
                Suppress fixture access logs.
                """
                return

        class Server(ThreadingHTTPServer):
            """
            Bind a threaded HTTP server for one fixture address.
            """
            allow_reuse_address = True
            daemon_threads = False

            def handle_error(self, request: Any, client_address: Any) -> None:
                """
                Suppress expected peer-close and TLS negotiation errors only.
                """
                error = sys.exc_info()[1]
                if isinstance(error, (OSError, ssl.SSLError)):
                    return
                super().handle_error(request, client_address)

        self.state = state
        Server.address_family = socket.AF_INET6 if ":" in address else socket.AF_INET
        self.server = Server((address, port), Handler)
        self.server.fixture_state = state
        if certificate and private_key:
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            context.load_cert_chain(certificate, private_key)

            def _record_sni(connection: Any, server_name: Optional[str], ssl_context: ssl.SSLContext) -> None:
                with state.lock:
                    state.sni.append(server_name)

            context.set_servername_callback(_record_sni)
            self.server.socket = context.wrap_socket(self.server.socket, server_side=True)
        self.thread = threading.Thread(target=self.server.serve_forever, args=(0.05,), daemon=True)
        self.thread.start()

    def close(self) -> None:
        """
        Stop the endpoint and wait for its serving thread.
        """
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=1)
        if self.thread.is_alive():
            raise VerificationFailure("HTTP fixture server did not stop")


class FixtureTCPServer:
    """
    Accept and close loopback TCP connections for protocol checks.
    """
    def __init__(self, address: str = "127.0.0.1") -> None:
        """
        Bind an ephemeral TCP port and start its accept loop.
        """
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.socket.bind((address, 0))
        self.socket.listen(64)
        self.socket.settimeout(0.2)
        self.port = self.socket.getsockname()[1]
        self.stopping = threading.Event()
        self.connections = 0
        self.lock = threading.Lock()
        self.thread = threading.Thread(target=self._serve, daemon=True)
        self.thread.start()

    def _serve(self) -> None:
        while not self.stopping.is_set():
            try:
                connection, _ = self.socket.accept()
            except socket.timeout:
                continue
            except OSError:
                return
            with self.lock:
                self.connections += 1
            connection.close()

    def close(self) -> None:
        """
        Stop the listener and join its accept thread.
        """
        self.stopping.set()
        self.socket.close()
        self.thread.join(timeout=1)
        if self.thread.is_alive():
            raise VerificationFailure("TCP fixture server did not stop")


class StreamBackendState:
    """
    Track raw stream probes, traffic, response bytes, and optional delays.
    """
    def __init__(self, identity: str, probe_send: bytes = b"", probe_reply: bytes = b"", probe_delay: float = 0.0) -> None:
        """
        Create a backend state with an identity and probe exchange contract.
        """
        self.identity = identity.encode()
        self.probe_send = probe_send
        self.probe_reply = probe_reply
        self.probe_delay = probe_delay
        self.response_chunks: List[bytes] = []
        self.chunk_delay = 0.0
        self.pause_after_first_chunk = False
        self.first_chunk_sent = threading.Event()
        self.gate = threading.Event()
        self.gate.set()
        self.repeat_response = b""
        self.repeat_delay = 0.01
        self.banner = b""
        self.hold_open = False
        self.connections = 0
        self.payloads: List[bytes] = []
        self.sni: List[Optional[str]] = []
        self.lock = threading.Lock()

    def set_probe_reply(self, reply: bytes, delay: Optional[float] = None) -> None:
        """
        Change the response used for subsequent healthcheck exchanges.
        """
        with self.lock:
            self.probe_reply = reply
            if delay is not None:
                self.probe_delay = delay

    def observe(self, payload: bytes) -> Tuple[bytes, float, List[bytes], float, bytes, bool]:
        """
        Record received bytes and select a health or traffic response.
        """
        with self.lock:
            self.connections += 1
            self.payloads.append(payload)
            if self.probe_send and payload == self.probe_send:
                return (
                    self.probe_reply,
                    self.probe_delay,
                    list(self.response_chunks),
                    self.chunk_delay,
                    self.repeat_response,
                    self.pause_after_first_chunk,
                )
            if not payload:
                return b"", 0.0, [], 0.0, b"", False
            return self.identity, 0.0, [], 0.0, b"", False


class FixtureStreamServer:
    """
    Serve bounded raw TCP exchanges on a loopback address.
    """
    def __init__(self, address: str, port: int, state: StreamBackendState, certificate: Optional[Path] = None, private_key: Optional[Path] = None) -> None:
        """
        Bind a TCP fixture endpoint and optionally configure server-side TLS.
        """
        self.state = state
        self.listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.listener.bind((address, port))
        self.listener.listen(64)
        self.listener.settimeout(0.2)
        self.stopping = threading.Event()
        self.client_lock = threading.Lock()
        self.client_threads: List[threading.Thread] = []
        self.context = None
        if certificate is not None and private_key is not None:
            self.context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            self.context.load_cert_chain(str(certificate), str(private_key))
            self.context.set_servername_callback(self._record_sni)
        self.thread = threading.Thread(target=self._serve, daemon=True)
        self.thread.start()

    def _record_sni(self, connection: Any, server_name: Optional[str], context: ssl.SSLContext) -> None:
        with self.state.lock:
            self.state.sni.append(server_name)

    def _serve(self) -> None:
        while not self.stopping.is_set():
            try:
                connection, _ = self.listener.accept()
            except socket.timeout:
                continue
            except OSError:
                return
            thread = threading.Thread(target=self._handle, args=(connection,), daemon=False)
            with self.client_lock:
                self.client_threads.append(thread)
            thread.start()

    def _handle(self, connection: socket.socket) -> None:
        try:
            connection.settimeout(2)
            if self.context is not None:
                connection = self.context.wrap_socket(connection, server_side=True)
            if self.state.banner:
                self.state.observe(b"")
                connection.sendall(self.state.banner)
                return
            if self.state.hold_open:
                self.state.observe(b"")
                self.state.gate.wait(timeout=5)
                return
            expected_length = len(self.state.probe_send)
            payload = bytearray()
            while expected_length == 0 or len(payload) < expected_length:
                try:
                    chunk = connection.recv(65536)
                except socket.timeout:
                    break
                if not chunk:
                    break
                payload.extend(chunk)
                if expected_length == 0:
                    break
            response, delay, chunks, chunk_delay, repeat, pause = self.state.observe(bytes(payload))
            if delay:
                time.sleep(delay)
            if response:
                try:
                    connection.sendall(response)
                except OSError:
                    return
            if pause and response:
                self.state.first_chunk_sent.set()
                self.state.gate.wait(timeout=30)
            for chunk in chunks:
                if chunk_delay:
                    time.sleep(chunk_delay)
                try:
                    connection.sendall(chunk)
                except OSError:
                    return
            while repeat:
                try:
                    connection.sendall(repeat)
                except OSError:
                    return
                time.sleep(self.state.repeat_delay)
        except (OSError, ssl.SSLError):
            return
        finally:
            try:
                connection.close()
            except OSError:
                pass

    def close(self) -> None:
        """
        Stop accepting and join the server and every connection thread.
        """
        self.state.gate.set()
        self.stopping.set()
        self.listener.close()
        self.thread.join(timeout=1)
        if self.thread.is_alive():
            raise VerificationFailure("stream fixture accept thread did not stop")
        for thread in self.client_threads:
            thread.join(timeout=3)
        alive = [thread.name for thread in self.client_threads if thread.is_alive()]
        if alive:
            raise VerificationFailure(f"stream fixture connections did not stop: {alive}")


def _start_stream_fixture(servers: List[FixtureStreamServer], ports: Dict[str, int], name: str, address: str, state: StreamBackendState, port: Optional[int] = None, certificate: Optional[Path] = None, private_key: Optional[Path] = None) -> int:
    """
    Bind one stream fixture and retain its server and selected port.
    """
    selected_port = port if port is not None else free_port(address)
    servers.append(FixtureStreamServer(address, selected_port, state, certificate, private_key))
    ports[name] = selected_port
    return selected_port


class Nginx:
    """
    Manage a foreground nginx master with a temporary prefix.
    """
    def __init__(self, binary: Path, prefix: Path, config: Path, log_path: Path) -> None:
        """
        Store the binary, runtime prefix, configuration, and log path.
        """
        self.binary = str(binary)
        self.prefix = prefix
        self.config = config
        self.log_path = log_path
        self.process = None

    def command(self, *arguments: str, timeout: float = 5) -> subprocess.CompletedProcess:
        """
        Run a bounded nginx control or validation command.
        """
        return subprocess.run(
            [self.binary, "-p", str(self.prefix), "-c", str(self.config), *arguments],
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            timeout=timeout,
            check=False,
        )

    def start(self) -> None:
        """
        Start nginx and wait for its loopback metrics endpoint.
        """
        self.log_file = self.log_path.open("ab", buffering=0)
        self.process = subprocess.Popen(
            [self.binary, "-p", str(self.prefix), "-c", str(self.config)],
            stdout=self.log_file,
            stderr=subprocess.STDOUT,
            close_fds=True,
        )
        wait_until("nginx HTTP listener becomes ready", lambda: self.process.poll() is None and self.request("/_healthcheck_metrics")[0] == 200, 8)

    def request(self, port: Union[int, str], path: Optional[str] = None) -> Tuple[int, str, List[Tuple[str, str]]]:
        """
        Issue one bounded HTTP request to nginx or a supplied port.
        """
        if path is None:
            path = port
            port = self.http_port
        connection = http.client.HTTPConnection("127.0.0.1", port, timeout=1.5)
        try:
            connection.request("GET", path)
            response = connection.getresponse()
            body = response.read()
            return response.status, body.decode("utf-8", errors="replace"), response.getheaders()
        finally:
            connection.close()

    def metrics(self) -> Dict[str, Any]:
        """
        Fetch and parse the native Prometheus healthcheck endpoint.
        """
        status, body, _ = self.request("/_healthcheck_metrics")
        if status != 200:
            raise VerificationFailure(f"native health metrics endpoint returned HTTP {status}")
        return parse_prometheus(body)

    def children(self) -> List[int]:
        """
        Return current child process identifiers for the nginx master.
        """
        path = Path(f"/proc/{self.process.pid}/task/{self.process.pid}/children")
        try:
            return [int(value) for value in path.read_text().split()]
        except (FileNotFoundError, ProcessLookupError):
            return []

    def close(self, require_graceful: bool = False) -> bool:
        """
        Stop nginx, optionally require a clean QUIT, and reject leftover workers.
        """
        if self.process is None:
            return True
        worker_pids = set(self.children())
        graceful = self.process.poll() is None
        if self.process.poll() is None:
            try:
                quit_result = self.command("-s", "quit", timeout=4)
                graceful = quit_result.returncode == 0
            except subprocess.SubprocessError:
                graceful = False
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                graceful = False
                self.process.send_signal(signal.SIGTERM)
                try:
                    self.process.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    graceful = False
                    self.process.kill()
                    self.process.wait(timeout=2)
        if self.process.returncode != 0:
            graceful = False
        self.log_file.close()
        for pid in worker_pids:
            try:
                os.kill(pid, 0)
            except ProcessLookupError:
                continue
            except PermissionError as error:
                raise VerificationFailure(f"could not verify nginx worker {pid} stopped: {error}") from error
            raise VerificationFailure(f"nginx worker {pid} remained after graceful shutdown")
        remaining_workers = self.children()
        if remaining_workers:
            raise VerificationFailure(f"nginx workers remained after shutdown: {remaining_workers}")
        if require_graceful and not graceful:
            raise VerificationFailure("nginx required forced termination or exited unsuccessfully instead of stopping on QUIT")
        return graceful


def wait_until(label: str, predicate: Callable[[], Any], timeout: float) -> Any:
    """
    Poll an assertion until it succeeds or its wall-clock bound expires.
    """
    deadline = time.monotonic() + timeout
    last_error = None
    while time.monotonic() < deadline:
        try:
            result = predicate()
            if result:
                print(f"PASS {label}", flush=True)
                return result
        except (OSError, http.client.HTTPException, VerificationFailure) as error:
            last_error = error
        time.sleep(0.035)
    detail = f": {last_error}" if last_error else ""
    raise VerificationFailure(f"timed out waiting for {label}{detail}")


def _print_ipv6_diagnostics(dns: Optional[AuthoritativeDNS], nginx: Optional[Nginx], state: Optional[BackendState]) -> None:
    """
    Print bounded resolver, peer-metric, and fixture evidence after a failed IPv6 assertion.
    """
    if dns is not None:
        print(
            "IPV6_DIAGNOSTIC dns_queries="
            f"A:{dns.query_count('ipv6.fixture.test', 1)} "
            f"AAAA:{dns.query_count('ipv6.fixture.test', 28)}",
            flush=True,
        )
    if state is not None:
        with state.lock:
            readiness_requests = state.paths["/readyz"]
        print(f"IPV6_DIAGNOSTIC fixture_readiness_requests={readiness_requests}", flush=True)
    if nginx is None or nginx.process is None or nginx.process.poll() is not None:
        print("IPV6_DIAGNOSTIC metrics=unavailable nginx_not_running", flush=True)
        return
    try:
        status, body, _ = nginx.request("/_healthcheck_metrics")
    except (OSError, http.client.HTTPException, VerificationFailure) as error:
        print(f"IPV6_DIAGNOSTIC metrics=unavailable error={error}", flush=True)
        return
    if status != 200:
        print(f"IPV6_DIAGNOSTIC metrics=unavailable status={status}", flush=True)
        return
    selected = [
        line
        for line in body.splitlines()
        if line.startswith((
            "nginx_healthcheck_peer_up{",
            "nginx_healthcheck_check_status{",
            "nginx_healthcheck_check_code{",
            "nginx_healthcheck_checks_total{",
            "nginx_healthcheck_check_failures_total{",
            "nginx_healthcheck_check_duration_seconds{",
            "nginx_healthcheck_last_check_timestamp_seconds{",
            "nginx_healthcheck_errors_total{",
        ))
        and 'upstream="ipv6"' in line
    ]
    print("IPV6_DIAGNOSTIC metrics_begin", flush=True)
    for line in selected[:20]:
        print(f"IPV6_DIAGNOSTIC {line[:500]}", flush=True)
    if not selected:
        print("IPV6_DIAGNOSTIC no_ipv6_peer_metric_lines", flush=True)
    print("IPV6_DIAGNOSTIC metrics_end", flush=True)


def parse_labels(value: str) -> Dict[str, str]:
    """
    Parse Prometheus labels while respecting escaped quoted values.
    """
    labels = {}
    current = []
    quoted = False
    escaped = False
    parts = []
    for character in value:
        if escaped:
            current.append(character)
            escaped = False
        elif quoted and character == "\\":
            current.append(character)
            escaped = True
        elif character == '"':
            current.append(character)
            quoted = not quoted
        elif character == "," and not quoted:
            parts.append("".join(current))
            current = []
        else:
            current.append(character)
    if current:
        parts.append("".join(current))
    for part in parts:
        key, separator, raw_value = part.partition("=")
        if not separator or key in labels:
            raise VerificationFailure(f"invalid or duplicate Prometheus label: {part}")
        labels[key.strip()] = json.loads(raw_value.strip())
    return labels


def parse_prometheus(body: str) -> Dict[str, Any]:
    """
    Parse metric metadata and samples while validating line structure.
    """
    samples = []
    help_lines = set()
    types = {}
    sample_pattern = re.compile(r"^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{(.*)\})?\s+([-+]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][-+]?\d+)?|NaN|[+-]Inf)(?:\s+\d+)?$")
    for line in body.splitlines():
        if not line:
            continue
        if line.startswith("# HELP "):
            metric = line.split(None, 3)[2]
            if metric in help_lines:
                raise VerificationFailure(f"duplicate Prometheus HELP line for {metric}")
            help_lines.add(metric)
            continue
        if line.startswith("# TYPE "):
            fields = line.split()
            if len(fields) != 4 or fields[2] in types:
                raise VerificationFailure(f"invalid or duplicate Prometheus TYPE line: {line}")
            types[fields[2]] = fields[3]
            continue
        if line.startswith("#"):
            continue
        match = sample_pattern.match(line)
        if match is None:
            raise VerificationFailure(f"invalid Prometheus sample: {line}")
        name, raw_labels, raw_value = match.groups()
        value = float(raw_value)
        if not math.isfinite(value):
            raise VerificationFailure(f"Prometheus sample {name} is not finite")
        labels = parse_labels(raw_labels) if raw_labels else {}
        if types.get(name) == "counter" and value < 0:
            raise VerificationFailure(f"counter {name} has a negative value")
        if name.endswith("_seconds") and not name.endswith("_timestamp_seconds") and value < 0:
            raise VerificationFailure(f"duration {name} has a negative value")
        samples.append((name, labels, value))
    return {"samples": samples, "types": types, "help": help_lines}


def samples_for(metrics: Dict[str, Any], name: str, **labels: str) -> List[MetricSample]:
    """
    Select samples by metric name and a subset of label values.
    """
    return [sample for sample in metrics["samples"] if sample[0] == name and all(sample[1].get(key) == value for key, value in labels.items())]


def peer_sample(metrics: Dict[str, Any], name: str, upstream: str, peer: str) -> MetricSample:
    """
    Find the unique sample for a resolved peer in one upstream group.
    """
    peer = str(peer)
    matches = [sample for sample in samples_for(metrics, name, upstream=upstream) if peer in sample[1].get("peer", "")]
    if len(matches) != 1:
        raise VerificationFailure(f"expected one {name} sample for {upstream}/{peer}, found {len(matches)}")
    return matches[0]


def peer_value_from(metrics: Dict[str, Any], name: str, upstream: str, peer: str) -> float:
    """
    Return the numeric sample value for an upstream peer.
    """
    return peer_sample(metrics, name, upstream, peer)[2]


def peer_value(nginx: Nginx, name: str, upstream: str, peer: str) -> float:
    """
    Fetch one native metric and return a peer's numeric value.
    """
    return peer_sample(nginx.metrics(), name, upstream, peer)[2]


def group_value(nginx: Nginx, name: str, upstream: str) -> float:
    """
    Fetch one native metric and return an upstream group's numeric value.
    """
    matches = samples_for(nginx.metrics(), name, upstream=upstream)
    if len(matches) != 1:
        raise VerificationFailure(f"expected one {name} sample for upstream {upstream}, found {len(matches)}")
    return matches[0][2]


def wait_peer(nginx: Nginx, upstream: str, peer: str, up: bool, timeout: float = 8) -> Any:
    """
    Wait for a peer's active health state to match the requested value.
    """
    return wait_until(
        f"{upstream} peer {peer} becomes {'up' if up else 'down'}",
        lambda: peer_value(nginx, "nginx_healthcheck_peer_up", upstream, peer) == (1 if up else 0),
        timeout,
    )


def wait_failed_peer(nginx: Nginx, upstream: str, peer: str, timeout: float = 8) -> Any:
    """
    Require a completed failed probe before accepting a down peer state.
    """
    def _failed() -> bool:
        """
        Check that metrics describe a completed failed probe and down state.
        """
        metrics = nginx.metrics()
        return (
            peer_value_from(metrics, "nginx_healthcheck_check_failures_total", upstream, peer) >= 1
            and peer_value_from(metrics, "nginx_healthcheck_last_check_timestamp_seconds", upstream, peer) > 0
            and peer_value_from(metrics, "nginx_healthcheck_check_status", upstream, peer) == 0
            and peer_value_from(metrics, "nginx_healthcheck_peer_up", upstream, peer) == 0
        )
    return wait_until(f"{upstream} peer {peer} completes a failed probe and remains down", _failed, timeout)


def wait_stream_peer(nginx: Nginx, upstream: str, peer: str, up: bool, timeout: float = 8) -> Any:
    """
    Wait for a stream peer's active health state to match the requested value.
    """
    return wait_until(
        f"stream {upstream} peer {peer} becomes {'up' if up else 'down'}",
        lambda: peer_value(nginx, "nginx_stream_healthcheck_peer_up", upstream, peer) == (1 if up else 0),
        timeout,
    )


def wait_failed_stream_peer(nginx: Nginx, upstream: str, peer: str, timeout: float = 8) -> Any:
    """
    Require a completed failed stream probe before accepting a down state.
    """
    def _failed() -> bool:
        """
        Check that stream metrics describe a completed failure and down state.
        """
        metrics = nginx.metrics()
        return (
            peer_value_from(metrics, "nginx_stream_healthcheck_check_failures_total", upstream, peer) >= 1
            and peer_value_from(metrics, "nginx_stream_healthcheck_last_check_timestamp_seconds", upstream, peer) > 0
            and peer_value_from(metrics, "nginx_stream_healthcheck_check_status", upstream, peer) == 0
            and peer_value_from(metrics, "nginx_stream_healthcheck_peer_up", upstream, peer) == 0
        )
    return wait_until(f"stream {upstream} peer {peer} completes a failed check and remains down", _failed, timeout)


def nginx_quote(value: Union[str, Path]) -> str:
    """
    Quote a filesystem path for an nginx string directive.
    """
    return '"' + str(value).replace("\\", "\\\\").replace('"', '\\"') + '"'


def upstream(name: str, peers: Iterable[str], healthcheck: str, extra: str = "", zone_name: Optional[str] = None) -> str:
    """
    Render one healthchecked upstream block for a temporary config.
    """
    selected_zone = zone_name or name
    return f"upstream {name} {{\n    zone {selected_zone} 128k;\n" + "".join(f"    server {peer};\n" for peer in peers) + f"    {healthcheck}\n" + (f"    {extra}\n" if extra else "") + "}\n"


def checked_stream(probe_type: str = "tcp", interval: str = "500ms", timeout: str = "300ms", fall: int = 1, rise: int = 1, max_response: int = 4096) -> str:
    """
    Render a bounded native stream probe directive.
    """
    return f"healthcheck_tcp type={probe_type} interval={interval} timeout={timeout} fall={fall} rise={rise} concurrency=16 shm_size=1m max_response={max_response};"


def stream_request(port: int, payload: bytes, timeout: float = 3) -> bytes:
    """
    Send one bounded client payload through a local nginx stream listener.
    """
    response = bytearray()
    with socket.create_connection(("127.0.0.1", port), timeout=timeout) as connection:
        connection.settimeout(timeout)
        connection.sendall(payload)
        while True:
            chunk = connection.recv(4096)
            if not chunk:
                break
            response.extend(chunk)
    return bytes(response)


def checked_http(interval: str = "500ms", timeout: str = "300ms", fall: int = 3, rise: int = 2, host: str = "healthcheck.example.test") -> str:
    """
    Render the verifier's HTTP probe contract and accepted statuses.
    """
    return f"healthcheck type=http interval={interval} timeout={timeout} fall={fall} rise={rise} uri=/readyz host={host} concurrency=16 shm_size=1m;\n    healthcheck_statuses 200 204;"


def config_text(dns: int, http_port: int, runtime_port: int, tls_port: int, tls_certificate: Path, bad_certificate: Path, ipv6_port: int, tcp_port: int, stream_endpoint_ports: Dict[str, int], stream_listener_ports: Dict[str, int]) -> str:
    """
    Build the integration configuration for all fixture protocols.
    """
    config = f"""worker_processes 2;
master_process on;
daemon off;
error_log logs/error.log notice;
pid logs/nginx.pid;
events {{ worker_connections 1024; }}
http {{
    access_log off;
    resolver 127.0.0.1:{dns} ipv4=on ipv6=on valid=1s;
    resolver_timeout 1s;
    vhost_traffic_status_zone shared:vhost_traffic_status:1m;
"""
    config += upstream("primary", [f"primary.fixture.test:{http_port} resolve", f"backup.fixture.test:{http_port} resolve backup", f"127.0.0.9:{http_port} down"], checked_http())
    config += upstream("isolated", [f"isolated.fixture.test:{http_port} resolve"], checked_http(fall=1, rise=1), zone_name="shared_health")
    config += upstream("single", [f"single.fixture.test:{http_port} resolve"], checked_http(fall=1, rise=1), zone_name="shared_health")
    config += upstream("refused", [f"127.0.0.14:{http_port}"], checked_http(fall=1, rise=1))
    config += upstream("malformed", [f"malformed.fixture.test:{http_port} resolve"], checked_http(fall=1, rise=1))
    config += upstream("stalled", [f"stalled.fixture.test:{http_port} resolve"], checked_http(timeout="200ms", fall=1, rise=1))
    config += upstream("ipv6", [f"ipv6.fixture.test:{ipv6_port} resolve"], checked_http(fall=1, rise=1))
    config += upstream("srv", ["srv.fixture.test service=http resolve"], checked_http(fall=1, rise=1))
    config += upstream("srv_admin", ["srv-admin.fixture.test service=http resolve down"], checked_http(fall=1, rise=1))
    config += upstream("tcp", [f"127.0.0.1:{tcp_port}"], "healthcheck type=tcp interval=500ms timeout=300ms fall=1 rise=1 concurrency=16 shm_size=1m;")
    config += upstream("least", [f"primary.fixture.test:{http_port} resolve", f"backup.fixture.test:{http_port} resolve"], checked_http(fall=1, rise=1), "least_conn;")
    config += upstream("hash", [f"primary.fixture.test:{http_port} resolve", f"backup.fixture.test:{http_port} resolve"], checked_http(fall=1, rise=1), "hash $request_uri consistent;")
    config += upstream("random", [f"primary.fixture.test:{http_port} resolve", f"backup.fixture.test:{http_port} resolve"], checked_http(fall=1, rise=1), "random two least_conn;")
    config += upstream("keepalive", [f"primary.fixture.test:{http_port} resolve", f"backup.fixture.test:{http_port} resolve"], checked_http(fall=1, rise=1), "keepalive 8;")
    for name, tls_name, ca in (("tls_good", "backend.example.test", tls_certificate), ("tls_bad_sni", "wrong.example.test", tls_certificate), ("tls_bad_ca", "backend.example.test", bad_certificate)):
        tls_check = checked_http(fall=1, rise=1).replace("type=http", "type=https")
        config += upstream(name, [f"tls.fixture.test:{tls_port} resolve"], tls_check, f"healthcheck_tls_name {tls_name};\n    healthcheck_trusted_certificate {nginx_quote(ca)};")
    churn_check = checked_http(interval="1s", timeout="30s", fall=3, rise=2)
    config += upstream("churn", [f"churn.fixture.test:{http_port} resolve"], churn_check)
    config += f"""    server {{
        listen 127.0.0.1:{runtime_port};
        server_name _;
        location = /_healthcheck_metrics {{ allow 127.0.0.1; deny all; healthcheck_metrics; }}
        location = /_vts_metrics {{ allow 127.0.0.1; deny all; vhost_traffic_status_display; vhost_traffic_status_display_format prometheus; }}
        location = /route {{ proxy_next_upstream off; proxy_set_header Host $http_host; proxy_pass http://primary; }}
        location = /single {{ proxy_next_upstream off; proxy_pass http://single; }}
        location = /isolated {{ proxy_next_upstream off; proxy_pass http://isolated; }}
        location = /srv {{ proxy_next_upstream off; proxy_pass http://srv; }}
        location = /least {{ proxy_next_upstream off; proxy_pass http://least; }}
        location = /hash {{ proxy_next_upstream off; proxy_pass http://hash; }}
        location = /random {{ proxy_next_upstream off; proxy_pass http://random; }}
        location = /keepalive {{ proxy_next_upstream off; proxy_pass http://keepalive; }}
        location = /worker {{ add_header X-Nginx-Worker $pid; return 200 $pid; }}
        location = / {{ return 200 "fixture\\n"; }}
    }}
}}
"""
    config += f"""stream {{
    resolver 127.0.0.1:{dns} ipv4=on ipv6=on valid=1s;
    resolver_timeout 1s;
"""
    config += upstream("stream_dynamic", [f"stream-dynamic.fixture.test:{stream_endpoint_ports['dynamic']} resolve"], checked_stream(rise=2))
    config += upstream(
        "stream_failover",
        [
            f"127.0.0.17:{stream_endpoint_ports['failover_a']}",
            f"127.0.0.18:{stream_endpoint_ports['failover_b']}",
            f"127.0.0.19:{stream_endpoint_ports['failover_backup']} backup",
            f"127.0.0.27:{stream_endpoint_ports['failover_admin']} down",
        ],
        checked_stream(),
        'healthcheck_send "PING\\r\\n";\n    healthcheck_expect "PONG" min_recv=4;',
    )
    config += upstream(
        "stream_text",
        [f"127.0.0.20:{stream_endpoint_ports['text']}"],
        checked_stream(),
        'healthcheck_send "PING\\r\\n";\n    healthcheck_expect "PONG" min_recv=4;',
    )
    config += upstream(
        "stream_send_only",
        [f"127.0.0.28:{stream_endpoint_ports['send_only']}"],
        checked_stream(),
        'healthcheck_send "PING\\r\\n";',
    )
    config += upstream(
        "stream_banner",
        [f"127.0.0.29:{stream_endpoint_ports['banner']}"],
        checked_stream(),
        'healthcheck_expect "READY" min_recv=5;',
    )
    config += upstream(
        "stream_binary",
        [f"127.0.0.21:{stream_endpoint_ports['binary']}"],
        checked_stream(),
        "healthcheck_send_hex 0050494E47FF;\n    healthcheck_expect_hex 504F004E47 min_recv=5;",
    )
    config += upstream(
        "stream_fragmented",
        [f"127.0.0.30:{stream_endpoint_ports['fragmented']}"],
        checked_stream(timeout="20s"),
        'healthcheck_send "PING\\r\\n";\n    healthcheck_expect "PONG" min_recv=8;',
    )
    config += upstream(
        "stream_incomplete",
        [f"127.0.0.22:{stream_endpoint_ports['incomplete']}"],
        checked_stream(),
        'healthcheck_send "PING\\r\\n";\n    healthcheck_expect "PONG" min_recv=8;',
    )
    config += upstream(
        "stream_timeout",
        [f"127.0.0.23:{stream_endpoint_ports['timeout']}"],
        checked_stream(timeout="200ms"),
        'healthcheck_send "PING\\r\\n";\n    healthcheck_expect "PONG" min_recv=4;',
    )
    config += upstream(
        "stream_unmatched",
        [f"127.0.0.32:{stream_endpoint_ports['unmatched']}"],
        checked_stream(timeout="1s", max_response=128),
        'healthcheck_send "PING\\r\\n";\n    healthcheck_expect "PONG" min_recv=4;',
    )
    config += upstream(
        "stream_stalled",
        [f"127.0.0.33:{stream_endpoint_ports['stalled']}"],
        checked_stream(timeout="200ms", max_response=64),
        'healthcheck_expect "READY" min_recv=5;',
    )
    config += upstream("stream_refused", [f"127.0.0.34:{stream_endpoint_ports['refused']}"], checked_stream())
    config += upstream(
        "stream_rise",
        [f"127.0.0.26:{stream_endpoint_ports['rise']}"],
        checked_stream(fall=3, rise=2),
        'healthcheck_send "PING\\r\\n";\n    healthcheck_expect "PONG" min_recv=4;',
    )
    for name, tls_name, ca in (
        ("stream_tls_good", "backend.example.test", tls_certificate),
        ("stream_tls_bad_sni", "wrong.example.test", tls_certificate),
        ("stream_tls_bad_ca", "backend.example.test", bad_certificate),
    ):
        stream_tls_check = checked_stream(probe_type="tls")
        stream_tls_options = (
            f"healthcheck_tls_name {tls_name};\n"
            f"    healthcheck_trusted_certificate {nginx_quote(ca)};\n"
            '    healthcheck_send "PING\\r\\n";\n'
            '    healthcheck_expect "PONG" min_recv=4;'
        )
        config += upstream(
            name,
            [f"127.0.0.25:{stream_endpoint_ports['tls']}"],
            stream_tls_check,
            stream_tls_options,
        )
    for listener_name, upstream_name in (("dynamic", "stream_dynamic"), ("failover", "stream_failover")):
        config += (
            f"    server {{ listen 127.0.0.1:{stream_listener_ports[listener_name]}; "
            f"proxy_connect_timeout 1s; proxy_timeout 2s; proxy_pass {upstream_name}; }}\n"
        )
    config += "}\n"
    return config


def generate_certificate(root: Path, name: str) -> Tuple[Path, Path]:
    """
    Create a one-day self-signed certificate with a DNS subject name.
    """
    key = root / f"{name}.key"
    certificate = root / f"{name}.pem"
    subprocess.run(
        ["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-keyout", str(key), "-out", str(certificate), "-days", "1", "-subj", f"/CN={name}.example.test", "-addext", f"subjectAltName=DNS:{name}.example.test"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=True,
        timeout=15,
    )
    return certificate, key


def negative_config(binary: Path, prefix: Path, content: str, label: str, expected_message: str) -> None:
    """
    Confirm nginx rejects one deliberately invalid configuration.
    """
    path = prefix / f"negative-{label}.conf"
    path.write_text(content)
    result = subprocess.run([str(binary), "-t", "-p", str(prefix), "-c", str(path)], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=5, check=False)
    require(result.returncode != 0 and expected_message.lower() in result.stdout.lower(), f"nginx rejects {label} with its expected configuration diagnostic")


def verify_negative_configs(binary: Path, prefix: Path, dns_port: int, port: int, certificate: Path) -> None:
    """
    Reject invalid zones, thresholds, types, statuses, headers, and TLS settings.
    """
    base = f"events {{}}\nhttp {{ resolver 127.0.0.1:{dns_port};\n"
    cases = {
        "unshared-zone": (
            f"upstream bad {{ server 127.0.0.2:{port}; healthcheck type=http; }}\n",
            "requires an upstream zone",
        ),
        "zero-fall": (
            f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=http fall=0; }}\n",
            "invalid healthcheck parameter",
        ),
        "unknown-type": (
            f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=udp; }}\n",
            "invalid healthcheck parameter",
        ),
        "invalid-status": (
            f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=http; healthcheck_statuses 99; }}\n",
            "healthcheck status must be between 200 and 599",
        ),
        "duplicate-status": (
            f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=http; healthcheck_statuses 200 200; }}\n",
            "duplicate healthcheck status",
        ),
        "crlf-uri": (
            f'upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=http uri="/readyz\\r\\nInjected: value"; }}\n',
            "invalid healthcheck parameter",
        ),
        "crlf-host": (
            f'upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=http host="healthcheck.example.test\\r\\nInjected: value"; }}\n',
            "invalid healthcheck parameter",
        ),
        "crlf-tls-name": (
            f'upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=https; healthcheck_tls_name "backend.example.test\\r\\nInjected: value"; healthcheck_trusted_certificate {nginx_quote(certificate)}; }}\n',
            "invalid healthcheck_tls_name",
        ),
        "https-no-ca": (
            f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=https; healthcheck_tls_name backend.example.test; }}\n",
            "requires healthcheck_tls_name and healthcheck_trusted_certificate",
        ),
        "https-no-sni": (
            f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck type=https; healthcheck_trusted_certificate {nginx_quote(certificate)}; }}\n",
            "requires healthcheck_tls_name and healthcheck_trusted_certificate",
        ),
        "duplicate-server": (
            f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; server 127.0.0.2:{port}; healthcheck type=http; }}\n",
            "duplicate server address",
        ),
    }
    for label, (block, expected_message) in cases.items():
        negative_config(binary, prefix, base + block + "}\n", label, expected_message)
    stream_base = f"events {{}}\nstream {{ resolver 127.0.0.1:{dns_port};\n"
    stream_cases = {
        "stream-send-hex-invalid-g": (
            "healthcheck_send_hex 0G;",
            "test failed",
        ),
        "stream-send-hex-invalid-bracket": (
            "healthcheck_send_hex 0[;",
            "test failed",
        ),
        "stream-expect-hex-invalid-g": (
            "healthcheck_expect_hex 0G;",
            "test failed",
        ),
        "stream-expect-hex-invalid-bracket": (
            "healthcheck_expect_hex 0[;",
            "test failed",
        ),
    }
    for label, (directive, expected_message) in stream_cases.items():
        block = f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck_tcp type=tcp; {directive} }}\n"
        negative_config(binary, prefix, stream_base + block + "}\n", label, expected_message)
    for label, tls_options in (
        ("stream-tls-no-ca", f"healthcheck_tls_name backend.example.test;"),
        ("stream-tls-no-sni", f"healthcheck_trusted_certificate {nginx_quote(certificate)};"),
    ):
        block = f"upstream bad {{ zone bad 64k; server 127.0.0.2:{port}; healthcheck_tcp type=tls; {tls_options} }}\n"
        negative_config(
            binary,
            prefix,
            stream_base + block + "}\n",
            label,
            "requires healthcheck_tls_name and healthcheck_trusted_certificate",
        )


def verify_stream_runtime(nginx: Nginx, dns: AuthoritativeDNS, states: Dict[str, StreamBackendState], endpoint_ports: Dict[str, int], listener_ports: Dict[str, int]) -> None:
    """
    Verify stream probe modes, thresholds, DNS selection, TLS, and routing.
    """
    fragmented_peer = f"127.0.0.30:{endpoint_ports['fragmented']}"
    wait_until("fragmented stream response sends its unmatched first chunk", states["fragmented"].first_chunk_sent.is_set, 4)

    def _fragment_waiting() -> Any:
        """
        Observe a fragmented response that has not met its match and byte count.
        """
        metrics = nginx.metrics()
        pending = (
            peer_value_from(metrics, "nginx_stream_healthcheck_check_status", "stream_fragmented", fragmented_peer) == -1
            and peer_value_from(metrics, "nginx_stream_healthcheck_peer_up", "stream_fragmented", fragmented_peer) == 0
            and peer_value_from(metrics, "nginx_stream_healthcheck_checks_total", "stream_fragmented", fragmented_peer) == 0
            and group_value(nginx, "nginx_stream_healthcheck_active_probes", "stream_fragmented") >= 1
        )
        return metrics if pending else False

    wait_until("fragmented expectation remains in flight before its second chunk", _fragment_waiting, 3)
    states["fragmented"].gate.set()
    wait_stream_peer(nginx, "stream_fragmented", fragmented_peer, True, 4)
    require(peer_value(nginx, "nginx_stream_healthcheck_check_status", "stream_fragmented", fragmented_peer) == 1, "fragmented response matches across read boundaries after min_recv")
    require(peer_value(nginx, "nginx_stream_healthcheck_checks_total", "stream_fragmented", fragmented_peer) >= 1, "fragmented stream result completes only after the second response chunk arrives")

    dynamic_peers = (f"127.0.0.15:{endpoint_ports['dynamic']}", f"127.0.0.16:{endpoint_ports['dynamic']}")
    for peer in dynamic_peers:
        wait_stream_peer(nginx, "stream_dynamic", peer, True)
    require(dns.query_count("stream-dynamic.fixture.test", 1) > 0, "stream resolver uses local authoritative A records")
    identities = set()
    for _ in range(24):
        identity = stream_request(listener_ports["dynamic"], b"CLIENT\n", timeout=2).decode("ascii", errors="replace")
        identities.add(identity)
        if {"dynamic-a", "dynamic-b"}.issubset(identities):
            break
    require({"dynamic-a", "dynamic-b"}.issubset(identities), f"stream DNS peers are both selected for live client traffic (observed: {sorted(identities)})")

    dynamic_a_peer = dynamic_peers[0]
    dynamic_b_peer = dynamic_peers[1]
    queries_before_removal = dns.query_count("stream-dynamic.fixture.test", 1)
    dns.set_addresses("stream-dynamic.fixture.test", ["127.0.0.16"])
    wait_until("stream resolver queries the reduced A record set", lambda: dns.query_count("stream-dynamic.fixture.test", 1) > queries_before_removal, 5)

    def _dynamic_a_removed() -> Any:
        """
        Detect A-peer removal while the other resolved address stays healthy.
        """
        metrics = nginx.metrics()
        peers = samples_for(metrics, "nginx_stream_healthcheck_peer_up", upstream="stream_dynamic")
        remaining = [sample for sample in peers if sample[1].get("peer") == dynamic_b_peer]
        return metrics if all(sample[1].get("peer") != dynamic_a_peer for sample in peers) and len(remaining) == 1 and remaining[0][2] == 1 else False

    wait_until("stream metrics remove the withdrawn A peer", _dynamic_a_removed, 5)
    post_removal_identities = {
        stream_request(listener_ports["dynamic"], b"CLIENT\n", timeout=2).decode("ascii", errors="replace")
        for _ in range(6)
    }
    require(post_removal_identities == {"dynamic-b"}, f"stream traffic uses only the remaining A peer (observed: {sorted(post_removal_identities)})")
    queries_before_readd = dns.query_count("stream-dynamic.fixture.test", 1)
    dns.set_addresses("stream-dynamic.fixture.test", ["127.0.0.15", "127.0.0.16"])
    wait_until("stream resolver queries the restored A record set", lambda: dns.query_count("stream-dynamic.fixture.test", 1) > queries_before_readd, 5)

    def _dynamic_a_first_success_pending() -> Any:
        """
        Observe a re-added stream peer after one fresh success but before rise.
        """
        metrics = nginx.metrics()
        try:
            pending = (
                peer_value_from(metrics, "nginx_stream_healthcheck_peer_up", "stream_dynamic", dynamic_a_peer) == 0
                and peer_value_from(metrics, "nginx_stream_healthcheck_check_status", "stream_dynamic", dynamic_a_peer) == 1
                and peer_value_from(metrics, "nginx_stream_healthcheck_checks_total", "stream_dynamic", dynamic_a_peer) == 1
                and peer_value_from(metrics, "nginx_stream_healthcheck_check_failures_total", "stream_dynamic", dynamic_a_peer) == 0
                and peer_value_from(metrics, "nginx_stream_healthcheck_rise_streak", "stream_dynamic", dynamic_a_peer) == 1
            )
        except VerificationFailure:
            return False
        return metrics if pending else False

    wait_until("re-added stream A peer starts fresh and remains pending at rise one", _dynamic_a_first_success_pending, 6)
    wait_stream_peer(nginx, "stream_dynamic", dynamic_a_peer, True, 6)
    readded_identity_seen = False
    for _ in range(16):
        identity = stream_request(listener_ports["dynamic"], b"CLIENT\n", timeout=2).decode("ascii", errors="replace")
        readded_identity_seen = readded_identity_seen or identity == "dynamic-a"
        if readded_identity_seen:
            break
    require(readded_identity_seen, "re-added stream A peer regains live traffic after its fresh rise threshold")

    failover_peers = {
        "failover_a": "127.0.0.17",
        "failover_b": "127.0.0.18",
        "failover_backup": "127.0.0.19",
    }
    for name, address in failover_peers.items():
        wait_stream_peer(nginx, "stream_failover", f"{address}:{endpoint_ports[name]}", True)
    down_peer = f"127.0.0.27:{endpoint_ports['failover_admin']}"
    admin_metrics = nginx.metrics()
    require(peer_value_from(admin_metrics, "nginx_stream_healthcheck_admin_down", "stream_failover", down_peer) == 1, "stream configured-down peer is marked administratively down")
    require(peer_value_from(admin_metrics, "nginx_stream_healthcheck_peer_up", "stream_failover", down_peer) == 0, "stream configured-down peer is excluded from readiness")
    require(peer_value_from(admin_metrics, "nginx_stream_healthcheck_check_status", "stream_failover", down_peer) == -1, "stream configured-down peer has no completed check")
    require(peer_value_from(admin_metrics, "nginx_stream_healthcheck_checks_total", "stream_failover", down_peer) == 0, "stream configured-down peer receives no health probe")
    require(states["failover_admin"].connections == 0, "healthy stream maintenance endpoint is never probed")
    require(peer_sample(admin_metrics, "nginx_stream_healthcheck_peer_up", "stream_failover", f"127.0.0.19:{endpoint_ports['failover_backup']}")[1]["backup"] == "true", "stream backup peer has its backup metric label")
    initial_traffic = stream_request(listener_ports["failover"], b"CLIENT\n", timeout=2)
    require(initial_traffic == b"failover-a", "stream client traffic initially selects a healthy primary")

    for name in ("failover_a", "failover_b"):
        states[name].set_probe_reply(b"BAD\r\n")
    for name, address in failover_peers.items():
        if name != "failover_backup":
            wait_failed_stream_peer(nginx, "stream_failover", f"{address}:{endpoint_ports[name]}")
    backup_traffic = stream_request(listener_ports["failover"], b"CLIENT\n", timeout=2)
    require(backup_traffic == b"failover-backup", "stream backup serves traffic after every primary fails its active exchange")
    require(states["failover_admin"].connections == 0, "stream traffic also excludes a healthy configured-down peer")
    states["failover_a"].set_probe_reply(b"PONG\r\n")
    failover_a_peer = f"127.0.0.17:{endpoint_ports['failover_a']}"
    wait_stream_peer(nginx, "stream_failover", failover_a_peer, True)
    recovered_traffic = stream_request(listener_ports["failover"], b"CLIENT\n", timeout=2)
    require(recovered_traffic == b"failover-a", "recovered stream primary regains preference over backup")

    for name, address in (
        ("text", "127.0.0.20"),
        ("send_only", "127.0.0.28"),
        ("banner", "127.0.0.29"),
        ("binary", "127.0.0.21"),
    ):
        wait_stream_peer(nginx, f"stream_{name}", f"{address}:{endpoint_ports[name]}", True)
    require(b"PING\r\n" in states["text"].payloads, "stream request-response check sends the configured text bytes")
    require(b"PING\r\n" in states["send_only"].payloads, "stream send-only mode passes after its exact request is sent")
    require(states["banner"].first_chunk_sent.is_set() or states["banner"].connections > 0, "stream expectation-only mode checks an unsolicited banner")
    require(b"\x00PING\xff" in states["binary"].payloads, "stream binary send preserves embedded NUL and high-bit bytes")
    require(states["binary"].probe_reply == b"PO\x00NG", "stream binary response fixture includes an embedded NUL in the expected token")

    failed_modes = (
        ("stream_incomplete", "127.0.0.22", "EOF before the expected stream response completes"),
        ("stream_timeout", "127.0.0.23", "stream response deadline expires before a delayed reply"),
        ("stream_unmatched", "127.0.0.32", "stream max_response bounds an endless unmatched body"),
        ("stream_stalled", "127.0.0.33", "stream response deadline expires on an open stalled connection"),
        ("stream_refused", "127.0.0.34", "stream connect-only probe records connection refusal"),
        ("stream_tls_bad_sni", "127.0.0.25", "stream TLS rejects an incorrect peer name"),
        ("stream_tls_bad_ca", "127.0.0.25", "stream TLS rejects an untrusted certificate"),
    )
    for upstream_name, address, assertion in failed_modes:
        peer = f"{address}:{endpoint_ports['tls'] if upstream_name.startswith('stream_tls') else endpoint_ports.get(upstream_name.removeprefix('stream_'), endpoint_ports.get('refused', 0))}"
        wait_failed_stream_peer(nginx, upstream_name, peer)
        require(peer_value(nginx, "nginx_stream_healthcheck_check_status", upstream_name, peer) == 0, assertion)
    require(peer_value(nginx, "nginx_stream_healthcheck_check_duration_seconds", "stream_timeout", f"127.0.0.23:{endpoint_ports['timeout']}") < 0.75, "stream timeout is bounded below the fixture's delayed response")
    require(peer_value(nginx, "nginx_stream_healthcheck_check_duration_seconds", "stream_unmatched", f"127.0.0.32:{endpoint_ports['unmatched']}") < 0.9, "stream max_response failure completes before its one-second deadline")
    require("backend.example.test" in states["tls"].sni, "verified stream TLS probe sends the configured SNI")
    require("wrong.example.test" in states["tls"].sni, "incorrect stream TLS SNI is observed and rejected")
    require(dns.query_count("stream-dynamic.fixture.test", 1) > 0, "stream health registry follows dynamic A resolution")

    rise_peer = f"127.0.0.26:{endpoint_ports['rise']}"
    wait_failed_stream_peer(nginx, "stream_rise", rise_peer, 5)
    wait_until(
        "initial stream failures reach the configured fall threshold without admission",
        lambda: peer_value(nginx, "nginx_stream_healthcheck_fall_streak", "stream_rise", rise_peer) >= 3
        and peer_value(nginx, "nginx_stream_healthcheck_peer_up", "stream_rise", rise_peer) == 0,
        5,
    )
    require(peer_value(nginx, "nginx_stream_healthcheck_peer_up", "stream_rise", rise_peer) == 0, "initial stream failures leave the never-admitted peer down")
    require(peer_value(nginx, "nginx_stream_healthcheck_check_up_down_total", "stream_rise", rise_peer) == 0, "initial stream admission does not increment the down-transition counter")
    states["rise"].set_probe_reply(b"PONG\r\n")
    wait_until(
        "first stream success remains pending below rise threshold",
        lambda: peer_value(nginx, "nginx_stream_healthcheck_check_status", "stream_rise", rise_peer) == 1
        and peer_value(nginx, "nginx_stream_healthcheck_rise_streak", "stream_rise", rise_peer) == 1
        and peer_value(nginx, "nginx_stream_healthcheck_peer_up", "stream_rise", rise_peer) == 0,
        5,
    )
    require(peer_value(nginx, "nginx_stream_healthcheck_check_up_down_total", "stream_rise", rise_peer) == 0, "first successful stream rise leaves the down-transition counter unchanged")
    wait_stream_peer(nginx, "stream_rise", rise_peer, True, 5)
    transition_count = peer_value(nginx, "nginx_stream_healthcheck_check_up_down_total", "stream_rise", rise_peer)
    require(transition_count == 0, "initial stream rise to ready does not count as an UP-to-DOWN transition")
    states["rise"].set_probe_reply(b"BAD\r\n")
    failure_count = peer_value(nginx, "nginx_stream_healthcheck_check_failures_total", "stream_rise", rise_peer)
    for streak in (1, 2):
        wait_until(
            f"stream fall threshold remains available at failure streak {streak}",
            lambda streak=streak: peer_value(nginx, "nginx_stream_healthcheck_check_failures_total", "stream_rise", rise_peer) == failure_count + streak
            and peer_value(nginx, "nginx_stream_healthcheck_fall_streak", "stream_rise", rise_peer) == streak
            and peer_value(nginx, "nginx_stream_healthcheck_peer_up", "stream_rise", rise_peer) == 1,
            4,
        )
    wait_failed_stream_peer(nginx, "stream_rise", rise_peer, 5)
    require(peer_value(nginx, "nginx_stream_healthcheck_fall_streak", "stream_rise", rise_peer) >= 3, "stream fall streak reaches its configured threshold")
    require(peer_value(nginx, "nginx_stream_healthcheck_check_up_down_total", "stream_rise", rise_peer) == 1, "stream UP-to-DOWN transition increments exactly once")
    states["rise"].set_probe_reply(b"PONG\r\n")
    wait_until(
        "first recovered stream result remains pending below rise threshold",
        lambda: peer_value(nginx, "nginx_stream_healthcheck_check_status", "stream_rise", rise_peer) == 1
        and peer_value(nginx, "nginx_stream_healthcheck_rise_streak", "stream_rise", rise_peer) == 1
        and peer_value(nginx, "nginx_stream_healthcheck_peer_up", "stream_rise", rise_peer) == 0,
        4,
    )
    require(peer_value(nginx, "nginx_stream_healthcheck_check_up_down_total", "stream_rise", rise_peer) == 1, "stream recovery does not increment the down-transition counter")
    wait_stream_peer(nginx, "stream_rise", rise_peer, True, 4)
    require(peer_value(nginx, "nginx_stream_healthcheck_check_up_down_total", "stream_rise", rise_peer) == 1, "stream rise to ready leaves the down-transition count unchanged")


def main() -> None:
    """
    Run bounded local HTTP, DNS, TLS, TCP, metrics, and nginx lifecycle checks.
    """
    parser = argparse.ArgumentParser(description="Exercise the native nginx upstream healthcheck module.")
    parser.add_argument("nginx", type=Path, help="nginx binary built with the native healthcheck module")
    arguments = parser.parse_args()
    binary = arguments.nginx.resolve()
    require(binary.is_file() and os.access(binary, os.X_OK), f"nginx binary is executable: {binary}")
    root_context = tempfile.TemporaryDirectory(prefix="nginx-healthcheck-")
    root = Path(root_context.name)
    (root / "logs").mkdir()
    dns = AuthoritativeDNS()
    servers = []
    stream_servers = []
    stream_states: Dict[str, StreamBackendState] = {}
    tcp = None
    nginx = None
    ipv6_state = None
    release_churn = threading.Event()
    try:
        primary_port = free_port("127.0.0.2")
        states = {
            "primary-a": BackendState("primary-a"),
            "primary-b": BackendState("primary-b"),
            "backup": BackendState("backup"),
            "admin": BackendState("admin"),
            "isolated": BackendState("isolated", 204),
            "single": BackendState("single", 503),
            "malformed": BackendState("malformed"),
            "stalled": BackendState("stalled"),
            "srv": BackendState("srv"),
            "tls": BackendState("tls"),
            "churn": BackendState("churn"),
            "replacement": BackendState("replacement"),
        }
        for address, state in (("127.0.0.2", states["primary-a"]), ("127.0.0.3", states["primary-b"]), ("127.0.0.4", states["backup"]), ("127.0.0.5", states["isolated"]), ("127.0.0.6", states["single"]), ("127.0.0.7", states["malformed"]), ("127.0.0.8", states["srv"]), ("127.0.0.9", states["admin"]), ("127.0.0.10", states["stalled"]), ("127.0.0.11", states["churn"]), ("127.0.0.13", states["replacement"])):
            servers.append(FixtureHTTPServer(address, primary_port, state))
        states["malformed"].malformed = True
        states["stalled"].ready_delay = 1.0
        ipv6_port = free_port("::1", socket.AF_INET6)
        states["ipv6"] = BackendState("ipv6")
        servers.append(FixtureHTTPServer("::1", ipv6_port, states["ipv6"]))
        certificate, private_key = generate_certificate(root, "backend")
        bad_certificate, _ = generate_certificate(root, "untrusted")
        tls_port = free_port("127.0.0.12")
        tls_state = states["tls"]
        servers.append(FixtureHTTPServer("127.0.0.12", tls_port, tls_state, certificate, private_key))
        tcp = FixtureTCPServer()
        stream_states = {
            "dynamic_a": StreamBackendState("dynamic-a"),
            "dynamic_b": StreamBackendState("dynamic-b"),
            "failover_a": StreamBackendState("failover-a", b"PING\r\n", b"PONG\r\n"),
            "failover_b": StreamBackendState("failover-b", b"PING\r\n", b"PONG\r\n"),
            "failover_backup": StreamBackendState("failover-backup", b"PING\r\n", b"PONG\r\n"),
            "failover_admin": StreamBackendState("stream-admin-down", b"PING\r\n", b"PONG\r\n"),
            "text": StreamBackendState("stream-text", b"PING\r\n", b"PONG\r\n"),
            "send_only": StreamBackendState("stream-send-only", b"PING\r\n"),
            "banner": StreamBackendState("stream-banner"),
            "binary": StreamBackendState("stream-binary", b"\x00PING\xff", b"PO\x00NG"),
            "fragmented": StreamBackendState("stream-fragmented", b"PING\r\n", b"NOPEPO"),
            "incomplete": StreamBackendState("stream-incomplete", b"PING\r\n", b"FAILFAIL"),
            "timeout": StreamBackendState("stream-timeout", b"PING\r\n", b"PONG\r\n", 0.8),
            "unmatched": StreamBackendState("stream-unmatched", b"PING\r\n"),
            "stalled": StreamBackendState("stream-stalled"),
            "rise": StreamBackendState("stream-rise", b"PING\r\n", b"BAD\r\n"),
            "tls": StreamBackendState("stream-tls", b"PING\r\n", b"PONG\r\n"),
        }
        ipv6_state = states["ipv6"]
        stream_states["fragmented"].response_chunks = [b"NG"]
        stream_states["fragmented"].chunk_delay = 0.05
        stream_states["fragmented"].pause_after_first_chunk = True
        stream_states["fragmented"].gate.clear()
        stream_states["banner"].banner = b"READY\r\n"
        stream_states["incomplete"].probe_reply = b"FAILFAIL"
        stream_states["unmatched"].repeat_response = b"x" * 64
        stream_states["stalled"].hold_open = True
        stream_states["stalled"].gate.clear()
        stream_endpoint_ports: Dict[str, int] = {}
        dynamic_port = free_port("127.0.0.15")
        _start_stream_fixture(stream_servers, stream_endpoint_ports, "dynamic_a", "127.0.0.15", stream_states["dynamic_a"], dynamic_port)
        _start_stream_fixture(stream_servers, stream_endpoint_ports, "dynamic_b", "127.0.0.16", stream_states["dynamic_b"], dynamic_port)
        stream_endpoint_ports["dynamic"] = dynamic_port
        for name, address in (
            ("failover_a", "127.0.0.17"),
            ("failover_b", "127.0.0.18"),
            ("failover_backup", "127.0.0.19"),
            ("failover_admin", "127.0.0.27"),
            ("text", "127.0.0.20"),
            ("binary", "127.0.0.21"),
            ("incomplete", "127.0.0.22"),
            ("timeout", "127.0.0.23"),
            ("rise", "127.0.0.26"),
            ("send_only", "127.0.0.28"),
            ("banner", "127.0.0.29"),
            ("fragmented", "127.0.0.30"),
            ("unmatched", "127.0.0.32"),
            ("stalled", "127.0.0.33"),
        ):
            _start_stream_fixture(stream_servers, stream_endpoint_ports, name, address, stream_states[name])
        _start_stream_fixture(stream_servers, stream_endpoint_ports, "tls", "127.0.0.25", stream_states["tls"], certificate=certificate, private_key=private_key)
        stream_endpoint_ports["refused"] = free_port("127.0.0.34")
        dns.set_addresses("primary.fixture.test", ["127.0.0.2", "127.0.0.3"])
        dns.set_addresses("backup.fixture.test", ["127.0.0.4"])
        dns.set_addresses("isolated.fixture.test", ["127.0.0.5"])
        dns.set_addresses("single.fixture.test", ["127.0.0.6"])
        dns.set_addresses("malformed.fixture.test", ["127.0.0.7"])
        dns.set_addresses("stalled.fixture.test", ["127.0.0.10"])
        dns.set_addresses("tls.fixture.test", ["127.0.0.12"])
        dns.set_addresses("churn.fixture.test", ["127.0.0.11"])
        dns.set_ipv6("ipv6.fixture.test", ["::1"])
        dns.set_srv(
            "_http._tcp.srv.fixture.test",
            [(0, 1, "srv-target.fixture.test", primary_port), (10, 1, "srv-backup.fixture.test", primary_port)],
        )
        dns.set_srv(
            "_http._tcp.srv-admin.fixture.test",
            [(0, 1, "srv-admin-primary.fixture.test", primary_port), (10, 1, "srv-admin-backup.fixture.test", primary_port)],
        )
        dns.set_addresses("srv-target.fixture.test", ["127.0.0.8"])
        dns.set_addresses("srv-backup.fixture.test", ["127.0.0.13"])
        dns.set_addresses("srv-admin-primary.fixture.test", ["127.0.0.2"])
        dns.set_addresses("srv-admin-backup.fixture.test", ["127.0.0.4"])
        dns.set_addresses("stream-dynamic.fixture.test", ["127.0.0.15", "127.0.0.16"])
        config = root / "nginx.conf"
        runtime_port = free_port()
        used_listeners = {runtime_port}
        stream_listener_ports = {}
        for name in ("dynamic", "failover"):
            stream_listener_ports[name] = free_port(exclude=used_listeners)
            used_listeners.add(stream_listener_ports[name])
        config.write_text(config_text(dns.port, primary_port, runtime_port, tls_port, certificate, bad_certificate, ipv6_port, tcp.port, stream_endpoint_ports, stream_listener_ports))
        nginx = Nginx(binary, root, config, root / "nginx.log")
        nginx.http_port = runtime_port
        test = nginx.command("-t", timeout=8)
        require(test.returncode == 0, f"nginx accepts generated native healthcheck config: {test.stdout.strip()}")
        verify_negative_configs(binary, root, dns.port, primary_port, certificate)
        nginx.start()
        native_status, native_body, native_headers = nginx.request("/_healthcheck_metrics")
        content_type = next((value.lower() for name, value in native_headers if name.lower() == "content-type"), "")
        require(native_status == 200 and "text/plain" in content_type, "native Prometheus endpoint returns a text metrics response")
        require("nginx_vts_" not in native_body, "native health and VTS metrics remain separate")
        verify_stream_runtime(nginx, dns, stream_states, stream_endpoint_ports, stream_listener_ports)

        primary_peers = (f"127.0.0.2:{primary_port}", f"127.0.0.3:{primary_port}")
        backup_peer = f"127.0.0.4:{primary_port}"
        admin_peer = f"127.0.0.9:{primary_port}"
        algorithm_peers = (*primary_peers, backup_peer)
        for peer in (*primary_peers, backup_peer):
            wait_peer(nginx, "primary", peer, True)
        for group in ("least", "hash", "random", "keepalive"):
            for peer in algorithm_peers:
                wait_peer(nginx, group, peer, True)
            status, body, _ = nginx.request(f"/{group}")
            require(status == 200 and body.strip() in {"primary-a", "primary-b", "backup"}, f"{group} upstream balancer serves a healthy peer (HTTP {status}, body={body.strip()!r})")
        admin_metrics = nginx.metrics()
        admin = peer_sample(admin_metrics, "nginx_healthcheck_peer_up", "primary", admin_peer)
        require(admin[2] == 0, "configured down peer stays out of rotation")
        require(all(peer_sample(admin_metrics, "nginx_healthcheck_peer_up", "primary", peer)[1]["backup"] == "false" for peer in primary_peers), "primary peers carry the primary metric label")
        require(peer_sample(admin_metrics, "nginx_healthcheck_peer_up", "primary", backup_peer)[1]["backup"] == "true", "backup peer is labelled separately")
        require(peer_sample(admin_metrics, "nginx_healthcheck_admin_down", "primary", admin_peer)[2] == 1, "admin-down state is reported separately")
        require(peer_sample(admin_metrics, "nginx_healthcheck_check_status", "primary", admin_peer)[2] == -1, "configured down peer has no successful probe state")
        require(peer_value_from(admin_metrics, "nginx_healthcheck_checks_total", "primary", admin_peer) == 0, "configured down peer receives no active probes")
        require(states["admin"].count() == 0, "healthy configured-down backend receives no active health request")
        require(all(host == "healthcheck.example.test" for state in states.values() for host in state.hosts_for()), "active HTTP probes send the configured Host header")
        wait_peer(nginx, "isolated", "127.0.0.5", True)
        wait_failed_peer(nginx, "single", "127.0.0.6")
        shared_zone_metrics = nginx.metrics()
        require(peer_value_from(shared_zone_metrics, "nginx_healthcheck_peer_up", "isolated", "127.0.0.5") == 1, "first health group sharing a native zone retains its healthy peer state")
        require(peer_value_from(shared_zone_metrics, "nginx_healthcheck_peer_up", "single", "127.0.0.6") == 0, "second health group sharing that zone retains its failed peer state")
        require(peer_value_from(shared_zone_metrics, "nginx_healthcheck_check_status", "isolated", "127.0.0.5") == 1, "shared-zone metrics preserve the healthy group's independent check status")
        require(peer_value_from(shared_zone_metrics, "nginx_healthcheck_check_status", "single", "127.0.0.6") == 0, "shared-zone metrics preserve the failed group's independent check status")
        wait_failed_peer(nginx, "refused", "127.0.0.14")
        wait_failed_peer(nginx, "malformed", "127.0.0.7")
        wait_failed_peer(nginx, "stalled", "127.0.0.10")
        require(peer_value(nginx, "nginx_healthcheck_check_code", "single", "127.0.0.6") == 503, "unaccepted HTTP status is preserved in metrics")
        require(peer_value(nginx, "nginx_healthcheck_check_code", "refused", "127.0.0.14") == 0, "connection refusal has no parsed HTTP status")
        require(peer_value(nginx, "nginx_healthcheck_check_code", "stalled", "127.0.0.10") == 0, "timed-out HTTP response has no parsed status")
        wait_peer(nginx, "tcp", tcp.port, True)
        with tcp.lock:
            require(tcp.connections > 0, "TCP health probes complete local handshakes")
        wait_peer(nginx, "ipv6", "::1", True)
        wait_peer(nginx, "srv", "127.0.0.8", True)
        srv_backup_peer = f"127.0.0.13:{primary_port}"
        wait_peer(nginx, "srv", srv_backup_peer, True)
        srv_admin_metrics = wait_until(
            "SRV maintenance templates resolve primary and priority-backup peers",
            lambda: (metrics if len(samples_for(metrics := nginx.metrics(), "nginx_healthcheck_peer_up", upstream="srv_admin")) == 2 else False),
            8,
        )
        for peer in ("127.0.0.2", "127.0.0.4"):
            require(peer_value_from(srv_admin_metrics, "nginx_healthcheck_admin_down", "srv_admin", peer) == 1, "SRV maintenance template marks every priority peer administratively down")
            require(peer_value_from(srv_admin_metrics, "nginx_healthcheck_peer_up", "srv_admin", peer) == 0, "SRV maintenance peer stays out of service")
            require(peer_value_from(srv_admin_metrics, "nginx_healthcheck_check_status", "srv_admin", peer) == -1, "SRV maintenance peer has no completed probe")
            require(peer_value_from(srv_admin_metrics, "nginx_healthcheck_checks_total", "srv_admin", peer) == 0, "SRV maintenance peer receives no active probe")
        srv_metrics = nginx.metrics()
        require(peer_sample(srv_metrics, "nginx_healthcheck_peer_up", "srv", "127.0.0.8")[1]["backup"] == "false", "SRV priority-zero target is labelled primary")
        require(peer_sample(srv_metrics, "nginx_healthcheck_peer_up", "srv", srv_backup_peer)[1]["backup"] == "true", "higher-priority SRV target is labelled backup")
        require(nginx.request("/srv")[0] == 200, "SRV upstream routes to its healthy priority-zero target")
        states["srv"].ready_status = 503
        wait_failed_peer(nginx, "srv", "127.0.0.8")
        srv_status, srv_body, _ = nginx.request("/srv")
        require(srv_status == 200 and srv_body.strip() == "replacement", "SRV priority backup serves traffic when its primary is unhealthy")
        states["srv"].ready_status = 200
        wait_peer(nginx, "srv", "127.0.0.8", True)
        srv_status, srv_body, _ = nginx.request("/srv")
        require(srv_status == 200 and srv_body.strip() == "srv", "recovered SRV primary regains preference over priority backup")
        wait_peer(nginx, "tls_good", "127.0.0.12", True)
        wait_failed_peer(nginx, "tls_bad_sni", "127.0.0.12")
        wait_failed_peer(nginx, "tls_bad_ca", "127.0.0.12")
        require("backend.example.test" in tls_state.sni, "HTTPS probes send the configured TLS SNI")
        require("wrong.example.test" in tls_state.sni, "HTTPS probe with incorrect SNI fails certificate hostname verification")
        require(peer_value(nginx, "nginx_healthcheck_check_code", "tls_good", "127.0.0.12") == 200, "trusted HTTPS readiness response is parsed")
        require(dns.query_count("primary.fixture.test", 1) > 0, "authoritative fixture served A lookups")
        require(dns.query_count("ipv6.fixture.test", 28) > 0, "authoritative fixture served AAAA lookups")
        require(dns.query_count("_http._tcp.srv.fixture.test", 33) > 0, "authoritative fixture served SRV lookups")
        require(dns.query_count("_http._tcp.srv-admin.fixture.test", 33) > 0, "authoritative fixture served SRV maintenance lookups")

        before_reorder_queries = dns.query_count("primary.fixture.test", 1)
        dns.set_addresses("primary.fixture.test", ["127.0.0.3", "127.0.0.2"])

        def _reordered_answers_seen() -> bool:
            """
            Detect a fresh resolver query after swapping the A answer order.
            """
            return dns.query_count("primary.fixture.test", 1) > before_reorder_queries

        wait_until("resolver observes reordered A answers", _reordered_answers_seen, 4)
        for peer in primary_peers:
            wait_peer(nginx, "primary", peer, True)
        require(all(peer_value(nginx, "nginx_healthcheck_check_status", "primary", peer) == 1 for peer in primary_peers), "DNS answer reordering preserves peer readiness")
        require(states["admin"].count() == 0, "healthy configured-down peer remains unprobed during traffic checks")
        dns.set_addresses("primary.fixture.test", ["127.0.0.2", "127.0.0.3"])

        metrics = nginx.metrics()
        required_metrics = {
            "nginx_healthcheck_peer_up",
            "nginx_healthcheck_admin_down",
            "nginx_healthcheck_checks_total",
            "nginx_healthcheck_check_failures_total",
            "nginx_healthcheck_check_up_down_total",
            "nginx_healthcheck_check_status",
            "nginx_healthcheck_check_code",
            "nginx_healthcheck_check_duration_seconds",
            "nginx_healthcheck_check_last_change_seconds",
            "nginx_healthcheck_last_check_timestamp_seconds",
            "nginx_healthcheck_rise_streak",
            "nginx_healthcheck_fall_streak",
            "nginx_healthcheck_last_scan_timestamp_seconds",
            "nginx_healthcheck_errors_total",
            "nginx_healthcheck_active_probes",
            "nginx_healthcheck_worker_pid",
            "nginx_healthcheck_concurrency",
            "nginx_healthcheck_concurrency_limited_total",
            "nginx_healthcheck_memory_failures_total",
        }
        stream_required_metrics = {
            "nginx_stream_healthcheck_peer_up",
            "nginx_stream_healthcheck_admin_down",
            "nginx_stream_healthcheck_checks_total",
            "nginx_stream_healthcheck_check_failures_total",
            "nginx_stream_healthcheck_check_up_down_total",
            "nginx_stream_healthcheck_check_status",
            "nginx_stream_healthcheck_check_duration_seconds",
            "nginx_stream_healthcheck_check_last_change_seconds",
            "nginx_stream_healthcheck_last_check_timestamp_seconds",
            "nginx_stream_healthcheck_rise_streak",
            "nginx_stream_healthcheck_fall_streak",
            "nginx_stream_healthcheck_last_scan_timestamp_seconds",
            "nginx_stream_healthcheck_errors_total",
            "nginx_stream_healthcheck_active_probes",
            "nginx_stream_healthcheck_worker_pid",
            "nginx_stream_healthcheck_concurrency",
            "nginx_stream_healthcheck_concurrency_limited_total",
            "nginx_stream_healthcheck_memory_failures_total",
        }
        require(required_metrics.issubset(metrics["types"]), "native metrics include state, counters, status, code, duration, and timestamps")
        require(required_metrics.issubset(metrics["help"]), "every required metric has HELP metadata")
        require(stream_required_metrics.issubset(metrics["types"]), "native stream metrics include peer state, counters, streaks, and group gauges")
        require(stream_required_metrics.issubset(metrics["help"]), "every required stream metric has HELP metadata")
        counter_metrics = {
            "nginx_healthcheck_checks_total",
            "nginx_healthcheck_check_failures_total",
            "nginx_healthcheck_check_up_down_total",
            "nginx_healthcheck_errors_total",
            "nginx_healthcheck_concurrency_limited_total",
            "nginx_healthcheck_memory_failures_total",
        }
        require(all(metrics["types"][name] == "counter" for name in counter_metrics), "cumulative health events use Prometheus counters")
        require(all(metrics["types"][name] == "gauge" for name in required_metrics - counter_metrics), "current health state and second-based values use Prometheus gauges")
        require(all(set(sample[1]) >= {"upstream", "server", "peer", "backup"} for sample in metrics["samples"] if sample[0] == "nginx_healthcheck_peer_up"), "peer metrics parse all four escaped identity labels")
        stream_counter_metrics = {
            "nginx_stream_healthcheck_checks_total",
            "nginx_stream_healthcheck_check_failures_total",
            "nginx_stream_healthcheck_check_up_down_total",
            "nginx_stream_healthcheck_errors_total",
            "nginx_stream_healthcheck_concurrency_limited_total",
            "nginx_stream_healthcheck_memory_failures_total",
        }
        require(all(metrics["types"][name] == "counter" for name in stream_counter_metrics), "stream cumulative events use Prometheus counters")
        require(all(metrics["types"][name] == "gauge" for name in stream_required_metrics - stream_counter_metrics), "stream state and second-based values use gauges")
        require(all(set(sample[1]) >= {"upstream", "server", "peer", "backup"} for sample in metrics["samples"] if sample[0] == "nginx_stream_healthcheck_peer_up"), "stream peer metrics parse all four identity labels")
        require(peer_value_from(metrics, "nginx_healthcheck_check_code", "isolated", "127.0.0.5") == 204, "accepted 204 response code is preserved in metrics")
        require(peer_value_from(metrics, "nginx_healthcheck_check_code", "malformed", "127.0.0.7") == 0, "malformed HTTP response has no parsed status code")
        require(samples_for(metrics, "nginx_healthcheck_last_scan_timestamp_seconds", upstream="primary"), "per-upstream health scan timestamp is exported")
        require(samples_for(metrics, "nginx_healthcheck_errors_total", upstream="primary"), "per-upstream health error counter is exported")
        require(samples_for(metrics, "nginx_healthcheck_worker_pid"), "healthcheck worker identity is exported")
        initial_route_status, initial_route_body, _ = nginx.request("/route")
        require(initial_route_status == 200 and initial_route_body.strip() in ("primary-a", "primary-b"), "healthy upstream returns an independent traffic identity")
        vts_status, vts_body, _ = nginx.request("/_vts_metrics")
        require(vts_status == 200 and "nginx_vts_" in vts_body, "VTS Prometheus endpoint remains independently reachable")

        states["primary-a"].ready_status = 503
        for group in ("least", "hash", "random", "keepalive"):
            wait_failed_peer(nginx, group, algorithm_peers[0])
            status, body, _ = nginx.request(f"/{group}")
            require(status == 200 and body.strip() in {"primary-b", "backup"}, f"{group} excludes the actively unhealthy peer (HTTP {status}, body={body.strip()!r})")
        states["primary-a"].ready_status = 200
        for group in ("least", "hash", "random", "keepalive"):
            wait_peer(nginx, group, algorithm_peers[0], True)

        failure_metric = "nginx_healthcheck_check_failures_total"
        baseline_failure = peer_value(nginx, failure_metric, "primary", primary_peers[0])
        baseline_transitions = peer_value(nginx, "nginx_healthcheck_check_up_down_total", "primary", primary_peers[0])
        states["primary-a"].ready_status = 503
        wait_until("first failed probe leaves peer available", lambda: peer_value(nginx, failure_metric, "primary", primary_peers[0]) == baseline_failure + 1 and peer_value(nginx, "nginx_healthcheck_peer_up", "primary", primary_peers[0]) == 1 and peer_value(nginx, "nginx_healthcheck_fall_streak", "primary", primary_peers[0]) == 1, 4)
        wait_until("second failed probe remains below the fall threshold", lambda: peer_value(nginx, failure_metric, "primary", primary_peers[0]) == baseline_failure + 2 and peer_value(nginx, "nginx_healthcheck_peer_up", "primary", primary_peers[0]) == 1 and peer_value(nginx, "nginx_healthcheck_fall_streak", "primary", primary_peers[0]) == 2, 4)
        wait_peer(nginx, "primary", primary_peers[0], False)
        require(peer_value(nginx, "nginx_healthcheck_check_status", "primary", primary_peers[0]) == 0, "failed active status is visible in metrics")
        require(peer_value(nginx, "nginx_healthcheck_check_code", "primary", primary_peers[0]) == 503, "failed HTTP status code is preserved in metrics")
        require(peer_value(nginx, "nginx_healthcheck_fall_streak", "primary", primary_peers[0]) >= 3, "fall streak reaches its configured threshold")
        require(peer_value(nginx, "nginx_healthcheck_check_up_down_total", "primary", primary_peers[0]) > baseline_transitions, "down transition increments its counter")
        for _ in range(12):
            status, body, _ = nginx.request("/route")
            require(status == 200 and "primary-a" not in body, "traffic excludes actively unhealthy peer while its data path returns 200")
        require(peer_value(nginx, "nginx_healthcheck_peer_up", "primary", admin_peer) == 0, "admin-down peer remains down while active peers fail")
        states["primary-b"].ready_status = 503
        wait_peer(nginx, "primary", primary_peers[1], False)
        status, body, _ = nginx.request("/route")
        require(status == 200 and body.strip() == "backup", "backup peer serves traffic when every primary is unhealthy")
        require(states["admin"].count() == 0 and states["admin"].count("/route") == 0, "healthy administratively down peer receives neither probes nor traffic")
        states["primary-a"].ready_status = 200
        wait_peer(nginx, "primary", primary_peers[0], True)
        require(peer_value(nginx, "nginx_healthcheck_rise_streak", "primary", primary_peers[0]) >= 2, "recovery requires the configured rise threshold")
        require(peer_value(nginx, "nginx_healthcheck_fall_streak", "primary", primary_peers[0]) == 0, "successful probes reset the fall streak")
        require(peer_value(nginx, "nginx_healthcheck_check_up_down_total", "primary", primary_peers[0]) == baseline_transitions + 1, "HTTP recovery leaves the down-transition count unchanged")
        status, body, _ = nginx.request("/route")
        require(status == 200 and body.strip() != "backup", "recovered primary peers regain preference over backup")
        single_status, _, _ = nginx.request("/single")
        require(single_status == 502, "single-peer upstream returns 502 after active health failure")
        isolated_status, isolated_body, _ = nginx.request("/isolated")
        require(isolated_status == 200 and isolated_body.strip() == "isolated", "separate upstream group remains independently routable")

        churn_peer = f"127.0.0.11:{primary_port}"
        wait_peer(nginx, "churn", churn_peer, True)
        with states["churn"].lock:
            states["churn"].block_next = (release_churn, 503)
        states["churn"].block_started.clear()
        states["churn"].block_write_succeeded.clear()
        wait_until("health probe is in flight before DNS replacement", states["churn"].block_started.is_set, 3)
        states["replacement"].ready_delay = 0.5
        dns.set_addresses("churn.fixture.test", ["127.0.0.13"])
        wait_until("DNS replacement removes the old peer", lambda: churn_peer not in "".join(sample[1].get("peer", "") for sample in samples_for(nginx.metrics(), "nginx_healthcheck_peer_up", upstream="churn")), 4)
        replacement = "127.0.0.13"
        def _replacement_pending():
            """
            Observe a DNS replacement before it meets its configured rise count.
            """
            metrics = nginx.metrics()
            try:
                pending = peer_value_from(metrics, "nginx_healthcheck_peer_up", "churn", replacement) == 0
                never_checked = peer_value_from(metrics, "nginx_healthcheck_check_status", "churn", replacement) == -1
            except VerificationFailure:
                return False
            return metrics if pending and never_checked else False

        replacement_metrics = wait_until("replacement peer appears with fresh pending state", _replacement_pending, 4)
        require(peer_value_from(replacement_metrics, "nginx_healthcheck_checks_total", "churn", replacement) == 0, "new DNS peer waits for its full rise threshold")
        wait_until(
            "new DNS peer remains unavailable after its first successful check",
            lambda: peer_value(nginx, "nginx_healthcheck_checks_total", "churn", replacement) == 1
            and peer_value(nginx, "nginx_healthcheck_rise_streak", "churn", replacement) == 1
            and peer_value(nginx, "nginx_healthcheck_peer_up", "churn", replacement) == 0,
            6,
        )
        wait_peer(nginx, "churn", replacement, True)
        states["replacement"].ready_delay = 0
        states["churn"].ready_delay = 1.5
        dns.set_addresses("churn.fixture.test", ["127.0.0.11"])

        def _recreated_peer_pending():
            """
            Detect a re-added DNS peer before it completes a fresh probe.
            """
            metrics = nginx.metrics()
            try:
                is_pending = peer_value_from(metrics, "nginx_healthcheck_peer_up", "churn", churn_peer) == 0
                never_checked = peer_value_from(metrics, "nginx_healthcheck_check_status", "churn", churn_peer) == -1
            except VerificationFailure:
                return False
            return metrics if is_pending and never_checked else False

        reset_metric = wait_until("re-added peer exposes fresh pending state", _recreated_peer_pending, 4)
        require(reset_metric is not False, "re-added DNS peer starts unhealthy with no inherited probe result")
        require(peer_value_from(reset_metric, "nginx_healthcheck_checks_total", "churn", churn_peer) == 0, "re-added peer receives a fresh probe counter")
        require(peer_value_from(reset_metric, "nginx_healthcheck_check_failures_total", "churn", churn_peer) == 0, "re-added peer receives a fresh failure counter")
        states["churn"].ready_delay = 0
        wait_until(
            "re-added DNS peer remains unavailable after its first successful check",
            lambda: peer_value(nginx, "nginx_healthcheck_checks_total", "churn", churn_peer) == 1
            and peer_value(nginx, "nginx_healthcheck_rise_streak", "churn", churn_peer) == 1
            and peer_value(nginx, "nginx_healthcheck_peer_up", "churn", churn_peer) == 0,
            6,
        )
        wait_until("re-added peer completes two successful checks", lambda: peer_value(nginx, "nginx_healthcheck_checks_total", "churn", churn_peer) >= 2 and peer_value(nginx, "nginx_healthcheck_rise_streak", "churn", churn_peer) >= 2, 8)
        wait_peer(nginx, "churn", churn_peer, True, 8)
        require(not states["churn"].block_write_succeeded.is_set(), "removed peer probe remains outstanding after its replacement recovers")
        require(group_value(nginx, "nginx_healthcheck_active_probes", "churn") >= 1, "removed peer probe remains active after the new peer recovers")
        failure_count_before_stale_result = peer_value(nginx, failure_metric, "churn", churn_peer)
        release_churn.set()
        wait_until("stale probe response reaches the fixture after peer recovery", states["churn"].block_write_succeeded.is_set, 2)
        wait_until("completed stale probe releases its active slot", lambda: group_value(nginx, "nginx_healthcheck_active_probes", "churn") == 0, 4)
        require(peer_value(nginx, failure_metric, "churn", churn_peer) == failure_count_before_stale_result, "stale probe from a removed DNS peer does not change its new generation")

        old_workers = nginx.children()
        require(len(old_workers) == 2, "nginx starts two worker processes")
        observed_workers = set()
        for _ in range(80):
            _, body, headers = nginx.request("/worker")
            worker_header = next((value for name, value in headers if name.lower() == "x-nginx-worker"), body.strip())
            observed_workers.add(worker_header)
            if len(observed_workers) == 2:
                break
        require(len(observed_workers) == 2, "requests are served by both nginx workers sharing health state")
        before_worker_restart = peer_value(nginx, "nginx_healthcheck_checks_total", "primary", primary_peers[0])
        timestamp_before_worker_restart = peer_value(nginx, "nginx_healthcheck_last_check_timestamp_seconds", "primary", primary_peers[0])
        checker_pids_before = samples_for(nginx.metrics(), "nginx_healthcheck_worker_pid")
        checker_pids = {int(sample[2]) for sample in checker_pids_before}
        require(len(checker_pids) == 1 and next(iter(checker_pids)) in old_workers, "healthcheck worker metric identifies the active nginx child")
        checker_pid = next(iter(checker_pids))
        os.kill(checker_pid, signal.SIGKILL)
        wait_until("nginx respawns the killed healthcheck worker", lambda: len(nginx.children()) == 2 and checker_pid not in nginx.children(), 5)
        wait_until("healthcheck ownership moves to the replacement worker", lambda: any(int(sample[2]) in nginx.children() and int(sample[2]) != checker_pid for sample in samples_for(nginx.metrics(), "nginx_healthcheck_worker_pid")), 5)
        wait_until("health probes progress after worker restart", lambda: peer_value(nginx, "nginx_healthcheck_checks_total", "primary", primary_peers[0]) > before_worker_restart, 5)
        wait_until("healthcheck timestamp advances after worker restart", lambda: peer_value(nginx, "nginx_healthcheck_last_check_timestamp_seconds", "primary", primary_peers[0]) > timestamp_before_worker_restart, 5)

        checks_before_reload = peer_value(nginx, "nginx_healthcheck_checks_total", "primary", primary_peers[0])
        states["primary-a"].ready_status = 503
        workers_before_reload = set(nginx.children())
        reload_result = nginx.command("-s", "reload", timeout=5)
        require(reload_result.returncode == 0, "nginx accepts a graceful reload")
        wait_until("workers are replaced and remain available after reload", lambda: nginx.process.poll() is None and len(nginx.children()) == 2 and not (set(nginx.children()) & workers_before_reload) and nginx.request("/_healthcheck_metrics")[0] == 200, 10)
        wait_until("reloaded healthcheck worker reports its new process id", lambda: any(int(sample[2]) in nginx.children() for sample in samples_for(nginx.metrics(), "nginx_healthcheck_worker_pid")), 5)
        wait_until(
            "reload resets peer health and counters before new admission",
            lambda: 1 <= peer_value(nginx, "nginx_healthcheck_checks_total", "primary", primary_peers[0]) < 3
            and peer_value(nginx, "nginx_healthcheck_check_failures_total", "primary", primary_peers[0]) >= 1
            and 1 <= peer_value(nginx, "nginx_healthcheck_fall_streak", "primary", primary_peers[0]) < 3
            and peer_value(nginx, "nginx_healthcheck_peer_up", "primary", primary_peers[0]) == 0,
            5,
        )
        require(peer_value(nginx, "nginx_healthcheck_checks_total", "primary", primary_peers[0]) < checks_before_reload, "reload starts fresh per-peer counters")
        states["primary-a"].ready_status = 200
        wait_peer(nginx, "primary", primary_peers[0], True, 8)
        graceful_shutdown = nginx.close(require_graceful=True)
        nginx = None
        require(graceful_shutdown, "nginx shuts down successfully on QUIT with no remaining workers")
        error_log = (root / "logs" / "error.log").read_text(errors="replace")
        require(not re.search(r"segfault|core dumped|signal \d+ \(SIGSEGV", error_log, re.IGNORECASE), "nginx error log contains no crash signature")
        expected_alerts = [line for line in error_log.splitlines() if re.search(r"alert", line, re.IGNORECASE)]
        unexpected_alerts = [line for line in expected_alerts if not re.search(r"worker process \d+ exited on signal 9", line)]
        require(not unexpected_alerts, "nginx error log contains no unexpected alert-level records")
    finally:
        had_failure = sys.exc_info()[1] is not None
        if had_failure:
            _print_ipv6_diagnostics(dns, nginx, ipv6_state)
            error_log_path = root / "logs" / "error.log"
            try:
                error_lines = error_log_path.read_text(errors="replace").splitlines()
            except OSError:
                error_lines = []
            print("NGINX_ERROR_LOG_TAIL_BEGIN", flush=True)
            for line in error_lines[-25:]:
                print(f"NGINX_ERROR_LOG {line[:500]}", flush=True)
            print("NGINX_ERROR_LOG_TAIL_END", flush=True)
        release_churn.set()
        for state in stream_states.values():
            state.gate.set()
        cleanup_errors = []
        for resource in (nginx, tcp, *reversed(stream_servers), *reversed(servers), dns):
            if resource is None:
                continue
            try:
                resource.close()
            except Exception as error:
                cleanup_errors.append(str(error))
        try:
            root_context.cleanup()
        except Exception as error:
            cleanup_errors.append(str(error))
        if cleanup_errors and not had_failure:
            raise VerificationFailure("fixture cleanup failed: " + "; ".join(cleanup_errors))
        if cleanup_errors:
            print("FIXTURE_CLEANUP_ERRORS " + "; ".join(cleanup_errors), flush=True)
    print("PASS native nginx active healthcheck runtime verification completed", flush=True)


if __name__ == "__main__":
    try:
        main()
    except (VerificationFailure, OSError, subprocess.SubprocessError) as error:
        print(f"FAIL {error}", flush=True)
        raise SystemExit(1)
