# VAULT

> A secret treasure box on your computer.
> A tiny app that keeps your passwords, keys, and codes safe and easy to find.
> One small file. No internet needed. No cloud. Your stuff stays on YOUR computer.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00add8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-34d399.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-22d3ee)](#)
[![Port](https://img.shields.io/badge/Port-7575-fbbf24)](#)
[![Version](https://img.shields.io/badge/version-4.0.1-34d399)](#)

---

## What is this?

Imagine a small box on your computer. Inside the box you can put little notes
with words you do not want anyone else to see:

- your game password
- your secret key for a website
- the code that lets your AI helper talk to your tools

The box is called **VAULT**. You open it in your browser at
`http://127.0.0.1:7575`. Only your computer can open it. It saves every note
as a tiny text file in a folder named `Vault` in your home folder.

---

## FOR EVERYONE: How to use it (5 easy steps)

### Step 1. Start the vault

1. Download the file for your computer from the [Releases](../../releases) page:
   - `vault.exe` for Windows
   - `vault-linux-amd64` for Linux
   - `vault-darwin-arm64` for Apple computers (Mac with M1/M2/M3)
   - `vault-darwin-amd64` for older Macs (Intel)
2. Put the file anywhere you like (your Desktop is fine).
3. Double-click it.
4. Your browser opens and shows the vault. If it does not, type this in the
   address bar: `http://127.0.0.1:7575`

That is it. The vault is now open.

### Step 2. Put a secret inside

1. Click the **New secret** button (or press the `N` key).
2. Give it a name. Example: `roblox` or `wifi_password`.
3. Type the secret value. Example: your real password.
4. (Optional) Add notes in the Details box. Example: `the password for my main account`.
5. Click **Save secret** (or press `Ctrl+Enter`).

Your secret now sits in the list. The vault hides it as dots, so someone
looking over your shoulder cannot read it.

### Step 3. Look at a secret

Click the **eye** button on the row of the secret. The dots turn back into the
real words for a moment. Click the eye again to hide it. The little slashed eye
means the secret is hidden.

### Step 4. Copy a secret

Click the **copy** button on the row. The value goes to your clipboard, ready
to paste. The vault wipes the clipboard after 30 seconds, so it cannot leak
later.

### Step 5. Remove a secret

Click the **trash** button on the row. The vault asks "are you sure?" first,
so you cannot delete by accident. If you delete by mistake, press **Undo**
on the little message that appears - you have 5 seconds.

---

## Env Vars: special secrets for your apps

The second tab, **Env Vars**, is for environment variables (programs use
these to read settings like a database address).

1. Click the **Env Vars** tab (or press `2`).
2. Click **New env var** (or press `E`).
3. Give it a big-name key. Example: `DATABASE_URL`.
4. Type the value. Example: `postgres://user:pass@host/db`.
5. Flip the **secret** switch if this value should be hidden with dots.
6. Click **Save env var**.

You can also:

- **Import a `.env` file** - add many env vars at once. Vault automatically
  marks keys like `PASSWORD`, `TOKEN`, and `API_KEY` as secret.
- **Export as `.env`** - download all env vars as one file.

## Gateways: groups of env vars

A **gateway** is a named bundle of env vars that belong together. Example: a
gateway named `stripe` with `STRIPE_KEY`, `WEBHOOK_SECRET`, and `MERCHANT_ID`.

1. Click the **Gateways** tab (or press `3`).
2. Click **New gateway** (or press `G`).
3. Type a name, and press **Generate** if you want a ready-made random gateway
   with sample variables. Edit the variables as you like.
4. Click **Save gateway**.

You can copy a gateway as a ready `.env` block anytime.

---

## Let your AI helper in (AI Access)

You can let a web AI (like Qwen or ChatGPT with code execution) read your
secrets. You give it two things: an **address** (the tunnel URL) and a
**key** (the token). The AI can then read your secrets, and nothing else on
your computer.

### Step 1. Make a key (token)

1. Click the robot button (AI Access) in the vault.
2. Pick how long the key should work: 1 day, 7 days, 30 days, or never.
3. Click **Generate token**.
4. Copy the token - it is shown only once!

### Step 2. Start the tunnel

1. In the same window, click **Start tunnel**.
2. Wait a few seconds. A public address appears, like
   `https://funny-words.trycloudflare.com`.
3. Click **Copy URL**.

### Step 3. Give the AI the address and the key

Paste both into your AI and tell it to use them like this:

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" https://funny-words.trycloudflare.com/api/ai/list
curl -H "Authorization: Bearer YOUR_TOKEN" "https://funny-words.trycloudflare.com/api/ai/read?name=roblox"
```

The tunnel address is safe: without the token, nobody can read anything.

### Step 4. Take the key away when you are done

Open AI Access, click the token row, and press **Delete**. The AI loses access
immediately. Tokens are also deleted automatically when they expire.

---

## What if something goes wrong?

| Problem | Fix |
|---------|-----|
| The page will not open | Make sure `vault.exe` is running (double-click it again) and the address is `http://127.0.0.1:7575` |
| The page says "v4.0.1" but nothing else | Wait 2 seconds - the vault loads itself automatically |
| I forgot my secret | Secrets are saved as plain text files in `~/Vault/Secrets/<name>/value.txt` - you can open the file directly |
| The tunnel says "off" | Click **Start tunnel** again. Restarting the vault stops the tunnel |
| The AI says "401" | The token is wrong or expired. Generate a new one |
| The AI says "403" | The tunnel URL changed. Copy the new URL from AI Access |

---

## All the buttons and keys

| Key | What it does |
|-----|--------------|
| `/` | Jump to the search box |
| `?` | Show this help |
| `1` | Go to Secrets |
| `2` | Go to Env Vars |
| `3` | Go to Gateways |
| `N` | New secret |
| `E` | New env var |
| `G` | New gateway |
| `R` | Refresh everything |
| `Ctrl+K` (or `Cmd+K`) | Open the command palette (type to find anything) |
| `Ctrl+Enter` | Save what you are editing |
| `Esc` | Close the window / back out |

---

## Where does vault keep your stuff?

Everything lives in a folder called `Vault` in your home folder
(`C:\Users\YOU\Vault` on Windows, `~/Vault` on Mac/Linux):

```
~/Vault/
Secrets/
  roblox/
    value.txt      <- the secret value
    details.txt    <- optional notes
    name.txt       <- the name (spelled exactly as you typed it)
  github_token/
EnvVars/
  DATABASE_URL/
    key.txt
    value.txt
    secret.txt     <- "true" or "false"
Gateways/
  stripe/
    meta.json      <- name, type, description
    vars.json      <- the key-value pairs
audit.log          <- who read what, and when
ai.tokens.json     <- your AI tokens
```

The files are not scrambled on purpose: you can read, edit, or delete them
directly. Delete a folder to delete an entry. Nothing is hidden from you.

---



### Starting from the terminal

```bash
vault.exe                    # Windows
./vault                      # Linux / Mac

vault.exe -port 8080         # use a different port
vault.exe -no-browser        # do not open the browser
vault.exe -version           # print the version and exit
vault.exe --lan              # also listen on your local network (phones, Termux)
VAULT_ALLOW_HOSTS=my-tunnel.trycloudflare.com vault.exe --lan   # allow a tunnel hostname
```

Other details:

- Port **7575** is the default. If something else uses it, vault finds that
  process, stops it, and takes the port.
- LAN mode (`--lan` / `--host 0.0.0.0`) prints all your network addresses in
  the terminal banner and re-scans every 30 seconds.
- The app auto-migrates old flat-file vaults to the folder layout on first run.

### API reference

All endpoints speak JSON over HTTP on `127.0.0.1:7575`.

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

AI Access (token-protected, no Origin check - works from curl / AI tools):

| Method | Path | Description |
|--------|------|-------------|
| `GET`    | `/api/ai/status` | List tokens (created / expiry / revoked) |
| `POST`   | `/api/ai/token` | Generate a token `{"days": 1, 7, 30, or 0 = never}` (default 7) |
| `DELETE` | `/api/ai/token?id=X` | Delete a token for good |
| `GET`    | `/api/ai/list` | List all entry names (bearer token, rate-limited) |
| `GET`    | `/api/ai/read?name=X` | Read a value (bearer token, rate-limited) |

Tunnel:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/tunnel` | Tunnel status (running / URL) |
| `POST` | `/api/tunnel/start` | Start the built-in HTTPS tunnel |
| `POST` | `/api/tunnel/stop` | Stop the tunnel |

Useful examples:

```bash
# list env vars
curl http://127.0.0.1:7575/api/env/list

# save an env var
curl -X POST http://127.0.0.1:7575/api/env/save \
  -H "Content-Type: application/json" \
  -d '{"key":"DATABASE_URL","value":"postgres://user:pass@host/db","secret":true}'

# export / import .env
curl -OJ http://127.0.0.1:7575/api/env/export
curl -X POST http://127.0.0.1:7575/api/env/import --data-binary @.env

# one-click backup (everything as JSON)
curl -OJ http://127.0.0.1:7575/api/backup

# AI access
curl -H "Authorization: Bearer TOKEN" http://127.0.0.1:7575/api/ai/list
curl -H "Authorization: Bearer TOKEN" "http://127.0.0.1:7575/api/ai/read?name=roblox"
```

### Security model

- **Origin/Host check (CSRF protection).** Browsers that send an `Origin`
  header must come from `127.0.0.1`, `localhost`, or `::1` - anything else
  gets `403 Forbidden`. Requests without an Origin (curl, scripts, terminal
  AIs) are allowed, which keeps automation working. `Referer` is checked as a
  fallback.
- **DNS-rebinding defense.** A Host allowlist: loopback plus your LAN IPs
  (`--lan` mode), plus anything in `VAULT_ALLOW_HOSTS`.
- **Security headers.** CSP, `nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`, `X-XSS-Protection`.
- **Server hardening.** Read/Write 30s, Idle 120s, 1MB max headers.
- **AI tokens.** 32-byte hex tokens, constant-time compare, rate-limited to
  60 requests/min per IP, per-token permanent delete, expiry enforced on
  every read.
- **Audit log.** Every save/delete and every AI read lands in `audit.log`
  with a timestamp and source IP.
- **Atomic writes.** Files are written to a temp file then renamed, so a
  crash cannot corrupt your vault.

Why no login or encryption? The vault is localhost-first and single-user.
Browser attacks are stopped by the Origin/Host checks, and AI/script reads by
bearer tokens. For remote or multi-user use, put TLS in front and gate the
whole API with a token. For encryption at rest, layer `gocryptfs` or
`VeraCrypt` over `~/Vault`.

**Backup hygiene:** do not sync `~/Vault/` to cloud storage in plaintext.
Encrypt backups first:

```bash
tar czf - ~/Vault | gpg --symmetric --cipher-algo AES256 -o vault-backup.tar.gz.gpg
```

### Architecture

A single Go binary with the UI embedded inside it (`//go:embed vault.html`).

```
vault.exe (Go binary)
  -> vault.html embedded (one file: HTML + CSS + JS, no build step)
  -> HTTP server (net/http, port takeover, graceful shutdown, atomic writes)
  -> Filesystem storage (~/Vault/{Secrets,EnvVars,Gateways}, plaintext by design)
```

- Backend: Go 1.21+, standard library only
- Frontend: vanilla HTML/CSS/JS, no framework
- Fonts: Space Grotesk, JetBrains Mono, Share Tech Mono
- Storage: one folder per entry, plaintext files

### Building from source

```bash
git clone https://github.com/AshesOfTheUndead/Vault.git
cd Vault
go build -o vault .

# or use the scripts
./build.sh        # linux/mac
build.bat         # windows
```

---

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
