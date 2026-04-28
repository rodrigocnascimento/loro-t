package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

const socketFileName = "loro.sock"

var ErrDaemonAlreadyRunning = errors.New("daemon already running")

// RuntimeInfo describes the resolved runtime directory and whether XDG fallback was used.
type RuntimeInfo struct {
	SocketPath   string
	UID          int
	UsedFallback bool
}

// ResolveSocketPath returns the default socket path for the current user:
//   - $XDG_RUNTIME_DIR/loro/loro.sock when XDG_RUNTIME_DIR is defined
//   - /tmp/loro-<uid>/loro.sock as a documented fallback
func ResolveSocketPath() (RuntimeInfo, error) {
	uid := os.Getuid()
	xdg := os.Getenv("XDG_RUNTIME_DIR")
	if xdg != "" {
		return RuntimeInfo{
			SocketPath:   filepath.Join(xdg, "loro", socketFileName),
			UID:          uid,
			UsedFallback: false,
		}, nil
	}

	return RuntimeInfo{
		SocketPath:   filepath.Join("/tmp", fmt.Sprintf("loro-%d", uid), socketFileName),
		UID:          uid,
		UsedFallback: true,
	}, nil
}

// ValidateSocketPath enforces absolute Unix socket path.
func ValidateSocketPath(p string) error {
	if p == "" {
		return errors.New("socket path cannot be empty")
	}
	if !filepath.IsAbs(p) {
		return fmt.Errorf("socket path must be absolute: %q", p)
	}
	return nil
}

// PrepareSocketPath creates parent directory with owner-only permissions and
// clears stale socket files when no process is listening.
func PrepareSocketPath(socketPath string) (cleanupStale bool, err error) {
	if err := ValidateSocketPath(socketPath); err != nil {
		return false, err
	}

	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("create socket directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return false, fmt.Errorf("chmod socket directory: %w", err)
	}

	info, err := os.Stat(socketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat socket path: %w", err)
	}

	if info.Mode()&os.ModeSocket == 0 {
		return false, fmt.Errorf("path exists and is not a socket: %s", socketPath)
	}

	conn, dialErr := net.Dial("unix", socketPath)
	if dialErr == nil {
		_ = conn.Close()
		return false, ErrDaemonAlreadyRunning
	}

	if errors.Is(dialErr, syscall.ECONNREFUSED) || errors.Is(dialErr, syscall.ENOENT) {
		if err := os.Remove(socketPath); err != nil {
			return false, fmt.Errorf("remove stale socket: %w", err)
		}
		return true, nil
	}

	var opErr *net.OpError
	if errors.As(dialErr, &opErr) {
		if scErr, ok := opErr.Err.(*os.SyscallError); ok {
			if errors.Is(scErr.Err, syscall.ECONNREFUSED) || errors.Is(scErr.Err, syscall.ENOENT) {
				if err := os.Remove(socketPath); err != nil {
					return false, fmt.Errorf("remove stale socket: %w", err)
				}
				return true, nil
			}
		}
	}

	return false, fmt.Errorf("probe existing socket: %w", dialErr)
}

func UIDString() string {
	return strconv.Itoa(os.Getuid())
}
