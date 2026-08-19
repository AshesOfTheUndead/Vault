package main

import (
        "context"
        "crypto/rand"
        "crypto/subtle"
        _ "embed"
        "encoding/hex"
        "encoding/json"
        "flag"
        "fmt"
        "io"
        "log"
        "net"
        "net/http"
        "net/url"
        "os"
        "os/exec"
        "os/signal"
        "path/filepath"
        "regexp"
        "runtime"
        "sort"
        "strconv"
        "strings"
        "sync"
        "syscall"
        "time"
)

//go:embed vault.html
var vaultHTML []byte

const (
        defaultPort = 7575
        version     = "4.0.1"
)

var (
        vaultDir     string
        secretsDir   string
        envVarsDir   string
        startTime    = time.Now()
        writeMu      sync.Mutex
        allowedMu    sync.RWMutex
        allowedHosts = map[string]bool{}
        lanHosts     = map[string]bool{}
)

var sanitizeRe = regexp.MustCompile(`[\\/:*?"<>|]`)

func sanitize(name string) string {
        s := strings.TrimSpace(name)
        s = sanitizeRe.ReplaceAllString(s, "_")
        s = strings.TrimLeft(s, ".")
        s = strings.TrimRight(s, ".")
        if len(s) > 100 {
                s = s[:100]
        }
        return s
}

func secretDir(name string) string {
        s := sanitize(name)
        if s == "" {
                return ""
        }
        return filepath.Join(secretsDir, s)
}

func envVarDir(key string) string {
        s := sanitize(key)
        if s == "" {
                return ""
        }
        return filepath.Join(envVarsDir, s)
}

// atomicWriteFile writes to a temp file then renames, preventing partial writes
func atomicWriteFile(path string, data []byte) error {
        tmp := path + ".tmp"
        if err := os.WriteFile(tmp, data, 0644); err != nil {
                return err
        }
        return os.Rename(tmp, path)
}

// migrateOldFormat moves flat .txt files into per-secret folders
func migrateOldFormat() {
        entries, err := os.ReadDir(vaultDir)
        if err != nil {
                return
        }
        for _, e := range entries {
                if e.IsDir() {
                        continue
                }
                name := e.Name()
                if !strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".details.txt") {
                        continue
                }
                baseName := strings.TrimSuffix(name, ".txt")
                valueFile := filepath.Join(vaultDir, name)
                detailsFile := filepath.Join(vaultDir, baseName+".details.txt")
                sani := sanitize(baseName)
                if sani == "" {
                        continue
                }
                newDir := filepath.Join(secretsDir, sani)
                if _, err := os.Stat(newDir); err == nil {
                        os.Remove(valueFile)
                        os.Remove(detailsFile)
                        continue
                }
                os.MkdirAll(newDir, 0755)
                data, err := os.ReadFile(valueFile)
                if err != nil {
                        continue
                }
                atomicWriteFile(filepath.Join(newDir, "value.txt"), data)
                atomicWriteFile(filepath.Join(newDir, "name.txt"), []byte(baseName))
                if ddata, err := os.ReadFile(detailsFile); err == nil {
                        atomicWriteFile(filepath.Join(newDir, "details.txt"), ddata)
                        os.Remove(detailsFile)
                }
                os.Remove(valueFile)
                log.Printf("migrated: %s", baseName)
        }
}

/* ---------- secrets ---------- */

type Entry struct {
        Name    string `json:"name"`
        Updated int64  `json:"updated"`
        Size    int64  `json:"size"`
}

func listEntries() []Entry {
        entries, err := os.ReadDir(secretsDir)
        if err != nil {
                return []Entry{}
        }
        result := []Entry{}
        for _, e := range entries {
                if !e.IsDir() {
                        continue
                }
                dir := filepath.Join(secretsDir, e.Name())
                valFile := filepath.Join(dir, "value.txt")
                detFile := filepath.Join(dir, "details.txt")
                var updated, size int64
                if info, err := os.Stat(valFile); err == nil {
                        updated = info.ModTime().UnixMilli()
                        size = info.Size()
                }
                if info, err := os.Stat(detFile); err == nil {
                        if m := info.ModTime().UnixMilli(); m > updated {
                                updated = m
                        }
                }
                result = append(result, Entry{
                        Name:    e.Name(),
                        Updated: updated,
                        Size:    size,
                })
        }
        sort.Slice(result, func(i, j int) bool {
                return result[i].Updated > result[j].Updated
        })
        return result
}

func readValue(name string) (string, bool) {
        dir := secretDir(name)
        if dir == "" {
                return "", false
        }
        data, err := os.ReadFile(filepath.Join(dir, "value.txt"))
        if err != nil {
                return "", false
        }
        return string(data), true
}

func readDetails(name string) (string, bool) {
        dir := secretDir(name)
        if dir == "" {
                return "", false
        }
        data, err := os.ReadFile(filepath.Join(dir, "details.txt"))
        if err != nil {
                return "", false
        }
        return string(data), true
}

func writeValue(name, value string) (string, error) {
        dir := secretDir(name)
        if dir == "" {
                return "", fmt.Errorf("invalid name")
        }
        if err := os.MkdirAll(dir, 0755); err != nil {
                return "", err
        }
        if err := atomicWriteFile(filepath.Join(dir, "value.txt"), []byte(value)); err != nil {
                return "", err
        }
        if err := atomicWriteFile(filepath.Join(dir, "name.txt"), []byte(name)); err != nil {
                return "", err
        }
        return sanitize(name), nil
}

func writeDetails(name, details string) {
        dir := secretDir(name)
        if dir == "" {
                return
        }
        p := filepath.Join(dir, "details.txt")
        if details == "" {
                os.Remove(p)
                return
        }
        os.MkdirAll(dir, 0755)
        atomicWriteFile(p, []byte(details))
}

func deleteValue(name string) error {
        dir := secretDir(name)
        if dir == "" {
                return nil
        }
        return os.RemoveAll(dir)
}

/* ---------- AI access API (token-protected read for AI tools / scripts) ---------- */

// AIToken is one entry in the token store. The full token value lives only
// on disk (ai.tokens.json); API responses omit it via publicAITokens.
type AIToken struct {
        ID       string `json:"id"`
        Token    string `json:"token,omitempty"`
        Created  string `json:"created"`
        Expires  string `json:"expires"`
        Revoked  bool   `json:"revoked"`
        LastUsed string `json:"lastUsed"`
}

// aiTokenFile is the legacy single-token path (migrated to the store below).
func aiTokenFile() string {
        return filepath.Join(vaultDir, "ai.token")
}

func aiTokensFile() string {
        return filepath.Join(vaultDir, "ai.tokens.json")
}

var tokensMu sync.Mutex

func loadAITokens() []AIToken {
        b, err := os.ReadFile(aiTokensFile())
        if err != nil {
                return nil
        }
        var store struct {
                Tokens []AIToken `json:"tokens"`
        }
        if json.Unmarshal(b, &store) != nil {
                return nil
        }
        return store.Tokens
}

func saveAITokens(tokens []AIToken) error {
        store := struct {
                Tokens []AIToken `json:"tokens"`
        }{Tokens: tokens}
        b, err := json.MarshalIndent(store, "", "  ")
        if err != nil {
                return err
        }
        return atomicWriteFile(aiTokensFile(), b)
}

// migrateLegacyAIToken imports the old single ai.token file into the store
// so existing setups keep working after the upgrade. The legacy token has
// no expiry (never expires) and keeps its original ID prefix.
func migrateLegacyAIToken() {
        if _, err := os.Stat(aiTokensFile()); err == nil {
                return
        }
        b, err := os.ReadFile(aiTokenFile())
        if err != nil {
                return
        }
        tok := strings.TrimSpace(string(b))
        if len(tok) < 16 {
                os.Remove(aiTokenFile())
                return
        }
        tokens := loadAITokens()
        tokens = append(tokens, AIToken{
                ID:      tok[:8],
                Token:   tok,
                Created: time.Now().Format(time.RFC3339),
        })
        if saveAITokens(tokens) == nil {
                os.Remove(aiTokenFile())
                log.Printf("ai-access: migrated legacy token %s", tok[:8])
        }
}

func generateAIToken() (string, error) {
        b := make([]byte, 32)
        if _, err := rand.Read(b); err != nil {
                return "", err
        }
        return hex.EncodeToString(b), nil
}

// validAIToken matches the bearer token against the store, skipping revoked
// and expired entries, and stamps LastUsed on a hit.
func validAIToken(r *http.Request) (*AIToken, bool) {
        auth := r.Header.Get("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") {
                return nil, false
        }
        got := strings.TrimPrefix(auth, "Bearer ")
        tokensMu.Lock()
        defer tokensMu.Unlock()
        tokens := loadAITokens()
        now := time.Now()
        for i := range tokens {
                t := &tokens[i]
                if t.Revoked || t.Token == "" {
                        continue
                }
                if t.Expires != "" {
                        if exp, err := time.Parse(time.RFC3339, t.Expires); err == nil && now.After(exp) {
                                continue
                        }
                }
                if subtle.ConstantTimeCompare([]byte(got), []byte(t.Token)) == 1 {
                        t.LastUsed = now.Format(time.RFC3339)
                        saveAITokens(tokens)
                        return t, true
                }
        }
        return nil, false
}

func aiBearerOK(r *http.Request) bool {
        _, ok := validAIToken(r)
        return ok
}

// publicAITokens strips the secret token value before serialization.
func publicAITokens(tokens []AIToken) []AIToken {
        out := make([]AIToken, 0, len(tokens))
        for _, t := range tokens {
                t.Token = ""
                out = append(out, t)
        }
        return out
}

// aiActiveCount counts non-revoked, non-expired tokens.
func aiActiveCount(tokens []AIToken) int {
        now := time.Now()
        n := 0
        for _, t := range tokens {
                if t.Revoked {
                        continue
                }
                if t.Expires != "" {
                        if exp, err := time.Parse(time.RFC3339, t.Expires); err == nil && now.After(exp) {
                                continue
                        }
                }
                n++
        }
        return n
}

/* ---------- tunnel (one-click HTTPS access for cloud AIs) ---------- */

var tunnelUrlRegex = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

var (
        tunnelMu      sync.Mutex
        tunnelCmd     *exec.Cmd
        tunnelProc    *os.Process
        tunnelURL     string
        tunnelStarted time.Time
)

// cloudflaredPath locates the cloudflared binary: VAULT_CLOUDFLARED env var,
// else next to the vault executable, else on PATH.
func cloudflaredPath() string {
        if p := os.Getenv("VAULT_CLOUDFLARED"); p != "" {
                return p
        }
        if runtime.GOOS == "windows" {
                if exe, err := os.Executable(); err == nil {
                        p := filepath.Join(filepath.Dir(exe), "cloudflared.exe")
                        if _, err := os.Stat(p); err == nil {
                                return p
                        }
                }
        }
        if p, err := exec.LookPath("cloudflared"); err == nil {
                return p
        }
        return "cloudflared"
}

// startTunnel launches a Cloudflare quick tunnel to the vault and auto-allows
// the reported hostname through the Host allowlist so no manual config is needed.
func startTunnel(port int) (string, error) {
        tunnelMu.Lock()
        defer tunnelMu.Unlock()
        if tunnelCmd != nil {
                return tunnelURL, nil
        }
        bin := cloudflaredPath()
        logFile := filepath.Join(vaultDir, "tunnel.log")
        f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
        if err != nil {
                return "", err
        }
        cmd := exec.Command(bin, "tunnel", "--url", fmt.Sprintf("http://127.0.0.1:%d", port), "--no-autoupdate")
        cmd.Stdout = f
        cmd.Stderr = f
        if err := cmd.Start(); err != nil {
                f.Close()
                return "", fmt.Errorf("could not start cloudflared: %w", err)
        }
        tunnelCmd = cmd
        tunnelProc = cmd.Process
        tunnelStarted = time.Now()
        go func() {
                cmd.Wait()
                tunnelMu.Lock()
                tunnelCmd = nil
                tunnelProc = nil
                tunnelURL = ""
                tunnelMu.Unlock()
                f.Close()
        }()
        deadline := time.Now().Add(30 * time.Second)
        for time.Now().Before(deadline) {
                if b, err := os.ReadFile(logFile); err == nil {
                        if m := tunnelUrlRegex.Find(b); m != nil {
                                u, perr := url.Parse(string(m))
                                if perr == nil && u.Host != "" {
                                        tunnelURL = "https://" + u.Host
                                        addAllowedHosts(u.Host)
                                        return tunnelURL, nil
                                }
                        }
                }
                time.Sleep(300 * time.Millisecond)
        }
        return "", fmt.Errorf("tunnel did not become ready in 30s (see %s)", logFile)
}

func stopTunnel() error {
        tunnelMu.Lock()
        defer tunnelMu.Unlock()
        if tunnelCmd == nil {
                return nil
        }
        proc := tunnelProc
        tunnelCmd = nil
        tunnelProc = nil
        tunnelURL = ""
        if proc != nil {
                return proc.Kill()
        }
        return nil
}

func clientIP(r *http.Request) string {
        host, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
                return r.RemoteAddr
        }
        return host
}

// aiRateLimit is a tiny in-memory per-IP limiter so a leaked token can't be
// hammered. 60 requests / minute per IP is generous for AI tools, stingy for
// a scraper.
type aiRateLimit struct {
        mu   sync.Mutex
        hits map[string][]int64
}

var aiLimiter = &aiRateLimit{hits: map[string][]int64{}}

func (l *aiRateLimit) allow(ip string, max int, window time.Duration) bool {
        now := time.Now().Unix()
        cut := now - int64(window/time.Second)
        l.mu.Lock()
        defer l.mu.Unlock()
        kept := l.hits[ip][:0]
        for _, h := range l.hits[ip] {
                if h > cut {
                        kept = append(kept, h)
                }
        }
        if len(kept) >= max {
                l.hits[ip] = kept
                return false
        }
        l.hits[ip] = append(kept, now)
        return true
}

/* ---------- env vars ---------- */

type EnvVar struct {
        Key     string `json:"key"`
        Value   string `json:"value"`
        Secret  bool   `json:"secret"`
        Updated int64  `json:"updated"`
}

func listEnvVars() []EnvVar {
        entries, err := os.ReadDir(envVarsDir)
        if err != nil {
                return []EnvVar{}
        }
        result := []EnvVar{}
        for _, e := range entries {
                if !e.IsDir() {
                        continue
                }
                dir := filepath.Join(envVarsDir, e.Name())
                valFile := filepath.Join(dir, "value.txt")
                secFile := filepath.Join(dir, "secret.txt")
                var updated int64
                var value string
                var secret bool
                if info, err := os.Stat(valFile); err == nil {
                        updated = info.ModTime().UnixMilli()
                        if data, err := os.ReadFile(valFile); err == nil {
                                value = string(data)
                        }
                }
                if info, err := os.Stat(secFile); err == nil {
                        if m := info.ModTime().UnixMilli(); m > updated {
                                updated = m
                        }
                        if data, err := os.ReadFile(secFile); err == nil {
                                s := strings.TrimSpace(string(data))
                                secret = s == "true" || s == "1"
                        }
                }
                result = append(result, EnvVar{
                        Key:     e.Name(),
                        Value:   value,
                        Secret:  secret,
                        Updated: updated,
                })
        }
        sort.Slice(result, func(i, j int) bool {
                return result[i].Key < result[j].Key
        })
        return result
}

func readEnvVar(key string) (string, bool, bool) {
        dir := envVarDir(key)
        if dir == "" {
                return "", false, false
        }
        data, err := os.ReadFile(filepath.Join(dir, "value.txt"))
        if err != nil {
                return "", false, false
        }
        secret := false
        if sd, err := os.ReadFile(filepath.Join(dir, "secret.txt")); err == nil {
                s := strings.TrimSpace(string(sd))
                secret = s == "true" || s == "1"
        }
        return string(data), secret, true
}

func writeEnvVar(key, value string, secret bool) (string, error) {
        dir := envVarDir(key)
        if dir == "" {
                return "", fmt.Errorf("invalid key")
        }
        if err := os.MkdirAll(dir, 0755); err != nil {
                return "", err
        }
        if err := atomicWriteFile(filepath.Join(dir, "value.txt"), []byte(value)); err != nil {
                return "", err
        }
        if err := atomicWriteFile(filepath.Join(dir, "key.txt"), []byte(key)); err != nil {
                return "", err
        }
        sv := "false"
        if secret {
                sv = "true"
        }
        atomicWriteFile(filepath.Join(dir, "secret.txt"), []byte(sv))
        return sanitize(key), nil
}

func deleteEnvVar(key string) error {
        dir := envVarDir(key)
        if dir == "" {
                return nil
        }
        return os.RemoveAll(dir)
}

/* ---------- gateways (named collections of key-value pairs) ---------- */

// GatewayVar is one key-value pair inside a gateway.
type GatewayVar struct {
        Key    string `json:"key"`
        Value  string `json:"value"`
        Secret bool   `json:"secret"`
}

// Gateway is a named configuration preset - a bundle of env vars you can
// create, view, and manage on demand. Think "Stripe gateway", "Email gateway",
// "Database gateway": each is a named .env file with metadata.
type Gateway struct {
        Name        string       `json:"name"`
        Type        string       `json:"type,omitempty"`
        Description string       `json:"description,omitempty"`
        Vars        []GatewayVar `json:"vars"`
        Updated     int64        `json:"updated"`
}

var gatewaysDir string

func gatewayDir(name string) string {
        s := sanitize(name)
        if s == "" {
                return ""
        }
        return filepath.Join(gatewaysDir, s)
}

func listGateways() []Gateway {
        entries, err := os.ReadDir(gatewaysDir)
        if err != nil {
                return []Gateway{}
        }
        result := []Gateway{}
        for _, e := range entries {
                if !e.IsDir() {
                        continue
                }
                g, ok := readGateway(e.Name())
                if !ok {
                        continue
                }
                result = append(result, g)
        }
        sort.Slice(result, func(i, j int) bool {
                return result[i].Updated > result[j].Updated
        })
        return result
}

func readGateway(name string) (Gateway, bool) {
        dir := gatewayDir(name)
        if dir == "" {
                return Gateway{}, false
        }
        metaPath := filepath.Join(dir, "meta.json")
        varsPath := filepath.Join(dir, "vars.json")
        var g Gateway
        g.Name = name
        if b, err := os.ReadFile(metaPath); err == nil {
                json.Unmarshal(b, &g)
                g.Name = name // canonical from folder
        }
        if b, err := os.ReadFile(varsPath); err == nil {
                json.Unmarshal(b, &g.Vars)
        }
        if info, err := os.Stat(varsPath); err == nil {
                g.Updated = info.ModTime().UnixMilli()
        } else if info, err := os.Stat(metaPath); err == nil {
                g.Updated = info.ModTime().UnixMilli()
        }
        return g, true
}

func writeGateway(name, gtype, desc string, vars []GatewayVar) (string, error) {
        dir := gatewayDir(name)
        if dir == "" {
                return "", fmt.Errorf("invalid name")
        }
        if err := os.MkdirAll(dir, 0755); err != nil {
                return "", err
        }
        meta := map[string]string{"name": name, "type": gtype, "description": desc}
        if mb, err := json.MarshalIndent(meta, "", "  "); err == nil {
                atomicWriteFile(filepath.Join(dir, "meta.json"), mb)
        }
        if vb, err := json.MarshalIndent(vars, "", "  "); err == nil {
                atomicWriteFile(filepath.Join(dir, "vars.json"), vb)
        }
        return sanitize(name), nil
}

func deleteGateway(name string) error {
        dir := gatewayDir(name)
        if dir == "" {
                return nil
        }
        return os.RemoveAll(dir)
}

// gatewayToEnv renders a gateway as a .env string for copy/export.
func gatewayToEnv(g Gateway) string {
        var sb strings.Builder
        sb.WriteString("# gateway: " + g.Name + "\n")
        if g.Description != "" {
                sb.WriteString("# " + g.Description + "\n")
        }
        sb.WriteString("\n")
        for _, v := range g.Vars {
                val := v.Value
                if strings.ContainsAny(val, "\n\r\" #") {
                        val = strconv.Quote(val)
                }
                sb.WriteString(v.Key + "=" + val + "\n")
        }
        return sb.String()
}

/* ---------- helpers ---------- */

func sendJSON(w http.ResponseWriter, code int, obj interface{}) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(code)
        json.NewEncoder(w).Encode(obj)
}

func uptimeSeconds() int64 {
        return int64(time.Since(startTime).Seconds())
}

/* ---------- audit log ---------- */

// auditEntry is a single line in the audit log.
type auditEntry struct {
        Time   string `json:"time"`
        Event  string `json:"event"`
        Name   string `json:"name"`
        Source string `json:"source"`
}

func auditFile() string {
        return filepath.Join(vaultDir, "audit.log")
}

// auditSafe strips anything that could break the " | " line format or inject
// fake entries: newlines, control chars, and the pipe separator itself.
func auditSafe(s string) string {
        s = strings.ReplaceAll(s, "|", "/")
        return strings.Map(func(r rune) rune {
                if r == '\n' || r == '\r' || (r < 0x20 && r != '\t') {
                        return -1
                }
                return r
        }, s)
}

var auditMu sync.Mutex

// auditLog appends one line to ~/Vault/audit.log. Format:
//
//      2026-08-19T15:04:05Z | secret.read | roblox | 127.0.0.1
//
// The file is created with 0600 perms so only the owner can read access history.
func auditLog(event, name, source string) {
        auditMu.Lock()
        defer auditMu.Unlock()
        if name == "" {
                name = "-"
        }
        if source == "" {
                source = "local"
        }
        line := fmt.Sprintf("%s | %s | %s | %s\n",
                time.Now().Format(time.RFC3339),
                auditSafe(event), auditSafe(name), auditSafe(source))
        f, err := os.OpenFile(auditFile(),
                os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
        if err != nil {
                return
        }
        defer f.Close()
        f.WriteString(line)
}

// readAuditLog returns the last `limit` entries from the audit log, newest last.
func readAuditLog(limit int) []auditEntry {
        data, err := os.ReadFile(auditFile())
        if err != nil {
                return []auditEntry{}
        }
        lines := strings.Split(strings.TrimSpace(string(data)), "\n")
        if len(lines) > limit {
                lines = lines[len(lines)-limit:]
        }
        result := make([]auditEntry, 0, len(lines))
        for _, line := range lines {
                parts := strings.Split(line, " | ")
                if len(parts) < 4 {
                        continue
                }
                result = append(result, auditEntry{
                        Time: parts[0], Event: parts[1],
                        Name: parts[2], Source: parts[3],
                })
        }
        return result
}

/* ---------- security middleware ---------- */

// addAllowedHosts merges extra comma-separated hostnames from VAULT_ALLOW_HOSTS
// into the Host allowlist, so the vault can sit behind a tunnel (ngrok,
// cloudflare) or a reverse proxy with a public hostname.
func addAllowedHosts(list string) {
        allowedMu.Lock()
        defer allowedMu.Unlock()
        for _, h := range strings.Split(list, ",") {
                if h = strings.TrimSpace(h); h != "" {
                        allowedHosts[normalizeHost(h)] = true
                }
        }
}

// buildAllowedHosts populates the Host/Origin allowlist. Loopback is always
// allowed; in LAN mode (bind to 0.0.0.0 / a LAN IP) every local interface IP
// plus the machine hostname is allowed, so phones/Termux can reach the vault.
// LAN IPs are re-scanned every 30s (updateLanHosts) so NIC reconnects and
// DHCP renewals never lock the phone out. Returns the displayable LAN URLs.
func buildAllowedHosts(port int, lanMode bool, bindHost string) []string {
        var lanURLs []string
        allowedMu.Lock()
        defer allowedMu.Unlock()
        add := func(host string) {
                if host == "" {
                        return
                }
                allowedHosts[normalizeHost(host)] = true
        }
        add("127.0.0.1")
        add("localhost")
        add("::1")
        add(bindHost)
        if lanMode {
                if hn, err := os.Hostname(); err == nil {
                        add(hn)
                }
                seen := map[string]bool{}
                for _, s := range scanLanIPs() {
                        add(s)
                        if ip := net.ParseIP(s); ip != nil && ip.To4() != nil && !seen[s] {
                                seen[s] = true
                                lanURLs = append(lanURLs, fmt.Sprintf("http://%s:%d", s, port))
                        }
                        lanHosts[normalizeHost(s)] = true
                }
        }
        return lanURLs
}

// scanLanIPs returns the machine's current non-loopback interface IPs.
// Note: on Windows, Go may not set FlagUp on interfaces that are perfectly
// functional (e.g. Ethernet), so only loopback is filtered out.
func scanLanIPs() []string {
        var ips []string
        ifaces, err := net.Interfaces()
        if err != nil {
                return ips
        }
        for _, ifc := range ifaces {
                if ifc.Flags&net.FlagLoopback != 0 {
                        continue
                }
                addrs, err := ifc.Addrs()
                if err != nil {
                        continue
                }
                for _, a := range addrs {
                        var ip net.IP
                        switch v := a.(type) {
                        case *net.IPNet:
                                ip = v.IP
                        case *net.IPAddr:
                                ip = v.IP
                        }
                        if ip != nil {
                                ips = append(ips, ip.String())
                        }
                }
        }
        return ips
}

// updateLanHosts re-scans interfaces and merges the diff into allowedHosts,
// so IPs that appear or disappear (NIC reconnect, DHCP, VPN) stay in sync.
func updateLanHosts() {
        allowedMu.Lock()
        defer allowedMu.Unlock()
        cur := map[string]bool{}
        for _, s := range scanLanIPs() {
                cur[normalizeHost(s)] = true
        }
        for h := range lanHosts {
                if !cur[h] {
                        delete(lanHosts, h)
                        delete(allowedHosts, h)
                }
        }
        for h := range cur {
                if !lanHosts[h] {
                        lanHosts[h] = true
                        allowedHosts[h] = true
                }
        }
}

// normalizeHost lowercases and strips IPv6 brackets for allowlist matching.
func normalizeHost(h string) string {
        h = strings.ToLower(strings.TrimSpace(h))
        h = strings.TrimPrefix(h, "[")
        h = strings.TrimSuffix(h, "]")
        return h
}

// hostAllowed reports whether a Host header / Origin host is one we actually
// serve. Matches both "host:port" and bare "host" forms, case-insensitively.
func hostAllowed(host string) bool {
        if host == "" {
                return true
        }
        host = normalizeHost(host)
        allowedMu.RLock()
        defer allowedMu.RUnlock()
        if allowedHosts[host] {
                return true
        }
        if h, _, err := net.SplitHostPort(host); err == nil {
                return allowedHosts[normalizeHost(h)]
        }
        return false
}

// isAllowedOrigin returns true if the URL's host is one of the hosts we serve
// (loopback, or LAN IPs when --lan/--host 0.0.0.0 is used), or the URL is empty
// (no Referer = direct API call from terminal/curl).
func isAllowedOrigin(rawURL string) bool {
        if rawURL == "" {
                return true
        }
        u, err := url.Parse(rawURL)
        if err != nil {
                return false
        }
        return hostAllowed(u.Host)
}

// securityMiddleware sets strict security headers and blocks any cross-origin
// request coming from a browser. Terminal AIs / curl don't send Origin, so they
// keep working. This is the single real attack surface for a localhost app:
// a malicious webpage making fetch() to 127.0.0.1:7575 to delete your vault.
func securityMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                h := w.Header()
                h.Set("X-Content-Type-Options", "nosniff")
                h.Set("X-Frame-Options", "DENY")
                h.Set("Referrer-Policy", "no-referrer")
                h.Set("X-XSS-Protection", "1; mode=block")
                // CSP: allow inline scripts/styles (single-file embedded UI), Google
                // Fonts for CSS + fonts, data: URIs for the favicon, and same-origin
                // for fetch. Everything else denied.
                h.Set("Content-Security-Policy",
                        "default-src 'self'; "+
                                "script-src 'self' 'unsafe-inline'; "+
                                "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
                                "font-src 'self' https://fonts.gstatic.com; "+
                                "img-src 'self' data:; "+
                                "connect-src 'self'")

                // Host header allowlist (DNS-rebinding defense in depth). Only the
                // hosts we actually serve (loopback, or local LAN IPs in LAN mode)
                // are accepted. Case-insensitive, "host:port" and bare "host" both match.
                if r.Host != "" && !hostAllowed(r.Host) {
                        log.Printf("blocked unknown host: host=%s path=%s", r.Host, r.URL.Path)
                        http.Error(w, "forbidden", http.StatusForbidden)
                        return
                }

                // Origin/Referer check
                origin := r.Header.Get("Origin")
                if origin != "" {
                        if !isAllowedOrigin(origin) {
                                log.Printf("blocked cross-origin request: origin=%s path=%s", origin, r.URL.Path)
                                http.Error(w, "forbidden", http.StatusForbidden)
                                return
                        }
                } else {
                        // No Origin header (terminal/curl/direct API). Still check Referer
                        // if a browser happened to send one without Origin.
                        referer := r.Header.Get("Referer")
                        if referer != "" && !isAllowedOrigin(referer) {
                                log.Printf("blocked cross-origin request: referer=%s path=%s", referer, r.URL.Path)
                                http.Error(w, "forbidden", http.StatusForbidden)
                                return
                        }
                }

                next.ServeHTTP(w, r)
        })
}

// statusWriter wraps http.ResponseWriter to capture the status code
type statusWriter struct {
        http.ResponseWriter
        status      int
        wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
        w.status = code
        w.wroteHeader = true
        w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
        if !w.wroteHeader {
                w.status = 200
                w.wroteHeader = true
        }
        return w.ResponseWriter.Write(b)
}

// loggingMiddleware logs every request as `METHOD path status duration`
func loggingMiddleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // skip noise from favicon and root html
                if r.URL.Path == "/favicon.ico" {
                        next.ServeHTTP(w, r)
                        return
                }
                start := time.Now()
                ww := &statusWriter{ResponseWriter: w, status: 200}
                next.ServeHTTP(ww, r)
                dur := time.Since(start)
                // dim color codes; terminal will render
                var color string
                switch {
                case ww.status >= 500:
                        color = "\033[31m" // red
                case ww.status >= 400:
                        color = "\033[33m" // amber
                case ww.status >= 300:
                        color = "\033[36m" // cyan
                default:
                        color = "\033[32m" // green
                }
                log.Printf("%s%-6s%-20s %d\033[0m %s",
                        color, r.Method+" ", r.URL.Path, ww.status, dur)
        })
}

/* ---------- port takeover ---------- */

func findPIDsOnPort(port int) []int {
        var out []int
        if runtime.GOOS == "windows" {
                // netstat -ano gives:  TCP  127.0.0.1:7575  0.0.0.0:0  LISTENING  12345
                // Parse the local address port exactly so ports like 75750 are not
                // matched by a naive substring search.
                cmd := exec.Command("cmd", "/c", "netstat -ano")
                if output, err := cmd.Output(); err == nil {
                        for _, line := range strings.Split(string(output), "\n") {
                                line = strings.TrimSpace(line)
                                if !strings.Contains(line, "LISTENING") {
                                        continue
                                }
                                fields := strings.Fields(line)
                                if len(fields) < 5 {
                                        continue
                                }
                                local := fields[1]
                                i := strings.LastIndex(local, ":")
                                if i < 0 {
                                        continue
                                }
                                if p, err := strconv.Atoi(local[i+1:]); err != nil || p != port {
                                        continue
                                }
                                pidStr := fields[len(fields)-1]
                                if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
                                        out = append(out, pid)
                                }
                        }
                }
        } else {
                // try lsof first, then ss
                cmd := exec.Command("sh", "-c",
                        fmt.Sprintf("lsof -i :%d -t 2>/dev/null || ss -tlnp 'sport = :%d' 2>/dev/null | grep -oE 'pid=[0-9]+' | grep -oE '[0-9]+'", port, port))
                if output, err := cmd.Output(); err == nil {
                        for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
                                line = strings.TrimSpace(line)
                                if pid, err := strconv.Atoi(line); err == nil && pid > 0 {
                                        out = append(out, pid)
                                }
                        }
                }
        }
        return out
}

func killPID(pid int) {
        if pid <= 0 {
                return
        }
        var cmd *exec.Cmd
        if runtime.GOOS == "windows" {
                cmd = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
        } else {
                cmd = exec.Command("kill", "-9", strconv.Itoa(pid))
        }
        cmd.Run()
}

func listenWithPortTakeover(addr string, port int) (net.Listener, error) {
        listener, err := net.Listen("tcp", addr)
        if err == nil {
                return listener, nil
        }
        // bind failed - try to find and kill the process holding the port
        log.Printf("port %d in use, attempting takeover...", port)
        pids := findPIDsOnPort(port)
        if len(pids) == 0 {
                return nil, fmt.Errorf("port %d in use and no PID found: %w", port, err)
        }
        for _, pid := range pids {
                log.Printf("  killing PID %d", pid)
                killPID(pid)
        }
        // give the OS a moment to release the port
        time.Sleep(700 * time.Millisecond)
        listener, err = net.Listen("tcp", addr)
        if err != nil {
                return nil, fmt.Errorf("port takeover failed: %w", err)
        }
        log.Printf("port %d taken over", port)
        return listener, nil
}

/* ---------- browser open ---------- */

func openBrowser(url string) {
        var cmd *exec.Cmd
        switch runtime.GOOS {
        case "windows":
                cmd = exec.Command("cmd", "/c", "start", "", url)
        case "darwin":
                cmd = exec.Command("open", url)
        default:
                cmd = exec.Command("xdg-open", url)
        }
        // run in background, don't block
        cmd.Stdin = nil
        cmd.Stdout = nil
        cmd.Stderr = nil
        cmd.Start()
}

/* ---------- HTTP handlers ---------- */

func main() {
        // CLI flags
        portFlag := flag.Int("port", defaultPort, "port to listen on")
        versionFlag := flag.Bool("version", false, "print version and exit")
        noBrowserFlag := flag.Bool("no-browser", false, "do not open browser on startup")
        hostFlag := flag.String("host", "127.0.0.1", "address to bind (0.0.0.0 allows phone/LAN access)")
        lanFlag := flag.Bool("lan", false, "shorthand for --host 0.0.0.0 (expose on your network)")
        flag.Parse()

        if *versionFlag {
                fmt.Printf("vault %s\n", version)
                return
        }

        port := *portFlag
        bindHost := *hostFlag
        if *lanFlag {
                bindHost = "0.0.0.0"
        }
        lanMode := bindHost != "127.0.0.1" && bindHost != "::1"
        lanURLs := buildAllowedHosts(port, lanMode, bindHost)
        addAllowedHosts(os.Getenv("VAULT_ALLOW_HOSTS"))
        if lanMode {
                go func() {
                        tick := time.NewTicker(30 * time.Second)
                        defer tick.Stop()
                        for range tick.C {
                                updateLanHosts()
                        }
                }()
        }
        home, err := os.UserHomeDir()
        if err != nil {
                log.Fatal(err)
        }
        vaultDir = filepath.Join(home, "Vault")
        secretsDir = filepath.Join(vaultDir, "Secrets")
        envVarsDir = filepath.Join(vaultDir, "EnvVars")
        gatewaysDir = filepath.Join(vaultDir, "Gateways")
        os.MkdirAll(vaultDir, 0755)
        os.MkdirAll(secretsDir, 0755)
        os.MkdirAll(envVarsDir, 0755)
        os.MkdirAll(gatewaysDir, 0755)
        migrateOldFormat()
        migrateLegacyAIToken()

        mux := http.NewServeMux()

        mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                if r.URL.Path != "/" {
                        http.NotFound(w, r)
                        return
                }
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                w.Header().Set("Cache-Control", "no-cache")
                w.Write(vaultHTML)
        })

        /* secrets endpoints */
        mux.HandleFunc("/api/list", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                writeMu.Lock()
                entries := listEntries()
                writeMu.Unlock()
                sendJSON(w, 200, map[string]interface{}{
                        "entries": entries,
                        "dir":     secretsDir,
                })
        })

        mux.HandleFunc("/api/get", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                name := r.URL.Query().Get("name")
                writeMu.Lock()
                v, ok := readValue(name)
                writeMu.Unlock()
                if !ok {
                        sendJSON(w, 404, map[string]string{"error": "not found"})
                        return
                }
                writeMu.Lock()
                details, _ := readDetails(name)
                writeMu.Unlock()
                sendJSON(w, 200, map[string]interface{}{
                        "value":   v,
                        "details": details,
                })
        })

        mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                r.Body = http.MaxBytesReader(w, r.Body, 131072)
                var req struct {
                        Name    string `json:"name"`
                        Value   string `json:"value"`
                        Details string `json:"details"`
                }
                if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                        sendJSON(w, 400, map[string]string{"error": "bad input"})
                        return
                }
                if req.Name == "" {
                        sendJSON(w, 400, map[string]string{"error": "name required"})
                        return
                }
                writeMu.Lock()
                stored, err := writeValue(req.Name, req.Value)
                writeMu.Unlock()
                if err != nil {
                        sendJSON(w, 400, map[string]string{"error": err.Error()})
                        return
                }
                writeMu.Lock()
                writeDetails(req.Name, req.Details)
                writeMu.Unlock()
                auditLog("secret.save", req.Name, clientIP(r))
                sendJSON(w, 200, map[string]interface{}{"ok": true, "stored": stored})
        })

        mux.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                name := r.URL.Query().Get("name")
                if name == "" {
                        sendJSON(w, 400, map[string]string{"error": "name required"})
                        return
                }
                writeMu.Lock()
                err := deleteValue(name)
                writeMu.Unlock()
                if err != nil {
                        sendJSON(w, 400, map[string]string{"error": err.Error()})
                        return
                }
                auditLog("secret.delete", name, clientIP(r))
                sendJSON(w, 200, map[string]bool{"ok": true})
        })

        /* env vars endpoints */
        mux.HandleFunc("/api/env/list", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                writeMu.Lock()
                entries := listEnvVars()
                writeMu.Unlock()
                sendJSON(w, 200, map[string]interface{}{
                        "entries": entries,
                        "dir":     envVarsDir,
                })
        })

        mux.HandleFunc("/api/env/get", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                key := r.URL.Query().Get("key")
                writeMu.Lock()
                v, secret, ok := readEnvVar(key)
                writeMu.Unlock()
                if !ok {
                        sendJSON(w, 404, map[string]string{"error": "not found"})
                        return
                }
                sendJSON(w, 200, map[string]interface{}{
                        "value":  v,
                        "secret": secret,
                })
        })

        mux.HandleFunc("/api/env/save", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                r.Body = http.MaxBytesReader(w, r.Body, 131072)
                var req struct {
                        Key    string `json:"key"`
                        Value  string `json:"value"`
                        Secret bool   `json:"secret"`
                }
                if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                        sendJSON(w, 400, map[string]string{"error": "bad input"})
                        return
                }
                if req.Key == "" {
                        sendJSON(w, 400, map[string]string{"error": "key required"})
                        return
                }
                writeMu.Lock()
                stored, err := writeEnvVar(req.Key, req.Value, req.Secret)
                writeMu.Unlock()
                if err != nil {
                        sendJSON(w, 400, map[string]string{"error": err.Error()})
                        return
                }
                auditLog("env.save", req.Key, clientIP(r))
                sendJSON(w, 200, map[string]interface{}{"ok": true, "stored": stored})
        })

        mux.HandleFunc("/api/env/delete", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                key := r.URL.Query().Get("key")
                if key == "" {
                        sendJSON(w, 400, map[string]string{"error": "key required"})
                        return
                }
                writeMu.Lock()
                err := deleteEnvVar(key)
                writeMu.Unlock()
                if err != nil {
                        sendJSON(w, 400, map[string]string{"error": err.Error()})
                        return
                }
                auditLog("env.delete", key, clientIP(r))
                sendJSON(w, 200, map[string]bool{"ok": true})
        })

        /* ---------- gateway endpoints ---------- */

        mux.HandleFunc("/api/gateway/list", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                writeMu.Lock()
                gateways := listGateways()
                writeMu.Unlock()
                sendJSON(w, 200, map[string]interface{}{
                        "entries": gateways,
                        "dir":     gatewaysDir,
                })
        })

        mux.HandleFunc("/api/gateway/get", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                name := r.URL.Query().Get("name")
                writeMu.Lock()
                g, ok := readGateway(name)
                writeMu.Unlock()
                if !ok {
                        sendJSON(w, 404, map[string]string{"error": "not found"})
                        return
                }
                sendJSON(w, 200, g)
        })

        mux.HandleFunc("/api/gateway/save", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
                var req struct {
                        Name        string       `json:"name"`
                        Type        string       `json:"type"`
                        Description string       `json:"description"`
                        Vars        []GatewayVar `json:"vars"`
                }
                if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                        sendJSON(w, 400, map[string]string{"error": "bad input"})
                        return
                }
                if req.Name == "" {
                        sendJSON(w, 400, map[string]string{"error": "name required"})
                        return
                }
                if req.Vars == nil {
                        req.Vars = []GatewayVar{}
                }
                writeMu.Lock()
                stored, err := writeGateway(req.Name, req.Type, req.Description, req.Vars)
                writeMu.Unlock()
                if err != nil {
                        sendJSON(w, 400, map[string]string{"error": err.Error()})
                        return
                }
                auditLog("gateway.save", req.Name, clientIP(r))
                sendJSON(w, 200, map[string]interface{}{"ok": true, "stored": stored})
        })

        mux.HandleFunc("/api/gateway/delete", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                name := r.URL.Query().Get("name")
                if name == "" {
                        sendJSON(w, 400, map[string]string{"error": "name required"})
                        return
                }
                writeMu.Lock()
                err := deleteGateway(name)
                writeMu.Unlock()
                if err != nil {
                        sendJSON(w, 400, map[string]string{"error": err.Error()})
                        return
                }
                auditLog("gateway.delete", name, clientIP(r))
                sendJSON(w, 200, map[string]bool{"ok": true})
        })

        /* export .env */
        mux.HandleFunc("/api/env/export", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                writeMu.Lock()
                vars := listEnvVars()
                writeMu.Unlock()
                var sb strings.Builder
                sb.WriteString("# exported from VAULT\n")
                sb.WriteString(fmt.Sprintf("# generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
                sb.WriteString("\n")
                for _, v := range vars {
                        sb.WriteString(envLine(v.Key, v.Value))
                }
                w.Header().Set("Content-Type", "text/plain; charset=utf-8")
                w.Header().Set("Content-Disposition", "attachment; filename=.env")
                w.Write([]byte(sb.String()))
        })

        /* import .env (raw text body) */
        mux.HandleFunc("/api/env/import", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
                body, err := io.ReadAll(r.Body)
                if err != nil {
                        sendJSON(w, 400, map[string]string{"error": "read error"})
                        return
                }
                lines := strings.Split(string(body), "\n")
                imported := 0
                for _, line := range lines {
                        line = strings.TrimSpace(line)
                        if line == "" || strings.HasPrefix(line, "#") {
                                continue
                        }
                        eq := strings.IndexByte(line, '=')
                        if eq < 0 {
                                continue
                        }
                        key := strings.TrimSpace(line[:eq])
                        val := strings.TrimSpace(line[eq+1:])
                        // strip optional surrounding quotes
                        if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"') {
                                val = val[1 : len(val)-1]
                        }
                        if key == "" {
                                continue
                        }
                        // auto-mark as secret if value looks sensitive
                        secret := looksSensitive(key, val)
                        writeMu.Lock()
                        writeEnvVar(key, val, secret)
                        writeMu.Unlock()
                        imported++
                }
                sendJSON(w, 200, map[string]interface{}{
                        "ok":       true,
                        "imported": imported,
                })
        })

        /* health check */
        mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                sendJSON(w, 200, map[string]interface{}{
                        "ok":      true,
                        "version": version,
                        "uptime":  uptimeSeconds(),
                })
        })

        /* version */
        mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                sendJSON(w, 200, map[string]interface{}{
                        "version":   version,
                        "goVersion": runtime.Version(),
                        "platform":  runtime.GOOS + "/" + runtime.GOARCH,
                        "port":      port,
                        "lanMode":   lanMode,
                })
        })

        /* audit log viewer - returns last 100 entries from ~/Vault/audit.log */
        mux.HandleFunc("/api/audit", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                entries := readAuditLog(100)
                sendJSON(w, 200, map[string]interface{}{
                        "entries": entries,
                        "count":   len(entries),
                })
        })

        /* one-click JSON backup of everything (secrets + env vars, values included) */
        mux.HandleFunc("/api/backup", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                writeMu.Lock()
                entries := listEntries()
                envVars := listEnvVars()
                writeMu.Unlock()
                type secretEntry struct {
                        Name    string `json:"name"`
                        Value   string `json:"value"`
                        Details string `json:"details,omitempty"`
                }
                type backupFile struct {
                        App        string        `json:"app"`
                        Version    string        `json:"version"`
                        ExportedAt int64         `json:"exportedAt"`
                        Secrets    []secretEntry `json:"secrets"`
                        EnvVars    []EnvVar      `json:"envVars"`
                }
                bf := backupFile{App: "vault", Version: version, ExportedAt: time.Now().UnixMilli()}
                for _, e := range entries {
                        writeMu.Lock()
                        v, _ := readValue(e.Name)
                        d, _ := readDetails(e.Name)
                        writeMu.Unlock()
                        bf.Secrets = append(bf.Secrets, secretEntry{Name: e.Name, Value: v, Details: d})
                }
                bf.EnvVars = envVars
                w.Header().Set("Content-Type", "application/json; charset=utf-8")
                w.Header().Set("Content-Disposition", `attachment; filename="vault-backup.json"`)
                w.Header().Set("Cache-Control", "no-store")
                json.NewEncoder(w).Encode(bf)
        })

        /* AI access: manage read tokens (UI only, browser origin-checked) */
        mux.HandleFunc("/api/ai/status", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                tokensMu.Lock()
                tokens := loadAITokens()
                tokensMu.Unlock()
                sendJSON(w, 200, map[string]interface{}{
                        "enabled": aiActiveCount(tokens) > 0,
                        "active":  aiActiveCount(tokens),
                        "tokens":  publicAITokens(tokens),
                })
        })

        mux.HandleFunc("/api/ai/token", func(w http.ResponseWriter, r *http.Request) {
                switch r.Method {
                case "GET":
                        // re-show a generated token (UI-only, origin-checked): the UI
                        // stores no token values, so the full value is fetched on demand
                        id := strings.TrimSpace(r.URL.Query().Get("id"))
                        tokensMu.Lock()
                        tokens := loadAITokens()
                        tokensMu.Unlock()
                        for _, t := range tokens {
                                if t.ID == id {
                                        sendJSON(w, 200, map[string]interface{}{
                                                "id": t.ID, "token": t.Token,
                                                "created": t.Created, "expires": t.Expires,
                                        })
                                        return
                                }
                        }
                        sendJSON(w, 404, map[string]string{"error": "token not found"})
                case "POST":
                        var req struct {
                                Days int `json:"days"`
                        }
                        json.NewDecoder(r.Body).Decode(&req)
                        if req.Days < 0 || req.Days > 365 {
                                req.Days = 7
                        }
                        tok, err := generateAIToken()
                        if err != nil {
                                sendJSON(w, 500, map[string]string{"error": "could not generate token"})
                                return
                        }
                        now := time.Now()
                        entry := AIToken{
                                ID:      tok[:8],
                                Token:   tok,
                                Created: now.Format(time.RFC3339),
                        }
                        if req.Days > 0 {
                                entry.Expires = now.AddDate(0, 0, req.Days).Format(time.RFC3339)
                        }
                        tokensMu.Lock()
                        tokens := append(loadAITokens(), entry)
                        err = saveAITokens(tokens)
                        tokensMu.Unlock()
                        if err != nil {
                                sendJSON(w, 500, map[string]string{"error": "could not save token"})
                                return
                        }
                        log.Printf("ai-access: token %s generated (expires %s)", entry.ID, entry.Expires)
                        sendJSON(w, 200, map[string]interface{}{
                                "token":   tok,
                                "id":      entry.ID,
                                "created": entry.Created,
                                "expires": entry.Expires,
                        })
                case "DELETE":
                        id := strings.TrimSpace(r.URL.Query().Get("id"))
                        tokensMu.Lock()
                        tokens := loadAITokens()
                        // BUG FIX: use make() instead of tokens[:0] to avoid
                        // slice aliasing - tokens[:0] shares the backing array
                        // and would corrupt the in-memory tokens slice during
                        // append, causing subtle state bugs on re-read.
                        kept := make([]AIToken, 0, len(tokens))
                        found := false
                        for _, t := range tokens {
                                if t.ID == id {
                                        found = true
                                        continue
                                }
                                kept = append(kept, t)
                        }
                        var err error
                        if found {
                                err = saveAITokens(kept)
                        }
                        tokensMu.Unlock()
                        if !found {
                                sendJSON(w, 404, map[string]string{"error": "token not found"})
                                return
                        }
                        if err != nil {
                                sendJSON(w, 500, map[string]string{"error": "could not save token"})
                                return
                        }
                        log.Printf("ai-access: token %s deleted", id)
                        sendJSON(w, 200, map[string]string{"ok": "true"})
                default:
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                }
        })

        /* AI access: token-protected read endpoints (no Origin needed, works from curl / AI tools) */
        mux.HandleFunc("/api/ai/list", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                if !aiBearerOK(r) {
                        sendJSON(w, 401, map[string]string{"error": "unauthorized"})
                        return
                }
                if !aiLimiter.allow(clientIP(r), 60, time.Minute) {
                        sendJSON(w, 429, map[string]string{"error": "rate limited"})
                        return
                }
                writeMu.Lock()
                secrets := listEntries()
                envVars := listEnvVars()
                writeMu.Unlock()
                type aiEntry struct {
                        Name string `json:"name"`
                        Kind string `json:"kind"`
                }
                entries := make([]aiEntry, 0, len(secrets)+len(envVars))
                for _, s := range secrets {
                        entries = append(entries, aiEntry{Name: s.Name, Kind: "secret"})
                }
                for _, e := range envVars {
                        entries = append(entries, aiEntry{Name: e.Key, Kind: "env"})
                }
                sendJSON(w, 200, map[string]interface{}{"entries": entries})
        })

        mux.HandleFunc("/api/ai/read", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                if !aiBearerOK(r) {
                        sendJSON(w, 401, map[string]string{"error": "unauthorized"})
                        return
                }
                if !aiLimiter.allow(clientIP(r), 60, time.Minute) {
                        sendJSON(w, 429, map[string]string{"error": "rate limited"})
                        return
                }
                name := strings.TrimSpace(r.URL.Query().Get("name"))
                if name == "" {
                        sendJSON(w, 400, map[string]string{"error": "name is required"})
                        return
                }
                writeMu.Lock()
                v, ok := readValue(name)
                if ok {
                        d, _ := readDetails(name)
                        writeMu.Unlock()
                        auditLog("ai.read.secret", name, clientIP(r))
                        log.Printf("ai-access: read secret %q from %s", name, clientIP(r))
                        sendJSON(w, 200, map[string]interface{}{"name": name, "kind": "secret", "value": v, "details": d})
                        return
                }
                ev, _, ok2 := readEnvVar(name)
                if ok2 {
                        writeMu.Unlock()
                        auditLog("ai.read.env", name, clientIP(r))
                        log.Printf("ai-access: read env var %q from %s", name, clientIP(r))
                        sendJSON(w, 200, map[string]interface{}{"name": name, "kind": "env", "value": ev})
                        return
                }
                writeMu.Unlock()
                sendJSON(w, 404, map[string]string{"error": "not found"})
        })

        /* tunnel: one-click HTTPS access for cloud AIs (UI only, browser origin-checked) */
        mux.HandleFunc("/api/tunnel", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                tunnelMu.Lock()
                u, started := tunnelURL, tunnelStarted
                running := tunnelCmd != nil
                tunnelMu.Unlock()
                sendJSON(w, 200, map[string]interface{}{
                        "running": running,
                        "url":     u,
                        "started": started.Format(time.RFC3339),
                })
        })

        mux.HandleFunc("/api/tunnel/start", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                u, err := startTunnel(port)
                if err != nil {
                        sendJSON(w, 500, map[string]string{"error": err.Error()})
                        return
                }
                log.Printf("tunnel: started %s", u)
                sendJSON(w, 200, map[string]interface{}{"running": true, "url": u})
        })

        mux.HandleFunc("/api/tunnel/stop", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "POST" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                stopTunnel()
                log.Printf("tunnel: stopped")
                sendJSON(w, 200, map[string]string{"ok": "true"})
        })

        /* stats */
        mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != "GET" {
                        sendJSON(w, 405, map[string]string{"error": "method not allowed"})
                        return
                }
                writeMu.Lock()
                secrets := listEntries()
                envVars := listEnvVars()
                writeMu.Unlock()
                var totalSize int64
                for _, s := range secrets {
                        totalSize += s.Size
                }
                for _, v := range envVars {
                        totalSize += int64(len(v.Value))
                }
                tokensMu.Lock()
                aiEnabled := aiActiveCount(loadAITokens()) > 0
                tokensMu.Unlock()
                sendJSON(w, 200, map[string]interface{}{
                        "secrets":         len(secrets),
                        "envVars":         len(envVars),
                        "totalSize":       totalSize,
                        "uptime":          uptimeSeconds(),
                        "version":         version,
                        "dir":             vaultDir,
                        "aiAccessEnabled": aiEnabled,
                        "lanMode":         lanMode,
                        "lanURLs":         lanURLs,
                })
        })

        // HTTP server with timeouts + middleware (security headers + origin check + logging)
        srv := &http.Server{
                Handler:           loggingMiddleware(securityMiddleware(mux)),
                ReadTimeout:       30 * time.Second,
                WriteTimeout:      30 * time.Second,
                IdleTimeout:       120 * time.Second,
                ReadHeaderTimeout: 10 * time.Second,
                MaxHeaderBytes:    1 << 20,
        }

        addr := fmt.Sprintf("%s:%d", bindHost, port)

        // port takeover: kill anything on this port before binding
        listener, err := listenWithPortTakeover(addr, port)
        if err != nil {
                log.Fatalf("cannot bind %s: %v", addr, err)
        }

        // graceful shutdown on signal
        go func() {
                sigChan := make(chan os.Signal, 1)
                signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
                <-sigChan
                fmt.Println()
                log.Printf("shutting down...")
                stopTunnel()
                ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
                defer cancel()
                srv.Shutdown(ctx)
        }()

        // colorful terminal banner
        endl := func() { fmt.Println() }
        green := func(s string) string { return "\033[32m" + s + "\033[0m" }
        dim := func(s string) string { return "\033[2m" + s + "\033[0m" }
        cyan := func(s string) string { return "\033[36m" + s + "\033[0m" }
        endl()
        fmt.Printf("  %sV%s A U L T   %ssecure local secrets%s   %sv%s%s\n",
                green(""), "", dim(""), "", dim(""), green(""), dim(version))
        fmt.Println("  ================================")
        fmt.Printf("  %s %shttp://127.0.0.1:%d%s\n", dim("listen"), green(""), port, dim(""))
        for _, u := range lanURLs {
                fmt.Printf("  %s %s%s%s\n", dim("lan   "), green(""), u, dim(""))
        }
        fmt.Printf("  %s %s%s%s\n", dim("secrets"), green(""), secretsDir, dim(""))
        fmt.Printf("  %s %s%s%s\n", dim("envvars"), green(""), envVarsDir, dim(""))
        fmt.Printf("  %s %s%s%s\n", dim("gateways"), green(""), gatewaysDir, dim(""))
        fmt.Printf("  %s %s%s%s\n", dim("os"), green(""), runtime.GOOS+"/"+runtime.GOARCH, dim(""))
        endl()
        if !*noBrowserFlag {
                fmt.Printf("  %s opening browser...%s\n", cyan(""), dim(""))
                go func() {
                        time.Sleep(800 * time.Millisecond)
                        openBrowser(fmt.Sprintf("http://127.0.0.1:%d", port))
                }()
        }
        fmt.Printf("  %sCtrl+C to stop%s\n", dim(""), dim(""))
        endl()

        log.Printf("vault %s listening on http://%s", version, addr)
        if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
                log.Fatal(err)
        }
        log.Printf("vault stopped")
}

// envLine renders one KEY=value line for .env export. Values containing
// whitespace or special characters are double-quoted; the value itself is
// written verbatim (UTF-8 preserved, no Go-style \uXXXX escapes).
func envLine(key, val string) string {
        if strings.ContainsAny(val, "\n\r\" ") || strings.HasPrefix(val, "#") {
                val = `"` + strings.ReplaceAll(val, `"`, `\"`) + `"`
        }
        return key + "=" + val + "\n"
}

// looksSensitive heuristically detects whether a key/value should be marked secret
func looksSensitive(key, val string) bool {
        k := strings.ToUpper(key)
        sensitiveWords := []string{"SECRET", "PASSWORD", "TOKEN", "API_KEY", "APIKEY", "PRIVATE", "CREDENTIAL", "AUTH", "ACCESS", "REFRESH"}
        for _, w := range sensitiveWords {
                if strings.Contains(k, w) {
                        return true
                }
        }
        return false
}
