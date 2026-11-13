"""Behavioral tests for the codeowner assignment sweep."""

from __future__ import annotations

import base64
import json
import re
import threading
import unittest
import urllib.parse
from datetime import datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).parent))
import assign_codeowner as sweep  # noqa: E402

NOW = datetime(2026, 11, 13, 7, 0, tzinfo=timezone.utc)


class FirstChoice:
    def choice(self, choices: list[str]) -> str:
        return choices[0]


def pull(
    number: int,
    *,
    author: str = "external",
    assignees: list[str] | None = None,
    age_days: int = 1,
) -> dict:
    return {
        "number": number,
        "created_at": (NOW - timedelta(days=age_days)).isoformat(),
        "user": {"login": author, "type": "User"},
        "assignees": [{"login": login} for login in (assignees or [])],
        "draft": False,
        "labels": [],
    }


class CoreBehaviorTests(unittest.TestCase):
    def test_codeowners_catch_all_duplicates_and_exclusions(self) -> None:
        groups = sweep.parse_codeowners(
            """
            # A comment is not a rule.
            * @Alice @alice @grafana/tempo
            /docs/ @grafana/docs-traces @Bob
            /tools/ @Bob
            """,
            "@grafana/docs-traces",
        )
        self.assertEqual(groups["user"], {"alice"})
        self.assertEqual(groups["team"], {"grafana/tempo"})
        self.assertEqual(groups["partial"], {"bob"})
        self.assertEqual(groups["excluded"], {"grafana/docs-traces"})

    def test_load_deduplicates_prs_and_ignores_stale_open_load(self) -> None:
        open_prs = [
            pull(1, assignees=["Alice", "Alice"]),
            pull(2, assignees=["Alice"], age_days=91),
            pull(4, author="dependabot[bot]", assignees=["Bob"]),
        ]
        open_prs[2]["user"]["type"] = "Bot"
        closed_prs = [pull(4, assignees=["Alice", "Alice"])]
        load, stale = sweep.calculate_load(
            open_prs, closed_prs, ["alice", "bob"], NOW, 90
        )
        self.assertEqual(load["alice"], {"open": 1, "closed": 1})
        self.assertEqual(load["bob"], {"open": 0, "closed": 0})
        self.assertEqual(stale, {"alice"})

    def test_load_uses_bot_type_not_author_name(self) -> None:
        bot_pr = pull(1, author="renovate[bot]", assignees=["Alice"])
        bot_pr["user"]["type"] = "Bot"
        load, _ = sweep.calculate_load([bot_pr], [], ["alice"], NOW, 90)
        self.assertEqual(load["alice"], {"open": 0, "closed": 0})

    def test_candidates_filter_and_order_oldest_first(self) -> None:
        marked = pull(2)
        marked["labels"] = [{"name": "codeowner-sweep"}]
        bot = pull(6, author="dependabot[bot]")
        bot["user"]["type"] = "Bot"
        deleted_author = pull(4)
        deleted_author["user"] = None
        draft = pull(3)
        draft["draft"] = True
        pulls = [
            pull(7, author="Alice"),
            pull(5, assignees=["Bob"]),
            marked,
            pull(1),
            bot,
            deleted_author,
            draft,
        ]
        candidates = sweep.find_candidates(pulls, ["alice", "bob"], "codeowner-sweep")
        self.assertEqual(
            [(candidate.number, candidate.marked) for candidate in candidates],
            [(1, False), (2, True)],
        )

    def test_assignee_tie_break_prefers_less_open_load_then_randomizes(self) -> None:
        load = {
            "alice": {"open": 1, "closed": 1},
            "bob": {"open": 0, "closed": 2},
            "carol": {"open": 3, "closed": 0},
        }
        self.assertEqual(sweep.choose_assignee(load, "external", FirstChoice()), "bob")
        load["bob"]["open"] = 1
        self.assertEqual(sweep.choose_assignee(load, "external", FirstChoice()), "alice")
        self.assertIsNone(sweep.choose_assignee({"alice": load["alice"]}, "Alice", FirstChoice()))

    def test_only_human_unassign_after_latest_bot_assignment_is_skipped(self) -> None:
        bot = sweep.Config(repo="grafana/tempo").bot_actor
        assigned = {"event": "assigned", "actor": {"login": bot}}
        human_unassigned = {"event": "unassigned", "actor": {"login": "maintainer"}}
        automatic_unassigned = {"event": "unassigned", "actor": None}
        self.assertTrue(sweep.deliberately_unassigned([assigned, human_unassigned], bot))
        self.assertFalse(
            sweep.deliberately_unassigned([human_unassigned, assigned], bot)
        )
        self.assertFalse(
            sweep.deliberately_unassigned([assigned, automatic_unassigned], bot)
        )


class GitHubSimulation:
    def __init__(self) -> None:
        self.repo = "grafana/tempo"
        self.pulls = {20: pull(20), 10: pull(10)}
        self.labels: dict[str, dict] = {}
        self.created_issues: list[dict] = []
        self.comments: list[tuple[int, str]] = []
        self.failure_issue: int | None = None
        self.fail_repo_requests = 1
        self.base_url = ""


class GitHubHandler(BaseHTTPRequestHandler):
    @property
    def simulation(self) -> GitHubSimulation:
        return self.server.simulation  # type: ignore[attr-defined]

    def log_message(self, format: str, *args) -> None:
        pass

    def reply(self, value, status: int = 200, link: str | None = None) -> None:
        data = json.dumps(value).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        if link:
            self.send_header("Link", link)
        self.end_headers()
        self.wfile.write(data)

    def body(self) -> dict:
        return json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))

    def do_GET(self) -> None:
        parsed = urllib.parse.urlsplit(self.path)
        path = parsed.path
        query = urllib.parse.parse_qs(parsed.query)
        sim = self.simulation
        repo_path = f"/repos/{sim.repo}"
        if path == repo_path:
            if sim.fail_repo_requests:
                sim.fail_repo_requests -= 1
                self.reply({"message": "temporary failure"}, 500)
            else:
                self.reply({"default_branch": "main"})
        elif path == f"{repo_path}/contents/CODEOWNERS":
            content = base64.b64encode(b"* @grafana/tempo\n").decode()
            self.reply({"content": content})
        elif path == f"{repo_path}/assignees":
            self.reply([{"login": "Alice"}, {"login": "Bob"}])
        elif path == f"{repo_path}/pulls":
            page = int(query.get("page", ["1"])[0])
            if page == 1:
                link = (
                    f'<{sim.base_url}{path}?state=open&per_page=100&page=2>; rel="next"'
                )
                self.reply([sim.pulls[20]], link=link)
            else:
                self.reply([sim.pulls[10]])
        elif path == "/search/issues":
            if "is:issue" in query["q"][0]:
                items = [{"number": sim.failure_issue}] if sim.failure_issue else []
                self.reply({"total_count": len(items), "items": items})
            else:
                self.reply({"total_count": 0, "items": []})
        elif match := re.fullmatch(rf"{repo_path}/issues/(\d+)", path):
            self.reply(sim.pulls[int(match.group(1))])
        elif path.endswith("/timeline"):
            self.reply([])
        else:
            self.reply({"message": f"unknown GET {path}"}, 404)

    def do_PATCH(self) -> None:
        path = urllib.parse.urlsplit(self.path).path
        sim = self.simulation
        if path == f"/repos/{sim.repo}/labels/codeowner-sweep":
            if path.rsplit("/", 1)[1] not in sim.labels:
                self.reply({"message": "not found"}, 404)
            else:
                self.reply(self.body())
        else:
            self.reply({"message": f"unknown PATCH {path}"}, 404)

    def do_POST(self) -> None:
        path = urllib.parse.urlsplit(self.path).path
        sim = self.simulation
        repo_path = f"/repos/{sim.repo}"
        body = self.body()
        if path == f"{repo_path}/labels":
            sim.labels[body["name"]] = body
            self.reply(body, 201)
        elif match := re.fullmatch(rf"{repo_path}/issues/(\d+)/assignees", path):
            pr = sim.pulls[int(match.group(1))]
            canonical = {"alice": "Alice", "bob": "Bob"}
            for login in body["assignees"]:
                pr["assignees"].append({"login": canonical[login]})
            self.reply(pr)
        elif match := re.fullmatch(rf"{repo_path}/issues/(\d+)/labels", path):
            pr = sim.pulls[int(match.group(1))]
            pr["labels"].extend({"name": name} for name in body["labels"])
            self.reply(pr["labels"])
        elif path == f"{repo_path}/issues":
            sim.created_issues.append(body)
            self.reply({"number": 100 + len(sim.created_issues)}, 201)
        elif match := re.fullmatch(rf"{repo_path}/issues/(\d+)/comments", path):
            sim.comments.append((int(match.group(1)), body["body"]))
            self.reply({"id": len(sim.comments)}, 201)
        else:
            self.reply({"message": f"unknown POST {path}"}, 404)


class EndToEndTests(unittest.TestCase):
    def setUp(self) -> None:
        self.simulation = GitHubSimulation()
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), GitHubHandler)
        self.server.simulation = self.simulation  # type: ignore[attr-defined]
        self.simulation.base_url = f"http://127.0.0.1:{self.server.server_port}"
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.addCleanup(self.server.server_close)
        self.addCleanup(self.server.shutdown)
        self.client = sweep.GitHubClient(
            "test-token", base_url=self.simulation.base_url, sleep=lambda _: None
        )

    def test_sweep_retries_paginates_and_assigns_oldest_pr_up_to_cap(self) -> None:
        config = sweep.Config(
            repo=self.simulation.repo,
            assignee_pool="Alice,Bob",
            max_assignments=1,
        )
        sweep.sweep(self.client, config, now=NOW, rng=FirstChoice())
        self.assertEqual(self.simulation.fail_repo_requests, 0)
        self.assertEqual(
            [item["login"] for item in self.simulation.pulls[10]["assignees"]],
            ["Alice"],
        )
        self.assertEqual(self.simulation.pulls[20]["assignees"], [])
        self.assertEqual(
            self.simulation.pulls[10]["labels"], [{"name": "codeowner-sweep"}]
        )
        self.assertIn("codeowner-sweep", self.simulation.labels)

    def test_sweep_spreads_a_batch_across_tied_owners(self) -> None:
        config = sweep.Config(
            repo=self.simulation.repo,
            assignee_pool="Alice,Bob",
            max_assignments=2,
        )
        sweep.sweep(self.client, config, now=NOW, rng=FirstChoice())
        self.assertEqual(self.simulation.pulls[10]["assignees"], [{"login": "Alice"}])
        self.assertEqual(self.simulation.pulls[20]["assignees"], [{"login": "Bob"}])

    def test_failure_report_creates_then_reuses_an_issue(self) -> None:
        run_url = "https://github.com/grafana/tempo/actions/runs/123"
        sweep.report_failure(self.client, self.simulation.repo, run_url)
        self.assertEqual(len(self.simulation.created_issues), 1)
        self.assertIn(run_url, self.simulation.created_issues[0]["body"])

        self.simulation.failure_issue = 42
        sweep.report_failure(self.client, self.simulation.repo, run_url)
        self.assertEqual(len(self.simulation.created_issues), 1)
        self.assertEqual(self.simulation.comments, [(42, self.simulation.created_issues[0]["body"])])


if __name__ == "__main__":
    unittest.main()
