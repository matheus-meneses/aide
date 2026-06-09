from abc import ABC, abstractmethod

from aide_sdk.models import ScraperEntry, TeamMemberEntry, MetricEntry


class BaseScraper(ABC):
    name: str = ""
    version: str = "0.1.0"
    categories: list[str] = []

    @abstractmethod
    def scrape(self, config: dict, secrets: dict) -> list[ScraperEntry]:
        ...

    def scrape_team(self, config: dict, secrets: dict) -> list[TeamMemberEntry]:
        return []

    def scrape_metrics(self, config: dict, secrets: dict) -> list[MetricEntry]:
        return []

    def authenticate(self, config: dict, secrets: dict) -> None:
        pass

    def validate_config(self, config: dict) -> None:
        pass

    def query(self, name: str, params: dict, config: dict, secrets: dict) -> str:
        raise NotImplementedError(f"query action '{name}' not implemented")

    def render(self, heading: str, items: list[dict], config: dict) -> list[str]:
        raise NotImplementedError("render action not implemented")
