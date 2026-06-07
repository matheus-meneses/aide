from datetime import date
from pydantic import BaseModel
from typing import Literal


class ScraperEntry(BaseModel):
    source: str
    member: str
    category: Literal["absence", "approval", "metric", "alert", "task", "event"]
    title: str
    detail: str | None = None
    entry_date: date
    priority: Literal["info", "warning", "critical"] = "info"
    metadata: dict | None = None
