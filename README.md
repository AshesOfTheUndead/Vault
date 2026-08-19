# 🔐 VAULT

> Secure local secrets and environment variables, kept in plain sight.
> A single self-contained binary. No runtime. No cloud. Just plaintext files on your machine.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00add8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-34d399.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-22d3ee)](#)
[![Port](https://img.shields.io/badge/Port-7575-fbbf24)](#)
[![Version](https://img.shields.io/badge/version-3.1.2-34d399)](#)

---

## ✨ Features

- **Single binary** - Go binary with embedded HTML, no runtime dependencies
- **Origin/Host check** - blocks cross-origin browser requests (the real attack against a localhost app); terminal AIs and curl keep working
- **DNS-rebinding defense** - Host header allowlist (only loopback + LAN IPs in `--lan` mode)
- **Strict security headers** - CSP, `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `X-XSS-Protection`
- **HTTP server timeouts** - Read/Write 30s, Idle 120s, MaxHeaderBytes 1MB (slowloris-resistant)
- **AI access tokens** - generate 32-byte hex tokens that expire after 1, 7 or 30 days (or never), each shown with its exact expiry timestamp in the UI; constant-time compare, rate-limited 60 req/min per IP, per-token revoke
- **Audit log** - every secret/env save/delete and every AI read is logged to `~/Vault/audit.log` with timestamp + source IP; viewable in the Access History modal
- **LAN mode** (`--lan` or `--host 0.0.0.0`) - exposes vault on your local network so phones/Termux can reach it; auto-scans NIC IPs every 30s for DHCP/reconnect resilience
- **Port takeover** - if port 7575 is in use, vault finds the process, kills it, and takes over
- **Auto-open browser** - launches your default browser 800ms after startup
- **Command palette** (Ctrl+K / Cmd+K) - fuzzy search across secrets, env vars, and actions; arrow-key navigation
- **Live connection status indicator** - sys-chip LED turns green/amber/red based on API health
- **Password strength meter** - real-time weak/fair/good/strong indicator below value fields
- **Clipboard auto-clear** - copied secrets auto-clear from clipboard after 30 seconds
- **Recently accessed** - last 6 accessed entries shown as quick-open chips above the list
- **Two storage types**:
  - **Secrets** - personal entries (name + value + details) for passwords, tokens, API keys
  - **Env Vars** - Render-style environment variables with key/value/secret toggle
- **Glass morphism UI** - premium dark hacker-terminal aesthetic with matrix rain, scanlines, and CRT vignette
- **Smooth animations** - letter-by-letter brand reveal, count-up stats, staggered row entrances, smooth tab transitions
- **Delete confirmation modal** - no accidental deletes
- **Undo delete** - 5-second undo window via toast action button
- **One-click backup** - download all secrets + env vars as a single JSON file
- **`.env` import/export** - bulk import env vars from `.env` files, export back to `.env`
- **Auto-mark sensitive** - keys like `PASSWORD`, `TOKEN`, `API_KEY` auto-marked as secret on import
- **Atomic file writes** - write to temp file then rename, prevents corruption
- **Request logging** - colorized per-request log (method, path, status, duration)
- **Graceful shutdown** - Ctrl+C cleanly shuts down the HTTP server
- **Auto-migration** - upgrades old flat-file format to new folder structure on first run
- **4-second auto-refresh** - multi-tab sync via polling
- **Keyboard shortcuts** - full keyboard navigation (`/`, `?`, `1`, `2`, `N`, `E`, `R`, `Ctrl+K`, `Ctrl+Enter`, `Esc`)
- **Health, stats, version, and audit endpoints** - `/api/health`, `/api/stats`, `/api/version`, `/api/audit`

---

## 🚀 Quick Start

### Option A: Download the binary

Pre-built binaries for all platforms are in the [Releases](../../releases) page:

| File | Platform |
|------|----------|
| `vault.exe` | Windows (amd64) |
| `vault-linux-amd64` | Linux (amd64) |
| `vault-darwin-amd64` | macOS (Intel) |
| `vault-darwin-arm64` | macOS (Apple Silicon) |

1. Download the right binary for your OS
2. Rename to `vault` (or `vault.exe` on Windows) if you like
3. On macOS/Linux, mark as executable: `chmod +x vault-darwin-arm64`
4. Double-click (or `./vault-darwin-arm64` in terminal)
5. Browser opens to `http://127.0.0.1:7575`

### Option B: Build from source

```bash
# requires Go 1.21+
git clone https://github.com/AshesOfTheUndead/Vault.git
cd vault

# build for your current platform
go build -o vault .

# build for Windows from any platform
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o vault.exe .

# run
./vault           # linux/mac
vault.exe         # windows
```

Or use the build scripts:

```bash
./build.sh        # linux/mac
build.bat         # windows
```

### CLI flags

```
vault.exe -port 8080           # use a different port
vault.exe -version              # print version
vault.exe -no-browser           # don't auto-open browser
VAULT_ALLOW_HOSTS=api.example.com vault.exe --lan   # allow a tunnel/reverse-proxy hostname (comma-separated)
```

---

## 📁 Storage Layout

Everything lives under `~/Vault/` (i.e. `C:\Users\<you>\Vault\` on Windows):

```
~/Vault/
├── Secrets/                    # personal secrets
│   ├── roblox/
│   │   ├── value.txt           # the secret value
│   │   ├── details.txt         # optional notes
│   │   └── name.txt            # original name (case-preserved)
│   └── github_token/
│       └── ...
└── EnvVars/                    # environment variables
    ├── DATABASE_URL/
    │   ├── value.txt
    │   ├── secret.txt          # "true" or "false"
    │   └── key.txt
    └── PORT/
        └── ...
```

Nothing is encrypted - by design. Delete a folder to delete an entry. Edit files directly if you want.

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
| `Ctrl+Enter` | Save current editor |
| `Esc` | Close editor / modal / palette |

---

## 🔌 API Reference

All endpoints are JSON over HTTP, listening on `127.0.0.1:7575`.

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
| `GET`  | `/api/stats` | Stats (counts, total size, uptime, AI status, LAN status) |
| `GET`  | `/api/version` | Version (go version, platform, port, LAN mode) |
| `GET`  | `/api/audit` | Last 100 audit log entries |
| `GET`  | `/api/backup` | One-click JSON backup of all secrets + env vars |

### AI Access (token-protected, no Origin check - works from curl / AI tools)

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/ai/status` | List all tokens with created / expiry / revoked state |
| `POST` | `/api/ai/token` | Generate a token, `{"days": 1, 7, 30, or 0 = never}` (default 7) |
| `DELETE` | `/api/ai/token?id=X` | Revoke a specific token |
| `GET`  | `/api/ai/list` | List all entry names (bearer token required, rate-limited) |
| `GET`  | `/api/ai/read?name=X` | Read a secret or env var value (bearer token required, rate-limited) |

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

# one-click backup (all secrets + env vars as JSON)
curl -OJ http://127.0.0.1:7575/api/backup

# view access history (last 100 events)
curl http://127.0.0.1:7575/api/audit
```

### AI access from terminal

```bash
# 1. generate a token (in the vault UI, click the AI button → Generate token)

# 2. list all entries
curl -H "Authorization: Bearer YOUR_TOKEN" http://127.0.0.1:7575/api/ai/list

# 3. read a specific secret
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://127.0.0.1:7575/api/ai/read?name=roblox"
```

### LAN mode (phone access)

```bash
# expose on your local network
vault.exe --lan
# or: vault.exe --host 0.0.0.0

# the terminal banner prints all LAN URLs:
#   lan    http://192.168.1.50:7575
#   lan    http://192.168.1.51:7575
# open one on your phone - the vault auto-allows your NIC IPs
```

### Exposing to a cloud AI (tunnel)

To let a web/cloud AI (with code execution) reach the vault, put it behind an HTTPS tunnel and tell the vault to accept the tunnel's hostname:

```bash
# 1. tunnel port 7575 to a public HTTPS URL (cloudflared example)
cloudflared tunnel --url http://localhost:7575
#    -> https://random-name.trycloudflare.com

# 2. restart the vault, allowing that hostname (comma-separated for several)
set VAULT_ALLOW_HOSTS=random-name.trycloudflare.com
vault.exe --lan

# 3. give the AI the public URL + a bearer token (AI Access -> Generate):
curl -H "Authorization: Bearer TOKEN" https://random-name.trycloudflare.com/api/ai/list
curl -H "Authorization: Bearer TOKEN" "https://random-name.trycloudflare.com/api/ai/read?name=Github"
```

The token still gates every read (expiry + rate limit + per-token revoke). The tunnel provides TLS, so the token is encrypted in transit. Revoke the token in the UI when done.

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────┐
│           vault.exe (Go binary)         │
│  ┌───────────────────────────────────┐  │
│  │   vault.html (embedded via        │  │
│  │   //go:embed directive)          │  │
│  │   - single-file HTML/CSS/JS      │  │
│  │   - glass morphism + matrix rain │  │
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
The server runs a security middleware that blocks **any cross-origin request** coming from a browser. This is the single real attack surface for a localhost app: a malicious webpage making `fetch("http://127.0.0.1:7575/api/delete?name=...")` to delete your vault.

- If the `Origin` header is present, it must be `127.0.0.1`, `localhost`, or `::1` - anything else gets `403 Forbidden`
- If `Origin` is absent (terminal AIs, curl, scripts), the request is allowed - terminal workflows keep working
- `Referer` is also checked as a fallback for browsers that omit `Origin`

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

### Why no login / encryption?

The vault is a localhost-first, single-user app: the UI endpoints are protected from malicious webpages by the Origin/Host checks above. AI and script reads are protected by the optional bearer token (see AI Access). If you need more:
- **For multi-user / remote access**: add TLS and put the whole API behind the token
- **For real encryption at rest**: layer `gocryptfs` or `VeraCrypt` on top of `~/Vault/`

### Backup hygiene
**Do not sync `~/Vault/` to cloud storage in plaintext.** If you back it up, encrypt the backup first:
```bash
tar czf - ~/Vault | gpg --symmetric --cipher-algo AES256 -o vault-backup.tar.gz.gpg
```

### Plaintext by design
Stored values are plaintext on disk. The server binds to `127.0.0.1` only (no remote access). No authentication, no TLS.

---

## 🤝 Contributing

PRs welcome. Keep it small and focused.

```bash
# setup
git clone https://github.com/AshesOfTheUndead/Vault.git
cd vault
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

