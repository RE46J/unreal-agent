import contextlib
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import call, patch

sys.path.insert(0, str(Path(__file__).parent))
import prepush


class ChangedPathsTests(unittest.TestCase):
    def test_head_includes_index_worktree_untracked_and_both_rename_paths(self):
        root = Path("/repo")
        with patch.object(prepush.subprocess, "check_output", side_effect=[
            b"old/file.go\0new/file.go\0staged.go\0unstaged.go\0",
            "new/日本語 file.go\0staged.go\0".encode(),
        ]) as output:
            paths = prepush.changed_paths(root)
        self.assertEqual(paths, {
            "old/file.go", "new/file.go", "staged.go", "unstaged.go", "new/日本語 file.go",
        })
        self.assertEqual(output.call_args_list, [
            call(["git", "diff", "--name-only", "--no-renames", "-z", "HEAD", "--"], cwd=root),
            call(["git", "ls-files", "--others", "--exclude-standard", "-z", "--"], cwd=root),
        ])

    def test_base_adds_committed_merge_base_diff_to_worktree(self):
        with patch.object(prepush.subprocess, "check_output", side_effect=[
            b"working.go\0", b"new.go\0", b"committed.go\0working.go\0",
        ]) as output:
            paths = prepush.changed_paths(Path("/repo"), "origin/main")
        self.assertEqual(paths, {"working.go", "new.go", "committed.go"})
        self.assertEqual(output.call_args.args[0][-2:], ["origin/main...HEAD", "--"])

    def test_invalid_base_is_an_error(self):
        with patch.object(prepush.subprocess, "check_output", side_effect=[
            b"", b"", subprocess.CalledProcessError(128, ["git", "diff"]),
        ]), self.assertRaises(subprocess.CalledProcessError):
            prepush.changed_paths(Path("/repo"), "missing")


class PackageSelectionTests(unittest.TestCase):
    def package(self, name, imports=(), **extra):
        return dict(
            Dir=f"/repo/{name}", ImportPath=f"example.test/repo/{name}",
            Module={"Main": True, "Path": "example.test/repo"},
            Imports=list(imports), **extra,
        )

    def test_transitive_production_internal_and_external_test_imports(self):
        leaf = "example.test/repo/leaf"
        packages = [
            self.package("leaf"),
            self.package("direct", [leaf]),
            self.package("transitive", ["example.test/repo/direct"]),
            self.package("internal"), self.package("external"),
            self.package("test_importer", ["example.test/repo/external"]),
            self.package("unrelated"),
            dict(Dir="/repo/internal", ImportPath="example.test/repo/internal [internal.test]",
                 ForTest="example.test/repo/internal", Imports=[leaf]),
            dict(Dir="/repo/external", ImportPath="example.test/repo/external_test [external.test]",
                 ForTest="example.test/repo/external", Imports=[leaf]),
        ]
        selected = prepush.affected_packages(Path("/repo"), {"leaf/leaf.go"}, packages)
        self.assertEqual(selected, [f"example.test/repo/{name}" for name in (
            "direct", "external", "internal", "leaf", "test_importer", "transitive",
        )])

    def test_unknown_or_deleted_go_package_selects_full_suite(self):
        self.assertEqual(prepush.affected_packages(
            Path("/repo"), {"deleted/old.go"}, [self.package("remaining")],
        ), ["./..."])

    def test_multiple_packages_and_import_cycles_terminate(self):
        packages = [self.package("a", ["example.test/repo/b"]),
                    self.package("b", ["example.test/repo/a"]), self.package("c")]
        self.assertEqual(prepush.affected_packages(
            Path("/repo"), {"a/a.go", "c/c_test.go"}, packages,
        ), ["example.test/repo/a", "example.test/repo/b", "example.test/repo/c"])

    def test_non_go_and_test_inputs_use_full_suite_without_listing(self):
        for path in (
            "go.mod", "go.sum", "internal/apigen/go.sum", "Makefile",
            ".github/workflows/ci.yml", "pkg/testdata/input.json", "pkg/testdata/input.go",
            "harness/contextbuilder/prompts/preamble.md", "unknown.dat",
            "benchmarks/harbor/src/harness_harbor/agent.py",
        ):
            with self.subTest(path=path), patch.object(prepush.subprocess, "check_output") as output:
                self.assertEqual(prepush.go_tests(Path("/repo"), {path}), ["./..."])
                output.assert_not_called()

    def test_unchanged_skips_go_test_and_listing(self):
        with patch.object(prepush.subprocess, "check_output") as output:
            self.assertEqual(prepush.go_tests(Path("/repo"), set()), [])
        output.assert_not_called()

    def test_real_go_list_includes_external_tests_and_their_importers(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "go.mod").write_text("module example.test/repo\n\ngo 1.20\n")
            files = {
                "leaf/leaf.go": "package leaf\nconst Value = 1\n",
                "consumer/consumer.go": 'package consumer\nimport _ "example.test/repo/leaf"\n',
                "testonly/testonly.go": "package testonly\n",
                "testonly/external_test.go": (
                    'package testonly_test\nimport (_ "example.test/repo/leaf"; "testing")\n'
                    'func TestExample(t *testing.T) {}\n'
                ),
                "chain/chain.go": 'package chain\nimport _ "example.test/repo/testonly"\n',
                "unrelated/unrelated.go": "package unrelated\n",
            }
            for path, content in files.items():
                target = root / path
                target.parent.mkdir(exist_ok=True)
                target.write_text(content)
            self.assertEqual(prepush.go_tests(root, {"leaf/leaf.go"}), [
                "example.test/repo/chain", "example.test/repo/consumer",
                "example.test/repo/leaf", "example.test/repo/testonly",
            ])

    def test_json_stream_handles_whitespace_and_concatenated_objects(self):
        self.assertEqual(list(prepush.json_objects(' \n{}' + json.dumps({"a": 2}) + '\n')),
                         [{}, {"a": 2}])


class PrepushTests(unittest.TestCase):
    def execute(self, paths, packages, formatted="", uv="/bin/uv"):
        with contextlib.ExitStack() as stack:
            stack.enter_context(patch.object(prepush, "changed_paths", return_value=paths))
            stack.enter_context(patch.object(prepush, "go_tests", return_value=packages))
            stack.enter_context(patch.object(prepush.shutil, "which", return_value=uv))
            stack.enter_context(patch.object(prepush.subprocess, "check_output", return_value=formatted))
            run = stack.enter_context(patch.object(prepush, "run"))
            stack.enter_context(contextlib.redirect_stdout(io.StringIO()))
            prepush.prepush(Path("/repo"), "base")
            return run.call_args_list

    def test_changed_package_checks_and_python_always_run_without_race(self):
        calls = self.execute({"pkg/file.go"}, ["example.test/repo/pkg"])
        self.assertEqual(calls, [
            call(["go", "vet", "-p", "1", "./..."], Path("/repo")),
            call(["go", "test", "-p", "1", "example.test/repo/pkg"], Path("/repo")),
            call(["python3", "-m", "unittest", "discover", "-p", "test_*.py"], Path("/repo/scripts")),
        ])

    def test_unchanged_still_runs_vet_and_python(self):
        calls = self.execute(set(), [])
        self.assertEqual(len(calls), 2)
        self.assertEqual(calls[0].args[0][:2], ["go", "vet"])
        self.assertEqual(calls[1].args[0][:3], ["python3", "-m", "unittest"])

    def test_harbor_checks_and_tests_without_smoke(self):
        calls = self.execute({"benchmarks/harbor/pyproject.toml"}, ["./..."])
        self.assertEqual(calls[-1], call(
            ["make", "-C", "benchmarks/harbor", "check", "test"], Path("/repo"),
        ))

    def test_missing_uv_explains_failure(self):
        with self.assertRaisesRegex(RuntimeError, "harbor changed, but uv is not on PATH"):
            self.execute({"benchmarks/harbor/uv.lock"}, ["./..."], uv=None)

    def test_unformatted_sources_fail(self):
        with self.assertRaisesRegex(RuntimeError, "gofmt is required:\\npkg/file.go"):
            self.execute({"pkg/file.go"}, [], formatted="pkg/file.go\n")

    def test_command_failure_propagates(self):
        with patch.object(prepush.subprocess, "run", side_effect=subprocess.CalledProcessError(1, "go")), \
                contextlib.redirect_stdout(io.StringIO()), self.assertRaises(subprocess.CalledProcessError):
            prepush.run(["go", "test", "-p", "1", "./..."], Path("/repo"))


if __name__ == "__main__":
    unittest.main()
