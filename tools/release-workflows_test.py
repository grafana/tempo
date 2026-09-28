"""Exercise release workflow shell steps without tokens, tags, or publication."""

import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest


ROOT = Path(__file__).resolve().parents[1]


def script(workflow, step):
    lines = (ROOT / ".github/workflows" / workflow).read_text().splitlines()
    start = lines.index("      - name: " + step)
    start = lines.index("        run: |", start) + 1
    end = start
    while end < len(lines) and (not lines[end] or lines[end].startswith("          ")):
        end += 1
    return textwrap.dedent("\n".join(lines[start:end]))


class ReleaseWorkflowsTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root / ".chloggen").mkdir()
        self.bin = self.root / "bin"
        self.bin.mkdir()
        # The validation step only reads history and calls changelog validation.
        # Keep these external operations local; exercise the actual workflow shell.
        for command in ("git", "make"):
            path = self.bin / command
            path.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> "$CALLS"\n')
            path.chmod(0o755)
        self.env = {
            **os.environ,
            "PATH": str(self.bin) + os.pathsep + os.environ["PATH"],
            "CALLS": str(self.root / "calls"),
            "GITHUB_OUTPUT": str(self.root / "output"),
        }

    def run_step(self, workflow, step, **env):
        shell = script(workflow, step).replace("/tmp/commits.txt", str(self.root / "commits"))
        return subprocess.run(
            ["bash", "-c", shell],
            cwd=self.root,
            env={**self.env, **env},
            text=True,
            capture_output=True,
        )

    def test_source_branch_selection(self):
        cases = [
            ("3.1.0-rc.0", "", "main", "false"),
            ("3.1.0-rc.2", "main", "main", "false"),
            ("3.1.0-rc.2", "release-v3.1", "release-v3.1", "false"),
            ("3.1.0", "", "release-v3.1", "true"),
            ("3.1.1", "release-v3.1", "release-v3.1", "true"),
            ("3.1.0-rc.2", "release-v3.2", None, None),
            ("3.1.0-rc.2", "feature/foo", None, None),
            ("3.1.0", "main", None, None),
            ("3.1.1", "release-v3.2", None, None),
            ("v3.1.0-rc.2", "", None, None),
        ]
        for version, requested, expected, bump in cases:
            with self.subTest(version=version, requested=requested):
                (self.root / "output").write_text("")
                result = self.run_step(
                    "release-prep.yml", "Validate version and derive release parameters",
                    VERSION=version, REQUESTED_BASE_BRANCH=requested,
                )
                if expected is None:
                    self.assertNotEqual(result.returncode, 0, result.stdout)
                else:
                    self.assertEqual(result.returncode, 0, result.stderr)
                    output = (self.root / "output").read_text().splitlines()
                    self.assertIn("base_branch=" + expected, output)
                    self.assertIn("bump=" + bump, output)

    def test_tag_guard(self):
        for version, branch, allowed in [
            ("3.1.0-rc.2", "main", True),
            ("3.1.0-rc.2", "release-v3.1", True),
            ("3.1.0-rc.2", "release-v3.2", False),
            ("3.1.0-rc.2", "feature/foo", False),
            ("3.1.0", "release-v3.1", True),
            ("3.1.1", "release-v3.1", True),
            ("3.1.0", "main", False),
            ("3.1.1", "release-v3.2", False),
            ("bad", "main", False),
        ]:
            with self.subTest(version=version, branch=branch):
                result = self.run_step(
                    "release-tag.yml", "Verify provenance and parse version",
                    HEAD_REF="release-prep/v" + version, BASE_REF=branch,
                )
                self.assertEqual(result.returncode == 0, allowed, result.stdout + result.stderr)

    def test_prepared_changelog_validation(self):
        section = "# v3.1.0-rc.2\n\n## Bug fixes\n\n- Backported fix\n"
        for content, pending, is_minor, allowed in [
            (section, False, "false", True),
            (section, True, "false", False),
            (section + section, False, "false", False),
            ("# v3.1.0-rc.2\n\n# v3.1.0-rc.1\n- Older fix\n", False, "false", False),
            ("# v3.1.0-rc.1\n- Older fix\n", False, "false", False),
            ("# v3.1.0-rc.1\n- Older fix\n", True, "false", True),
            ("# v3.1.0-rc.1\n- Older fix\n", False, "true", True),
        ]:
            with self.subTest(content=content, pending=pending, is_minor=is_minor):
                (self.root / "CHANGELOG.md").write_text(content)
                entry = self.root / ".chloggen/fix.yaml"
                if pending:
                    entry.write_text("note: fix\n")
                else:
                    entry.unlink(missing_ok=True)
                result = self.run_step(
                    "release-prep.yml", "Validate pending changelog entries",
                    TAG="v3.1.0-rc.2", BASE_BRANCH="release-v3.1", IS_MINOR=is_minor,
                )
                self.assertEqual(result.returncode == 0, allowed, result.stdout + result.stderr)

    def test_prepared_rc_generation_preserves_changelog(self):
        changelog = "# v3.1.0-rc.2\n\n- Backported fix\n"
        (self.root / "CHANGELOG.md").write_text(changelog)
        result = self.run_step(
            "release-prep.yml", "Generate changelog, bump image tag, regenerate jsonnet",
            TAG="v3.1.0-rc.2", IMAGE_TAG="3.1.0", BUMP="false",
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.root / "VERSION").read_text(), "3.1.0-rc.2\n")
        self.assertEqual((self.root / "CHANGELOG.md").read_text(), changelog)
        self.assertFalse((self.root / "calls").exists())


if __name__ == "__main__":
    unittest.main()
