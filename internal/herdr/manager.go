package herdr

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ExecFunc func(cmd string) (stdout string, stderr string, exitCode int, err error)

type Manager struct {
	exec          ExecFunc
	sessionName   string
	bootstrap     bool
	enabled       bool
	mu            sync.RWMutex
	initialized   bool
	available     bool
	defaultPaneID string
	pathEnv       string
}

func NewManager(exec ExecFunc, sessionName string, bootstrap, enabled bool) *Manager {
	if sessionName == "" {
		sessionName = "mcp-workspace"
	}
	return &Manager{
		exec:        exec,
		sessionName: sessionName,
		bootstrap:   bootstrap,
		enabled:     enabled,
		pathEnv:     `export PATH="$HOME/.local/bin:$HOME/.cargo/bin:/usr/local/bin:$PATH"; `,
	}
}

func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

func (m *Manager) IsAvailable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.available
}

func (m *Manager) herdrCmd(args ...string) string {
	var escapedArgs []string
	for _, a := range args {
		escapedArgs = append(escapedArgs, fmt.Sprintf("%q", a))
	}
	return fmt.Sprintf("%sherdr --session %q %s", m.pathEnv, m.sessionName, strings.Join(escapedArgs, " "))
}

func (m *Manager) rawCmd(cmd string) string {
	return m.pathEnv + cmd
}

// EnsureReady checks if herdr is installed, bootstraps it if requested,
// and ensures a running headless server and workspace/pane exist.
func (m *Manager) EnsureReady() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return fmt.Errorf("herdr terminal engine is disabled")
	}

	if m.initialized && m.available {
		return nil
	}

	// 1. Detect if herdr is installed on remote host
	checkCmd := m.rawCmd("command -v herdr || test -x \"$HOME/.local/bin/herdr\" || test -x \"$HOME/.cargo/bin/herdr\"")
	_, _, code, err := m.exec(checkCmd)
	if err != nil || code != 0 {
		if !m.bootstrap {
			m.initialized = true
			m.available = false
			return fmt.Errorf("herdr is not installed on remote host and auto-bootstrap is disabled")
		}

		// 2. Auto-bootstrap Herdr via official installer
		installCmd := m.rawCmd("mkdir -p \"$HOME/.local/bin\" && (curl -fsSL https://herdr.dev/install.sh | sh)")
		_, installStderr, installCode, installErr := m.exec(installCmd)
		if installErr != nil || installCode != 0 {
			m.initialized = true
			m.available = false
			return fmt.Errorf("failed to auto-bootstrap herdr on remote host: %v (stderr: %s)", installErr, installStderr)
		}

		// Re-verify installation
		_, _, verifyCode, verifyErr := m.exec(checkCmd)
		if verifyErr != nil || verifyCode != 0 {
			m.initialized = true
			m.available = false
			return fmt.Errorf("herdr installation completed but binary was not found in PATH or ~/.local/bin")
		}
	}

	m.available = true

	// 3. Ensure headless server is running for this persistent session
	snapCmd := m.herdrCmd("api", "snapshot")
	stdout, _, snapCode, _ := m.exec(snapCmd)

	if snapCode != 0 || !strings.Contains(stdout, "session_snapshot") {
		// Server is not running; start headless server in background
		startServerCmd := m.rawCmd(fmt.Sprintf("nohup herdr --session %q server >/dev/null 2>&1 &", m.sessionName))
		_, _, _, _ = m.exec(startServerCmd)

		// Poll for up to 3 seconds for the socket to become ready
		serverReady := false
		for i := 0; i < 10; i++ {
			time.Sleep(300 * time.Millisecond)
			stdout, _, snapCode, _ = m.exec(snapCmd)
			if snapCode == 0 && strings.Contains(stdout, "session_snapshot") {
				serverReady = true
				break
			}
		}

		if !serverReady {
			m.initialized = true
			return fmt.Errorf("herdr server failed to start for session %q", m.sessionName)
		}
	}

	// 4. Ensure at least one workspace and pane exists
	var snap SnapshotResponse
	if err := json.Unmarshal([]byte(stdout), &snap); err == nil {
		if len(snap.Result.Snapshot.Workspaces) == 0 {
			wsCreateCmd := m.herdrCmd("workspace", "create", "--label", "mcp-agent")
			wsOut, _, _, _ := m.exec(wsCreateCmd)
			var wsResp WorkspaceCreateResponse
			if err := json.Unmarshal([]byte(wsOut), &wsResp); err == nil && wsResp.Result.RootPane.PaneID != "" {
				m.defaultPaneID = wsResp.Result.RootPane.PaneID
			}
		} else {
			if snap.Result.Snapshot.FocusedPaneID != "" {
				m.defaultPaneID = snap.Result.Snapshot.FocusedPaneID
			} else if len(snap.Result.Snapshot.Panes) > 0 {
				m.defaultPaneID = snap.Result.Snapshot.Panes[0].PaneID
			}
		}
	}

	m.initialized = true
	return nil
}

// GetSnapshot retrieves the full runtime state (workspaces, tabs, panes, agents)
func (m *Manager) GetSnapshot() (*SnapshotData, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}

	cmd := m.herdrCmd("api", "snapshot")
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to get herdr snapshot: %v (stderr: %s)", err, stderr)
	}

	var snap SnapshotResponse
	if err := json.Unmarshal([]byte(stdout), &snap); err != nil {
		return nil, fmt.Errorf("failed to parse herdr snapshot JSON: %w (raw: %s)", err, stdout)
	}
	if snap.Error != nil {
		return nil, fmt.Errorf("herdr error (%s): %s", snap.Error.Code, snap.Error.Message)
	}

	return &snap.Result.Snapshot, nil
}

// ListPanes returns all active panes across the session
func (m *Manager) ListPanes() ([]Pane, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}

	cmd := m.herdrCmd("pane", "list")
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to list panes: %v (stderr: %s)", err, stderr)
	}

	var resp PaneListResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse pane list JSON: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("herdr error (%s): %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result.Panes, nil
}

// ListWorkspaces returns all active workspaces in the session
func (m *Manager) ListWorkspaces() ([]Workspace, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}

	cmd := m.herdrCmd("workspace", "list")
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to list workspaces: %v (stderr: %s)", err, stderr)
	}

	var resp WorkspaceListResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse workspace list JSON: %w (raw: %s)", err, stdout)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("herdr error (%s): %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result.Workspaces, nil
}

// CreateWorkspace creates a new full-screen isolated workspace/terminal
func (m *Manager) CreateWorkspace(label, cwd string, noFocus bool) (*WorkspaceCreateResponse, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}

	args := []string{"workspace", "create"}
	if label != "" {
		args = append(args, "--label", label)
	}
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	if noFocus {
		args = append(args, "--no-focus")
	}

	cmd := m.herdrCmd(args...)
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to create workspace: %v (stderr: %s)", err, stderr)
	}

	var resp WorkspaceCreateResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse workspace create JSON: %w (raw: %s)", err, stdout)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("herdr error (%s): %s", resp.Error.Code, resp.Error.Message)
	}

	if resp.Result.RootPane.PaneID != "" {
		m.SetDefaultPaneID(resp.Result.RootPane.PaneID)
	}

	return &resp, nil
}

// CloseWorkspace cleanly terminates a workspace and all its panes
func (m *Manager) CloseWorkspace(workspaceID string) error {
	if err := m.EnsureReady(); err != nil {
		return err
	}
	if workspaceID == "" {
		return fmt.Errorf("workspaceID is required")
	}

	cmd := m.herdrCmd("workspace", "close", workspaceID)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return fmt.Errorf("failed to close workspace %s: %v (stderr: %s)", workspaceID, err, stderr)
	}

	m.mu.Lock()
	if strings.HasPrefix(m.defaultPaneID, workspaceID+":") {
		m.defaultPaneID = ""
	}
	m.mu.Unlock()

	if panes, err := m.ListPanes(); err == nil && len(panes) > 0 {
		m.SetDefaultPaneID(panes[0].PaneID)
	}

	return nil
}

// FocusWorkspace focuses a workspace
func (m *Manager) FocusWorkspace(workspaceID string) error {
	if err := m.EnsureReady(); err != nil {
		return err
	}
	if workspaceID == "" {
		return fmt.Errorf("workspaceID is required")
	}

	cmd := m.herdrCmd("workspace", "focus", workspaceID)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return fmt.Errorf("failed to focus workspace %s: %v (stderr: %s)", workspaceID, err, stderr)
	}

	return nil
}

// SplitPane splits an existing pane horizontally (right) or vertically (down)
func (m *Manager) SplitPane(direction, cwd, targetPaneID string, noFocus bool) (*Pane, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}

	if direction == "" {
		direction = "right"
	}

	args := []string{"pane", "split", "--direction", direction}
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	if noFocus {
		args = append(args, "--no-focus")
	}
	if targetPaneID != "" {
		args = append(args, "--pane", targetPaneID)
	} else if m.defaultPaneID != "" {
		args = append(args, "--pane", m.defaultPaneID)
	}

	cmd := m.herdrCmd(args...)
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to split pane: %v (stderr: %s)", err, stderr)
	}

	var resp PaneSplitResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse pane split JSON: %w (raw: %s)", err, stdout)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("herdr error (%s): %s", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result.Pane, nil
}

// RunInPane executes a command inside a Herdr pane with real PTY/TUI support
func (m *Manager) RunInPane(paneID, command, cwd, waitMatch, waitRegex string, timeoutMs int) (*TerminalRunResult, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}

	start := time.Now()
	targetPane := paneID
	if targetPane == "" {
		m.mu.RLock()
		targetPane = m.defaultPaneID
		m.mu.RUnlock()
	}

	if targetPane == "" {
		// Fallback: lookup from pane list
		panes, err := m.ListPanes()
		if err != nil || len(panes) == 0 {
			return nil, fmt.Errorf("no active terminal pane available")
		}
		targetPane = panes[0].PaneID
	}

	// Change directory if requested
	if cwd != "" {
		cdCmd := m.herdrCmd("pane", "run", targetPane, fmt.Sprintf("cd %q", cwd))
		_, _, _, _ = m.exec(cdCmd)
	}

	// Run command inside pane
	runCmd := m.herdrCmd("pane", "run", targetPane, command)
	_, stderr, code, err := m.exec(runCmd)
	if (err != nil || code != 0) && paneID == "" && strings.Contains(stderr, "pane_not_found") {
		if panes, listErr := m.ListPanes(); listErr == nil && len(panes) > 0 {
			targetPane = panes[0].PaneID
			m.SetDefaultPaneID(targetPane)
			runCmd = m.herdrCmd("pane", "run", targetPane, command)
			_, stderr, code, err = m.exec(runCmd)
		}
	}
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to run command in pane %s: %v (stderr: %s)", targetPane, err, stderr)
	}

	// Wait for expected output if requested
	if waitMatch != "" || waitRegex != "" {
		waitArgs := []string{"pane", "wait-output", targetPane}
		if waitMatch != "" {
			waitArgs = append(waitArgs, "--match", waitMatch)
		} else {
			waitArgs = append(waitArgs, "--regex", waitRegex)
		}
		if timeoutMs > 0 {
			waitArgs = append(waitArgs, "--timeout", fmt.Sprintf("%d", timeoutMs))
		}

		waitCmd := m.herdrCmd(waitArgs...)
		_, waitErrOut, waitCode, waitErr := m.exec(waitCmd)
		if waitErr != nil || waitCode != 0 {
			return nil, fmt.Errorf("wait-output failed on pane %s: %v (stderr: %s)", targetPane, waitErr, waitErrOut)
		}
	} else if timeoutMs > 0 {
		// Brief pause for quick commands to produce initial output
		sleepDuration := time.Duration(timeoutMs) * time.Millisecond
		if sleepDuration > 500*time.Millisecond {
			sleepDuration = 500 * time.Millisecond
		}
		time.Sleep(sleepDuration)
	}

	// Read pane buffer output (default capped to 50 lines to conserve tokens)
	readOut, _ := m.ReadPane(targetPane, "recent-unwrapped", 50, "text")

	return &TerminalRunResult{
		Stdout:     readOut,
		ExitCode:   0,
		PaneID:     targetPane,
		Cwd:        cwd,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

type SessionStatus struct {
	SessionName   string `json:"session_name"`
	DefaultPaneID string `json:"default_pane_id"`
	ServerRunning bool   `json:"server_running"`
}

// SessionStatus returns the current live Herdr session status
func (m *Manager) SessionStatus() (*SessionStatus, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &SessionStatus{
		SessionName:   m.sessionName,
		DefaultPaneID: m.defaultPaneID,
		ServerRunning: m.available,
	}, nil
}

// ReadPane reads output from a pane's screen buffer or scrollback with smart token truncation and optional ANSI stripping
func (m *Manager) ReadPane(paneID, source string, lines int, format string) (string, error) {
	if err := m.EnsureReady(); err != nil {
		return "", err
	}

	if source == "" {
		source = "recent-unwrapped"
	}
	if format == "" {
		format = "text"
	}

	targetPane := paneID
	if targetPane == "" {
		m.mu.RLock()
		targetPane = m.defaultPaneID
		m.mu.RUnlock()
	}

	fetchLines := lines
	if fetchLines <= 0 {
		fetchLines = 50
	}

	args := []string{"pane", "read", targetPane, "--source", source, "--format", format}
	if fetchLines > 100 {
		args = append(args, "--lines", fmt.Sprintf("%d", fetchLines))
	}
	cmd := m.herdrCmd(args...)

	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return "", fmt.Errorf("failed to read pane %s: %v (stderr: %s)", targetPane, err, stderr)
	}

	if format != "ansi" {
		stdout = StripAnsi(stdout)
	}

	if fetchLines > 0 {
		stdout = SmartTruncate(stdout, fetchLines)
	}

	return stdout, nil
}

// SendKeys sends logical keys (e.g. "ctrl+c", "esc", "enter", "up", "down") to a pane
func (m *Manager) SendKeys(paneID string, keys ...string) error {
	if err := m.EnsureReady(); err != nil {
		return err
	}

	targetPane := paneID
	if targetPane == "" {
		m.mu.RLock()
		targetPane = m.defaultPaneID
		m.mu.RUnlock()
	}

	args := append([]string{"pane", "send-keys", targetPane}, keys...)
	cmd := m.herdrCmd(args...)

	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return fmt.Errorf("failed to send keys to pane %s: %v (stderr: %s)", targetPane, err, stderr)
	}
	return nil
}

// SendText sends literal text to a pane
func (m *Manager) SendText(paneID string, text string) error {
	if err := m.EnsureReady(); err != nil {
		return err
	}

	targetPane := paneID
	if targetPane == "" {
		m.mu.RLock()
		targetPane = m.defaultPaneID
		m.mu.RUnlock()
	}

	cmd := m.herdrCmd("pane", "send-text", targetPane, text)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return fmt.Errorf("failed to send text to pane %s: %v (stderr: %s)", targetPane, err, stderr)
	}
	return nil
}

// WaitOutput waits until output in the pane matches a string or regex
func (m *Manager) WaitOutput(paneID, match, regex string, timeoutMs int) error {
	if err := m.EnsureReady(); err != nil {
		return err
	}

	targetPane := paneID
	if targetPane == "" {
		m.mu.RLock()
		targetPane = m.defaultPaneID
		m.mu.RUnlock()
	}

	args := []string{"pane", "wait-output", targetPane}
	if match != "" {
		args = append(args, "--match", match)
	} else if regex != "" {
		args = append(args, "--regex", regex)
	} else {
		return fmt.Errorf("either match or regex must be provided")
	}

	if timeoutMs > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", timeoutMs))
	}

	cmd := m.herdrCmd(args...)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return fmt.Errorf("wait-output failed on pane %s: %v (stderr: %s)", targetPane, err, stderr)
	}
	return nil
}

// ClosePane closes a pane and terminates its process
func (m *Manager) ClosePane(paneID string) error {
	if err := m.EnsureReady(); err != nil {
		return err
	}

	targetPane := paneID
	if targetPane == "" {
		m.mu.RLock()
		targetPane = m.defaultPaneID
		m.mu.RUnlock()
	}

	cmd := m.herdrCmd("pane", "close", targetPane)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return fmt.Errorf("failed to close pane %s: %v (stderr: %s)", targetPane, err, stderr)
	}
	return nil
}

// DefaultPaneID returns the currently selected default pane
func (m *Manager) DefaultPaneID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultPaneID
}

// SetDefaultPaneID updates the default target pane
func (m *Manager) SetDefaultPaneID(paneID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultPaneID = paneID
}

// CleanupSession cleanly stops and deletes the Herdr session on remote host
func (m *Manager) CleanupSession() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stopCmd := fmt.Sprintf("%sherdr session stop %q 2>/dev/null; herdr session delete %q 2>/dev/null", m.pathEnv, m.sessionName, m.sessionName)
	_, _, _, _ = m.exec(stopCmd)
	m.initialized = false
	m.defaultPaneID = ""
	return nil
}
