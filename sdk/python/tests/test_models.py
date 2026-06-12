from datetime import date

from aide_sdk.models import MetricEntry, ScraperEntry, TeamMemberEntry


def test_scraper_entry_minimal():
    entry = ScraperEntry(
        member="alice",
        category="task",
        title="Fix the bug",
        entry_date=date.today(),
        priority="info",
    )
    assert entry.member == "alice"
    assert entry.priority == "info"


def test_team_member_entry():
    member = TeamMemberEntry(
        name="Alice",
        email="alice@example.com",
        role="Engineer",
        department="Engineering",
        branch="Main",
        registration="A001",
        manager_registration="ROOT",
    )
    assert member.name == "Alice"


def test_metric_entry():
    metric = MetricEntry(name="inbox_unread", value=5.0)
    assert metric.value == 5.0
