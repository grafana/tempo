#!/usr/bin/env python3
"""Assign unassigned pull requests to code owners using recent review load."""

from __future__ import annotations

import argparse
import base64
import json
import os
import random
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Any

CODEOWNERS_PATHS = ("CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS")


class APIError(RuntimeError):
    def __init__(self, message: str, status: int | None = None) -> None:
        super().__init__(message)
        self.status = status


class GitHubClient:
    def __init__(
        self,
        token: str,
        base_url: str = "https://api.github.com",
        sleep: Any = time.sleep,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.sleep = sleep
        self.headers = {
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "User-Agent": "tempo-codeowner-sweep",
            "X-GitHub-Api-Version": "2022-11-28",
        }

    def request(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        body: dict[str, Any] | None = None,
    ) -> Any:
        result, _ = self._request(method, path, query=query, body=body)
        return result

    def _request(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        body: dict[str, Any] | None = None,
    ) -> tuple[Any, Any]:
        for attempt in range(1, 4):
            try:
                return self._request_once(method, path, query=query, body=body)
            except APIError:
                if attempt == 3:
                    raise
                self.sleep(attempt * 2)
        raise AssertionError("unreachable")

    def _request_once(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None,
        body: dict[str, Any] | None,
    ) -> tuple[Any, Any]:
        url = (
            path
            if path.startswith(("http://", "https://"))
            else f"{self.base_url}/{path.lstrip('/')}"
        )
        if query:
            url += ("&" if "?" in url else "?") + urllib.parse.urlencode(query)
        data = json.dumps(body).encode() if body is not None else None
        headers = dict(self.headers)
        if data is not None:
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(url, data=data, headers=headers, method=method)

        try:
            with urllib.request.urlopen(request) as response:
                raw = response.read()
                response_headers = response.headers
        except urllib.error.HTTPError as error:
            detail = error.read().decode(errors="replace")[:500]
            raise APIError(
                f"{method} {path} failed with HTTP {error.code}: {detail}", error.code
            ) from error
        except urllib.error.URLError as error:
            raise APIError(f"{method} {path} failed: {error.reason}") from error

        if not raw:
            return None, response_headers
        try:
            return json.loads(raw), response_headers
        except json.JSONDecodeError as error:
            raise APIError(f"{method} {path} returned invalid JSON: {error}") from error

    def paginate(
        self,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        items_key: str | None = None,
    ) -> list[dict[str, Any]]:
        results: list[dict[str, Any]] = []
        next_path: str | None = path
        while next_path:
            page, headers = self._request("GET", next_path, query=query)
            query = None
            if items_key:
                items = page.get(items_key, [])
            elif isinstance(page, list):
                items = page
            else:
                raise APIError(f"GET {next_path} returned a non-list response")
            results.extend(items)
            next_path = None
            for link in urllib.request.parse_http_list(headers.get("Link", "")):
                target, separator, parameters = link.partition(";")
                if separator and 'rel="next"' in parameters:
                    next_path = target.strip().strip("<>")
                    break
        return results


@dataclass(frozen=True)
class Config:
    repo: str
    assignee_pool: str = ""
    excluded_owners: str = ""
    max_assignments: int = 25
    open_load_max_age_days: int = 90
    marker_label: str = "codeowner-sweep"
    bot_actor: str = "github-actions[bot]"

    @classmethod
    def from_env(cls) -> "Config":
        return cls(
            repo=os.environ["REPO"],
            assignee_pool=os.environ.get("ASSIGNEE_POOL", ""),
            excluded_owners=os.environ.get("EXCLUDED_OWNERS", ""),
            max_assignments=int(os.environ.get("MAX_ASSIGNMENTS", "25")),
            open_load_max_age_days=int(os.environ.get("OPEN_LOAD_MAX_AGE_DAYS", "90")),
            marker_label=os.environ.get("MARKER_LABEL", "codeowner-sweep"),
            bot_actor=os.environ.get("BOT_ACTOR", "github-actions[bot]"),
        )


@dataclass(frozen=True)
class Candidate:
    number: int
    author: str
    marked: bool


def logins(value: str, *, strip_leading_slash: bool = False) -> set[str]:
    result = set()
    for value in re.split(r"[,\s]+", value):
        value = value.lower().removeprefix("@")
        if strip_leading_slash:
            value = value.lstrip("/")
        if value:
            result.add(value)
    return result


def parse_codeowners(text: str, excluded: str) -> dict[str, set[str]]:
    counts = {"user": {}, "team": {}}
    catch_all: set[str] = set()
    rules = 0
    for line in text.splitlines():
        fields = line.split()
        if not fields or fields[0].startswith("#"):
            continue
        rules += 1
        seen: set[str] = set()
        for field in fields[1:]:
            if not field.startswith("@"):
                continue
            owner = field[1:].lower()
            if owner in seen:
                continue
            seen.add(owner)
            if fields[0] == "*":
                catch_all.add(owner)
            kind = "team" if "/" in owner else "user"
            counts[kind][owner] = counts[kind].get(owner, 0) + 1

    groups = {name: set() for name in ("user", "team", "partial", "excluded")}
    excluded_owners = logins(excluded, strip_leading_slash=True)
    for kind in ("user", "team"):
        for owner, count in counts[kind].items():
            if owner in excluded_owners:
                groups["excluded"].add(owner)
            elif count == rules or owner in catch_all:
                groups[kind].add(owner)
            else:
                groups["partial"].add(owner)
    return groups


def resolve_pool(client: GitHubClient, config: Config, groups: dict[str, set[str]]) -> list[str]:
    assignable = {
        item["login"].lower()
        for item in client.paginate(
            f"repos/{config.repo}/assignees", query={"per_page": 100}
        )
    }
    if config.assignee_pool:
        pool = logins(config.assignee_pool)
        print("Using ASSIGNEE_POOL for the code owner pool.")
    else:
        pool = set(groups["user"])
        org = config.repo.split("/", 1)[0]
        for team in sorted(groups["team"]):
            slug = team.split("/", 1)[1]
            try:
                members = client.paginate(
                    f"orgs/{org}/teams/{slug}/members", query={"per_page": 100}
                )
            except APIError:
                print(
                    f"::warning::Could not expand team @{team}: listing org team members "
                    "needs an org-scoped token. Set the CODEOWNER_ASSIGNEES repository "
                    "variable to supply the pool directly."
                )
                continue
            member_logins = {member["login"].lower() for member in members}
            pool.update(member_logins)
            print(f"Expanded @{team} to {len(member_logins)} member(s).")

    excluded = logins(config.excluded_owners, strip_leading_slash=True)
    return sorted((pool - excluded) & assignable)


def calculate_load(
    open_prs: list[dict[str, Any]],
    closed_prs: list[dict[str, Any]],
    eligible: list[str],
    now: datetime,
    max_open_age_days: int,
) -> tuple[dict[str, dict[str, int]], set[str]]:
    load = {owner: {"open": 0, "closed": 0} for owner in eligible}
    stale_owners: set[str] = set()
    for state, pulls in (("open", open_prs), ("closed", closed_prs)):
        seen: set[tuple[int, str]] = set()
        for pr in pulls:
            if (pr.get("user") or {}).get("type") == "Bot":
                continue
            for assignee in pr.get("assignees") or []:
                owner = assignee["login"].lower()
                key = (pr["number"], owner)
                if key in seen:
                    continue
                seen.add(key)
                if state == "open":
                    created = datetime.fromisoformat(pr["created_at"].replace("Z", "+00:00"))
                    age = int((now - created).total_seconds() // 86400)
                    if age > max_open_age_days:
                        stale_owners.add(owner)
                        continue
                if owner in load:
                    load[owner][state] += 1
    return load, stale_owners


def find_candidates(
    pulls: list[dict[str, Any]], eligible: list[str], marker_label: str
) -> list[Candidate]:
    owners = set(eligible)
    candidates: dict[int, Candidate] = {}
    for pr in pulls:
        author = pr.get("user")
        if (
            pr.get("assignees")
            or pr.get("draft")
            or not author
            or author.get("type") == "Bot"
            or author["login"].lower() in owners
        ):
            continue
        if pr["number"] not in candidates:
            marked = any(label.get("name") == marker_label for label in pr.get("labels", []))
            candidates[pr["number"]] = Candidate(
                number=pr["number"], author=author["login"], marked=marked
            )
    return sorted(candidates.values(), key=lambda candidate: candidate.number)


def deliberately_unassigned(events: list[dict[str, Any]], bot_actor: str) -> bool:
    last_bot = -1
    last_human = -1
    for index, event in enumerate(events):
        actor = (event.get("actor") or {}).get("login", "")
        if event.get("event") == "assigned" and actor == bot_actor:
            last_bot = index
        elif event.get("event") == "unassigned" and actor and actor != bot_actor:
            last_human = index
    return last_human > last_bot


def choose_assignee(
    load: dict[str, dict[str, int]], author: str, rng: Any
) -> str | None:
    best_key: tuple[int, int] | None = None
    choices: list[str] = []
    for owner in sorted(load):
        if owner == author.lower():
            continue
        counts = load[owner]
        key = (counts["open"] + counts["closed"], counts["open"])
        if best_key is None or key < best_key:
            best_key = key
            choices = [owner]
        elif key == best_key:
            choices.append(owner)
    return rng.choice(choices) if choices else None


def ensure_marker_label(client: GitHubClient, config: Config) -> None:
    label = urllib.parse.quote(config.marker_label, safe="")
    try:
        client.request(
            "PATCH",
            f"repos/{config.repo}/labels/{label}",
            body={
                "color": "ededed",
                "description": "Assigned by the daily code owner sweep",
            },
        )
    except APIError as error:
        if error.status != 404:
            raise
        client.request(
            "POST",
            f"repos/{config.repo}/labels",
            body={
                "name": config.marker_label,
                "color": "ededed",
                "description": "Assigned by the daily code owner sweep",
            },
        )


def read_codeowners(client: GitHubClient, repo: str) -> str:
    default_branch = client.request("GET", f"repos/{repo}")["default_branch"]
    for path in CODEOWNERS_PATHS:
        try:
            result = client.request(
                "GET", f"repos/{repo}/contents/{path}", query={"ref": default_branch}
            )
        except APIError:
            continue
        print(f"Reading {path} from {default_branch}.")
        return base64.b64decode(result["content"]).decode()
    raise RuntimeError("No CODEOWNERS file found at any known location.")


def sweep(
    client: GitHubClient,
    config: Config,
    *,
    now: datetime | None = None,
    rng: Any = None,
) -> None:
    now = now or datetime.now(timezone.utc)
    rng = rng or random.SystemRandom()
    codeowners = read_codeowners(client, config.repo)
    groups = parse_codeowners(codeowners, config.excluded_owners)
    if groups["excluded"]:
        print(
            "::notice::Excluded from the pool by EXCLUDED_OWNERS: "
            + " ".join(sorted(groups["excluded"]))
        )
    if groups["partial"]:
        print(
            "::notice::These CODEOWNERS owners appear in only some rules and are excluded "
            "from the pool: " + " ".join(sorted(groups["partial"]))
        )

    eligible = resolve_pool(client, config, groups)
    if not eligible:
        raise RuntimeError(
            "No assignable code owner was found. Pool: users "
            f"[{' '.join(sorted(groups['user']))}], teams "
            f"[{' '.join(sorted(groups['team']))}]; excluded as partial "
            f"[{' '.join(sorted(groups['partial']))}]. Restore the hardcoded ASSIGNEE_POOL "
            "default, set the CODEOWNER_ASSIGNEES repository variable, or expand a team "
            "with an org-scoped token."
        )
    print(f"Eligible owners: {len(eligible)}.")

    open_prs = client.paginate(
        f"repos/{config.repo}/pulls", query={"state": "open", "per_page": 100}
    )
    since = (now - timedelta(days=7)).strftime("%Y-%m-%dT%H:%M:%SZ")
    closed_query = f"repo:{config.repo} is:pr is:closed closed:>={since}"
    closed_total = client.request(
        "GET", "search/issues", query={"q": closed_query, "per_page": 1}
    )["total_count"]
    if closed_total > 1000:
        print(
            f"::warning::{closed_total} PRs closed in the last seven days, but the search "
            "API returns at most 1000. Closed load is undercounted."
        )
    closed_prs = client.paginate(
        "search/issues", query={"q": closed_query, "per_page": 100}, items_key="items"
    )
    load, stale_owners = calculate_load(
        open_prs, closed_prs, eligible, now, config.open_load_max_age_days
    )
    for owner in sorted(stale_owners):
        print(
            f"Open PRs by {owner} older than {config.open_load_max_age_days} days no longer "
            "count as load.",
            file=sys.stderr,
        )

    # Fetch again because assignments may have changed while load was calculated.
    current_prs = client.paginate(
        f"repos/{config.repo}/pulls", query={"state": "open", "per_page": 100}
    )
    candidates = find_candidates(current_prs, eligible, config.marker_label)
    print(f"Found {len(candidates)} unassigned PR(s).")
    if not candidates:
        return

    try:
        ensure_marker_label(client, config)
    except APIError:
        print(f"::warning::Could not create the {config.marker_label} label.")

    assigned = 0
    skipped = 0
    failed = 0
    unassignable = 0
    for candidate in candidates:
        if assigned >= config.max_assignments:
            print(
                f"Reached the per-run cap of {config.max_assignments}; the rest are left "
                "for the next run."
            )
            break

        if candidate.marked:
            try:
                events = client.paginate(
                    f"repos/{config.repo}/issues/{candidate.number}/timeline",
                    query={"per_page": 100},
                )
            except APIError:
                print(
                    f"::warning::Could not read the timeline for PR #{candidate.number}; "
                    "leaving it for the next run."
                )
                failed += 1
                continue
            relevant = [
                event for event in events if event.get("event") in ("assigned", "unassigned")
            ]
            if deliberately_unassigned(relevant, config.bot_actor):
                print(
                    f"PR #{candidate.number} was auto-assigned before and unassigned by a "
                    "person; skipping."
                )
                skipped += 1
                continue

        assignee = choose_assignee(load, candidate.author, rng)
        if assignee is None:
            print(
                f"::warning::No eligible owner other than @{candidate.author} for PR "
                f"#{candidate.number}."
            )
            unassignable += 1
            continue

        try:
            current = client.request(
                "GET", f"repos/{config.repo}/issues/{candidate.number}"
            )
        except APIError:
            print(f"::warning::Could not re-read PR #{candidate.number} before assigning.")
            failed += 1
            continue
        if current.get("assignees"):
            print(
                f"PR #{candidate.number} was assigned elsewhere while the sweep ran; "
                "leaving it alone."
            )
            skipped += 1
            continue

        try:
            response = client.request(
                "POST",
                f"repos/{config.repo}/issues/{candidate.number}/assignees",
                body={"assignees": [assignee]},
            )
        except APIError:
            print(f"::warning::Could not assign PR #{candidate.number} to @{assignee}.")
            failed += 1
            continue
        returned_assignees = {
            item["login"].lower() for item in response.get("assignees", [])
        }
        if assignee not in returned_assignees:
            print(
                f"::warning::PR #{candidate.number} assignment to @{assignee} was not "
                "reflected in the response."
            )
            failed += 1
            continue

        try:
            client.request(
                "POST",
                f"repos/{config.repo}/issues/{candidate.number}/labels",
                body={"labels": [config.marker_label]},
            )
        except APIError:
            print(
                f"::warning::Could not mark PR #{candidate.number} with "
                f"{config.marker_label}."
            )

        load[assignee]["open"] += 1
        counts = load[assignee]
        print(
            f"Assigned @{assignee} to PR #{candidate.number} (load: "
            f"{counts['open'] + counts['closed']} = {counts['open']} open + "
            f"{counts['closed']} closed in the past seven days)."
        )
        assigned += 1

    print(
        f"Done: assigned {assigned}, skipped {skipped} deliberately unassigned, "
        f"{unassignable} with no eligible owner, {failed} deferred by errors."
    )
    if failed:
        raise RuntimeError(
            f"{failed} PR(s) could not be processed; they will be retried on the next run."
        )


def report_failure(client: GitHubClient, repo: str, run_url: str) -> None:
    title = "assign-codeowner sweep failed"
    body = (
        f"The daily code owner sweep failed.\n\nRun: {run_url}\n\n"
        "This is expected when the code owner pool resolves to nobody: CODEOWNERS may "
        "name teams only, and a team cannot be assigned to a pull request. Set the "
        "CODEOWNER_ASSIGNEES repository variable, or name individuals in CODEOWNERS.\n"
    )
    existing = None
    try:
        results = client.request(
            "GET",
            "search/issues",
            query={"q": f'repo:{repo} is:issue is:open in:title "{title}"'},
        )
        if results.get("items"):
            existing = results["items"][0]["number"]
    except APIError:
        pass

    if existing is not None:
        client.request(
            "POST", f"repos/{repo}/issues/{existing}/comments", body={"body": body}
        )
    else:
        client.request(
            "POST", f"repos/{repo}/issues", body={"title": title, "body": body}
        )


def workflow_error(error: BaseException) -> None:
    message = str(error).replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
    print(f"::error::{message}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", nargs="?", default="sweep", choices=("sweep", "report-failure"))
    args = parser.parse_args()
    try:
        config = Config.from_env()
        client = GitHubClient(os.environ["GH_TOKEN"])
        if args.command == "report-failure":
            report_failure(client, config.repo, os.environ["RUN_URL"])
        else:
            sweep(client, config)
    except (KeyError, RuntimeError, ValueError) as error:
        if isinstance(error, KeyError):
            error = RuntimeError(f"Missing required environment variable: {error.args[0]}")
        workflow_error(error)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
