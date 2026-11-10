"""
Verify release resolution, artifact checks, and publication gating with isolated fixtures.
"""

import json
import os
from pathlib import Path
import subprocess
import tempfile


def _run(command, argument, directory, environment):
    return subprocess.run(
        [str(command), argument],
        cwd=directory,
        env=environment,
        capture_output=True,
        text=True,
        timeout=15,
        check=False,
    )


def _resolve(commands, directory, environment, metadata):
    target = {
        "tag_prefix": "nginx",
        "version_env_var": "VERSION",
        "env": {"VERSION": "1.2.3"},
        "release_files": ["sbin/fixture", "conf"],
    }
    targets = {
        "nginx": target,
        "nginx-upstream-healthcheck": {**target, "tag_prefix": "nginx-upstream-healthcheck"},
        "apache-httpd": {**target, "tag_prefix": "httpd"},
    }
    metadata.write_text(json.dumps(targets))
    output = directory / "output"
    sentinel = "existing=value\n"
    for tag, expected in (
        ("nginx-1.2.3.0", "nginx"),
        ("nginx-1.2.3.001", "nginx"),
        ("nginx-1.2.3-beta.0", "nginx"),
        ("nginx-1.2.3-rc1.extra-2.12", "nginx"),
        ("nginx-upstream-healthcheck-1.2.3.0", "nginx-upstream-healthcheck"),
        ("httpd-1.2.3.0", "apache-httpd"),
    ):
        output.write_text(sentinel)
        result = _run(commands / "release-resolve", tag, directory, {**environment, "GITHUB_OUTPUT": str(output)})
        assert result.returncode == 0, (tag, result.stderr)
        assert output.read_text() == sentinel + f"target={expected}\n", tag
    for tag in (
        "unknown-1.2.3.0", "apache-httpd-1.2.3.0", "nginx", "nginx-1.2.4.0",
        "nginx-1.2.3", "nginx-1.2.3.", "nginx-1.2.3.-1", "nginx-1.2.3.x",
        "nginx-1.2.3-.0", "nginx-1.2.3-beta..rc.0", "nginx-1.2.3-beta_1.0",
    ):
        output.write_text(sentinel)
        result = _run(commands / "release-resolve", tag, directory, {**environment, "GITHUB_OUTPUT": str(output)})
        assert result.returncode != 0, tag
        assert output.read_text() == sentinel, tag
    metadata.write_text(json.dumps({**targets, "duplicate": target}))
    result = _run(commands / "release-resolve", "nginx-1.2.3.0", directory, {**environment, "GITHUB_OUTPUT": str(output)})
    assert result.returncode != 0, result
    assert output.read_text() == sentinel
    metadata.write_text(json.dumps(targets))
    for binding in (None, ""):
        missing_output = {key: value for key, value in environment.items() if key != "GITHUB_OUTPUT"}
        if binding is not None:
            missing_output["GITHUB_OUTPUT"] = binding
        result = _run(commands / "release-resolve", "nginx-1.2.3.0", directory, missing_output)
        assert result.returncode != 0, result
        assert "target=" not in result.stdout, result.stdout
        assert output.read_text() == sentinel
    print("PASS target resolution, tag validation, and output mutation boundaries")


def _verify(commands, directory, environment, metadata):
    files = directory / ".out/fixture"
    (files / "sbin").mkdir(parents=True)
    (files / "sbin/fixture").write_text("fixture")
    (files / "conf").mkdir()
    for release_files, expected_status in (
        (["sbin/fixture", "conf"], 0),
        (["absent", "conf"], 1),
        (["sbin/fixture", "absent"], 1),
        ([], 1),
        ([""], 1),
        (None, 1),
    ):
        target = {} if release_files is None else {"release_files": release_files}
        metadata.write_text(json.dumps({"fixture": target}))
        result = _run(commands / "release-verify", "fixture", directory, environment)
        assert result.returncode == expected_status, (release_files, result.stderr)
    metadata.write_text(json.dumps({"other": {"release_files": ["sbin/fixture"]}}))
    result = _run(commands / "release-verify", "fixture", directory, environment)
    assert result.returncode != 0, result
    print("PASS files and directories, missing metadata and paths")


def _package(commands, directory, environment, metadata):
    metadata.write_text(json.dumps({"fixture": {"release_files": ["sbin/fixture", "conf"]}}))
    result = subprocess.run(
        [str(commands / "package-release"), "fixture", "fixture-release"],
        cwd=directory, env=environment, capture_output=True, text=True, check=False,
    )
    assert result.returncode == 0, result.stderr
    members = subprocess.check_output(["tar", "-tzf", str(directory / "fixture-release.tar.gz")], text=True).splitlines()
    assert members == ["sbin/fixture", "conf/"], members
    print("PASS metadata-selected package members and order")


def _main():
    repository = Path(__file__).resolve().parents[2]
    with tempfile.TemporaryDirectory(prefix="release-verification-") as temporary:
        directory = Path(temporary)
        commands = directory / "commands"
        commands.mkdir()
        for name in ("release-resolve", "release-verify", "release-publish", "package-release"):
            subprocess.run(["go", "build", "-o", str(commands / name), "./cmd/" + name], cwd=repository, check=True)
        metadata = directory / "metadata.json"
        environment = {**os.environ, "METADATA_PATH_OVERRIDE": str(metadata)}
        _resolve(commands, directory, environment, metadata)
        _verify(commands, directory, environment, metadata)
        _package(commands, directory, environment, metadata)
        result = _run(commands / "release-publish", "absent", directory, environment)
        assert result.returncode == 1 and "not found" in result.stderr, result
    subprocess.run(["go", "test", "./cmd/release-publish"], cwd=repository, check=True)
    print("PASS publisher API creation, replacement, failure propagation, and absent package")


if __name__ == "__main__":
    _main()
