"""
Exercise the shared nginx verifier with local process fixtures and no Docker daemon.

GNU Coreutils timeout or gtimeout is required for real process supervision.
"""

import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from typing import Final


_EXPECTED_ENGINE: Final = "njs 1.0.2 engine QuickJS"
_SCENARIOS: Final = (
    ("success", 0, None, False),
    ("http_error_first", 22, None, False),
    ("http_error_second", 22, None, False),
    ("http_deadline", 137, None, False),
    ("signal_int", 130, signal.SIGINT, False),
    ("signal_term", 143, signal.SIGTERM, False),
    ("active_probe_cancellation", 143, signal.SIGTERM, False),
    ("resistant_nginx", 0, None, True),
)


def _write_executable(path: Path, content: str) -> None:
    path.write_text(content)
    path.chmod(0o755)


def _read_process_state(process_id: int) -> str | None:
    stat_path = Path(f"/proc/{process_id}/stat")
    try:
        stat_line = stat_path.read_text()
    except FileNotFoundError:
        result = subprocess.run(
            ["ps", "-o", "stat=", "-p", str(process_id)],
            capture_output=True,
            text=True,
            check=False,
        )
        state = result.stdout.strip().split(maxsplit=1)[0] if result.stdout.strip() else None
    else:
        stat_tail = stat_line[stat_line.rfind(")") + 2 :]
        state = stat_tail.split(maxsplit=1)[0]
    return state


def _process_running(process_id: int) -> bool:
    return _read_process_state(process_id) not in (None, "Z")


def _process_groups():
    if Path("/proc").is_dir():
        for stat_path in Path("/proc").glob("[0-9]*/stat"):
            try:
                stat_line = stat_path.read_text()
            except FileNotFoundError:
                pass
            else:
                stat_tail = stat_line[stat_line.rfind(")") + 2 :].split()
                if len(stat_tail) >= 3:
                    yield stat_tail[2], stat_tail[0]
    else:
        result = subprocess.run(
            ["ps", "-axo", "pgid=,stat="],
            capture_output=True,
            text=True,
            check=False,
        )
        for line in result.stdout.splitlines():
            fields = line.split()
            if len(fields) >= 2:
                yield fields[0], fields[1]


def _group_running(group_id: int) -> bool:
    return any(
        group == str(group_id) and not state.startswith("Z")
        for group, state in _process_groups()
    )


def _fixture_processes(directory: Path) -> tuple[set[int], set[int]]:
    process_ids: set[int] = set()
    group_ids: set[int] = set()
    for name in ("timeout.pids", "curl.pids"):
        path = directory / name
        if path.exists():
            recorded = {int(value) for value in path.read_text().splitlines()}
            process_ids.update(recorded)
            group_ids.update(recorded)
    for name in ("nginx.pid", "curl.pid"):
        path = directory / name
        if path.exists():
            process_ids.add(int(path.read_text()))
    return process_ids, group_ids


def _signal_fixtures(process: subprocess.Popen[str], directory: Path, signum: int) -> None:
    process_ids, group_ids = _fixture_processes(directory)
    for group_id in group_ids | {process.pid}:
        if _group_running(group_id):
            try:
                os.killpg(group_id, signum)
            except ProcessLookupError:
                pass
    for process_id in process_ids:
        if _process_running(process_id):
            try:
                os.kill(process_id, signum)
            except ProcessLookupError:
                pass


def _fixtures_running(process: subprocess.Popen[str], directory: Path) -> bool:
    process_ids, group_ids = _fixture_processes(directory)
    return any(_process_running(process_id) for process_id in process_ids) or any(
        _group_running(group_id) for group_id in group_ids | {process.pid}
    )


def _terminate_group(process: subprocess.Popen[str], directory: Path) -> None:
    _signal_fixtures(process, directory, signal.SIGTERM)
    deadline = time.monotonic() + 2
    while time.monotonic() < deadline and _fixtures_running(process, directory):
        time.sleep(0.05)
    if _fixtures_running(process, directory):
        _signal_fixtures(process, directory, signal.SIGKILL)
    try:
        process.wait(timeout=3)
    except subprocess.TimeoutExpired:
        _signal_fixtures(process, directory, signal.SIGKILL)
        process.wait(timeout=3)
    deadline = time.monotonic() + 2
    while time.monotonic() < deadline and _fixtures_running(process, directory):
        time.sleep(0.05)
    if _fixtures_running(process, directory):
        raise AssertionError(f"Scenario fixture processes remain after cleanup for group {process.pid}")


def _create_tools(directory: Path) -> Path:
    tools = directory / "bin"
    tools.mkdir()
    _write_executable(
        tools / "readelf",
        "#!/bin/sh\n"
        "case \"$1\" in\n"
        "  -h) printf '  Type: DYN (Position-Independent Executable file)\\n  Machine: fixture\\n' ;;\n"
        "  -l) printf 'No interpreter or dynamic section\\n' ;;\n"
        "  -d) printf 'There is no dynamic section in this file.\\n' ;;\n"
        "esac\n",
    )
    _write_executable(tools / "file", "#!/bin/sh\nprintf 'ELF fixture statically linked\\n'\n")
    _write_executable(tools / "ldd", "#!/bin/sh\nprintf 'not a dynamic executable\\n'\n")
    _write_executable(tools / "strace", "#!/bin/sh\nexec \"$@\"\n")
    _write_executable(
        tools / "timeout",
        "#!/usr/bin/env python3\n"
        "import os\n"
        "import sys\n"
        "arguments = sys.argv[1:]\n"
        "if arguments == ['3', '/usr/local/bin/nginx-stream-read.sh']:\n"
        "    print(os.environ['NGINX_EXPECTED_ENGINE'])\n"
        "    raise SystemExit(0)\n"
        "from pathlib import Path\n"
        "with (Path(os.environ['NGINX_FIXTURE_DIR']) / 'timeout.pids').open('a') as records:\n"
        "    records.write(str(os.getpid()) + '\\n')\n"
        "timeout = os.environ['REAL_TIMEOUT']\n"
        "os.execv(timeout, [timeout, *arguments])\n",
    )
    _write_executable(
        tools / "curl",
        "#!/usr/bin/env python3\n"
        "import os\n"
        "from pathlib import Path\n"
        "import signal\n"
        "import sys\n"
        "import time\n"
        "scenario = os.environ['NGINX_SCENARIO']\n"
        "directory = Path(os.environ['NGINX_FIXTURE_DIR'])\n"
        "count_path = directory / 'curl-count'\n"
        "with (directory / 'curl.pids').open('a') as records:\n"
        "    records.write(str(os.getpid()) + '\\n')\n"
        "count = int(count_path.read_text()) + 1 if count_path.exists() else 1\n"
        "count_path.write_text(str(count))\n"
        "if scenario in ('http_deadline', 'signal_int', 'signal_term', 'active_probe_cancellation'):\n"
        "    (directory / 'curl.pid').write_text(str(os.getpid()))\n"
        "    signal.signal(signal.SIGTERM, signal.SIG_IGN)\n"
        "    while True:\n"
        "        time.sleep(0.1)\n"
        "if scenario == 'http_error_first' and count == 1 or scenario == 'http_error_second' and count == 2:\n"
        "    raise SystemExit(22)\n"
        "output_path = Path(sys.argv[sys.argv.index('-o') + 1])\n"
        "output_path.write_text(os.environ['NGINX_EXPECTED_ENGINE'] + '\\n')\n",
    )
    _write_executable(
        tools / "nginx",
        "#!/usr/bin/env python3\n"
        "import os\n"
        "from pathlib import Path\n"
        "import signal\n"
        "import sys\n"
        "import time\n"
        "directory = Path(os.environ['NGINX_FIXTURE_DIR'])\n"
        "scenario = os.environ['NGINX_SCENARIO']\n"
        "resistant = os.environ['NGINX_RESISTANT'] == '1'\n"
        "if '-V' in sys.argv:\n"
        "    print('nginx fixture')\n"
        "    raise SystemExit(0)\n"
        "if '-t' in sys.argv:\n"
        "    raise SystemExit(0)\n"
        "if '-s' in sys.argv:\n"
        "    server_pid = int((directory / 'nginx.pid').read_text())\n"
        "    os.kill(server_pid, signal.SIGQUIT)\n"
        "    raise SystemExit(0)\n"
        "(directory / 'nginx.pid').write_text(str(os.getpid()))\n"
        "def stop(signum, frame):\n"
        "    raise SystemExit(0)\n"
        "if resistant:\n"
        "    signal.signal(signal.SIGQUIT, signal.SIG_IGN)\n"
        "    signal.signal(signal.SIGTERM, signal.SIG_IGN)\n"
        "else:\n"
        "    signal.signal(signal.SIGQUIT, stop)\n"
        "    signal.signal(signal.SIGTERM, stop)\n"
        "while True:\n"
        "    time.sleep(0.1)\n",
    )
    return tools


def _wait_for_probe(marker_path: Path, process: subprocess.Popen[str]) -> int:
    deadline = time.monotonic() + 8
    probe_id = None
    while probe_id is None and time.monotonic() < deadline:
        if marker_path.exists():
            probe_id = int(marker_path.read_text())
        elif process.poll() is not None:
            output, error = process.communicate()
            raise AssertionError(f"Verifier exited before the active probe started: {output}\n{error}")
        else:
            time.sleep(0.02)
    if probe_id is None:
        raise AssertionError("Verifier did not start the active HTTP probe")
    return probe_id


def _run_scenario(repository: Path, scenario: tuple[str, int, signal.Signals | None, bool], real_timeout: str) -> None:
    name, expected_status, interrupt, resistant = scenario
    with tempfile.TemporaryDirectory(prefix="nginx-verify-") as temporary:
        directory = Path(temporary)
        target = directory / "target"
        (target / "sbin").mkdir(parents=True)
        tools = _create_tools(directory)
        shutil.copy2(tools / "nginx", target / "sbin" / "nginx")
        fixture_dir = directory / "fixture"
        fixture_dir.mkdir()
        environment = {
            **os.environ,
            "PATH": os.pathsep.join((str(tools), os.environ.get("PATH", ""))),
            "TARGET_PREFIX": str(target),
            "NJS_VERSION": "1.0.2",
            "NGINX_EXPECTED_ENGINE": _EXPECTED_ENGINE,
            "NGINX_SCENARIO": name,
            "NGINX_RESISTANT": "1" if resistant else "0",
            "NGINX_FIXTURE_DIR": str(fixture_dir),
            "REAL_TIMEOUT": real_timeout,
        }
        verifier = repository / "nginx" / "nginx-verify.sh"
        process = subprocess.Popen(
            ["/bin/sh", str(verifier)],
            env=environment,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            start_new_session=True,
        )
        try:
            if interrupt is not None:
                probe_pid = _wait_for_probe(fixture_dir / "curl.pid", process)
                os.kill(process.pid, interrupt)
            elif name == "http_deadline":
                probe_pid = _wait_for_probe(fixture_dir / "curl.pid", process)
            output, error = process.communicate(timeout=30)
            if process.returncode != expected_status:
                raise AssertionError(
                    f"{name}: exit {process.returncode}, expected {expected_status}\nstdout:\n{output}\nstderr:\n{error}"
                )
            server_pid_path = fixture_dir / "nginx.pid"
            if server_pid_path.exists():
                server_pid = int(server_pid_path.read_text())
                if _process_running(server_pid):
                    raise AssertionError(f"{name}: nginx process {server_pid} is still running")
                if _read_process_state(server_pid) == "Z":
                    raise AssertionError(f"{name}: nginx process {server_pid} was not reaped")
            if interrupt is not None or name == "http_deadline":
                if _process_running(probe_pid):
                    raise AssertionError(f"{name}: active curl process {probe_pid} is still running")
                if _read_process_state(probe_pid) == "Z":
                    raise AssertionError(f"{name}: active curl process {probe_pid} was not reaped")
            print(f"PASS {name}")
        finally:
            _terminate_group(process, fixture_dir)


def _main() -> int:
    real_timeout = shutil.which("timeout") or shutil.which("gtimeout")
    if real_timeout is None:
        raise RuntimeError("This verification requires GNU Coreutils timeout or gtimeout.")
    version = subprocess.run([real_timeout, "--version"], capture_output=True, text=True, check=False)
    if version.returncode != 0 or "GNU coreutils" not in version.stdout:
        raise RuntimeError("This verification requires GNU Coreutils timeout or gtimeout.")
    repository = Path(__file__).resolve().parents[2]
    for scenario in _SCENARIOS:
        _run_scenario(repository, scenario, real_timeout)
    return 0


if __name__ == "__main__":
    sys.exit(_main())
