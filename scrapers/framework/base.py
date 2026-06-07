from abc import ABC, abstractmethod

from framework.models import ScraperEntry


class BaseScraper(ABC):
    name: str = ""
    version: str = "0.1.0"
    categories: list[str] = []

    @abstractmethod
    def scrape(self, config: dict) -> list[ScraperEntry]:
        ...

    def authenticate(self, config: dict) -> None:
        pass

    def validate_config(self, config: dict) -> None:
        pass
