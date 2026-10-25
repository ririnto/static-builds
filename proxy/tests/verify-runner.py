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
        "almalinux:"
        + json.loads((repository / "metadata.json").read_text())["nginx-upstream-healthcheck"]["env"][
            "ALMALINUX10_MINIMAL_TAG"
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


if __name__ == "__main__":
    sys.exit(_docker() if Path(sys.argv[0]).name == "docker" else _main())
