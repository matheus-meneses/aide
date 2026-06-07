import os
import sys
from datetime import date, datetime
from framework.base import BaseScraper
from framework.models import ScraperEntry
from pathlib import Path
from playwright.sync_api import sync_playwright, BrowserContext, Page, Playwright

SESSIONS_DIR = Path(__file__).parent.parent / ".sessions"
SESSION_FILE = SESSIONS_DIR / "rh_portal.json"
BASE_DOMAIN = "portalrh.bancointer.com.br"
LOGIN_HASH = "#/login"
VACATIONS_PATH = "#/request/notifications/vacation"
ABSENCE_PATH = "#/absence"
BASE_URL_PREFIX = "https://portalrh.bancointer.com.br/FrameHTML/web/app/RH/PortalMeuRH/"


class RHPortalScraper(BaseScraper):
    name = "rh_portal"
    version = "0.1.0"
    categories = ["approval", "absence"]

    def validate_config(self, config: dict) -> None:
        if "base_url" not in config:
            raise ValueError("Missing 'base_url' in config")

    def scrape(self, config: dict) -> list[ScraperEntry]:
        SESSIONS_DIR.mkdir(parents=True, exist_ok=True)
        base_url = config["base_url"]

        username = os.environ.get("AIDE_RH_PORTAL_USERNAME", "")
        password = os.environ.get("AIDE_RH_PORTAL_PASSWORD", "")

        self._log("Starting browser...")
        with sync_playwright() as p:
            context = self._get_context(p)
            page = context.new_page()

            self._log("Navigating to portal...")
            page.goto(base_url, wait_until="domcontentloaded", timeout=60000)
            page.wait_for_timeout(3000)

            if self._needs_login(page):
                if not username or not password:
                    context.close()
                    raise ValueError("Session expired and no credentials available (AIDE_RH_PORTAL_USERNAME/PASSWORD)")
                self._log("Session expired. Logging in with AD credentials...")
                self._do_login(page, username, password)

            if self._needs_login(page):
                context.close()
                raise ValueError("Login failed - still on login page after authentication attempt")

            self._log("Authenticated. Saving session...")
            self._save_session(context)

            entries = []

            self._log("Extracting vacation approvals...")
            vacation_entries = self._extract_vacations(page, config)
            entries.extend(vacation_entries)
            self._log(f"  Found {len(vacation_entries)} vacation approvals")

            self._log("Extracting absences...")
            absence_entries = self._extract_absences(page, config)
            entries.extend(absence_entries)
            self._log(f"  Found {len(absence_entries)} absences")

            self._log(f"Done. {len(entries)} entries collected.")
            context.close()
            return entries

    def _log(self, msg: str) -> None:
        print(msg, file=sys.stderr, flush=True)

    def _get_context(self, p: Playwright) -> BrowserContext:
        browser = p.chromium.launch(headless=True)
        if SESSION_FILE.exists():
            context = browser.new_context(storage_state=str(SESSION_FILE))
        else:
            context = browser.new_context()
        return context

    def _needs_login(self, page: Page) -> bool:
        return LOGIN_HASH in page.url or BASE_DOMAIN not in page.url

    def _do_login(self, page: Page, username: str, password: str) -> None:
        page.wait_for_timeout(2000)

        username_field = page.locator('input[name="user"]')
        password_field = page.locator('input[name="password"]')

        username_field.wait_for(state="visible", timeout=15000)
        username_field.fill(username)

        password_field.wait_for(state="visible", timeout=5000)
        password_field.fill(password)

        page.locator('button.po-button:has-text("Enter")').click()

        page.wait_for_timeout(5000)

        try:
            page.wait_for_function(
                "!window.location.hash.includes('/login')",
                timeout=15000,
            )
        except Exception:
            pass

    def _save_session(self, context: BrowserContext) -> None:
        context.storage_state(path=str(SESSION_FILE))

    def _extract_vacations(self, page: Page, config: dict) -> list[ScraperEntry]:
        page.goto(f"{BASE_URL_PREFIX}{VACATIONS_PATH}", wait_until="domcontentloaded", timeout=30000)
        page.wait_for_timeout(5000)

        entries = []
        rows = page.locator("tr.po-table-row").all()

        if not rows:
            self._log("  No vacation rows found in table")
            return entries

        for row in rows:
            try:
                cells = row.locator("td.po-table-column").all()
                if len(cells) < 5:
                    continue

                cell_texts = [c.inner_text().strip() for c in cells]
                name = cell_texts[0]
                vac_type = cell_texts[1] if len(cell_texts) > 1 else ""
                vesting = cell_texts[2] if len(cell_texts) > 2 else ""
                start = cell_texts[3] if len(cell_texts) > 3 else ""
                end = cell_texts[4] if len(cell_texts) > 4 else ""
                days = cell_texts[5] if len(cell_texts) > 5 else ""

                if not name:
                    continue

                title = f"{name} - {vac_type} ({start} to {end}, {days} days)"
                entry_date = self._parse_date_str(start) or date.today()

                entries.append(ScraperEntry(
                    source="rh_portal",
                    member=name,
                    category="approval",
                    title=title,
                    detail=f"Vesting: {vesting}",
                    entry_date=entry_date,
                    priority="warning",
                    metadata={"name": name, "type": vac_type, "vesting": vesting, "start": start, "end": end,
                              "days": days},
                ))
            except Exception as e:
                self._log(f"  Error parsing vacation row: {e}")

        return entries

    def _extract_absences(self, page: Page, config: dict) -> list[ScraperEntry]:
        page.goto(f"{BASE_URL_PREFIX}{ABSENCE_PATH}", wait_until="domcontentloaded", timeout=30000)
        page.wait_for_timeout(8000)

        entries = []

        cards = page.locator("div.timeline-block").all()
        if not cards:
            self._log("  No absence timeline cards found")
            return entries

        for card in cards:
            try:
                text = card.inner_text().strip()
                if not text or "Vacation balance" not in text:
                    continue

                lines = [l.strip() for l in text.split("\n") if l.strip()]

                name = ""
                role = ""
                status_tag = ""
                balance = ""
                grant_period = ""
                ref_period = ""
                stage = ""

                i = 0
                while i < len(lines):
                    line = lines[i]
                    if i <= 2 and line.isdigit():
                        i += 1
                        continue
                    if i <= 3 and len(line) <= 3 and line.isalpha():
                        i += 1
                        continue

                    if not name and line[0].isalpha() and line == line.title():
                        name = line
                        i += 1
                        continue
                    if not role and name and line == line.upper() and len(line) > 3:
                        role = line
                        i += 1
                        if i < len(lines) and lines[i] == lines[i].upper() and len(lines[i]) > 3 and "BALANCE" not in \
                                lines[i].upper():
                            status_tag = lines[i]
                            i += 1
                        continue
                    if "Vacation balance" in line or "balance" in line.lower():
                        i += 1
                        if i < len(lines):
                            balance = lines[i]
                            i += 1
                        continue
                    if "Grant vacation" in line or "grant" in line.lower():
                        i += 1
                        if i < len(lines):
                            grant_period = lines[i]
                            i += 1
                        continue
                    if "Period referring" in line or "referring" in line.lower():
                        i += 1
                        if i < len(lines):
                            ref_period = lines[i]
                            i += 1
                        continue
                    if "Stage" in line or "stage" in line.lower():
                        i += 1
                        if i < len(lines):
                            stage = lines[i]
                            i += 1
                        continue
                    i += 1

                if not name:
                    continue

                priority = "info"
                if status_tag in ("EXPIRED", "DOUBLE RISK"):
                    priority = "warning"
                elif status_tag == "TO EXPIRE":
                    priority = "warning"

                grant_end = ""
                if "until" in grant_period:
                    parts = grant_period.split("until")
                    grant_end = parts[-1].strip() if len(parts) > 1 else ""

                entry_date = self._parse_date_str(grant_end) or date.today()

                title = f"{name} - {balance} ({status_tag})" if status_tag else f"{name} - {balance}"
                detail = f"Grant: {grant_period} | Ref: {ref_period} | Stage: {stage}"

                entries.append(ScraperEntry(
                    source="rh_portal",
                    member=name,
                    category="absence",
                    title=title,
                    detail=detail,
                    entry_date=entry_date,
                    priority=priority,
                    metadata={
                        "name": name,
                        "role": role,
                        "status": status_tag,
                        "balance": balance,
                        "grant_period": grant_period,
                        "ref_period": ref_period,
                        "stage": stage,
                    },
                ))
            except Exception as e:
                self._log(f"  Error parsing absence card: {e}")

        return entries

    def _parse_date_str(self, s: str) -> date | None:
        for fmt in ("%d/%m/%Y", "%d/%m/%y", "%m/%d/%Y"):
            try:
                return datetime.strptime(s.strip(), fmt).date()
            except ValueError:
                continue
        return None

    def _parse_date_from_cells(self, cells: list[str]) -> date | None:
        for cell in cells:
            result = self._parse_date_str(cell)
            if result:
                return result
        return None
