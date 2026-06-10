from __future__ import annotations

import json
import sys
from datetime import UTC, datetime
from typing import Any

_LEVELS = {"debug": 10, "info": 20, "warn": 30, "error": 40}


def _now() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


class Logger:
    def __init__(self, level: str = "info", fmt: str = "text", scope: str = "") -> None:
        self.threshold = _LEVELS.get(level, 20)
        self.fmt = "json" if fmt == "json" else "text"
        self.scope = scope

    @classmethod
    def from_context(cls, context: dict[str, Any], scope: str = "") -> Logger:
        context = context or {}
        return cls(
            level=str(context.get("log_level", "info")),
            fmt=str(context.get("log_format", "text")),
            scope=scope,
        )

    def _emit(self, level: str, msg: str) -> None:
        if _LEVELS[level] < self.threshold:
            return
        if self.fmt == "json":
            payload: dict[str, Any] = {"ts": _now(), "level": level, "scope": self.scope, "msg": msg}
            if not self.scope:
                payload.pop("scope")
            line = json.dumps(payload, default=str)
        else:
            prefix = f"{self.scope}: " if self.scope else ""
            line = f"{_now()} [{level}] {prefix}{msg}"
        print(line, file=sys.stderr, flush=True)

    def debug(self, msg: str) -> None:
        self._emit("debug", msg)

    def info(self, msg: str) -> None:
        self._emit("info", msg)

    def warning(self, msg: str) -> None:
        self._emit("warn", msg)

    def error(self, msg: str) -> None:
        self._emit("error", msg)
