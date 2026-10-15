"""
Verify release resolution, artifact checks, and publication with isolated fixtures.
"""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


def _gh():
    state_path = Path(os.environ["RELEASE_MOCK_STATE"])
    state = json.loads(state_path.read_text())
    arguments = sys.argv[1:]
    assert arguments[:1] == ["release"], arguments
    operation = arguments[1]
    assert operation in ("view", "create", "upload"), arguments
    state["calls"].append({
        "argv": arguments,
        "token": os.environ.get("GH_TOKEN"),
        "repo": os.environ.get("GH_REPO"),
    })
    state_path.write_text(json.dumps(state))
    return state[operation]


def _run(script, argument, directory, environment):
    return subprocess.run(
        ["sh", str(script), argument],
        cwd=directory,
        env=environment,
        capture_output=True,
        text=True,
        timeout=15,
        check=False,
    )


def _resolve(repository, directory, environment, metadata):
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
    script = repository / ".github/release-resolve.sh"
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
        result = _run(script, tag, directory, {**environment, "GITHUB_OUTPUT": str(output)})
        assert result.returncode == 0, (tag, result.stderr)
        assert output.read_text() == sentinel + f"target={expected}\n", tag
    for tag in (
        "unknown-1.2.3.0", "apache-httpd-1.2.3.0", "nginx", "nginx-1.2.4.0",
        "nginx-1.2.3", "nginx-1.2.3.", "nginx-1.2.3.-1", "nginx-1.2.3.x",
        "nginx-1.2.3-.0", "nginx-1.2.3-beta..rc.0", "nginx-1.2.3-beta_1.0",
    ):
        output.write_text(sentinel)
        result = _run(script, tag, directory, {**environment, "GITHUB_OUTPUT": str(output)})
        assert result.returncode != 0, tag
        assert output.read_text() == sentinel, tag
    metadata.write_text(json.dumps({**targets, "duplicate": target}))
    result = _run(script, "nginx-1.2.3.0", directory, {**environment, "GITHUB_OUTPUT": str(output)})
    assert result.returncode != 0, result
    assert output.read_text() == sentinel
    metadata.write_text(json.dumps(targets))
    for binding in (None, ""):
        missing_output = {key: value for key, value in environment.items() if key != "GITHUB_OUTPUT"}
        if binding is not None:
            missing_output["GITHUB_OUTPUT"] = binding
        result = _run(script, "nginx-1.2.3.0", directory, missing_output)
        assert result.returncode != 0, result
        assert "target=" not in result.stdout, result.stdout
        assert output.read_text() == sentinel
    print("PASS target resolution, tag validation, and output mutation boundaries")


def _verify(repository, directory, environment, metadata, state_path):
    files = directory / ".out/fixture"
    (files / "sbin").mkdir(parents=True)
    (files / "sbin/fixture").write_text("fixture")
    (files / "conf").mkdir()
    (directory / "fixture-1.2.3.0.tar.gz").write_text("package")
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
        state_path.write_text(json.dumps({"view": 0, "create": 0, "upload": 0, "calls": []}))
        result = subprocess.run(
            [
                "sh", "-e", "-c", 'sh "$1" fixture\nsh "$2" fixture-1.2.3.0',
                "release-verification", str(repository / ".github/release-verify.sh"),
                str(repository / ".github/release-publish.sh"),
            ],
            cwd=directory,
            env=environment,
            capture_output=True,
            text=True,
            timeout=15,
            check=False,
        )
        assert result.returncode == expected_status, (release_files, result.stderr)
        calls = json.loads(state_path.read_text())["calls"]
        assert [call["argv"][1] for call in calls] == (["view", "upload"] if expected_status == 0 else []), calls
    metadata.write_text(json.dumps({"other": {"release_files": ["sbin/fixture"]}}))
    result = _run(repository / ".github/release-verify.sh", "fixture", directory, environment)
    assert result.returncode != 0, result
    print("PASS files and directories, missing metadata and paths, and publication gating")


def _publish(repository, directory, environment, state_path):
    tag = "fixture-1.2.3.0"
    package = directory / f"{tag}.tar.gz"
    view = ["release", "view", tag]
    create = ["release", "create", tag, "--title", tag, "--notes", f"Automated static binary release from tag {tag}."]
    upload = ["release", "upload", tag, f"{tag}.tar.gz", "--clobber"]
    for view_status, create_status, upload_status, expected_status, expected_calls in (
        (0, 0, 0, 0, [view, upload]),
        (1, 0, 0, 0, [view, create, upload]),
        (42, 0, 0, 0, [view, create, upload]),
        (1, 43, 0, 43, [view, create]),
        (0, 0, 44, 44, [view, upload]),
        (1, 0, 44, 44, [view, create, upload]),
    ):
        package.write_text("package")
        state_path.write_text(json.dumps({"view": view_status, "create": create_status, "upload": upload_status, "calls": []}))
        result = _run(repository / ".github/release-publish.sh", tag, directory, environment)
        assert result.returncode == expected_status, (expected_calls, result.stderr)
        calls = json.loads(state_path.read_text())["calls"]
        assert calls == [
            {"argv": arguments, "token": "fixture-token", "repo": "fixture/repository"}
            for arguments in expected_calls
        ], calls
    package.unlink()
    state_path.write_text(json.dumps({"view": 0, "create": 0, "upload": 0, "calls": []}))
    result = _run(repository / ".github/release-publish.sh", tag, directory, environment)
    assert result.returncode != 0, result
    assert json.loads(state_path.read_text())["calls"] == []
    print("PASS exact gh requests, environment, fallback, failure statuses, and absent package")


def _main():
    repository = Path(__file__).resolve().parents[2]
    with tempfile.TemporaryDirectory(prefix="release-verification-") as temporary:
        directory = Path(temporary)
        gh = directory / "gh"
        gh.write_text(f"#!{sys.executable}\n" + Path(__file__).read_text())
        gh.chmod(0o755)
        metadata = directory / "metadata.json"
        state_path = directory / "state.json"
        environment = {
            **os.environ,
            "PATH": str(directory) + os.pathsep + os.environ.get("PATH", ""),
            "METADATA_PATH_OVERRIDE": str(metadata),
            "RELEASE_MOCK_STATE": str(state_path),
            "GH_TOKEN": "fixture-token",
            "GH_REPO": "fixture/repository",
        }
        _resolve(repository, directory, environment, metadata)
        _verify(repository, directory, environment, metadata, state_path)
        _publish(repository, directory, environment, state_path)


if __name__ == "__main__":
    sys.exit(_gh() if Path(sys.argv[0]).name == "gh" else _main())
