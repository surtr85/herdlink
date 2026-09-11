package herdr

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AgentStartResult struct {
	Name    string `json:"name"`
	PaneID  string `json:"pane_id"`
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type AgentPromptResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Output     string `json:"output,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

type AgentWaitResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AgentListResponse struct {
	ID     string `json:"id"`
	Result struct {
		Type   string  `json:"type"`
		Agents []Agent `json:"agents"`
	} `json:"result"`
	Error *HerdrError `json:"error,omitempty"`
}

type AgentGetResponse struct {
	ID     string `json:"id"`
	Result struct {
		Type  string `json:"type"`
		Agent Agent  `json:"agent"`
	} `json:"result"`
	Error *HerdrError `json:"error,omitempty"`
}

// AgentStart starts a supported agent (e.g. codex, claude) in a target pane
func (m *Manager) AgentStart(name, kind, paneID string, extraArgs ...string) (*AgentStartResult, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if kind == "" {
		kind = "codex"
	}
	targetPane := paneID
	if targetPane == "" {
		m.mu.RLock()
		targetPane = m.defaultPaneID
		m.mu.RUnlock()
	}

	args := []string{"agent", "start", name, "--kind", kind, "--pane", targetPane}
	if len(extraArgs) > 0 {
		args = append(args, "--")
		args = append(args, extraArgs...)
	}

	cmd := m.herdrCmd(args...)
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to start agent %s: %v (stderr: %s)", name, err, stderr)
	}

	return &AgentStartResult{
		Name:    name,
		PaneID:  targetPane,
		Kind:    kind,
		Status:  "idle",
		Message: strings.TrimSpace(stdout),
	}, nil
}

// AgentPrompt submits a prompt to an agent and optionally waits for settled response
func (m *Manager) AgentPrompt(name, prompt string, wait bool, timeoutMs int) (*AgentPromptResult, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	start := time.Now()
	args := []string{"agent", "prompt", name, prompt}
	if wait {
		args = append(args, "--wait")
	}
	if timeoutMs > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", timeoutMs))
	} else if wait {
		args = append(args, "--timeout", "120000") // default 2 min
	}

	cmd := m.herdrCmd(args...)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to prompt agent %s: %v (stderr: %s)", name, err, stderr)
	}

	// Read agent response output
	out, _ := m.AgentRead(name, "recent-unwrapped", 60)

	// Get agent current status
	status := "done"
	if info, err := m.AgentGet(name); err == nil && info != nil {
		status = info.Status
	}

	return &AgentPromptResult{
		Name:       name,
		Status:     status,
		Output:     out,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// AgentRead reads the clean transcript/output from a live agent
func (m *Manager) AgentRead(name, source string, lines int) (string, error) {
	if err := m.EnsureReady(); err != nil {
		return "", err
	}
	if name == "" {
		return "", fmt.Errorf("agent name is required")
	}
	if source == "" {
		source = "recent-unwrapped"
	}

	args := []string{"agent", "read", name, "--source", source}
	if lines > 0 {
		args = append(args, "--lines", fmt.Sprintf("%d", lines))
	}

	cmd := m.herdrCmd(args...)
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return "", fmt.Errorf("failed to read agent %s: %v (stderr: %s)", name, err, stderr)
	}

	return stdout, nil
}

// AgentWait waits for an agent to reach a specific lifecycle state (idle, done, blocked)
func (m *Manager) AgentWait(name, until string, timeoutMs int) (*AgentWaitResult, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	args := []string{"agent", "wait", name}
	if until != "" {
		args = append(args, "--until", until)
	}
	if timeoutMs > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", timeoutMs))
	}

	cmd := m.herdrCmd(args...)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed waiting for agent %s: %v (stderr: %s)", name, err, stderr)
	}

	status := "idle"
	if info, err := m.AgentGet(name); err == nil && info != nil {
		status = info.Status
	}

	return &AgentWaitResult{
		Name:   name,
		Status: status,
	}, nil
}

// AgentGet inspects a specific agent's metadata and state
func (m *Manager) AgentGet(name string) (*Agent, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	cmd := m.herdrCmd("agent", "get", name)
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to get agent %s: %v (stderr: %s)", name, err, stderr)
	}

	var resp AgentGetResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse agent json: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("herdr agent error (%s): %s", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result.Agent, nil
}

// AgentList lists all live agents across the Herdr session
func (m *Manager) AgentList() ([]Agent, error) {
	if err := m.EnsureReady(); err != nil {
		return nil, err
	}

	cmd := m.herdrCmd("agent", "list")
	stdout, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return nil, fmt.Errorf("failed to list agents: %v (stderr: %s)", err, stderr)
	}

	var resp AgentListResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse agent list json: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("herdr error (%s): %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result.Agents, nil
}

// AgentSendKeys sends logical keys to an interactive agent UI (e.g. esc, ctrl+c)
func (m *Manager) AgentSendKeys(name string, keys ...string) error {
	if err := m.EnsureReady(); err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("agent name is required")
	}

	args := append([]string{"agent", "send-keys", name}, keys...)
	cmd := m.herdrCmd(args...)
	_, stderr, code, err := m.exec(cmd)
	if err != nil || code != 0 {
		return fmt.Errorf("failed to send keys to agent %s: %v (stderr: %s)", name, err, stderr)
	}
	return nil
}
