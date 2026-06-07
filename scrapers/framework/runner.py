import argparse
import json
import sys
import traceback

from framework.registry import get_scraper


def main():
    parser = argparse.ArgumentParser(description="Scraper framework runner")
    parser.add_argument("source", help="Source name to run")
    parser.add_argument("--config", default="{}", help="JSON config string")
    args = parser.parse_args()

    scraper = get_scraper(args.source)
    if scraper is None:
        print(f"Unknown source: {args.source}", file=sys.stderr)
        sys.exit(1)

    try:
        config = json.loads(args.config)
    except json.JSONDecodeError as e:
        print(f"Invalid config JSON: {e}", file=sys.stderr)
        sys.exit(1)

    try:
        scraper.validate_config(config)
        scraper.authenticate(config)
        entries = scraper.scrape(config)
    except Exception as e:
        traceback.print_exc(file=sys.stderr)
        print(f"Scraper error: {e}", file=sys.stderr)
        sys.exit(1)

    output = [entry.model_dump(mode="json") for entry in entries]
    json.dump(output, sys.stdout, default=str)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
