import contextlib
import io
import json
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import call, patch

sys.path.insert(0, str(Path(__file__).parent))
import verify_release_ci as release_ci


class ReleaseCITests(unittest.TestCase):
    def setUp(self):
        self.output = contextlib.redirect_stdout(io.StringIO())
        self.output.__enter__()
        self.addCleanup(self.output.__exit__, None, None, None)

    def run_info(self, run_id=1, sha="commit", status="completed", conclusion="success"):
        return dict(id=run_id, head_sha=sha, status=status, conclusion=conclusion,
                    html_url=f"https://github.com/owner/repo/actions/runs/{run_id}")

    def test_api_uses_get_and_preserves_head_sha_filter(self):
        with patch.object(release_ci.subprocess, "check_output", return_value=json.dumps({"workflow_runs": []})) as output:
            self.assertEqual(release_ci.api("endpoint", "-f", "head_sha=commit"), {"workflow_runs": []})
        self.assertEqual(output.call_args.args[0], [
            "gh", "api", "--method", "GET", "endpoint", "-f", "head_sha=commit",
        ])

    def test_success_without_wait_or_repeated_completed_queries(self):
        with patch.object(release_ci, "api", return_value={"workflow_runs": [self.run_info()]}) as api, \
                patch.object(release_ci.time, "sleep") as sleep:
            release_ci.wait_for_ci("owner/repo", "commit")
        sleep.assert_not_called()
        self.assertEqual(api.call_count, 1)
        self.assertEqual(api.call_args, call(
            "repos/owner/repo/actions/workflows/ci.yml/runs", "-f", "head_sha=commit", "-f", "per_page=100",
        ))

    def test_waits_through_registration_queue_and_running_until_success(self):
        with patch.object(release_ci, "api", side_effect=[
            {"workflow_runs": []},
            {"workflow_runs": [self.run_info(status="queued", conclusion=None)]},
            self.run_info(status="in_progress", conclusion=None), self.run_info(),
        ]) as api, patch.object(release_ci.time, "sleep") as sleep:
            release_ci.wait_for_ci("owner/repo", "commit")
        self.assertEqual(sleep.call_count, 3)
        self.assertEqual(api.call_args_list[2:], [
            call("repos/owner/repo/actions/runs/1"), call("repos/owner/repo/actions/runs/1"),
        ])

    def test_failure_cancel_and_other_unsuccessful_conclusions_stop_with_reason(self):
        for conclusion in ("failure", "cancelled", "timed_out", "skipped", "neutral", "action_required"):
            with self.subTest(conclusion=conclusion), \
                    patch.object(release_ci, "api", return_value={"workflow_runs": [self.run_info(conclusion=conclusion)]}), \
                    patch.object(release_ci.time, "sleep") as sleep, \
                    self.assertRaisesRegex(RuntimeError, f"CI {conclusion} for commit: https://"):
                release_ci.wait_for_ci("owner/repo", "commit")
            sleep.assert_not_called()

    def test_latest_run_selected_and_other_sha_not_accepted(self):
        with patch.object(release_ci, "api", side_effect=[
            {"workflow_runs": [self.run_info(3, sha="another"), self.run_info(),
                               self.run_info(2, status="queued", conclusion=None)]},
            self.run_info(2, conclusion="cancelled"),
        ]) as api, patch.object(release_ci.time, "sleep"), self.assertRaisesRegex(RuntimeError, "cancelled"):
            release_ci.wait_for_ci("owner/repo", "commit")
        self.assertEqual(api.call_args, call("repos/owner/repo/actions/runs/2"))

    def test_resolves_commit_and_checks_ancestry_before_querying_ci(self):
        root = Path("/repo")
        with patch.object(release_ci.subprocess, "check_output", return_value="peeled-commit\n") as output, \
                patch.object(release_ci.subprocess, "run", return_value=subprocess.CompletedProcess([], 0)) as run, \
                patch.object(release_ci, "wait_for_ci") as wait:
            release_ci.verify(root, "owner/repo")
        output.assert_called_once_with(["git", "rev-parse", "HEAD^{commit}"], cwd=root, text=True)
        run.assert_called_once_with(["git", "merge-base", "--is-ancestor", "peeled-commit", "origin/main"], cwd=root)
        wait.assert_called_once_with("owner/repo", "peeled-commit")

    def test_non_ancestor_stops_before_querying_ci(self):
        with patch.object(release_ci.subprocess, "check_output", return_value="commit\n"), \
                patch.object(release_ci.subprocess, "run", return_value=subprocess.CompletedProcess([], 1)), \
                patch.object(release_ci, "wait_for_ci") as wait, \
                self.assertRaisesRegex(RuntimeError, "not an ancestor of main"):
            release_ci.verify(Path("/repo"), "owner/repo")
        wait.assert_not_called()

    def test_git_or_api_failure_propagates(self):
        with patch.object(release_ci.subprocess, "check_output", return_value="commit\n"), \
                patch.object(release_ci.subprocess, "run", return_value=subprocess.CompletedProcess([], 128)), \
                self.assertRaises(subprocess.CalledProcessError):
            release_ci.verify(Path("/repo"), "owner/repo")
        with patch.object(release_ci, "api", side_effect=subprocess.CalledProcessError(1, "gh")), \
                self.assertRaises(subprocess.CalledProcessError):
            release_ci.wait_for_ci("owner/repo", "commit")


if __name__ == "__main__":
    unittest.main()
