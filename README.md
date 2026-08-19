# 🔐 VAULT

> A tiny local app that keeps your secrets and environment variables in plain sight.
> One binary, no runtime, no cloud, no database. Everything is just plaintext files on your machine.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00add8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-34d399.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-22d3ee)](#)
[![Port](https://img.shields.io/badge/Port-7575-fbbf24)](#)

**What it does**: stores two kinds of things for you - **Secrets** (passwords, tokens, API keys) and **Env Vars** (key/value pairs with a secret toggle). You manage them in a browser UI at `http://127.0.0.1:7575`, or straight from a terminal with `curl`. For AI tools, generate a token in the UI and read any secret with one `curl` call.

---

## ✨ Features

- **Single binary** - one executable with the UI embedded. No install, no dependencies
- **Terminal-first** - every action works over plain HTTP, so `curl` and terminal AIs can read and write without the UI
- **Origin/Host check** - blocks cross-origin browser requests (the real attack against a localhost app); terminal AIs and curl keep working
- **Strict security headers** - CSP, `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `X-XSS-Protection`
- **HTTP server timeouts** - Read/Write 30s, Idle 120s, MaxHeaderBytes 1MB (slowloris-resistant)
- **Port takeover** - if port 7575 is already in use, vault finds the process, kills it, and takes over
- **Auto-open browser** - launches your default browser 800ms after startup
- **Command palette** (Ctrl+K / Cmd+K) - fuzzy search across secrets, env vars, and actions
- **Live connection indicator** - the SYS LED turns green/amber/red based on API health
- **Two storage types**:
  - **Secrets** - personal entries (name + value + details) for passwords, tokens, API keys
  - **Env Vars** - Render-style environment variables with key/value/secret toggle
- **Dark hacker-terminal UI** - glassmorphism, matrix rain, scanlines, CRT vignette
- **Undo delete** - 5-second undo window after deleting
- **Keyboard shortcuts** - `/`, `?`, `1`, `2`, `N`, `E`, `R`, `Ctrl+K`, `Ctrl+Enter`, `Esc`
- **`.env` import/export** - bulk import env vars from `.env` files, export back to `.env`
- **Auto-mark sensitive** - keys like `PASSWORD`, `TOKEN`, `API_KEY` are auto-marked secret on import
- **Atomic file writes** - write to temp file then rename, prevents corruption
- **Diff-based auto-sync** - multi-tab sync via polling that only re-renders when something actually changed
- **One-click JSON backup** - downloads everything (secrets + env vars, values included) as a single file
- **Password generator** - one click fills a strong 20-character random password in any editor
- **Smarter search** - the filter matches details (secrets) and values (env vars), not just names
- **Command palette copy** - Shift+Enter in the palette copies the selected item's value straight to your clipboard
- **AI access** - generate a token and let Claude, Codex, Cursor, or any script read secrets via a small REST API
- **Graceful shutdown** - Ctrl+C cleanly shuts down the HTTP server

---

## 🚀 Quick Start

### Windows

1. Download `vault.exe` from [Releases](../../releases)
2. Double-click it (Windows may show a SmartScreen warning - click **More info** then **Run anyway**)
3. Your browser opens `http://127.0.0.1:7575` - you're in

### macOS / Linux

Download the matching binary (`vault-darwin-arm64`, `vault-darwin-amd64` or `vault-linux-amd64`) or build from source:

```bash
git clone https://github.com/AshesOfTheUndead/Vault.git
cd Vault
go build -o vault .        # requires Go 1.21+
./vault
```

### Where your data lives

Everything is stored as plaintext files under `~/Vault/`:

```
~/Vault/
├── Secrets/                    # personal secrets
│   ├── github_token/
│   │   ├── value.txt           # the secret value
│   │   ├── details.txt         # optional notes
│   │   └── name.txt            # original name (case-preserved)
│   └── ...
└── EnvVars/                    # environment variables
    ├── DATABASE_URL/
    │   ├── value.txt
    │   ├── secret.txt          # "true" or "false"
    │   └── key.txt
    └── ...
```

Nothing is encrypted, by design. Delete a folder to delete an entry. You can edit the files directly and the UI will pick it up within seconds.

### CLI flags

```
vault.exe              # start on port 7575, opens browser
vault.exe -port 8080   # use a different port
vault.exe -no-browser  # don't auto-open the browser
vault.exe -lan         # bind to all interfaces (phone/Termux access)
vault.exe -host 0.0.0.0# same as -lan
vault.exe -version     # print version and exit
```

### Access from your phone (Android / Termux)

By default the vault only listens on `127.0.0.1`, so your phone can't reach it. Start it in LAN mode:

```bash
vault.exe -lan
```

The startup banner prints your LAN URL(s), e.g. `http://192.168.1.5:7575`.

- **Android browser**: open that URL - the full UI works on mobile
- **Termux**: plain `curl` works with no extra setup:

```bash
curl http://192.168.1.5:7575/api/health
```

- On Windows, the first LAN run asks for a firewall rule - click **Allow**
- Security stays the same in LAN mode: cross-origin requests and unknown hosts still get `403`. There is no login though, so only run `-lan` on a network you trust

---

## 🤖 AI Access

AI tools (Claude Code, Codex, Cursor, any script) can read your secrets over HTTP. Reads are protected by a token you generate in the UI:

1. Click the **AI** button in the toolbar
2. **Generate token** - copy it somewhere safe (it is shown only once)
3. Use it with any AI tool or terminal:

```bash
# list everything the vault knows about
curl -H "Authorization: Bearer TOKEN" http://127.0.0.1:7575/api/ai/list

# read one secret or env var
curl -H "Authorization: Bearer TOKEN" "http://127.0.0.1:7575/api/ai/read?name=Github%20Token"
```

The token is a master key: anyone holding it can read every value, so treat it like a password. Rotate or revoke it anytime from the same panel. Requests are rate-limited per IP (60/min) and every read is logged to the vault's console output.

---

## ⌨️ Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `/` | Focus search |
| `?` | Show keyboard shortcuts help |
| `1` | Switch to Secrets tab |
| `2` | Switch to Env Vars tab |
| `N` | New secret (in Secrets tab) |
| `E` | New env var (in Env Vars tab) |
| `R` | Manual refresh |
| `Ctrl+K` / `Cmd+K` | Open command palette |
| `Shift+Enter` | In palette: copy the selected item's value |
| `Ctrl+Enter` | Save current editor |
| `Esc` | Close editor / modal / palette |

---

## 🔌 API Reference

All endpoints are JSON over HTTP on `127.0.0.1:7575`.

### Secrets

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/list` | List all secrets |
| `GET`  | `/api/get?name=X` | Get one secret's value + details |
| `POST` | `/api/save` | Save a secret `{name, value, details}` |
| `POST` | `/api/delete?name=X` | Delete a secret |

### Env Vars

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/env/list` | List all env vars |
| `GET`  | `/api/env/get?key=X` | Get one env var's value + secret flag |
| `POST` | `/api/env/save` | Save an env var `{key, value, secret}` |
| `POST` | `/api/env/delete?key=X` | Delete an env var |
| `GET`  | `/api/env/export` | Download all env vars as `.env` file |
| `POST` | `/api/env/import` | Import env vars from `.env` text body |

### System

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/health` | Health check (uptime, version) |
| `GET`  | `/api/stats` | Stats (counts, total size, uptime) |
| `GET`  | `/api/version` | Version (go version, platform, port) |
| `GET`  | `/api/backup` | Download everything as one JSON file (attachment) |
| `GET`  | `/api/ai/status` | AI access status (enabled or not) |
| `POST` | `/api/ai/token` | Generate / rotate the AI access token |
| `DELETE` | `/api/ai/token` | Revoke the AI access token |
| `GET`  | `/api/ai/list` | List secret + env var names (requires `Authorization: Bearer TOKEN`) |
| `GET`  | `/api/ai/read` | Read one value: `?name=X` (requires `Authorization: Bearer TOKEN`) |

### Examples

```bash
# list env vars
curl http://127.0.0.1:7575/api/env/list

# save an env var
curl -X POST http://127.0.0.1:7575/api/env/save \
  -H "Content-Type: application/json" \
  -d '{"key":"DATABASE_URL","value":"postgres://user:pass@host/db","secret":true}'

# export as .env
curl -OJ http://127.0.0.1:7575/api/env/export

# import from .env file
curl -X POST http://127.0.0.1:7575/api/env/import \
  --data-binary @.env
```

---

## 🛠️ Troubleshooting

### "The site can't be reached" / nothing loads in the browser

1. Is the vault actually running? Check with:
   ```bash
   curl http://127.0.0.1:7575/api/health
   ```
   If you get a connection error, the vault isn't running. Start it (`vault.exe`) and retry.
2. Are you on the right URL? It's `http://127.0.0.1:7575` (not `https`, not port 80).
3. Browser caching an old page? Hard-refresh with `Ctrl+Shift+R`.

### I get `403 Forbidden`

This is the security middleware doing its job. It happens when the `Host` or `Origin` header isn't one of the addresses the vault serves:

- **From a terminal (curl, AI)**: no `Origin` header is sent, so requests are always allowed. If you're seeing 403 from curl, you're sending a `Host` header that isn't your machine - stop overriding it.
- **From a webpage**: a browser sends `Origin`. It must be the exact address you're browsing on (`http://127.0.0.1:7575` or the LAN URL shown in the banner). A page opened from `https://evil.com` will always get 403 - that's intended.
- **From your phone in LAN mode**: make sure the vault was started with `-lan` **and** you're using the exact LAN URL from the banner. The allowlist only accepts your machine's real interface IPs.

### My phone can't reach the vault (LAN mode)

1. Start the vault with `-lan` and read the banner - use exactly the URL it prints.
2. Make sure phone and PC are on the **same network** (same Wi-Fi, not mobile data).
3. Windows Firewall: on the first `-lan` run you must click **Allow** on the prompt. If you missed it, allow inbound on TCP port 7575 manually:
   ```bash
   netsh advfirewall firewall add rule name="VAULT" dir=in action=allow program="C:\path\to\vault.exe"
   ```
   (run as Administrator)
4. If your PC has both Wi-Fi and Ethernet, the phone must be on the same one as the IP you're using.
5. The allowlist re-scans every 30 seconds, so if you just connected the network, wait a moment and retry.

### Double-clicking `vault.exe` does nothing

- Check the terminal: run `vault.exe` from a command prompt - any error prints there.
- SmartScreen: click **More info** then **Run anyway** (the binary is not code-signed).
- If the port is stuck from a previous run, vault should take it over automatically. If not, kill the old process: `taskkill /F /IM vault.exe`.
- The first launch opens a browser after ~1 second. If it doesn't, open `http://127.0.0.1:7575` manually.

### "Failed to load vault" shown in the UI

The page loaded but the API calls failed. The server process may have crashed, or another program took port 7575. Restart `vault.exe` and refresh. The SYS LED goes red when the API is unreachable.

### The UI looks old / characters look like "Ã¢â‚¬Â¦"

Hard-refresh with `Ctrl+Shift+R` to bust the browser cache. The UI is served fresh on every load, but browsers sometimes keep the old page around.

### Env var values look like garbage when exported as `.env`

Values containing newlines, quotes, spaces or a leading `#` are quoted automatically on export - that's correct `.env` syntax, not a bug.

### Where are my secrets? I want a backup

Everything is plaintext under `~/Vault/`. **Do not sync that folder to cloud storage as-is** - encrypt it first:

```bash
tar czf - ~/Vault | gpg --symmetric --cipher-algo AES256 -o vault-backup.tar.gz.gpg
```

### I want a password/encryption on the vault

It's not built in, on purpose: the vault exists so terminal AIs can read it without a token gate. If you need more:
- **Multi-user / remote access**: add a bearer token + TLS in front
- **Encryption at rest**: layer `gocryptfs` or `VeraCrypt` on top of `~/Vault/`

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────┐
│           vault.exe (Go binary)         │
│  ┌───────────────────────────────────┐  │
│  │   vault.html (embedded via        │  │
│  │   //go:embed directive)          │  │
│  │   - single-file HTML/CSS/JS      │  │
│  │   - glassmorphism + matrix rain  │  │
│  │   - vanilla JS, no build step    │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │   HTTP server (net/http)         │  │
│  │   - port takeover on bind fail   │  │
│  │   - graceful shutdown (SIGINT)   │  │
│  │   - atomic file writes + mutex   │  │
│  │   - timeouts on read/write/idle  │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │   Filesystem storage             │  │
│  │   - ~/Vault/Secrets/{name}/...   │  │
│  │   - ~/Vault/EnvVars/{key}/...    │  │
│  │   - plaintext by design          │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

### Tech Stack

- **Backend**: [Go 1.21+](https://go.dev) - stdlib `net/http`, `embed`, `os`, `os/exec`
- **Frontend**: vanilla HTML/CSS/JS - no framework, no build step
- **Fonts**: Space Grotesk (display), JetBrains Mono (body), Share Tech Mono (terminal labels)
- **Storage**: filesystem, one folder per entry, plaintext files

---

## 🛡️ Security

### Origin/Host check (CSRF protection)
The server blocks **any cross-origin request** coming from a browser. This is the single real attack surface for a localhost app: a malicious webpage making `fetch("http://127.0.0.1:7575/api/delete?name=...")` to delete your vault.

- If the `Origin` header is present, its host must be one the server actually serves: loopback (`127.0.0.1`, `localhost`, `::1`) always, plus your machine's real interface IPs and hostname when running with `-lan`. Anything else gets `403 Forbidden` - matching is case-insensitive and accepts both `host` and `host:port` forms
- The `Host` header is verified against the same allowlist (DNS-rebinding defense), so a request aimed at your LAN IP is only served if that IP is genuinely local
- If `Origin` is absent (terminal AIs, curl, scripts), the request is allowed - terminal workflows keep working
- `Referer` is also checked as a fallback for browsers that omit `Origin`
- LAN IPs are re-scanned every 30 seconds, so NIC reconnects and DHCP renewals can't lock you out

```bash
# terminal (no Origin header) - works
curl http://127.0.0.1:7575/api/health

# same-origin from the vault UI - works
curl -H "Origin: http://127.0.0.1:7575" http://127.0.0.1:7575/api/health

# cross-origin from a malicious page - BLOCKED
curl -H "Origin: https://evil.com" http://127.0.0.1:7575/api/health
# => 403 Forbidden
```

### Security headers
Every response includes:
- `Content-Security-Policy` - strict allowlist (`'self'`, Google Fonts, `data:` URIs)
- `X-Content-Type-Options: nosniff` - no MIME sniffing
- `X-Frame-Options: DENY` - no clickjacking
- `Referrer-Policy: no-referrer` - no referrer leakage
- `X-XSS-Protection: 1; mode=block` - legacy XSS filter

### HTTP server timeouts
Read/Write 30s, Idle 120s, ReadHeader 10s, MaxHeaderBytes 1MB - prevents slowloris-style resource exhaustion.

### Plaintext by design
Values are plaintext on disk, and by default the server binds to `127.0.0.1` only (no remote access). With `-lan` it also listens on your network, with the same origin/host protections - but no authentication, so keep it on a trusted network.

---

## 🤝 Contributing

PRs welcome. Keep it small and focused.

```bash
# setup
git clone https://github.com/AshesOfTheUndead/Vault.git
cd Vault
go build -o vault .

# before submitting, make sure these pass:
gofmt -w main.go
go vet ./...
go build -o vault .
```

---

## 📜 License

MIT - see [LICENSE](LICENSE).

---

## 🙏 Acknowledgments

- [Space Grotesk](https://fonts.google.com/specimen/Space+Grotesk) by Florian Karsten
- [JetBrains Mono](https://www.jetbrains.com/lp/mono/) by JetBrains
- [Share Tech Mono](https://fonts.google.com/specimen/Share+Tech+Mono) by Carrois Apostille
- Inspired by [Render](https://render.com)'s environment variable UI