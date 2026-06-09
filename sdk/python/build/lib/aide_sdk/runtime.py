import json
import sys
import traceback

_real_stdout = sys.stdout
sys.stdout = sys.stderr


def _emit(response: dict) -> None:
    _real_stdout.write(json.dumps(response, default=str))
    _real_stdout.write("\n")
    _real_stdout.flush()


def serve(scraper_class):
    try:
        raw = sys.stdin.read()
        request = json.loads(raw)
    except Exception as e:
        _emit({"protocol_version": "1", "ok": False, "error": f"failed to parse request: {e}"})
        sys.exit(1)

    action = request.get("action", "scrape")
    config = request.get("config") or {}
    secrets = request.get("secrets") or {}
    scraper = scraper_class()

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
            traceback.print_exc(file=sys.stderr)
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
        items = request.get("items") or []
        try:
            lines = scraper.render(heading, items, config)
        except Exception as e:
            traceback.print_exc(file=sys.stderr)
            _emit({"protocol_version": "1", "ok": False, "error": str(e)})
            sys.exit(1)
        _emit({"protocol_version": "1", "ok": True, "lines": lines})
        return

    if action == "query":
        name = request.get("name", "")
        params = request.get("params") or {}
        try:
            text = scraper.query(name, params, config, secrets)
        except Exception as e:
            traceback.print_exc(file=sys.stderr)
            _emit({"protocol_version": "1", "ok": False, "error": str(e)})
            sys.exit(1)
        _emit({"protocol_version": "1", "ok": True, "text": text})
        return

    _emit({"protocol_version": "1", "ok": False, "error": f"unknown action: {action}"})
    sys.exit(1)
