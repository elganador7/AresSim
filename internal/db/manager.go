// Package db manages the SurrealDB subprocess lifecycle and connection for AresSim.
//
// SurrealDB runs as a child process bound to loopback only. The binary is
// either bundled alongside the application executable or resolved from PATH.
// Storage uses file-based persistence in the OS app data directory so the
// simulation state survives process restarts.
package db

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultPort     = 18765
	defaultUsername = "root"
	defaultPassword = "aressim"
	defaultNS       = "aressim"
	defaultDB       = "engine"

	// healthCheckInterval is how often we poll the TCP port while waiting for ready.
	healthCheckInterval = 100 * time.Millisecond
	// healthCheckTimeout is the maximum time we wait for SurrealDB to become ready.
	healthCheckTimeout = 15 * time.Second
	// shutdownGracePeriod is how long we wait for a clean shutdown before SIGKILL.
	shutdownGracePeriod = 5 * time.Second

	// UseInMemoryDB controls whether SurrealDB runs with the in-memory backend
	// instead of the file-based surrealkv store. Set to true during development
	// to avoid LOCK file conflicts and schema migration friction. All repository
	// and schema code remains active — only the storage layer changes.
	UseInMemoryDB = true
)

// Config holds all tunable parameters for the SurrealDB subprocess.
type Config struct {
	// BinaryPath is the absolute path to the surreal executable.
	// If empty, Manager will search the bundle directory then $PATH.
	BinaryPath string

	// DataDir is the directory where the database files will be stored.
	// If empty, the OS app data directory is used.
	DataDir string

	// Port is the TCP port SurrealDB will bind to on 127.0.0.1.
	// If the preferred port is busy, Manager will try up to 9 adjacent ports.
	Port int

	// Username and Password are the root credentials for the local instance.
	// These are only accessible from the loopback interface.
	Username string
	Password string

	// Namespace and Database select the SurrealDB logical context.
	Namespace string
	Database  string
}

// DefaultConfig returns a Config with sensible defaults for a desktop deployment.
func DefaultConfig() (Config, error) {
	dataDir, err := AppDataDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolve app data dir: %w", err)
	}
	return Config{
		DataDir:   dataDir,
		Port:      DefaultPort,
		Username:  defaultUsername,
		Password:  defaultPassword,
		Namespace: defaultNS,
		Database:  defaultDB,
	}, nil
}

// Manager owns the SurrealDB OS process. Exactly one Manager should exist per
// application instance. Obtain a connected *Client via Manager.Client() after
// calling Start.
type Manager struct {
	cfg    Config
	mu     sync.Mutex
	proc   *os.Process
	stopCh chan struct{}
}

// NewManager creates a Manager. Call Start to spawn the process.
func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg, stopCh: make(chan struct{})}
}

// Start spawns the SurrealDB process, waits for it to accept connections,
// and verifies it is healthy. Returns an error if any step fails.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Kill any orphaned surreal processes from previous unclean exits.
	// This is always needed — in-memory mode leaves no LOCK file, so port
	// exhaustion is the only symptom of leaked processes.
	killOrphanedSurrealProcesses(m.cfg.Port)

	if !UseInMemoryDB {
		if err := os.MkdirAll(m.cfg.DataDir, 0o700); err != nil {
			return fmt.Errorf("create data dir %q: %w", m.cfg.DataDir, err)
		}

		// Remove a stale LOCK file left behind by a previous unclean shutdown.
		lockPath := filepath.Join(m.cfg.DataDir, "surreal.db", "LOCK")
		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale lock file: %w", err)
		}
	}

	binary, err := m.resolveBinary()
	if err != nil {
		return err
	}

	port, err := findFreePort(m.cfg.Port)
	if err != nil {
		return fmt.Errorf("find free port: %w", err)
	}
	m.cfg.Port = port

	args := m.buildArgs()
	cmd := exec.Command(binary, args...)

	// Redirect SurrealDB output to our stderr so Wails can capture it.
	// In production builds this will be suppressed by --log warn.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	// Isolate SurrealDB in its own process group so that Ctrl-C on the
	// terminal sends SIGINT only to the Wails process. Our signal handler
	// (registered in app.go startup) will then call Manager.Stop() which
	// sends SIGINT to SurrealDB gracefully before the parent exits.
	setProcGroup(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start surreal process: %w", err)
	}
	m.proc = cmd.Process

	// Monitor the process: if it exits early, capture the error.
	exitCh := make(chan error, 1)
	go func() {
		exitCh <- cmd.Wait()
	}()

	// Wait for the port to be accepting connections.
	if err := m.waitReady(ctx, port, exitCh); err != nil {
		_ = m.proc.Kill()
		return fmt.Errorf("surreal did not become ready: %w", err)
	}

	return nil
}

// Stop sends SIGINT to SurrealDB and waits for it to exit cleanly.
// If the process does not exit within shutdownGracePeriod, it is killed.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.proc == nil {
		return
	}

	// Graceful: interrupt signal allows SurrealDB to flush RocksDB/SurrealKV.
	_ = m.proc.Signal(os.Interrupt)

	done := make(chan struct{})
	go func() {
		_, _ = m.proc.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(shutdownGracePeriod):
		_ = m.proc.Kill()
	}
	m.proc = nil
}

// Addr returns the WebSocket RPC address for this Manager's SurrealDB instance.
func (m *Manager) Addr() string {
	return fmt.Sprintf("ws://127.0.0.1:%d/rpc", m.cfg.Port)
}

// Credentials returns the username and password for this instance.
func (m *Manager) Credentials() (username, password string) {
	return m.cfg.Username, m.cfg.Password
}

// NS returns the namespace name.
func (m *Manager) NS() string { return m.cfg.Namespace }

// DB returns the database name.
func (m *Manager) DBName() string { return m.cfg.Database }

// buildArgs constructs the surreal start argument list.
// Compatible with SurrealDB 2.x and 3.x.
func (m *Manager) buildArgs() []string {
	var datastore string
	if UseInMemoryDB {
		datastore = "memory"
	} else {
		dbPath := filepath.Join(m.cfg.DataDir, "surreal.db")
		// SurrealDB 3.x requires an explicit engine prefix. The path must be a
		// valid URI — spaces (e.g. "Application Support" on macOS) must be
		// percent-encoded or SurrealDB exits immediately with status 1.
		datastore = (&url.URL{Scheme: "surrealkv", Host: "", Path: dbPath}).String()
	}
	return []string{
		"start",
		"--bind", fmt.Sprintf("127.0.0.1:%d", m.cfg.Port),
		"--username", m.cfg.Username,
		"--password", m.cfg.Password,
		"--log", "warn",
		"--no-banner",
		datastore,
	}
}

// resolveBinary returns the path to the surreal executable.
// It checks for a bundled binary adjacent to this process first,
// then falls back to $PATH.
func (m *Manager) resolveBinary() (string, error) {
	if m.cfg.BinaryPath != "" {
		if _, err := os.Stat(m.cfg.BinaryPath); err == nil {
			return m.cfg.BinaryPath, nil
		}
	}

	// Check for a binary bundled next to the app executable.
	if exe, err := os.Executable(); err == nil {
		bundled := filepath.Join(filepath.Dir(exe), "surreal")
		if runtime.GOOS == "windows" {
			bundled += ".exe"
		}
		if _, err := os.Stat(bundled); err == nil {
			return bundled, nil
		}
	}

	// Check the official SurrealDB install location (~/.surrealdb/surreal).
	if home, err := os.UserHomeDir(); err == nil {
		installed := filepath.Join(home, ".surrealdb", "surreal")
		if runtime.GOOS == "windows" {
			installed += ".exe"
		}
		if _, err := os.Stat(installed); err == nil {
			return installed, nil
		}
	}

	// Fall back to $PATH.
	path, err := exec.LookPath("surreal")
	if err != nil {
		return "", fmt.Errorf(
			"surreal binary not found in bundle or PATH\n"+
				"  macOS:   brew install surrealdb/tap/surreal\n"+
				"  Linux:   curl -sSf https://install.surrealdb.com | sh\n"+
				"  Windows: https://surrealdb.com/install",
		)
	}
	return path, nil
}

// waitReady polls the TCP port until SurrealDB accepts connections.
func (m *Manager) waitReady(ctx context.Context, port int, exitCh <-chan error) error {
	deadline := time.Now().Add(healthCheckTimeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-exitCh:
			if err != nil {
				return fmt.Errorf("process exited early: %w", err)
			}
			return fmt.Errorf("process exited early with no error")
		default:
		}

		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(healthCheckInterval)
	}
	return fmt.Errorf("timeout after %s", healthCheckTimeout)
}

// killOrphanedSurrealProcesses kills any `surreal` processes listening on ports
// in the range [preferredPort, preferredPort+9]. This covers both the file-based
// and in-memory cases — in-memory mode leaves no LOCK file, so port occupation
// is the only evidence of a leaked process from a previous unclean exit.
func killOrphanedSurrealProcesses(preferredPort int) {
	if runtime.GOOS == "windows" {
		return
	}
	for port := preferredPort; port < preferredPort+10; port++ {
		// lsof -ti tcp:<port> returns the PID of the process bound to that port.
		out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).Output()
		if err != nil || len(out) == 0 {
			continue
		}
		for _, field := range strings.Fields(string(out)) {
			pid, err := strconv.Atoi(field)
			if err != nil {
				continue
			}
			proc, err := os.FindProcess(pid)
			if err != nil {
				continue
			}
			slog.Info("killing orphaned surreal process", "pid", pid, "port", port)
			_ = proc.Signal(os.Interrupt)
			time.Sleep(300 * time.Millisecond)
			_ = proc.Kill()
		}
	}
}

// killStaleProcess checks whether any OS process is holding the SurrealDB LOCK
// file open and, if so, terminates it. This handles the case where a previous
// app instance was killed with Ctrl-C before the graceful shutdown could run.
//
// On macOS/Linux we use lsof(8) which is available on all supported platforms.
// On Windows we skip the check (the stale lock removal is sufficient because
// Windows releases file locks when the process dies).
func killStaleProcess(lockPath string) {
	if runtime.GOOS == "windows" {
		return
	}
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return // No lock file — nothing to do.
	}

	// lsof -t returns one PID per line for every process that has the file open.
	out, err := exec.Command("lsof", "-t", lockPath).Output()
	if err != nil || len(out) == 0 {
		return // lsof not available or no process holds the file.
	}

	for _, field := range strings.Fields(string(out)) {
		pid, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		slog.Info("killing stale surreal process", "pid", pid)
		_ = proc.Signal(os.Interrupt) // Graceful first.
		time.Sleep(600 * time.Millisecond)
		_ = proc.Kill() // Force-kill if still alive.
	}
}

// findFreePort returns the first free TCP port in [preferred, preferred+9].
func findFreePort(preferred int) (int, error) {
	for port := preferred; port < preferred+10; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port in range %d–%d", preferred, preferred+9)
}

// AppDataDir returns the OS-appropriate directory for persistent app data.
//
//	macOS:   ~/Library/Application Support/AresSim
//	Windows: %APPDATA%\AresSim
//	Linux:   ~/.local/share/aressim
func AppDataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "AresSim"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "AresSim"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "aressim"), nil
	}
}
