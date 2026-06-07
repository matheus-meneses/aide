import sys
from framework.base import BaseScraper
from framework.models import ScraperEntry
from pathlib import Path
from playwright.sync_api import sync_playwright, BrowserContext, Page, Playwright

SESSIONS_DIR = Path(__file__).parent.parent / ".sessions"
SESSION_FILE = SESSIONS_DIR / "rh_management_portal.json"
LOGIN_DOMAIN = "login.microsoftonline.com"
PORTAL_DOMAIN = "rhgestao.guardianrh.com.br"


class RHManagementPortalScraper(BaseScraper):
    name = "rh_management_portal"
    version = "0.1.0"
    categories = ["absence", "event", "task"]

    def validate_config(self, config: dict) -> None:
        if "base_url" not in config:
            raise ValueError("Missing 'base_url' in config")

    def scrape(self, config: dict) -> list[ScraperEntry]:
        SESSIONS_DIR.mkdir(parents=True, exist_ok=True)
        base_url = config["base_url"]

        self._log("Starting browser...")
        with sync_playwright() as p:
            context = self._get_context(p)
            page = context.new_page()
            self._log("Navigating to portal...")
            page.goto(base_url, wait_until="domcontentloaded", timeout=60000)
            page.wait_for_timeout(3000)

            if self._needs_login(page):
                self._log("Session expired. Attempting headless auto-auth...")
                self._try_auto_select_account(page)
                try:
                    page.wait_for_url(f"**/{PORTAL_DOMAIN}/**", timeout=30000)
                    page.wait_for_timeout(3000)
                except Exception:
                    pass

                if self._needs_login(page):
                    self._log("Headless auth failed. Opening visible browser...")
                    context.close()
                    context = self._manual_login(p, base_url)
                    page = context.pages[0] if context.pages else context.new_page()
                else:
                    self._log("Headless auto-auth successful!")

            self._log("Authenticated. Extracting data...")
            self._save_session(context)
            entries = self._extract_data(page, config)
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
        url = page.url
        return LOGIN_DOMAIN in url or PORTAL_DOMAIN not in url

    def _manual_login(self, p: Playwright, base_url: str) -> BrowserContext:
        self._log("Opening browser for authentication (auto-SSO or manual)...")

        browser = p.chromium.launch(headless=False)
        if SESSION_FILE.exists():
            context = browser.new_context(storage_state=str(SESSION_FILE))
        else:
            context = browser.new_context()
        page = context.new_page()
        page.goto(base_url, wait_until="domcontentloaded", timeout=60000)
        page.wait_for_timeout(3000)

        if self._needs_login(page):
            self._try_auto_select_account(page)
            page.wait_for_url(f"**/{PORTAL_DOMAIN}/**", timeout=300000)
            page.wait_for_timeout(3000)

        self._log("Login successful!")
        self._save_session(context)
        return context

    def _try_auto_select_account(self, page: Page) -> None:
        try:
            account_tile = page.locator(
                '[data-test-id="list-item"], .table[role="presentation"] td, [id*="tilesHolder"] > div').first
            if account_tile.is_visible(timeout=3000):
                self._log("  Auto-selecting account in SSO picker...")
                account_tile.click()
                page.wait_for_timeout(2000)
        except Exception:
            pass

    def _save_session(self, context: BrowserContext) -> None:
        context.storage_state(path=str(SESSION_FILE))

    def _extract_data(self, page: Page, config: dict) -> list[ScraperEntry]:
        return []
