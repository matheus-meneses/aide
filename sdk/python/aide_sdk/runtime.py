from __future__ import annotations

import json
import os
import sys
import traceback
from typing import Any

from aide_sdk.log import Logger

_real_stdout = sys.stdout
sys.stdout = sys.stderr


def _emit(response: dict[str, Any]) -> None:
    _real_stdout.write(json.dumps(response, default=str))
    _real_stdout.write("\n")
    _real_stdout.flush()


def _log_exc(log: Logger, e: Exception) -> None:
    log.error(f"{type(e).__name__}: {e}")
    log.debug(traceback.format_exc().strip())


def _apply_tls(context: dict[str, Any]) -> None:
    if not bool(context.get("verify_ssl", True)):
        try:
            import urllib3

            urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        except Exception:
            pass
        return

    ca_bundle = str(context.get("ca_bundle") or "")
    if ca_bundle:
        os.environ.setdefault("REQUESTS_CA_BUNDLE", ca_bundle)
        os.environ.setdefault("SSL_CERT_FILE", ca_bundle)
        os.environ.setdefault("CURL_CA_BUNDLE", ca_bundle)
        return

    try:
        import truststore

        truststore.inject_into_ssl()
    except Exception:
        pass


def serve(scraper_class: type) -> None:
    try:
        raw = sys.stdin.read()
        request: dict[str, Any] = json.loads(raw)
    except Exception as e:
        _emit({"protocol_version": "1", "ok": False, "error": f"failed to parse request: {e}"})
        sys.exit(1)

    action = request.get("action", "scrape")
    config: dict[str, Any] = request.get("config") or {}
    secrets: dict[str, Any] = request.get("secrets") or {}
    context: dict[str, Any] = request.get("context") or {}
    _apply_tls(context)
    scraper = scraper_class()
    scraper.context = context
    scraper.log = Logger.from_context(context, scope=getattr(scraper, "name", "") or "")

    if action == "describe":
        _emit({
            "protocol_version": "1",
            "ok": True,
            "text": json.dumps({
                "name": scraper.name,
                "version": scraper.version,
                "categories": scraper.categories,
            }),
        })
        return

    if action == "scrape":
        try:
            scraper.validate_config(config)
            scraper.authenticate(config, secrets)
            entries = scraper.scrape(config, secrets)
            team = scraper.scrape_team(config, secrets)
            metrics = scraper.scrape_metrics(config, secrets)
        except Exception as e:
            _log_exc(scraper.log, e)
            _emit({"protocol_version": "1", "ok": False, "error": str(e)})
            sys.exit(1)

        _emit({
            "protocol_version": "1",
            "ok": True,
            "entries": [e.model_dump(mode="json") for e in entries],
            "team_members": [t.model_dump(mode="json") for t in team],
            "metrics": [m.model_dump(mode="json") for m in metrics],
        })
        return

    if action == "render":
        heading = request.get("heading", "")
        items: list[dict[str, Any]] = request.get("items") or []
        try:
            lines = scraper.render(heading, items, config)
        except Exception as e:
            _log_exc(scraper.log, e)
            _emit({"protocol_version": "1", "ok": False, "error": str(e)})
            sys.exit(1)
        _emit({"protocol_version": "1", "ok": True, "lines": lines})
        return

    if action == "query":
        name = request.get("name", "")
        params: dict[str, Any] = request.get("params") or {}
        try:
            text = scraper.query(name, params, config, secrets)
        except Exception as e:
            _log_exc(scraper.log, e)
            _emit({"protocol_version": "1", "ok": False, "error": str(e)})
            sys.exit(1)
        _emit({"protocol_version": "1", "ok": True, "text": text})
        return

    _emit({"protocol_version": "1", "ok": False, "error": f"unknown action: {action}"})
    sys.exit(1)
