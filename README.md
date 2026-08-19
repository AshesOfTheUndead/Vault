# VAULT

> A secret treasure box on your computer.
> A tiny app that keeps your passwords, keys, and codes safe and easy to find.
> One small file. No internet needed. No cloud. Your stuff stays on YOUR computer.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00add8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-34d399.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-22d3ee)](#)
[![Port](https://img.shields.io/badge/Port-7575-fbbf24)](#)
[![Version](https://img.shields.io/badge/version-4.0.1-34d399)](#)

This guide has three tracks. Pick the one that fits you:

1. **Beginner** - first time using the vault. Everything in plain words.
2. **Advanced** - you use terminals, curl, and your own scripts. Everything in commands.
3. **Professional** - you run servers, care about security, or deploy at work. Everything in depth.

Each track ends with its own **Problems, trials, and errors** section, so you can
find the exact fix for whatever went wrong.

## Report a bug (please!)

Found something that is broken, confusing, or just plain wrong? Tell us - every
report makes the vault better.

1. Go to the [Issues page](https://github.com/AshesOfTheUndead/Vault/issues).
2. Click **New issue** -> **Bug report** (a form opens with the right questions).
3. Fill in: what happened, how to reproduce it, your version (the badge in the
   top-left, e.g. `v4.0.1`), and your operating system.
4. Paste the error message and console output. **Never paste secret values or
   tokens** - remove them first.

A good report is answered fast. Reports without a version and reproduction
steps are hard to fix - the more detail, the better.

---

# 1. BEGINNER TRACK

## What is this?

Imagine a small box on your computer. Inside the box you can put little notes
with words you do not want anyone else to see:

- your game password
- your secret key for a website
- the code that lets your AI helper talk to your tools

The box is called **VAULT**. You open it in your browser at
`http://127.0.0.1:7575`. Only your computer can open it. It saves every note
as a tiny text file in a folder named `Vault` in your home folder.

## How to start the vault

1. Download the file for your computer from the [Releases](../../releases) page:
   - `vault.exe` for Windows
   - `vault-linux-amd64` for Linux
   - `vault-darwin-arm64` for Apple computers (Mac with M1/M2/M3)
   - `vault-darwin-amd64` for older Macs (Intel)
2. Put the file anywhere you like (your Desktop is fine).
3. Double-click it.
4. Your browser opens and shows the vault. If it does not, type this in the
   address bar: `http://127.0.0.1:7575`

## How to put a secret inside

1. Click the **New secret** button (or press the `N` key).
2. Give it a name. Example: `roblox` or `wifi_password`.
3. Type the secret value. Example: your real password.
4. (Optional) Add notes in the Details box. Example: `the password for my main account`.
5. Click **Save secret** (or press `Ctrl+Enter`).

Your secret now sits in the list, hidden as dots so nobody can read it over
your shoulder.

## How to look at, copy, and remove a secret

- **Look**: click the **eye** button on the row. Click it again to hide.
  A slashed eye means the value is hidden.
- **Copy**: click the **copy** button. The value goes to your clipboard and
  wipes itself after 30 seconds.
- **Remove**: click the **trash** button, then confirm. Deleted by mistake?
  Press **Undo** on the message - you have 5 seconds.
- **Generate a strong password**: in the secret editor, click the little star
  (dice) button. Vault writes a 20-character random password for you.

## Env vars (settings for your apps)

The **Env Vars** tab stores environment variables - settings that programs
read at startup.

1. Click the **Env Vars** tab (or press `2`).
2. Click **New env var** (or press `E`).
3. Key: `DATABASE_URL`. Value: `postgres://user:pass@host/db`.
4. Flip the **secret** switch if the value should be hidden.
5. Click **Save env var**.

## Gateways (groups of settings)

A **gateway** is a named bundle of env vars that belong together, like a
gateway named `stripe` holding `STRIPE_KEY`, `WEBHOOK_SECRET`, and
`MERCHANT_ID`.

1. Click the **Gateways** tab (or press `3`).
2. Click **New gateway** (or press `G`).
3. Type a name. Press **Generate** for a ready-made random gateway with
   sample variables, then edit them.
4. Click **Save gateway**.

## Letting your AI helper in (AI Access)

You can let a web AI (like Qwen or ChatGPT with code execution) read your
secrets. You give it two things: an **address** (the tunnel URL) and a
**key** (the token).

1. Click the robot button (AI Access).
2. Pick how long the key works: 1 day, 7 days, 30 days, or never.
3. Click **Generate token** and copy it - it is shown only once!
4. Click **Start tunnel**, wait a few seconds, and copy the URL.
5. Paste both into your AI and tell it to use them.
6. When you are done, click the token row and press **Delete**. The AI loses
   access immediately.

## Beginner problems, trials, and errors

| Problem | What you tried | The fix |
|---------|----------------|---------|
| The page will not open | Double-clicked, nothing happened | Start the vault again and wait 2 seconds. The address must be exactly `http://127.0.0.1:7575` |
| The page opened but it is empty | Waited, nothing appeared | Press `R` to refresh. If it stays empty, check the search box - a filter hides everything else |
| I forgot my secret | Looked everywhere in the app | Open the folder `Vault/Secrets/<name>/value.txt` directly on your computer. Vault keeps everything as normal text files |
| My secret shows as dots | Tried to read it in the list | That is the hide feature. Click the eye button on the row to peek |
| I deleted something by mistake | Panicked | Click **Undo** on the popup message within 5 seconds |
| My copied secret vanished | Pasting it somewhere else | Vault clears the clipboard after 30 seconds on purpose. Copy it again right before pasting |
| I generated a token but lost it | Closed the window | Tokens are shown once for safety. Generate a new one and copy it immediately |
| The tunnel says "off" | Restarted the vault | Restarting the vault stops the tunnel. Click **Start tunnel** again and use the new URL |
| The AI says "401" | Gave it my token | The token is wrong or expired. Generate a fresh one |
| The AI says "403" | Gave it the old address | The tunnel URL changed. Copy the new one from AI Access |

Not in the list? Open a [bug report](https://github.com/AshesOfTheUndead/Vault/issues/new?template=bug_report.md) - tell us exactly what you tried.

---

# 2. ADVANCED TRACK

## Running vault from the terminal

```bash
vault.exe                    # Windows
./vault                      # Linux / Mac

vault.exe -port 8080         # different port
vault.exe -no-browser        # do not open the browser
vault.exe -version           # print version and exit
vault.exe --lan              # also listen on your local network
VAULT_ALLOW_HOSTS=my-tunnel.trycloudflare.com vault.exe --lan   # allow a tunnel hostname
```

Behavior notes:

- Default port is **7575**. If something else holds the port, vault finds the
  process, stops it, and takes over.
- `--lan` (or `--host 0.0.0.0`) prints every NIC address in the banner and
  re-scans every 30 seconds, so DHCP changes never break phone access.
- Ctrl+C shuts down cleanly and stops the tunnel.
- Old flat-file vaults are migrated automatically on first run.

## Terminal workflows

```bash
# list / read / save / delete secrets
curl http://127.0.0.1:7575/api/list
curl "http://127.0.0.1:7575/api/get?name=roblox"
curl -X POST http://127.0.0.1:7575/api/save \
  -H "Content-Type: application/json" \
  -d '{"name":"roblox","value":"s3cret","details":"main account"}'
curl -X POST "http://127.0.0.1:7575/api/delete?name=roblox"

# env vars
curl http://127.0.0.1:7575/api/env/list
curl -X POST http://127.0.0.1:7575/api/env/save \
  -H "Content-Type: application/json" \
  -d '{"key":"DATABASE_URL","value":"postgres://user:pass@host/db","secret":true}'

# .env import / export (import auto-marks PASSWORD/TOKEN/API_KEY as secret)
curl -OJ http://127.0.0.1:7575/api/env/export
curl -X POST http://127.0.0.1:7575/api/env/import --data-binary @.env

# one-click backup of everything
curl -OJ http://127.0.0.1:7575/api/backup

# audit trail (last 100 events)
curl http://127.0.0.1:7575/api/audit
```

## AI access from a script

```bash
# 1. generate a token in the UI (AI Access -> Generate token)

# 2. list everything
curl -H "Authorization: Bearer YOUR_TOKEN" http://127.0.0.1:7575/api/ai/list

# 3. read one entry
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://127.0.0.1:7575/api/ai/read?name=roblox"
```

Token rules:

- TTL: 1, 7, or 30 days, or never. Expiry is enforced server-side.
- Rate limit: **60 requests/min per IP**. Over that: `429`.
- Deleting a token is permanent and immediate. Old tokens still appear in the
  list as Revoked/Expired for visibility.
- AI reads are written to the audit log with the source IP.

## LAN mode (phones, tablets, Termux)

```bash
vault.exe --lan
# banner prints:  lan  http://192.168.1.50:7575
# open that address on your phone
```

The Host allowlist accepts loopback plus your own NIC addresses, so phone
access works without configuration. Everything else is rejected with `403`.

## Manual tunnel (when you want to run cloudflared yourself)

```bash
# terminal 1: the tunnel
cloudflared tunnel --url http://localhost:7575
# -> https://random-name.trycloudflare.com

# terminal 2: restart the vault, allowing that hostname
set VAULT_ALLOW_HOSTS=random-name.trycloudflare.com
vault.exe --lan

# from the AI side
curl -H "Authorization: Bearer TOKEN" https://random-name.trycloudflare.com/api/ai/list
curl -H "Authorization: Bearer TOKEN" "https://random-name.trycloudflare.com/api/ai/read?name=Github"
```

Comma-separate several hostnames if you use more than one tunnel.

## Keyboard shortcuts (full list)

| Key | Action |
|-----|--------|
| `/` | Focus search |
| `?` | Keyboard shortcuts help |
| `1` / `2` / `3` | Secrets / Env Vars / Gateways tab |
| `N` / `E` / `G` | New secret / env var / gateway |
| `R` | Manual refresh |
| `Ctrl+K` (Cmd+K) | Command palette (fuzzy search over everything) |
| `Ctrl+Enter` | Save the open editor |
| `Esc` | Close editor / modal / palette |

## API reference

Secrets:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/list` | List all secrets |
| `GET`  | `/api/get?name=X` | Get one secret (value + details) |
| `POST` | `/api/save` | Save a secret `{name, value, details}` |
| `POST` | `/api/delete?name=X` | Delete a secret |

Env vars:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/env/list` | List all env vars |
| `GET`  | `/api/env/get?key=X` | Get one env var (value + secret flag) |
| `POST` | `/api/env/save` | Save an env var `{key, value, secret}` |
| `POST` | `/api/env/delete?key=X` | Delete an env var |
| `GET`  | `/api/env/export` | Download all env vars as `.env` |
| `POST` | `/api/env/import` | Import env vars from `.env` text |

Gateways:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/gateway/list` | List all gateways |
| `GET`  | `/api/gateway/get?name=X` | Get one gateway |
| `POST` | `/api/gateway/save` | Save a gateway `{name, type, description, vars}` |
| `POST` | `/api/gateway/delete?name=X` | Delete a gateway |

System:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/health` | Health check (uptime, version) |
| `GET`  | `/api/stats` | Stats (counts, size, uptime, AI + LAN status) |
| `GET`  | `/api/version` | Version (go, platform, port, LAN mode) |
| `GET`  | `/api/audit` | Last 100 audit log entries |
| `GET`  | `/api/backup` | One-click JSON backup of everything |

AI Access (bearer token required, no Origin check - works from curl / AI tools):

| Method | Path | Description |
|--------|------|-------------|
| `GET`    | `/api/ai/status` | List tokens (created / expiry / revoked) |
| `POST`   | `/api/ai/token` | Generate a token `{"days": 1, 7, 30, or 0 = never}` (default 7) |
| `DELETE` | `/api/ai/token?id=X` | Delete a token for good |
| `GET`    | `/api/ai/list` | List all entry names |
| `GET`    | `/api/ai/read?name=X` | Read a value |

Tunnel:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/tunnel` | Tunnel status (running / URL) |
| `POST` | `/api/tunnel/start` | Start the built-in HTTPS tunnel |
| `POST` | `/api/tunnel/stop` | Stop the tunnel |

## Advanced problems, trials, and errors

| Problem | What you tried | The fix |
|---------|----------------|---------|
| `403` on the UI API calls | Opened the vault from `192.168.x.x:7575` and the page half-loads | The browser sends an Origin your vault does not trust. Open `http://127.0.0.1:7575` (or a host you added to `VAULT_ALLOW_HOSTS`) |
| `403` from a script with `curl -H "Origin: ..."` | Added an Origin header by habit | Scripts must NOT send an Origin header. Vault allows Origin-free requests; foreign Origins get rejected |
| `401` on `/api/ai/*` | Passed a revoked or expired token | Check `/api/ai/status` for the token state, then generate a new token |
| `429` on `/api/ai/*` | Ran a loop that polls fast | Slow down - the limit is 60 requests/min per IP |
| `404 token not found` | Deleted a token twice | Delete is permanent. A second delete has nothing to remove |
| Port 7575 already in use | Another app grabbed it | Vault kills the offending process and takes the port. If you do NOT want that, start with `-port 8080` |
| The page shows stale data | Two tabs open, or old browser tab | Vault re-syncs every 4 seconds. Press `R` for an instant refresh |
| Tunnel URL changed | Vault restarted | Vault owns the tunnel it starts. Restart it with **Start tunnel** and update your scripts |
| `.env` import failed silently | Pasted a `.env` with weird quotes | Values with `"`, `#`, or spaces are quoted/imported as JSON-escaped. Check the imported rows for quoting |
| Clipboard cleared too fast | Copying a long value | The 30-second auto-clear is by design. Re-copy right before pasting |
| AI reads show in audit log | Investigated a breach | That is a feature: every AI read records source IP + timestamp in `audit.log` |

Not in the list? Open a [bug report](https://github.com/AshesOfTheUndead/Vault/issues/new?template=bug_report.md) - include the endpoint, the status code, and how you call it.

---

# 3. PROFESSIONAL TRACK

## Architecture

A single Go binary with the UI embedded inside it (`//go:embed vault.html`).
No runtime, no database, no node_modules, no build step for the frontend.

```
vault.exe (Go binary)
  -> vault.html embedded (one file: HTML + CSS + JS, vanilla, no framework)
  -> HTTP server (net/http, port takeover, graceful shutdown, atomic writes)
  -> Filesystem storage, plaintext by design:
       ~/Vault/Secrets/{name}/    value.txt, details.txt, name.txt
       ~/Vault/EnvVars/{key}/     key.txt, value.txt, secret.txt
       ~/Vault/Gateways/{name}/   meta.json, vars.json
       ~/Vault/audit.log          every save/delete/AI read
       ~/Vault/ai.tokens.json     AI token store
```

- Backend: Go 1.21+, standard library only (`net/http`, `embed`, `os`, `os/exec`).
- Frontend: vanilla HTML/CSS/JS. Fonts: Space Grotesk, JetBrains Mono, Share Tech Mono.
- Writes are atomic (temp file + rename), so a crash cannot corrupt entries.
- The version badge in the UI is read live from `/api/version` - it always
  matches the running binary.

## Security model

### Origin/Host check (CSRF protection)

The single realistic attack against a localhost app is a malicious webpage
doing `fetch("http://127.0.0.1:7575/api/delete?name=...")` in the background.
Vault blocks that:

- Any request with an `Origin` header must have `127.0.0.1`, `localhost`, or
  `::1`. Anything else: `403 Forbidden`.
- Requests WITHOUT an Origin (curl, scripts, terminal AIs) are allowed - this
  is what keeps automation working.
- `Referer` is checked as a fallback for browsers that omit `Origin`.

```bash
curl http://127.0.0.1:7575/api/health                          # 200 (no Origin)
curl -H "Origin: http://127.0.0.1:7575" http://127.0.0.1:7575/api/health   # 200
curl -H "Origin: https://evil.com" http://127.0.0.1:7575/api/health        # 403
```

### DNS-rebinding defense

The Host allowlist only accepts loopback, your LAN NIC IPs (`--lan` mode,
re-scanned every 30s), and hostnames listed in `VAULT_ALLOW_HOSTS`. A rebinding
attack that resolves `attacker.com` to `127.0.0.1` still fails the Host check.

### Security headers

Every response carries:

- `Content-Security-Policy` - strict allowlist (`'self'`, Google Fonts, `data:` URIs)
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: no-referrer`
- `X-XSS-Protection: 1; mode=block`

### Server hardening

Read/Write timeouts 30s, Idle 120s, ReadHeader 10s, MaxHeaderBytes 1MB -
slowloris and header-bomb resistance.

### AI token security

- 32-byte hex tokens (256 bits of entropy), generated with `crypto/rand`.
- Constant-time comparison (no timing side channel).
- Rate limit 60 req/min per IP.
- Per-token TTL enforced server-side; `DELETE` is permanent.
- Tokens travel over the tunnel's TLS when you use the built-in tunnel.

### Audit logging

`audit.log` records every save, delete, and AI read with timestamp + source
IP. `/api/audit` exposes the last 100 events for the UI's Access History.

## Hardening checklist

1. **Remote access**: put vault behind a TLS reverse proxy (Caddy or nginx),
   add `VAULT_ALLOW_HOSTS=<your-domain>`, and gate the whole API with a token.
2. **Encryption at rest**: layer `gocryptfs` or `VeraCrypt` over `~/Vault/`.
3. **Backups**: never sync `~/Vault/` to cloud storage in plaintext.
   Encrypt first:
   ```bash
   tar czf - ~/Vault | gpg --symmetric --cipher-algo AES256 -o vault-backup.tar.gz.gpg
   ```
4. **Secrets rotation**: after any suspected leak, delete the AI token
   immediately and rotate the affected secrets.
5. **Least privilege**: run vault as a dedicated user, not root/admin.
6. **Monitoring**: watch `audit.log` for unexpected reads or 4xx storms.

## Professional problems, trials, and errors

| Problem | What you tried | The fix |
|---------|----------------|---------|
| Slowloris-style exhaustion | Kept connections half-open | The server closes reads after 30s and idles after 120s. If you still see issues, put vault behind a proxy that enforces its own timeouts |
| Header bomb / huge request | Sent oversized headers | `MaxHeaderBytes` is 1MB. Requests beyond it are rejected before parsing |
| DNS-rebinding probe | Resolved a domain to 127.0.0.1 and connected | Expected: `403` unless the Host is loopback, a NIC IP, or in `VAULT_ALLOW_HOSTS` |
| Cross-origin fetch from a compromised page | Malicious site fires `fetch` at the vault | Expected: `403` (Origin check). Verify with the curl examples above |
| Disk full mid-write | Vault froze during a save | Atomic writes mean the old file survives and the temp rename fails cleanly - no corruption. Free disk space and retry |
| Two vault processes fighting | Started vault twice | Port takeover kills the loser. If you intentionally run two instances, give them different `-port` values AND different homes (different users or `HOME`) |
| Token leaked | Token appeared in a log or chat | Delete the token immediately (UI or `DELETE /api/ai/token?id=X`), then rotate every secret the token could read. Check `audit.log` for when the token was used |
| Audit log grew huge | Long-running vault | Rotate `audit.log` with your normal log rotation. Vault appends; it never truncates |
| Multi-user access | Several people need the vault | Vault is single-user by design. Front it with a token-gated reverse proxy + TLS, or move to a shared secrets manager |
| Compliance needs encryption-at-rest proof | Auditor asks how data is stored | Vault stores plaintext by design - document it. Add `gocryptfs`/`VeraCrypt` for at-rest encryption and re-verify the backup pipeline |

Not in the list? Open a [bug report](https://github.com/AshesOfTheUndead/Vault/issues/new?template=bug_report.md) - include the exact error, version, and your setup.

## Building from source

```bash
git clone https://github.com/AshesOfTheUndead/Vault.git
cd Vault
go build -o vault .

# cross-compile
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o vault.exe .
GOOS=linux   GOARCH=amd64 go build -o vault-linux-amd64 .
GOOS=darwin  GOARCH=amd64 go build -o vault-darwin-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o vault-darwin-arm64 .

# or use the scripts
./build.sh        # linux/mac
build.bat         # windows
```

## Contributing

PRs welcome. Keep them small and focused. Before submitting:

```bash
gofmt -w main.go
go vet ./...
go build -o vault .
```

---

## License

MIT - see [LICENSE](LICENSE).

## Acknowledgments

- [Space Grotesk](https://fonts.google.com/specimen/Space+Grotesk) by Florian Karsten
- [JetBrains Mono](https://www.jetbrains.com/lp/mono/) by JetBrains
- [Share Tech Mono](https://fonts.google.com/specimen/Share+Tech+Mono) by Carrois Apostille