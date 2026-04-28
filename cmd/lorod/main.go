package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/example/loro-t/internal/daemon"
)

type config struct {
	SocketPath string
}

func parseConfig(args []string) (config, error) {
	defaultInfo, err := daemon.ResolveSocketPath()
	if err != nil {
		return config{}, err
	}

	cfg := config{}
	fs := flag.NewFlagSet("lorod", flag.ContinueOnError)
	fs.StringVar(&cfg.SocketPath, "socket-path", defaultInfo.SocketPath, "absolute path for Unix socket")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	if err := daemon.ValidateSocketPath(cfg.SocketPath); err != nil {
		return config{}, err
	}

	return cfg, nil
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cleanupStale, err := daemon.PrepareSocketPath(cfg.SocketPath)
	if err != nil {
		if errors.Is(err, daemon.ErrDaemonAlreadyRunning) {
			slog.Error("daemon already running", "socket_path", cfg.SocketPath, "uid", daemon.UIDString(), "cleanup_stale", false)
			fmt.Fprintln(os.Stderr, daemon.ErrDaemonAlreadyRunning.Error())
			os.Exit(1)
		}
		slog.Error("socket setup failed", "socket_path", cfg.SocketPath, "uid", daemon.UIDString(), "cleanup_stale", cleanupStale, "error", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	l, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		slog.Error("socket bind failed", "socket_path", cfg.SocketPath, "uid", daemon.UIDString(), "cleanup_stale", cleanupStale, "error", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.Remove(cfg.SocketPath)
	defer l.Close()

	if err := os.Chmod(cfg.SocketPath, 0o600); err != nil {
		slog.Error("chmod socket failed", "socket_path", cfg.SocketPath, "uid", daemon.UIDString(), "cleanup_stale", cleanupStale, "error", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	slog.Info("daemon listening", "socket_path", cfg.SocketPath, "uid", daemon.UIDString(), "cleanup_stale", cleanupStale)
	select {}
}
