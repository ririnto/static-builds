"""
Verify wget download deadlines and cleanup without a Docker daemon.

Run with Python 3, OpenSSL, wget, and GNU Coreutils or BusyBox timeout.
GNU Coreutils gtimeout is also supported on macOS.
"""

import http.server
import json
import os
from pathlib import Path
import shutil
import signal
import ssl
import subprocess
import sys
import tempfile
import threading
import time
from typing import Final


_SCENARIOS: Final = ("success", "slow", "resistant", "gtimeout", "signal", "failure", "missing_timeout", "fixture_timeout")
_HOST_COMMANDS: Final = ("sh", "awk", "dirname", "mkdir", "mktemp", "rm", "rmdir", "sleep", "mv")


class _Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        """
        Serve a complete archive, an HTTP error, or a slow-drip body.
        """
        scenario = self.path.split("/")[1]
        self.send_response(503 if scenario == "failure" else 200)
        self.send_header("Content-Length", "100" if scenario in ("slow", "resistant", "gtimeout", "signal", "fixture_timeout") else "7")
        self.end_headers()
        try:
            if scenario in ("slow", "resistant", "gtimeout", "signal", "fixture_timeout"):
                remaining = 100
                while remaining > 0:
                    self.wfile.write(b"x")
                    self.wfile.flush()
                    remaining = 0 if self.server.stopping.wait(0.1) else remaining - 1
            else:
                self.wfile.write(b"archive")
        except (BrokenPipeError, ConnectionResetError, ssl.SSLError):
            pass

    def log_message(self, format, *args):
        """
        Keep expected fixture HTTP failures out of the command output.
        """


def _wget():
    if os.environ["DOWNLOAD_FIXTURE_SCENARIO"] == "resistant":
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
    Path(os.environ["DOWNLOAD_FIXTURE_PID"]).write_text(str(os.getpid()))
    os.execv(os.environ["DOWNLOAD_FIXTURE_WGET"], ["wget", *sys.argv[1:]])


def _timeout():
    Path(os.environ["DOWNLOAD_FIXTURE_TIMEOUT_PID"]).write_text(str(os.getpid()))
    os.execv(os.environ["DOWNLOAD_FIXTURE_TIMEOUT"], ["timeout", *sys.argv[1:]])


def _running(pid):
    result = subprocess.run(
        ["ps", "-p", str(pid), "-o", "stat="], capture_output=True, text=True, check=False,
    )
    return result.returncode == 0 and not result.stdout.strip().startswith("Z")


def _stop_fixtures(process, wget_pid_file, timeout_pid_file):
    recorded = tuple(int(path.read_text()) for path in (wget_pid_file, timeout_pid_file) if path.exists())
    groups = (process.pid,)
    if timeout_pid_file.exists():
        groups += (int(timeout_pid_file.read_text()),)
    for group in groups:
        try:
            os.killpg(group, signal.SIGKILL)
        except ProcessLookupError:
            pass
    for pid in recorded:
        try:
            os.kill(pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    process.communicate(timeout=3)
    deadline = time.monotonic() + 3
    while any(_running(pid) for pid in recorded) and time.monotonic() < deadline:
        time.sleep(0.02)
    if any(_running(pid) for pid in recorded):
        raise AssertionError("The fixture left a downloader or deadline process running.")


def _verify(repository, directory, server, wget, limiter, scenario):
    directory.mkdir()
    for name in ("download.sh", "metadata.sh", "metadata.awk"):
        shutil.copyfile(repository / name, directory / name)
    binary = directory / "bin"
    binary.mkdir()
    for name in _HOST_COMMANDS:
        source = shutil.which(name)
        if source is None:
            raise RuntimeError(f"Required fixture command is missing: {name}")
        (binary / name).symlink_to(source)
    if scenario != "missing_timeout":
        wrapper = binary / ("gtimeout" if scenario == "gtimeout" else "timeout")
        wrapper.write_text(f"#!{sys.executable}\n" + Path(__file__).read_text())
        wrapper.chmod(0o755)
    wrapper = binary / "wget"
    wrapper.write_text(f"#!{sys.executable}\n" + Path(__file__).read_text())
    wrapper.chmod(0o755)
    metadata = directory / "metadata.json"
    metadata.write_text(json.dumps({"fixture": {
        "env": {"VERSION": "1"},
        "downloads": [{
            "url": f"https://127.0.0.1:{server.server_port}/{scenario}/archive/refs/heads/main.tar.gz",
            "name": "fixture.tar.gz",
        }],
    }}))
    wgetrc = directory / "wgetrc"
    wgetrc.write_text(f"ca_certificate = {server.certificate}\n")
    pid_file = directory / "wget.pid"
    timeout_pid_file = directory / "timeout.pid"
    environment = {
        **os.environ,
        "PATH": str(binary),
        "WGETRC": str(wgetrc),
        "METADATA_PATH_OVERRIDE": str(metadata),
        "DOWNLOAD_FIXTURE_PID": str(pid_file),
        "DOWNLOAD_FIXTURE_WGET": wget,
        "DOWNLOAD_FIXTURE_TIMEOUT_PID": str(timeout_pid_file),
        "DOWNLOAD_FIXTURE_TIMEOUT": limiter,
        "DOWNLOAD_FIXTURE_SCENARIO": scenario,
        "DOWNLOAD_RETRIES": "1",
        "DOWNLOAD_CONNECT_TIMEOUT": "1",
        "DOWNLOAD_MAX_TIME": "1" if scenario in ("slow", "resistant", "gtimeout") else "10",
    }
    started = time.monotonic()
    process = subprocess.Popen(
        [str(binary / "sh"), str(directory / "download.sh"), "fixture"],
        env=environment, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        start_new_session=True,
    )
    try:
        if scenario == "signal":
            deadline = time.monotonic() + 5
            while not pid_file.exists() and time.monotonic() < deadline:
                time.sleep(0.02)
            if not pid_file.exists():
                raise AssertionError("The cancellation fixture never started wget.")
            process.terminate()
        if scenario == "fixture_timeout":
            try:
                process.communicate(timeout=1)
            except subprocess.TimeoutExpired:
                if not pid_file.exists() or not timeout_pid_file.exists():
                    raise AssertionError("The failure-cleanup fixture never started both processes.")
            else:
                raise AssertionError("The failure-cleanup fixture did not exceed its driver deadline.")
        else:
            _, error = process.communicate(timeout=6)
            expected = 0 if scenario == "success" else 1
            if process.returncode != expected:
                raise AssertionError(f"{scenario}: exit {process.returncode}, expected {expected}\n{error}")
            elapsed = time.monotonic() - started
            if scenario in ("slow", "resistant", "gtimeout") and elapsed > 4:
                raise AssertionError(f"The one-second deadline took {elapsed:.2f} seconds.")
            if scenario == "missing_timeout" and "require timeout" not in error:
                raise AssertionError(f"Missing timeout prerequisite was not diagnosed: {error}")
            if pid_file.exists() and _running(int(pid_file.read_text())):
                raise AssertionError(f"{scenario}: wget survived downloader completion.")
            if timeout_pid_file.exists() and _running(int(timeout_pid_file.read_text())):
                raise AssertionError(f"{scenario}: timeout survived downloader completion.")
            remaining = list((directory / ".tmp").iterdir())
            expected_files = [directory / ".tmp" / "fixture.tar.gz"] if scenario == "success" else []
            if remaining != expected_files:
                raise AssertionError(f"{scenario}: download temporary files or locks remain: {remaining}")
            print(f"PASS {scenario}")
    finally:
        _stop_fixtures(process, pid_file, timeout_pid_file)
        if scenario == "fixture_timeout":
            print("PASS fixture_timeout cleanup")


def _main():
    repository = Path(__file__).resolve().parents[2]
    wget = shutil.which("wget")
    limiter = shutil.which("timeout") or shutil.which("gtimeout")
    if wget is None or limiter is None:
        raise RuntimeError("This verification requires wget and timeout or gtimeout.")
    with tempfile.TemporaryDirectory(prefix="download-deadline-") as temporary:
        directory = Path(temporary)
        certificate = directory / "certificate.pem"
        key = directory / "key.pem"
        subprocess.run([
            "openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
            "-keyout", str(key), "-out", str(certificate), "-days", "1",
            "-subj", "/CN=localhost", "-addext", "subjectAltName=IP:127.0.0.1",
        ], capture_output=True, check=True)
        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
        server.certificate = certificate
        server.stopping = threading.Event()
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        context.load_cert_chain(certificate, key)
        server.socket = context.wrap_socket(server.socket, server_side=True)
        worker = threading.Thread(target=server.serve_forever)
        worker.start()
        try:
            for scenario in _SCENARIOS:
                _verify(repository, directory / scenario, server, wget, limiter, scenario)
        finally:
            server.stopping.set()
            server.shutdown()
            server.server_close()
            worker.join(timeout=3)
            if worker.is_alive():
                raise AssertionError("The fixture HTTPS server did not stop.")


if __name__ == "__main__":
    match Path(sys.argv[0]).name:
        case "wget":
            _wget()
        case "timeout" | "gtimeout":
            _timeout()
        case _:
            _main()
