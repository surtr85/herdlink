package herdr

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// IsLocalHerdrAvailable checks if the herdr executable is in PATH
func IsLocalHerdrAvailable() bool {
	_, err := exec.LookPath("herdr")
	return err == nil
}

// LocalHerdrStatus checks status of local Herdr client and server
func LocalHerdrStatus() (string, error) {
	if !IsLocalHerdrAvailable() {
		return "", fmt.Errorf("local herdr CLI is not installed or not in PATH")
	}

	cmd := exec.Command("herdr", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("herdr status error: %v (output: %s)", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// LocalHerdrWorkspaceCreate creates a new workspace in local Herdr and optionally runs an initial command
func LocalHerdrWorkspaceCreate(label, cwd, runCommand string, noFocus bool) (string, string, error) {
	if !IsLocalHerdrAvailable() {
		return "", "", fmt.Errorf("local herdr CLI is not installed")
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

	cmd := exec.Command("herdr", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("failed to create local workspace: %v (output: %s)", err, string(out))
	}

	var resp WorkspaceCreateResponse
	workspaceID := ""
	rootPaneID := ""
	if jsonErr := json.Unmarshal(out, &resp); jsonErr == nil {
		workspaceID = resp.Result.Workspace.WorkspaceID
		rootPaneID = resp.Result.RootPane.PaneID
	}

	if runCommand != "" && rootPaneID != "" {
		runCmd := exec.Command("herdr", "pane", "run", rootPaneID, runCommand)
		_ = runCmd.Run()
	}

	return workspaceID, rootPaneID, nil
}

// LocalHerdrFindWorkspaceByLabel checks if a workspace with the given label already exists
func LocalHerdrFindWorkspaceByLabel(label string) (string, bool) {
	if !IsLocalHerdrAvailable() || label == "" {
		return "", false
	}

	cmd := exec.Command("herdr", "workspace", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}

	var resp WorkspaceListResponse
	if jsonErr := json.Unmarshal(out, &resp); jsonErr == nil {
		for _, ws := range resp.Result.Workspaces {
			if strings.EqualFold(ws.Label, label) {
				return ws.WorkspaceID, true
			}
		}
	}
	return "", false
}
