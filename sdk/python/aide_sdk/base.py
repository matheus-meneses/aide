from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any, ClassVar

from aide_sdk.log import Logger
from aide_sdk.models import MetricEntry, ScraperEntry, TeamMemberEntry


class BaseScraper(ABC):
    name: str = ""
    version: str = "0.1.0"
    categories: ClassVar[list[str]] = []

    def __init__(self) -> None:
        self.log = Logger()
        self.context: dict[str, Any] = {}

    @property
    def verify_ssl(self) -> bool:
        return bool(self.context.get("verify_ssl", True))

    @abstractmethod
    def scrape(self, config: dict[str, Any], secrets: dict[str, Any]) -> list[ScraperEntry]: ...

    def scrape_team(self, config: dict[str, Any], secrets: dict[str, Any]) -> list[TeamMemberEntry]:
        return []

    def scrape_metrics(self, config: dict[str, Any], secrets: dict[str, Any]) -> list[MetricEntry]:
        return []

    def authenticate(self, config: dict[str, Any], secrets: dict[str, Any]) -> None: ...  # noqa: B027

    def validate_config(self, config: dict[str, Any]) -> None: ...  # noqa: B027

    def query(self, name: str, params: dict[str, Any], config: dict[str, Any], secrets: dict[str, Any]) -> str:
        raise NotImplementedError(f"query action '{name}' not implemented")

    def render(self, heading: str, items: list[dict[str, Any]], config: dict[str, Any]) -> list[str]:
        raise NotImplementedError("render action not implemented")
