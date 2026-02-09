---
name: web-login
description: Complete login via magic link on the house-finder web UI. Use when the dev server is running and you need to authenticate to test protected pages.
---

# Web Login

Complete magic link authentication on the house-finder web UI. Uses dev mode console logging to extract the magic link (no real email needed).

## Prerequisites

- Dev server running (`make dev`)
- `HF_ADMIN_EMAIL` and `HF_DEV_MODE=true` set in `.env`
- Playwright venv at `/tmp/pw-venv` (create with `python -m venv /tmp/pw-venv && /tmp/pw-venv/bin/pip install playwright && /tmp/pw-venv/bin/playwright install chromium`)

## Steps

### 1. Get Admin Email

```bash
ADMIN_EMAIL=$(grep HF_ADMIN_EMAIL .env | cut -d= -f2)
```

If empty, ask the user what email to use.

### 2. Run Login via Playwright

Write a Python script using Playwright to:

1. Navigate to `http://localhost:8080/login`
2. Fill the email input and submit
3. Extract the magic link from `make dev-tail`
4. Navigate to the magic link to complete login

### Example Script

```python
import subprocess, re, time
from playwright.sync_api import sync_playwright

BASE = "http://localhost:8080"
EMAIL = "ADMIN_EMAIL_HERE"

def get_magic_link():
    result = subprocess.run(
        ["make", "dev-tail"],
        capture_output=True, text=True, timeout=5,
        cwd="/home/erik/Private/code/github/evcraddock/house-finder"
    )
    output = result.stdout.replace("\n", "")
    match = re.search(r'http://localhost:8080/auth/verify\?token=[a-f0-9]+', output)
    return match.group(0) if match else None

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page()

    # Navigate to login
    page.goto(f"{BASE}/login")
    page.wait_for_load_state("networkidle")

    # Fill and submit
    page.fill("input[type=email]", EMAIL)
    page.click("button[type=submit]")
    page.wait_for_load_state("networkidle")

    # Get magic link from dev logs
    time.sleep(1)
    link = get_magic_link()
    assert link, "No magic link found in dev-tail output"

    # Navigate to magic link — creates session
    page.goto(link)
    page.wait_for_load_state("networkidle")
    assert "/login" not in page.url, "Still on login page"

    page.screenshot(path="/tmp/hf_logged_in.png")
    browser.close()
```

Run with:
```bash
/tmp/pw-venv/bin/python /tmp/test_login.py
```

### Key Selectors

| Element | Selector |
|---------|----------|
| Email input | `input[type=email]` |
| Submit button | `button[type=submit]` |
| Success flash | `.flash` |
| Error flash | `.flash.flash-error` |

### Magic Link Format

```
http://localhost:8080/auth/verify?token=<64-char-hex>
```

Logged in dev mode as: `[DEV] Magic link for <email>: <url>`

## Extracting Magic Link from Logs

The magic link appears in `make dev-tail` output. Use:

```bash
make dev-tail 2>&1 | tr -d '\n' | grep -oP 'http://localhost:8080/auth/verify\?token=[a-f0-9]+' | tail -1
```

**Important:** Use `tr -d '\n'` to join lines — long URLs may wrap in terminal output.

## Troubleshooting

### No magic link in logs

- Confirm `HF_DEV_MODE=true` in `.env`
- Confirm `HF_ADMIN_EMAIL` matches the email entered
- Check `make dev-tail` manually for `[DEV] Magic link` output
- Unknown emails produce no magic link (silent fail by design)

### Token expired or already used

Magic link tokens expire after 15 minutes and are single-use. Submit the form again for a fresh one.

### dev-tail hangs or returns empty

The `dev-tail` target finds the overmind tmux socket automatically. If the dev server was restarted, old sockets may confuse it. Run `make dev-stop` and `make dev` to get a clean start.
