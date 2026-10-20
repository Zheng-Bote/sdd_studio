package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/ui"
)

func runUI(args []string, repoRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 0, "Port auf 127.0.0.1 (0 = freier Port)")
	open := fs.Bool("open", false, "Oberfläche im Standardbrowser öffnen")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if _, err := config.Load(repoRoot); err != nil && !errors.Is(err, config.ErrConfigNotFound) {
		fmt.Fprintf(stderr, "sdd ui: %v\n", err)
		return 2
	}

	ln, err := ui.Listen(*port)
	if err != nil {
		fmt.Fprintf(stderr, "sdd ui: %v\n", err)
		return 2
	}
	token, err := ui.NewToken()
	if err != nil {
		fmt.Fprintf(stderr, "sdd ui: %v\n", err)
		return 2
	}
	project := "Projekt"
	if abs, err := filepath.Abs(repoRoot); err == nil {
		project = filepath.Base(abs)
	}
	addr := ln.Addr().String()
	srv := &http.Server{Handler: ui.New(repoRoot, addr, token, project), ReadHeaderTimeout: 5 * time.Second}

	url := "http://" + addr + "/?token=" + token
	fmt.Fprintf(stdout, "SDD Studio läuft auf %s (nur lokal). Beenden mit Strg+C.\n%s\n", addr, url)
	if *open {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(stderr, "sdd ui: Browser nicht startbar: %v\n", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "sdd ui: %v\n", err)
		return 2
	}
	return 0
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
