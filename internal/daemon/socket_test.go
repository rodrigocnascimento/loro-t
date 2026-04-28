package daemon

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSocketPath(t *testing.T) {
	if err := ValidateSocketPath("relative.sock"); err == nil {
		t.Fatal("expected absolute path validation error")
	}
	if err := ValidateSocketPath("/tmp/loro.sock"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareSocketPathHandlesClosedSocketPath(t *testing.T) {
	tmp := t.TempDir()
	sock := filepath.Join(tmp, "daemon.sock")

	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	l.Close()

	_, statErr := os.Stat(sock)
	cleanup, err := PrepareSocketPath(sock)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	if statErr == nil && !cleanup {
		t.Fatal("expected stale cleanup when socket file exists")
	}
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatalf("expected socket removed or absent, stat err: %v", err)
	}
}

func TestPrepareSocketPathDetectsActiveSocket(t *testing.T) {
	tmp := t.TempDir()
	sock := filepath.Join(tmp, "daemon.sock")

	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	cleanup, err := PrepareSocketPath(sock)
	if !errors.Is(err, ErrDaemonAlreadyRunning) {
		t.Fatalf("expected ErrDaemonAlreadyRunning, got: %v", err)
	}
	if cleanup {
		t.Fatal("did not expect stale cleanup when daemon is active")
	}
}
