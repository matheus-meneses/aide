import os
import requests
import sys
import urllib3
from datetime import date, datetime

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

from framework.base import BaseScraper
from framework.models import ScraperEntry


class JiraScraper(BaseScraper):
    name = "jira"
    version = "0.1.0"
    categories = ["task", "metric"]

    def validate_config(self, config: dict) -> None:
        if "base_url" not in config:
            raise ValueError("Missing 'base_url' in config")
        if "queries" not in config:
            raise ValueError("Missing 'queries' in config")

    def scrape(self, config: dict) -> list[ScraperEntry]:
        base_url = config["base_url"].rstrip("/")
        email = os.environ.get("MGMT_JIRA_EMAIL", "")
        token = os.environ.get("MGMT_JIRA_TOKEN", "")
        if not email or not token:
            raise ValueError(
                "MGMT_JIRA_EMAIL and MGMT_JIRA_TOKEN must be set. "
                "Run: mgmt credential set jira email <email> && mgmt credential set jira token <token>"
            )

        self._log(f"Connecting to Jira at {base_url}...")
        session = requests.Session()
        session.auth = (email, token)
        session.verify = False
        session.headers["Accept"] = "application/json"

        myself = session.get(f"{base_url}/rest/api/3/myself")
        myself.raise_for_status()
        display_name = myself.json().get("displayName", email)
        self._log(f"Authenticated as {display_name}")

        queries = config.get("queries", [])
        entries: list[ScraperEntry] = []

        for q in queries:
            name = q.get("name", "unnamed")
            jql = q.get("jql", "")
            mode = q.get("mode", "items")

            if not jql:
                self._log(f"  Skipping '{name}': empty JQL")
                continue

            self._log(f"  Running query: {name} (mode={mode})...")

            if mode == "metric":
                entries.extend(self._run_metric_query(session, base_url, name, jql))
            else:
                entries.extend(self._run_items_query(session, base_url, name, jql))

        self._log(f"Done. {len(entries)} entries collected.")
        return entries

    def _run_items_query(
            self, session: requests.Session, base_url: str, name: str, jql: str
    ) -> list[ScraperEntry]:
        entries = []
        next_page_token = None

        while True:
            body = {
                "jql": jql,
                "maxResults": 100,
                "fields": ["summary", "assignee", "reporter", "status", "priority", "created", "updated", "issuetype",
                           "project"],
            }
            if next_page_token:
                body["nextPageToken"] = next_page_token

            resp = session.post(f"{base_url}/rest/api/3/search/jql", json=body)
            resp.raise_for_status()
            data = resp.json()

            issues = data.get("issues", [])
            for issue in issues:
                entries.append(self._issue_to_entry(base_url, name, issue))

            if data.get("isLast", True) or not issues:
                break
            next_page_token = data.get("nextPageToken")
            if not next_page_token:
                break

        self._log(f"    {name}: {len(entries)} tickets")
        return entries

    def _run_metric_query(
            self, session: requests.Session, base_url: str, name: str, jql: str
    ) -> list[ScraperEntry]:
        resp = session.post(
            f"{base_url}/rest/api/3/search/approximate-count",
            json={"jql": jql},
        )
        resp.raise_for_status()
        total = resp.json().get("count", 0)
        self._log(f"    {name}: count={total}")

        return [
            ScraperEntry(
                source="jira",
                member="",
                category="metric",
                title=name,
                detail=str(total),
                entry_date=date.today(),
                priority="info",
                metadata={
                    "mode": "metric",
                    "metric_value": total,
                    "jql": jql,
                },
            )
        ]

    def _issue_to_entry(self, base_url: str, query_name: str, issue: dict) -> ScraperEntry:
        fields = issue.get("fields", {})
        key = issue.get("key", "")
        summary = fields.get("summary", "")
        assignee = fields.get("assignee") or {}
        reporter = fields.get("reporter") or {}
        status = fields.get("status", {}).get("name", "")
        priority = fields.get("priority", {}).get("name", "")
        created = fields.get("created", "")
        project = fields.get("project", {}).get("key", "")

        member = assignee.get("displayName", "unassigned")
        browse_url = f"{base_url}/browse/{key}"

        entry_priority = "info"
        if priority and priority.lower() in ("highest", "critical", "blocker"):
            entry_priority = "critical"
        elif priority and priority.lower() in ("high",):
            entry_priority = "warning"

        return ScraperEntry(
            source="jira",
            member=member,
            category="task",
            title=f"[{key}] {summary}",
            detail=f"{project} | {status} | {priority}",
            entry_date=self._parse_date(created),
            priority=entry_priority,
            metadata={
                "mode": "items",
                "web_url": browse_url,
                "query_name": query_name,
                "key": key,
                "status": status,
                "priority": priority,
                "reporter": reporter.get("displayName", ""),
                "created": created,
            },
        )

    def _parse_date(self, iso_str: str) -> date:
        try:
            return datetime.fromisoformat(iso_str.replace("Z", "+00:00")).date()
        except (ValueError, AttributeError, TypeError):
            return date.today()

    def _log(self, msg: str) -> None:
        print(msg, file=sys.stderr, flush=True)
