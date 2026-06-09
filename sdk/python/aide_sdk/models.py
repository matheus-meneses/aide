from __future__ import annotations

from datetime import date
from typing import Any, Literal

from pydantic import BaseModel


class ScraperEntry(BaseModel):
    member: str
    category: Literal["absence", "approval", "metric", "alert", "task", "event"]
    title: str
    detail: str | None = None
    entry_date: date
    priority: Literal["info", "warning", "critical"] = "info"
    link: str | None = None
    metadata: dict[str, Any] | None = None


class TeamMemberEntry(BaseModel):
    name: str
    email: str = ""
    role: str = ""
    department: str = ""
    branch: str = ""
    registration: str = ""
    manager_registration: str = ""


class MetricEntry(BaseModel):
    name: str
    value: float


PluginEntry = ScraperEntry
