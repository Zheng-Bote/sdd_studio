// Command sdd-desk is the optional AI desktop for sdd (ADR 0002). It is built and run separately from sdd.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/desk"
)

const systemPreamble = `Du bist der Begleiter von SDD Studio. Du führst den Nutzer durch Spec-Driven Development: von der Produkt-Idee über PRD, Tech Spec, ADR und Plan bis zu Tests, Gate und Archiv. Antworte auf Deutsch, knapp und konkret. Stelle eine Frage nach der anderen und erfinde keine Anforderungen. Nutze die sdd-Werkzeuge, um den Stand zu prüfen und Entwürfe zu validieren; jeder Aufruf wird dem Nutzer zur Erlaubnis vorgelegt.`

func main() {
	os.Exit(run())
}

func run() int {
	project := flag.String("project", ".", "Projektverzeichnis (Repository mit sdd.config.yaml)")
	configPath := flag.String("config", "", "llm.env (Standard: config/llm.env neben sdd-desk)")
	sddPath := flag.String("sdd", "", "Pfad zu sdd (Standard: neben sdd-desk, sonst PATH)")
	port := flag.Int("port", 0, "Port auf 127.0.0.1 (0 = freier Port)")
	open := flag.Bool("open", false, "im Standardbrowser öffnen")
	flag.Parse()

	root, err := filepath.Abs(*project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdd-desk: %v\n", err)
		return 2
	}
	explicit := *configPath != ""
	if !explicit {
		*configPath = desk.DefaultConfigPath()
	}
	cfg, err := desk.LoadConfig(*configPath, explicit, os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdd-desk: %v\n", err)
		return 2
	}

	bin := findSdd(*sddPath)
	mcp, err := desk.StartMCP(bin, root, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdd-desk: %v\n", err)
		return 2
	}
	defer mcp.Close()
	tools, err := mcp.Tools()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdd-desk: %v\n", err)
		return 2
	}

	httpClient := &http.Client{Timeout: 10 * time.Minute}
	agent := &desk.Agent{
		Tools: tools, Runner: mcp, Providers: cfg.Providers,
		NewChat: func(p desk.Provider) desk.Chat { return desk.NewChat(p, httpClient) },
		System:  systemPreamble + "\n\n" + mcp.Instructions,
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdd-desk: %v\n", err)
		return 2
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintf(os.Stderr, "sdd-desk: %v\n", err)
		return 2
	}
	token := hex.EncodeToString(b)
	srv := &desk.Server{Addr: ln.Addr().String(), Token: token, Project: filepath.Base(root), Config: cfg, Agent: agent, MCP: mcp,
		HTTP: &http.Client{Timeout: 3 * time.Second}}
	httpSrv := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second}

	url := "http://" + srv.Addr + "/?token=" + token
	fmt.Printf("sdd-desk läuft auf %s für %s (Konfiguration: %s). Beenden mit Strg+C.\n%s\n", srv.Addr, root, cfg.Path, url)
	for _, w := range cfg.Warnings {
		fmt.Fprintf(os.Stderr, "sdd-desk: %s\n", w)
	}
	if *open {
		_ = openBrowser(url)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdown)
	}()
	if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "sdd-desk: %v\n", err)
		return 2
	}
	return 0
}

// findSdd prefers an explicit path, then sdd next to sdd-desk (Tauri sidecar layout), then PATH.
func findSdd(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if exe, err := os.Executable(); err == nil {
		name := "sdd"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if p := filepath.Join(filepath.Dir(exe), name); fileExists(p) {
			return p
		}
	}
	if p, err := exec.LookPath("sdd"); err == nil {
		return p
	}
	return "sdd"
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
