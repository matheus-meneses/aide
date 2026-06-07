import importlib
import sys
from framework.base import BaseScraper
from pathlib import Path

_registry: dict[str, type[BaseScraper]] = {}


def _load_module(module_name: str) -> None:
    try:
        module = importlib.import_module(f"sources.{module_name}")
    except Exception as e:
        print(f"[registry] skipping sources.{module_name}: {e}", file=sys.stderr)
        return

    for attr_name in dir(module):
        attr = getattr(module, attr_name)
        if (
                isinstance(attr, type)
                and issubclass(attr, BaseScraper)
                and attr is not BaseScraper
                and attr.name
        ):
            _registry[attr.name] = attr


def get_scraper(name: str) -> BaseScraper | None:
    if name in _registry:
        return _registry[name]()

    _load_module(name)
    if name in _registry:
        return _registry[name]()

    discover_scrapers()
    cls = _registry.get(name)
    if cls is None:
        return None
    return cls()


def discover_scrapers() -> dict[str, type[BaseScraper]]:
    if _registry:
        return _registry

    sources_path = Path(__file__).parent.parent / "sources"
    if not sources_path.exists():
        return _registry

    for f in sorted(sources_path.glob("*.py")):
        module_name = f.stem
        if module_name.startswith("_"):
            continue
        _load_module(module_name)

    return _registry


def list_scrapers() -> list[str]:
    return list(discover_scrapers().keys())
