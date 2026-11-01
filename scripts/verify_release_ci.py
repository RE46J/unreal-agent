#!/usr/bin/env python3
"""Require a tagged commit on main with successful CI before release builds."""

import json
import os
from pathlib import Path
import subprocess
import sys
import time


def api(endpoint, *fields):
    return json.loads(subprocess.check_output(
        ["gh", "api", "--method", "GET", endpoint, *fields], text=True,
    ))


def wait_for_ci(repository, sha):
    endpoint = f"repos/{repository}/actions"
    run = None
    while run is None:
        result = api(
            f"{endpoint}/workflows/ci.yml/runs",
            "-f", f"head_sha={sha}", "-f", "per_page=100",
        )
        runs = [item for item in result["workflow_runs"] if item["head_sha"] == sha]
        if runs:
            run = max(runs, key=lambda item: item["id"])
        else:
            print(f"CI for {sha} is not registered yet; waiting.", flush=True)
            time.sleep(15)
    while True:
        if run["status"] == "completed":
            if run["conclusion"] != "success":
                raise RuntimeError(f"CI {run['conclusion']} for {sha}: {run['html_url']}")
            print(f"CI success for {sha}: {run['html_url']}", flush=True)
            return
        print(f"CI {run['status']} for {sha}: {run['html_url']}; waiting.", flush=True)
        time.sleep(15)
        run = api(f"{endpoint}/runs/{run['id']}")


def verify(root, repository):
    # HEAD^{commit} also handles annotated tags; GITHUB_SHA can name the tag object.
    sha = subprocess.check_output(
        ["git", "rev-parse", "HEAD^{commit}"], cwd=root, text=True,
    ).strip()
    result = subprocess.run(
        ["git", "merge-base", "--is-ancestor", sha, "origin/main"], cwd=root,
    )
    if result.returncode == 1:
        raise RuntimeError(f"Tagged commit {sha} is not an ancestor of main.")
    result.check_returncode()
    wait_for_ci(repository, sha)


def main():
    try:
        verify(Path(__file__).resolve().parent.parent, os.environ["GITHUB_REPOSITORY"])
    except (OSError, RuntimeError, subprocess.CalledProcessError) as error:
        print(f"release CI: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
