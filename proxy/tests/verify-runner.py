"""
Verify Docker runner image selection and cleanup with a local CLI fixture and no Docker daemon.
"""

import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from typing import Final


_SCENARIOS: Final = (
    ("success", 0),
    ("load_error", 41),
    ("create_error", 42),
    ("load_signal", 143),
    ("create_signal", 143),
    ("foreign_image", 41),
    ("foreign_container", 42),
    ("suite_error", 17),
)


def _option(arguments, name):
    return arguments[arguments.index(name) + 1]


def _docker():
    state_path = Path(os.environ["HEALTHCHECK_MOCK_STATE"])
    state = json.loads(state_path.read_text())
    arguments = sys.argv[1:]
    scenario = state["scenario"]
    status = 0
    output = ""
    if arguments[:2] == ["buildx", "version"]:
        output = "mock buildx"
    elif arguments[:2] == ["buildx", "build"]:
        state["functional_base_image"] = _option(arguments, "--build-arg").split("=", 1)[1]
        owner = _option(arguments, "--label").split("=", 1)[1]
        state["image"] = {
            "tag": _option(arguments, "--tag"),
            "owner": "foreign" if scenario == "foreign_image" else owner,
        }
        if scenario in ("load_error", "foreign_image"):
            status = 41
    elif arguments[:1] == ["create"]:
        owner = _option(arguments, "--label").split("=", 1)[1]
        state["container"] = {
            "name": _option(arguments, "--name"),
            "owner": "foreign" if scenario == "foreign_container" else owner,
        }
        output = "mock-container-id"
        if scenario in ("create_error", "foreign_container"):
            status = 42
    elif arguments[:2] == ["container", "ls"]:
        container = state["container"]
        if container is not None:
            filters = [arguments[i + 1] for i, value in enumerate(arguments) if value == "--filter"]
            if (
                f'name=^/{container["name"]}$' in filters
                and f'label=static-builds.healthcheck.run={container["owner"]}' in filters
            ):
                output = "mock-container-id"
    elif arguments[:2] == ["image", "ls"]:
        if state["image"] is not None and arguments[-1] == state["image"]["tag"]:
            output = "mock-image-id"
    elif arguments[:2] == ["image", "inspect"]:
        if state["image"] is None:
            status = 1
        else:
            output = state["image"]["owner"]
    elif arguments[:1] == ["rm"]:
        state["container"] = None
    elif arguments[:2] == ["image", "rm"]:
        state["image"] = None
    elif arguments[:1] == ["wait"]:
        output = "17" if scenario == "suite_error" else "0"
    elif arguments[:1] not in (["start"], ["logs"]):
        raise AssertionError(f"Unexpected Docker fixture command: {arguments!r}")
    state_path.write_text(json.dumps(state))
    if (
        scenario == "load_signal" and arguments[:2] == ["buildx", "build"]
        or scenario == "create_signal" and arguments[:1] == ["create"]
    ):
        os.kill(os.getppid(), signal.SIGTERM)
    if output:
        print(output)
    return status


def _daemonless_command():
    directory = Path(os.environ["DAEMONLESS_FIXTURE"])
    name = Path(sys.argv[0]).name
    arguments = sys.argv[1:]
    scenario = os.environ["DAEMONLESS_SCENARIO"]
    if name == "setsid":
        os.setsid()
        os.execvp(arguments[0], arguments)
    if name == "timeout":
        os.execv(shutil.which("gtimeout") or "/usr/bin/timeout", ["timeout", *arguments])
    if name == "go":
        if arguments == ["tool", "buildctl", "--version"]:
            return 0
        if arguments == ["tool", "-n", "buildkitd"]:
            print(directory / "bin/buildkitd")
            return 0
        if arguments[:2] != ["tool", "buildctl"]:
            raise AssertionError(arguments)
        arguments = arguments[2:]
        (directory / ("workers.json" if "workers" in arguments else "client.json")).write_text(json.dumps(arguments))
        if "workers" in arguments:
            if scenario == "startup_failure":
                return 19
            if scenario == "readiness_deadline":
                time.sleep(120)
            if not (directory / "daemon.json").exists():
                return 1
            print("fixture-worker")
            return 0
        (directory / "client.pid").write_text(str(os.getpid()))
        if scenario in ("interrupt_hup", "interrupt_int", "interrupt_term", "escalation", "concurrent"):
            while True:
                time.sleep(1)
        return 37 if scenario == "client_failure" else 0
    if name == "rootlesskit":
        (directory / "rootlesskit.json").write_text(json.dumps(arguments))
        arguments = arguments[3:]
        os.execv(arguments[0], arguments)
    if name == "buildkitd":
        (directory / "daemon.json").write_text(json.dumps(arguments))
        (directory / "daemon.pid").write_text(str(os.getpid()))
        if scenario == "startup_failure":
            return 23
        child = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(120)"])
        (directory / "descendant.pid").write_text(str(child.pid))
        def stop(signum, frame):
            child.terminate()
            child.wait(timeout=5)
            sys.exit(0)
        if scenario == "escalation":
            signal.signal(signal.SIGTERM, signal.SIG_IGN)
        else:
            signal.signal(signal.SIGTERM, stop)
        while True:
            time.sleep(0.1)
            child.poll()
    if name == "runc":
        return 0
    raise AssertionError(name)


def _verify_daemonless(repository):
    with tempfile.TemporaryDirectory(prefix="daemonless-runner-") as temporary:
        root = Path(temporary)
        daemonless = root / "build-daemonless"
        subprocess.run(["go", "build", "-o", str(daemonless), "./cmd/build"], cwd=repository, check=True)
        for scenario, expected in (
            ("success", 0), ("client_failure", 37), ("startup_failure", 1),
            ("interrupt_hup", 129), ("interrupt_int", 130), ("interrupt_term", 143),
            ("escalation", 143), ("readiness_deadline", 1), ("concurrent", 143),
        ):
            instances = []
            for index in range(2 if scenario == "concurrent" else 1):
                directory = root / f"{scenario}-{index}"
                (directory / "bin").mkdir(parents=True)
                (directory / "tmp").mkdir()
                for command in ("go", "rootlesskit", "buildkitd", "setsid", "timeout", "runc"):
                    executable = directory / "bin" / command
                    executable.write_text(f"#!{sys.executable}\n" + Path(__file__).read_text())
                    executable.chmod(0o755)
                environment = {
                    **os.environ,
                    "PATH": str(directory / "bin") + os.pathsep + os.environ["PATH"],
                    "DAEMONLESS_FIXTURE": str(directory), "DAEMONLESS_SCENARIO": scenario,
                    "TMPDIR": str(directory / "tmp"),
                    "BUILD_OUTPUT_DEST": str(directory / "output with spaces"),
                    "CACHE_DIR": str(directory / "cache with spaces"),
                    "BUILDKIT_NETWORK": "host" if scenario == "success" else "none",
                    "BUILDKIT_CACHE_BACKEND": "gha" if scenario == "client_failure" else "local",
                    "BUILDKIT_PLATFORM": "linux/amd64",
                    "BUILDKIT_HOST": "unix:///foreign.sock", "BUILDKITD_FLAGS": "--oci-worker-no-process-sandbox",
                }
                process = subprocess.Popen(
                    [str(daemonless), "nginx"],
                    cwd=repository, env=environment, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
                )
                instances.append((directory, process))
            try:
                if scenario.startswith("interrupt_") or scenario in ("escalation", "concurrent"):
                    deadline = time.monotonic() + 15
                    while not all((directory / "client.pid").exists() for directory, _ in instances) and time.monotonic() < deadline:
                        time.sleep(0.05)
                    for directory, process in instances:
                        if not (directory / "client.pid").exists():
                            raise AssertionError(f"{scenario}: client did not start")
                        process.send_signal({"interrupt_hup": signal.SIGHUP, "interrupt_int": signal.SIGINT}.get(scenario, signal.SIGTERM))
                addresses = []
                for directory, process in instances:
                    stdout, stderr = process.communicate(timeout=80)
                    if process.returncode != expected:
                        raise AssertionError(f"daemonless {scenario}: {process.returncode}, expected {expected}\n{stdout}\n{stderr}")
                    for pidfile in ("daemon.pid", "client.pid", "descendant.pid"):
                        if (directory / pidfile).exists():
                            deadline = time.monotonic() + 5
                            alive = True
                            while alive and time.monotonic() < deadline:
                                try:
                                    os.kill(int((directory / pidfile).read_text()), 0)
                                except ProcessLookupError:
                                    alive = False
                                else:
                                    time.sleep(0.05)
                            if alive:
                                raise AssertionError(f"{scenario}: live {pidfile}")
                    if list((directory / "tmp").iterdir()):
                        raise AssertionError(f"{scenario}: private state leaked")
                    daemon = json.loads((directory / "daemon.json").read_text())
                    rootlesskit = json.loads((directory / "rootlesskit.json").read_text())
                    if rootlesskit[:2] != ["--pidns", "--reaper=true"]:
                        raise AssertionError(f"{scenario}: missing namespace/reaper")
                    address = next(value for value in daemon if value.startswith("--addr="))
                    addresses.append(address)
                    if "--oci-worker-no-process-sandbox" in daemon or "foreign.sock" in address:
                        raise AssertionError(f"{scenario}: inherited daemon override")
                    if (directory / "client.json").exists():
                        client = json.loads((directory / "client.json").read_text())
                        if client[0] != address:
                            raise AssertionError(f"{scenario}: client address mismatch")
                        network = "host" if scenario == "success" else "none"
                        if f"force-network-mode={network}" not in client:
                            raise AssertionError(f"{scenario}: network mapping")
                        if ("network.host" in client) != (scenario == "success") or ("network.host" in daemon) != (scenario == "success"):
                            raise AssertionError(f"{scenario}: entitlement mismatch")
                        if "build-arg:NGINX_VERSION=" + json.loads((repository / "metadata.json").read_text())["nginx"]["env"]["NGINX_VERSION"] not in client:
                            raise AssertionError(f"{scenario}: metadata argument missing")
                    if not (directory / "output with spaces").is_dir():
                        raise AssertionError(f"{scenario}: output removed")
                    if (directory / "cache with spaces").exists() == (scenario == "client_failure"):
                        raise AssertionError(f"{scenario}: cache creation differs from backend")
                if len(addresses) != len(set(addresses)):
                    raise AssertionError("Concurrent invocations shared an address")
                print(f"PASS daemonless {scenario}")
            finally:
                for _, process in instances:
                    if process.poll() is None:
                        process.terminate()
                        process.communicate(timeout=20)


def _main():
    runner = Path(__file__).resolve().with_name("run-docker.sh")
    repository = runner.parents[2]
    expected_functional_base_image = (
        "registry.access.redhat.com/ubi10/ubi-minimal:"
        + json.loads((repository / "metadata.json").read_text())["nginx-upstream-healthcheck"]["env"][
            "UBI10_MINIMAL_VERSION"
        ]
    )
    binary = shutil.which("true")
    if binary is None:
        raise RuntimeError("The true command is required as the executable input fixture.")
    with tempfile.TemporaryDirectory(prefix="healthcheck-runner-") as temporary:
        directory = Path(temporary).resolve()
        docker = directory / "docker"
        docker.write_text(f"#!{sys.executable}\n" + Path(__file__).read_text())
        docker.chmod(0o755)
        state_path = directory / "state.json"
        for scenario, expected_status in _SCENARIOS:
            state_path.write_text(
                json.dumps(
                    {
                        "scenario": scenario,
                        "image": None,
                        "container": None,
                        "functional_base_image": None,
                    }
                )
            )
            environment = {
                **os.environ,
                "PATH": str(directory) + os.pathsep + os.environ.get("PATH", ""),
                "HEALTHCHECK_MOCK_STATE": str(state_path),
                "METADATA_PATH_OVERRIDE": str(repository / "metadata.json"),
            }
            result = subprocess.run(
                ["bash", str(runner), binary],
                env=environment,
                capture_output=True,
                text=True,
                timeout=15,
                check=False,
            )
            if result.returncode != expected_status:
                raise AssertionError(f"{scenario}: exit {result.returncode}, expected {expected_status}\n{result.stderr}")
            if not result.stdout.startswith(f"{docker}\nmock buildx\n"):
                raise AssertionError(f"{scenario}: Docker command-check output is not visible")
            state = json.loads(state_path.read_text())
            if state["functional_base_image"] != expected_functional_base_image:
                raise AssertionError(
                    f"{scenario}: base image {state['functional_base_image']!r}, "
                    f"expected {expected_functional_base_image!r}"
                )
            for resource in ("image", "container"):
                remaining = state[resource]
                foreign_expected = scenario == f"foreign_{resource}"
                if foreign_expected:
                    if remaining is None or remaining["owner"] != "foreign":
                        raise AssertionError(f"{scenario}: foreign {resource} was removed")
                elif remaining is not None:
                    raise AssertionError(f"{scenario}: task-owned {resource} leaked")
            print(f"PASS {scenario}")
    with tempfile.TemporaryDirectory(prefix="rootless-build-output-") as temporary:
        directory = Path(temporary).resolve()
        rootless_repository = directory / "repo"
        rootless_repository.mkdir()
        command = rootless_repository / "build-rootless"
        subprocess.run(["go", "build", "-o", str(command), "./cmd/build"], cwd=repository, check=True)
        shutil.copy2(repository / "metadata.json", rootless_repository / "metadata.json")
        (rootless_repository / "nginx").mkdir()
        (rootless_repository / "nginx/Dockerfile").write_text("FROM scratch\n")
        mock_bin = directory / "bin"
        mock_bin.mkdir()
        for name in ("go", "rootlesskit", "buildkitd", "setsid", "timeout", "runc"):
            executable = mock_bin / name
            executable.write_text(f"#!{sys.executable}\n" + Path(__file__).read_text())
            executable.chmod(0o755)
        exporter_args = directory / "client.json"
        for scenario, destination, expected_status in (
            ("external_absolute", str(directory / "outside_absolute"), 1),
            ("parent_traversal", ".out/nginx/../../escaped-output", 1),
            ("nested_traversal", ".out/nginx/nested/../outside_nested", 1),
            ("default_absolute", None, 0),
            ("relative_target", ".out/nginx", 0),
            ("relative_nested", ".out/nginx/nested", 0),
            ("dot_prefix_nested", "./.out/nginx/nested", 0),
            ("absolute_inside_nested", str(rootless_repository / ".out/nginx/absolute"), 0),
        ):
            environment = {
                **os.environ,
                "PATH": str(mock_bin) + os.pathsep + os.environ.get("PATH", ""),
                "DAEMONLESS_FIXTURE": str(directory),
                "DAEMONLESS_SCENARIO": "success",
                "GITLAB_CI": "true",
                "BUILDKIT_CACHE_BACKEND": "local",
                "BUILDKIT_NETWORK": "default",
                "BUILDKIT_PLATFORM": "linux/amd64",
            }
            environment.pop("BUILD_OUTPUT_DEST", None)
            environment.pop("METADATA_PATH_OVERRIDE", None)
            if destination is None:
                expected_destination = rootless_repository / ".out/nginx"
                expected_destination_argument = str(expected_destination)
            else:
                environment["BUILD_OUTPUT_DEST"] = destination
                expected_destination = Path(destination)
                if not expected_destination.is_absolute():
                    expected_destination = rootless_repository / expected_destination
                expected_destination_argument = destination
            result = subprocess.run(
                [str(command), "nginx"],
                cwd=rootless_repository,
                env=environment,
                capture_output=True,
                text=True,
                timeout=15,
                check=False,
            )
            if result.returncode != expected_status:
                raise AssertionError(f"rootless {scenario}: exit {result.returncode}, expected {expected_status}\n{result.stderr}")
            if expected_status == 1:
                if result.stdout:
                    raise AssertionError(f"rootless {scenario}: command launched before containment rejection")
                if result.stderr != "Error: Build output must be under .out/nginx\n":
                    raise AssertionError(f"rootless {scenario}: missing containment diagnostic\n{result.stderr}")
                if expected_destination.exists():
                    raise AssertionError(f"rootless {scenario}: invalid destination was created")
                if (rootless_repository / ".cache/nginx").exists():
                    raise AssertionError(f"rootless {scenario}: cache directory was created")
                if exporter_args.exists():
                    raise AssertionError(f"rootless {scenario}: exporter was invoked")
            else:
                if not exporter_args.is_file():
                    raise AssertionError(f"rootless {scenario}: exporter was not invoked")
                for name in ("rootlesskit", "setsid", "timeout", "runc"):
                    if str(mock_bin / name) + "\n" not in result.stdout:
                        raise AssertionError(f"rootless {scenario}: command-check output is not visible")
                exporter_arguments = json.loads(exporter_args.read_text())
                output_argument = exporter_arguments[exporter_arguments.index("--output") + 1]
                if output_argument != f"type=local,dest={expected_destination_argument}":
                    raise AssertionError(f"rootless {scenario}: output destination {output_argument!r} is incorrect")
                if not expected_destination.is_dir():
                    raise AssertionError(f"rootless {scenario}: valid destination was not created")
                if not (rootless_repository / ".cache/nginx").is_dir():
                    raise AssertionError(f"rootless {scenario}: cache directory was not created")
                exporter_args.unlink()
            print(f"PASS rootless {scenario}")

    _verify_daemonless(repository)


if __name__ == "__main__":
    if "DAEMONLESS_FIXTURE" in os.environ:
        sys.exit(_daemonless_command())
    sys.exit(_docker() if Path(sys.argv[0]).name == "docker" else _main())
