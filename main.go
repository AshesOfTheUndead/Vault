package main

import (
        _ "embed"
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
        "context"
)

//go:embed vault.html
var vaultHTML []byte

const (
        defaultPort = 7575
        version     = "2.2.0"
)

var (
        vaultDir     string
        secretsDir   string
        envVarsDir   string
        startTime    = time.Now()
        writeMu      sync.Mutex
        allowedHosts = map[string]bool{}
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

/* ---------- helpers ---------- */

func sendJSON(w http.ResponseWriter, code int, obj interface{}) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(code)
        json.NewEncoder(w).Encode(obj)
}

func uptimeSeconds() int64 {
        return int64(time.Since(startTime).Seconds())
}

/* ---------- security middleware ---------- */

// isLocalhostOrigin returns true if the URL's host is a loopback address
// or if the URL is empty (no Referer = direct API call from terminal/curl)
func isLocalhostOrigin(rawURL string) bool {
        if rawURL == "" {
                return true
        }
        u, err := url.Parse(rawURL)
        if err != nil {
                return false
        }
        host := u.Host
        if h, _, err := net.SplitHostPort(host); err == nil {
                host = h
        }
        switch host {
        case "127.0.0.1", "localhost", "::1", "[::1]", "0.0.0.0", "[::]":
                return true
        }
        return false
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
                // loopback hostnames we actually serve are accepted.
                if r.Host != "" && !allowedHosts[r.Host] {
                        log.Printf("blocked unknown host: host=%s path=%s", r.Host, r.URL.Path)
                        http.Error(w, "forbidden", http.StatusForbidden)
                        return
                }

                // Origin/Referer check
                origin := r.Header.Get("Origin")
                if origin != "" {
                        if !isLocalhostOrigin(origin) {
                                log.Printf("blocked cross-origin request: origin=%s path=%s", origin, r.URL.Path)
                                http.Error(w, "forbidden", http.StatusForbidden)
                                return
                        }
                } else {
                        // No Origin header (terminal/curl/direct API). Still check Referer
                        // if a browser happened to send one without Origin.
                        referer := r.Header.Get("Referer")
                        if referer != "" && !isLocalhostOrigin(referer) {
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
        flag.Parse()

        if *versionFlag {
                fmt.Printf("vault %s\n", version)
                return
        }

        port := *portFlag
        for _, h := range []string{
                fmt.Sprintf("127.0.0.1:%d", port),
                fmt.Sprintf("localhost:%d", port),
                fmt.Sprintf("[::1]:%d", port),
        } {
                allowedHosts[h] = true
        }
        home, err := os.UserHomeDir()
        if err != nil {
                log.Fatal(err)
        }
        vaultDir = filepath.Join(home, "Vault")
        secretsDir = filepath.Join(vaultDir, "Secrets")
        envVarsDir = filepath.Join(vaultDir, "EnvVars")
        os.MkdirAll(vaultDir, 0755)
        os.MkdirAll(secretsDir, 0755)
        os.MkdirAll(envVarsDir, 0755)
        migrateOldFormat()

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
                })
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
                sendJSON(w, 200, map[string]interface{}{
                        "secrets":   len(secrets),
                        "envVars":   len(envVars),
                        "totalSize": totalSize,
                        "uptime":    uptimeSeconds(),
                        "version":   version,
                        "dir":       vaultDir,
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

        addr := fmt.Sprintf("127.0.0.1:%d", port)

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
        fmt.Printf("  %s %s%s%s\n", dim("secrets"), green(""), secretsDir, dim(""))
        fmt.Printf("  %s %s%s%s\n", dim("envvars"), green(""), envVarsDir, dim(""))
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

        log.Printf("vault %s listening on http://127.0.0.1:%d", version, port)
        if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
                log.Fatal(err)
        }
        log.Printf("vault stopped")
}

// envLine renders one KEY=value line for .env export, quoting the value
// when it contains characters that would break the format.
func envLine(key, val string) string {
        if strings.ContainsAny(val, "\n\r\" ") || strings.HasPrefix(val, "#") {
                val = strconv.Quote(val)
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
