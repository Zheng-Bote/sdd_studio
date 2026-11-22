# Code Review & Security Audit: SDD Studio

**Datum:** 22. November 2026  
**Ziel:** Überprüfung des aktuellen Quellcodes hinsichtlich Security, Clean Code, Architektur und Best Practices.  
**Umfang:**
- Go Core & CLI (`cmd/sdd`, `cmd/sdd-desk`, `internal/sddcore`, `internal/desk`, `internal/mcp`, `internal/ui`)
- Desktop Shell & Frontend (`apps/desk`: Angular 19/22, Tauri v2, Rust Sidecar)
- Konfiguration & Templates (`sdd.config.yaml`, `config/`, `templates/`)

---

## 1. Management Summary & Gesamtbewertung

SDD Studio demonstriert ein **außerordentlich hohes Niveau an Softwarequalität, Architekturdisziplin und sicherheitsbewusstem Design**. 

### Wesentliche Stärken:
- **Minimale Angriffsfläche (Supply Chain):** Das Go-Backend verzichtet fast vollständig auf externe Abhängigkeiten (`gopkg.in/yaml.v3` ist die einzige Dependency). Keine transitive Abhängigkeitshölle.
- **Strikte architektonische Trennung (ADR 0002):** Der SDD-Kern (`sddcore`, `sdd CLI`) ist vollständig deterministisch und enthält **keinen KI- oder Netzwerk-Code**. Der KI-Desktop (`sdd-desk`) kommuniziert mit SDD ausschließlich über einen isolierten Kindprozess (`sdd mcp`) via Stdio-JSON-RPC 2.0.
- **Human-in-the-Loop & Kritische Aktionen (ADR 0003, ADR 0004, ADR 0008):** Schreibende und zustandsverändernde Operationen (Baseline, ADR-Entscheidungen, Config-Änderungen, Commits) sind gestaffelt abgesichert und verlangen zwingend menschliche Freigabe (keine Session-Grants für kritische Tools).
- **Robuste Web-Sicherheit:** Lokale HTTP-Server binden ausschließlich an `127.0.0.1`, erzwingen kryptografische Session-Tokens (`crypto/rand`, Zeitkonstanten-Vergleich mit `subtle.ConstantTimeCompare`), validieren den `Host`-Header gegen DNS-Rebinding und nutzen strikte Content Security Policies.
- **XSS-Freiheit:** Sowohl im Angular-Frontend als auch in der leichtgewichtigen UI (`internal/ui/web`) wird **konsequent auf `innerHTML` verzichtet**. Alle Inhalte werden sicher über DOM-Textknoten (`createTextNode` / `textContent`) gerendert.
- **Keine Shell-Injektion:** Alle Systemaufrufe (`exec.Command`) verzichten auf Shells (`sh -c`) und nutzen explizite Argumentlisten; Git-Befehle verwenden `--` zur Trennung von Parametern und Pfaden.

### Handlungsbedarf / Identifizierte Schwachstellen:
Es wurden **keine kritischen Sicherheitslücken (Critical/High)** gefunden. Es gibt jedoch einen Befund mittlerer Schwere bezüglich Symlink-Handling beim Öffnen von Dateien im Editor sowie mehrere Punkte zur Verbesserung der Robustheit, Speicherbereinigung und des Clean Codes.

---

## 2. Befundmatrix

| ID | Bereich | Kategorie | Schweregrad | Betroffene Datei(en) | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SEC-01** | Security | Path Traversal / Symlink Follow | **Mittel** | `internal/ui/api.go` (handleOpen) | Empfehlung |
| **SEC-02** | Security | Defense-in-Depth / CSRF | **Niedrig** | `internal/desk/server.go` (ServeHTTP) | Empfehlung |
| **RES-01** | Clean Code | Resource Leak / Memory | **Niedrig** | `internal/desk/mcpclient.go` (Call) | Empfehlung |
| **RES-02** | Clean Code | Concurrency / Idempotenz | **Niedrig** | `internal/desk/mcpclient.go` (Close) | Empfehlung |
| **CC-01** | Clean Code | Dead Code | **Hinweis** | `internal/mcp/write.go` (toolWrite) | Empfehlung |
| **CC-02** | Clean Code | Code Smell / String Parsing | **Hinweis** | `internal/ui/markdown.go` (inline) | Empfehlung |
| **CC-03** | Clean Code | Default Fallthrough | **Hinweis** | `internal/mcp/critical.go` (runCritical) | Empfehlung |
| **CC-04** | Clean Code | DRY (Code-Duplikation) | **Hinweis** | `cmd/sdd/ui.go`, `cmd/sdd-desk/main.go` | Empfehlung |

---

## 3. Detaillierte Befunde & Analyse

### SEC-01: Symlink-Auflösung in `handleOpen` umgeht Pfadgrenzen
- **Datei:** [internal/ui/api.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/ui/api.go#L259-L288)
- **Kategorie:** Path Traversal / Information Disclosure
- **Schweregrad:** Mittel
- **Beschreibung:**  
  Im Endpunkt `/api/open` wird `scan.CleanRel(req.Path)` aufgerufen. Dies stellt sicher, dass der Pfad syntaktisch keine `..`-Sequenzen enthält. Anschließend wird der Pfad mit `abs := filepath.Join(s.Root, filepath.FromSlash(rel))` gebildet und mit `os.Stat(abs)` geprüft:
  ```go
  abs := filepath.Join(s.Root, filepath.FromSlash(rel))
  if info, err := os.Stat(abs); err != nil || !info.Mode().IsRegular() {
      writeError(w, http.StatusNotFound, fmt.Errorf("%s ist keine Datei im Repository", rel))
      return
  }
  ...
  if err := s.Exec(editor[0], args...); err != nil { ... }
  ```
  `os.Stat` folgt symbolischen Links. Befindet sich im Repository ein symbolischer Link, der auf eine Datei außerhalb des Repositories zeigt (z. B. `/etc/passwd` oder ein privater SSH-Schlüssel), liefert `os.Stat` Erfolg und `info.Mode().IsRegular()` ist wahr. Anschließend wird der konfigurierte `$EDITOR` mit dieser externen Datei gestartet.  
  Im Vergleich dazu prüfen `readExisting` in [internal/mcp/write.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/mcp/write.go#L95-L141) und `Plan.Apply` in [internal/sddcore/scaffold/plan.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/sddcore/scaffold/plan.go#L150-L161) Symlinks explizit via `os.Lstat` und `insideRoot` / `filepath.EvalSymlinks`.
- **Empfehlung:**  
  In `handleOpen` sollte vor dem Starten des Editors entweder symbolische Links via `os.Lstat` abgewiesen werden (`if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0`), oder `insideRoot(s.Root, rel)` aufgerufen werden.

---

### SEC-02: Fehlende Origin-Prüfung in `sdd-desk` (Defense-in-Depth)
- **Datei:** [internal/desk/server.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/desk/server.go#L62-L85)
- **Kategorie:** Cross-Origin Request Forgery (CSRF)
- **Schweregrad:** Niedrig
- **Beschreibung:**  
  Während `internal/ui/server.go:94` für alle schreibenden Anfragen explizit prüft:
  ```go
  if r.Header.Get("Origin") != "http://"+s.Addr {
      writeError(w, http.StatusForbidden, errors.New("unerwarteter Origin"))
      return
  }
  ```
  prüft `sdd-desk` in `ServeHTTP` lediglich den `Host`-Header und das Vorhandensein des `X-SDD-Token`. Da moderne Webbrowser das Setzen von benutzerdefinierten Headern wie `X-SDD-Token` bei Cross-Origin-Requests ohne CORS-Freigabe (die nicht vorhanden ist) unterbinden, ist das Risiko gering. Als Defense-in-Depth-Maßnahme gegen Browser-Edge-Cases oder Fehlkonfigurationen sollte aber auch in `sdd-desk` die `Origin`-Validierung implementiert werden.
- **Empfehlung:**  
  Ebenso wie in `internal/ui` den `Origin`-Header bei `/api/`-POST-Anfragen gegen `"http://" + s.Addr` prüfen.

---

### RES-01: Leak von pending Callbacks bei Timeout in `MCPClient.Call`
- **Datei:** [internal/desk/mcpclient.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/desk/mcpclient.go#L101-L132)
- **Kategorie:** Memory Leak / Resource Management
- **Schweregrad:** Niedrig
- **Beschreibung:**  
  In `MCPClient.Call` wird eine Anfrage mit ID im `pending`-Map registriert:
  ```go
  c.pending[id] = ch
  ...
  select {
  case r := <-ch:
      ...
  case <-c.closed:
      return nil, c.err
  case <-time.After(c.Timeout):
      return nil, fmt.Errorf("sdd mcp: keine Antwort auf %s", method)
  }
  ```
  Läuft der Aufruf in den Timeout (`time.After`), wird die Funktion beendet, ohne dass `id` aus `c.pending` entfernt wird. Antwortet der Kindprozess verspätet nie mehr, verbleibt der Channel `ch` dauerhaft in `c.pending`. Zudem allokiert `time.After` einen Timer, der erst nach Ablauf der vollen Timeout-Dauer (Standard 2 Minuten) Garbage-Collected wird.
- **Empfehlung:**  
  Aufräumen per `defer`:
  ```go
  defer func() {
      c.mu.Lock()
      delete(c.pending, id)
      c.mu.Unlock()
  }()
  ```

---

### RES-02: Fehlende Idempotenz bei `MCPClient.Close`
- **Datei:** [internal/desk/mcpclient.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/desk/mcpclient.go#L210-L223)
- **Kategorie:** Concurrency / Goroutine Lifecycle
- **Schweregrad:** Niedrig
- **Beschreibung:**  
  `Close()` startet bei vorhandenem `c.cmd` eine Goroutine, die `c.cmd.Wait()` ausführt:
  ```go
  if c.cmd != nil {
      done := make(chan error, 1)
      go func() { done <- c.cmd.Wait() }()
      select {
      case <-done:
      case <-time.After(3 * time.Second):
          _ = c.cmd.Process.Kill()
      }
  }
  ```
  Wird `Close()` mehrfach aufgerufen (z. B. explizit im Shutdown und zusätzlich per `defer`), wird `cmd.Wait()` mehrfach für denselben Prozess aufgerufen, was zu einem Fehler führt. Zudem wird nach `Kill()` nicht garantiert abgewartet, bis der Prozess-Tabelleneintrag (Zombie) freigegeben wird.
- **Empfehlung:**  
  `MCPClient.Close()` mit `sync.Once` absichern.

---

### CC-01: Toter Code in `internal/mcp/write.go:toolWrite`
- **Datei:** [internal/mcp/write.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/mcp/write.go#L231-L253)
- **Kategorie:** Clean Code / Dead Code
- **Schweregrad:** Hinweis
- **Beschreibung:**  
  In Zeile 231–233 wird bereits abgefangen:
  ```go
  if class == classConfig {
      return nil, fmt.Errorf("nicht beschreibbar mit sdd_write: %s ändert nur sdd_config (kritisch, ADR 0004)", rel)
  }
  ```
  Dadurch ist der nachfolgende `switch`-Zweig in Zeile 248–252 unerreichbar (Dead Code):
  ```go
  case classConfig:
      if _, err := config.Parse([]byte(a.Content)); err != nil {
          return nil, fmt.Errorf("sdd.config.yaml nicht geschrieben: %w", err)
      }
  ```
- **Empfehlung:**  
  Den toten `case classConfig:`-Block entfernen.

---

### CC-02: ASCII-Arithmetik für Code-Platzhalter in `internal/ui/markdown.go`
- **Datei:** [internal/ui/markdown.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/ui/markdown.go#L116-L135)
- **Kategorie:** Clean Code / Robustheit
- **Schweregrad:** Hinweis
- **Beschreibung:**  
  Beim Rendern von Inline-Codeblöcken wird ein Platzhalter erzeugt:
  ```go
  return "\x00" + string(rune(len(codes)-1+'A')) + "\x00"
  ```
  Wenn ein Handbuchdokument mehr als 26 Code-Schnipsel in einem Textblock enthält, läuft `rune('A' + 26)` über das lateinische Alphabet hinaus in ASCII-Sonderzeichen (`[`, `\`, `]`, `^`, etc.).
- **Empfehlung:**  
  Statt Buchstaben-Offsets einen expliziten Integer-Index verwenden (z. B. `fmt.Sprintf("\x00C%d\x00", len(codes)-1)`).

---

### CC-03: `default`-Zweig in `runCritical` führt implizit `toolAdoptApply` aus
- **Datei:** [internal/mcp/critical.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/internal/mcp/critical.go#L48-L60)
- **Kategorie:** Clean Code / Defensive Programming
- **Schweregrad:** Hinweis
- **Beschreibung:**  
  `runCritical` schaltet über den Namen:
  ```go
  switch name {
  case "sdd_baseline":
      return s.toolBaseline(raw)
  case "sdd_adr_accept":
      return s.toolADRAccept(raw)
  case "sdd_config":
      return s.toolConfig(raw)
  case "sdd_commit":
      return s.toolCommit(raw)
  default:
      return s.toolAdoptApply(raw)
  }
  ```
  Wird später ein weiteres Tool zur Whitelist hinzugefügt oder ein Tippfehler gemacht, fällt der Aufruf automatisch auf `toolAdoptApply` zurück.
- **Empfehlung:**  
  Expliziter Fall `case "sdd_adopt_apply": return s.toolAdoptApply(raw)` und im `default:` einen Fehler `fmt.Errorf("%w %q", errUnknownTool, name)` zurückgeben.

---

### CC-04: Code-Duplikation bei `openBrowser`
- **Dateien:** [cmd/sdd/ui.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/cmd/sdd/ui.go#L74-L85) und [cmd/sdd-desk/main.go](file:///Users/zb_bamboo/Documents/DEV/sdd_studio/cmd/sdd-desk/main.go#L170-L181)
- **Kategorie:** Clean Code (DRY)
- **Schweregrad:** Hinweis
- **Beschreibung:**  
  Die Funktion `openBrowser` (mit Plattform-Switch für `darwin`, `windows` und `xdg-open`) ist in beiden CLI-Entrypoints zeilengleich dupliziert.
- **Empfehlung:**  
  In ein gemeinsames Hilfspaket auslagern oder wiederverwenden.

---

## 4. Sicherheits- und Architektur-Prüfungen im Detail

### 4.1 Secrets-Management & Vertraulichkeit
- `config/llm.env` ist korrekt in `.gitignore` hinterlegt und wurde laut Git-Historie nie eingecheckt.
- `desk.LoadConfig` prüft die Dateiberechtigung unter Unix/macOS strikt auf `0600` (`info.Mode().Perm() & 0o077 != 0` wirft `ErrInsecureConfig`).
- API-Schlüssel sind in Go-Strukturen mit `json:"-"` versehen, sodass sie in `/api/meta` oder Fehlermeldungen nie serialisiert werden.
- In der Tauri-Shell (`apps/desk/src-tauri/src/sidecar.rs`) werden Session-Tokens über die Funktion `redact` aus allen Log- und Konsolenausgaben getilgt.

### 4.2 Git CLI & Injection-Schutz
- Alle Git-Aufrufe nutzen `exec.Command("git", ...)` ohne Shell-Wrapper.
- `CLIGit.Commit` trennt Dateipfade explizit mit `--` ab (`git commit -m "..." -- <paths>`), was verhindert, dass Dateinamen als Git-Flags interpretiert werden.
- Branch-Namen werden vor der Erstellung strikt mit `git check-ref-format --branch` validiert.

### 4.3 DoS & Ressourcenbegrenzung
- Maximale Request-Body-Größen sind fest definiert (`maxBody = 64 KB` in `ui`, `8 MB` in `sdd-desk`).
- Streaming-Puffer sind begrenzt (`maxStreamLine = 1 MB`, `maxLine = 16 MB` in MCP).
- Die Agenten-Schleife in `sdd-desk` (`Agent.run`) ist strikt auf maximal 8 Iterationen pro Turn beschränkt (`maxModelSteps = 8`), wodurch Endlosschleifen zwischen LLM und Tools zuverlässig verhindert werden.

### 4.4 Frontend & Tauri v2 Sandbox
- **Tauri Navigation Guard:** Die Rust-Hülle (`navigation` in `lib.rs`) erlaubt Navigation ausschließlich auf die interne Startseite und den lokalen Sidecar-Origin (`http://127.0.0.1:<port>`). Externe URLs werden im Systembrowser geöffnet, alle anderen Schemas (`file:`, `javascript:`, `data:`) werden geblockt.
- **Token Scrubbing:** Das Angular-Frontend (`DeskApi.takeToken`) extrahiert den Token aus der URL, legt ihn im flüchtigen `sessionStorage` ab und entfernt den Query-Parameter sofort mittels `history.replaceState` aus der Browser-Adressleiste.
- **Keine XSS-Sinks:** Die Analyse des Angular- und UI-Codes zeigt 0 Treffer für `innerHTML`, `dangerouslySetInnerHTML`, `bypassSecurityTrust*` oder `eval`.

---

## 5. Fazit & Priorisierte Handlungsschritte

Der Code befindet sich in einem **hervorragenden Gesamtzustand**. Die Softwarearchitektur ist durchdacht, die ADRs sind vorbildlich umgesetzt und der Sicherheitsfokus ist im gesamten Projekt spürbar.

Zur weiteren Härtung werden folgende, risikoarme Optimierungen empfohlen:
1. **Kurzfristig (Security):** In `internal/ui/api.go` (`handleOpen`) `os.Lstat` oder `insideRoot` verwenden, um das Verfolgen von Symlinks außerhalb des Repositories auszuschließen (SEC-01).
2. **Mittelfristig (Robustheit):** In `internal/desk/mcpclient.go` die `pending`-Map bei Timeouts per `defer delete(...)` bereinigen (RES-01) und `Close()` mit `sync.Once` absichern (RES-02).
3. **Refactoring (Clean Code):** Den toten `case classConfig:`-Zweig in `internal/mcp/write.go` entfernen (CC-01) und `runCritical` mit explizitem `default: return error` ausstatten (CC-03).
