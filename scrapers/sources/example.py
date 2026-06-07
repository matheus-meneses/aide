from datetime import date

from framework.base import BaseScraper
from framework.models import ScraperEntry


class ExampleScraper(BaseScraper):
    name = "example"
    version = "0.1.0"
    categories = ["task", "event"]

    def validate_config(self, config: dict) -> None:
        if "url" not in config:
            raise ValueError("Missing 'url' in config")

    def scrape(self, config: dict) -> list[ScraperEntry]:
        return [
            ScraperEntry(
                source=self.name,
                member="John Doe",
                category="task",
                title="Complete quarterly review",
                detail="Due by end of week",
                entry_date=date.today(),
                priority="info",
            ),
            ScraperEntry(
                source=self.name,
                member="Jane Smith",
                category="event",
                title="Team sync meeting",
                detail="Scheduled for 3pm",
                entry_date=date.today(),
                priority="info",
            ),
        ]
