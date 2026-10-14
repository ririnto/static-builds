"""Verify metadata queries, first-match lookups, and malformed input rejection."""

import json
import os
from pathlib import Path
import subprocess
import tempfile


def _query(repository, metadata, command, argument):
    return subprocess.run(
        ["sh", str(repository / "metadata.sh"), command, argument],
        env={**os.environ, "METADATA_PATH_OVERRIDE": str(metadata)},
        capture_output=True,
        text=True,
        check=False,
    )


def _expect(repository, metadata, command, argument, output):
    result = _query(repository, metadata, command, argument)
    assert result.returncode == 0, result.stderr
    assert result.stdout == output, (command, result.stdout, output)


def _main():
    repository = Path(__file__).resolve().parents[2]
    with tempfile.TemporaryDirectory(prefix="metadata-verification-") as temporary:
        metadata = Path(temporary) / "metadata.json"
        target = {
            "tag_prefix": "fixture",
            "version_env_var": "VERSION",
            "env": {"VERSION": "1.2", "BRANCH": "main"},
            "release_files": ["sbin/fixture", "conf/fixture.conf"],
            "downloads": [{"url": "https://example.invalid/{BRANCH}/{VERSION}", "name": "{VERSION}.tgz"}],
            "unknown": [{"nested": [None, True, False, -1.25e3, "escaped\\value\"/"]}],
        }
        metadata.write_text(json.dumps({"fixture": target, "extended": {**target, "tag_prefix": "fixture-extra"}}))
        _expect(repository, metadata, "get-env", "fixture", "VERSION=1.2\nBRANCH=main\n")
        _expect(repository, metadata, "get-official-version", "fixture", "1.2\n")
        _expect(repository, metadata, "get-tag-prefix", "fixture", "fixture\n")
        _expect(repository, metadata, "get-release-files", "fixture", "sbin/fixture\nconf/fixture.conf\n")
        _expect(repository, metadata, "get-downloads", "fixture", "https://example.invalid/main/1.2\t1.2.tgz\n")
        _expect(repository, metadata, "get-target-from-tag", "fixture-extra-1.2.0", "extended\n")
        assert _query(repository, metadata, "get-env", "absent").returncode == 1

        for template in ("literal{", "literal}", "{VERSION}{BRANCH}", "{VERSION}{"):
            target["downloads"][0]["name"] = template
            metadata.write_text(json.dumps({"fixture": target}))
            expected = template.replace("{VERSION}", "1.2").replace("{BRANCH}", "main")
            _expect(repository, metadata, "get-downloads", "fixture", f"https://example.invalid/main/1.2\t{expected}\n")

        target["downloads"][0]["name"] = "{MISSING}"
        metadata.write_text(json.dumps({"fixture": target}))
        assert "unknown placeholder" in _query(repository, metadata, "get-downloads", "fixture").stderr
        target["downloads"] = []
        metadata.write_text(json.dumps({"fixture": target}))
        _expect(repository, metadata, "get-downloads", "fixture", "")

        metadata.write_text('{"fixture":{"version_env_var":"VERSION","env":{"VERSION":"first","VERSION":"last"},"downloads":[{"url":"{VERSION}","name":"archive"}]}}')
        _expect(repository, metadata, "get-official-version", "fixture", "first\n")
        _expect(repository, metadata, "get-downloads", "fixture", "first\tarchive\n")

        malformed = (
            "{}", "[]", '{"fixture":0}', '{"":{}}', '{"fixture":{',
            '{"fixture":{},}', '{"fixture":{"env":{"A":"1",}}}',
            '{"fixture":{"release_files":["file",]}}',
            '{"fixture":{"downloads":[{"url":"url"}]}}',
            '{"fixture":{"unknown":[1,]}}', '{"fixture":{"unknown":{"A":1,}}}',
            '{"fixture":{"unknown":1e}}', '{"fixture":{"unknown":1.}}',
            '{"fixture":{"unknown":"\\u1234"}}', '{"fixture":{"unknown":"\\q"}}',
            '{"fixture":{"unknown":"unterminated}}', '{"fixture":{}} trailing',
        )
        for content in malformed:
            metadata.write_text(content)
            result = _query(repository, metadata, "get-env", "fixture")
            assert result.returncode == 1, content
            assert "invalid metadata schema" in result.stderr, (content, result.stderr)

        metadata.write_text('{"first":{"tag_prefix":"same"},"second":{"tag_prefix":"same"}}')
        assert "ambiguous tag" in _query(repository, metadata, "get-target-from-tag", "same-1.0.0").stderr
    print("PASS metadata queries, templates, first-match lookups, and malformed input")


if __name__ == "__main__":
    _main()
