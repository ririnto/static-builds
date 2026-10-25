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
        directory = Path(temporary)
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
        directory = Path(temporary)
        rootless_repository = directory / "repo"
        rootless_repository.mkdir()
        (rootless_repository / ".gitlab").mkdir()
        shutil.copy2(repository / ".gitlab/build-rootless.sh", rootless_repository / ".gitlab/build-rootless.sh")
        for filename in ("build-env.sh", "metadata.sh", "metadata.awk", "metadata.json"):
            shutil.copy2(repository / filename, rootless_repository / filename)
        (rootless_repository / "nginx").mkdir()
        (rootless_repository / "nginx/Dockerfile").write_text("FROM scratch\n")
        mock_bin = directory / "bin"
        mock_bin.mkdir()
        id = mock_bin / "id"
        id.write_text(
            "#!/bin/sh\n"
            "if [ \"$#\" -eq 1 ] && [ \"$1\" = \"-u\" ]; then\n"
            "  echo 1000\n"
            "else\n"
            "  exit 1\n"
            "fi\n"
        )
        id.chmod(0o755)
        buildctl = mock_bin / "buildctl-daemonless.sh"
        buildctl.write_text("#!/bin/sh\nprintf '%s\\n' \"$@\" >\"$BUILDCTL_MOCK_ARGS\"\n")
        buildctl.chmod(0o755)
        exporter_args = directory / "buildctl-arguments.txt"
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
                "BUILDCTL_MOCK_ARGS": str(exporter_args),
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
                ["sh", str(rootless_repository / ".gitlab/build-rootless.sh"), "nginx"],
                cwd=rootless_repository,
                env=environment,
                capture_output=True,
                text=True,
                timeout=15,
                check=False,
            )
            if result.returncode != expected_status:
                raise AssertionError(f"rootless {scenario}: exit {result.returncode}, expected {expected_status}\n{result.stderr}")
            if result.stdout != f"{buildctl}\n":
                raise AssertionError(f"rootless {scenario}: command-check output is not visible")
            if expected_status == 1:
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
                exporter_arguments = exporter_args.read_text().splitlines()
                output_argument = exporter_arguments[exporter_arguments.index("--output") + 1]
                if output_argument != f"type=local,dest={expected_destination_argument}":
                    raise AssertionError(f"rootless {scenario}: output destination {output_argument!r} is incorrect")
                if not expected_destination.is_dir():
                    raise AssertionError(f"rootless {scenario}: valid destination was not created")
                if not (rootless_repository / ".cache/nginx").is_dir():
                    raise AssertionError(f"rootless {scenario}: cache directory was not created")
                exporter_args.unlink()
            print(f"PASS rootless {scenario}")


if __name__ == "__main__":
    sys.exit(_docker() if Path(sys.argv[0]).name == "docker" else _main())
