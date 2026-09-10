package herdr

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSnapshotParsing(t *testing.T) {
	rawJSON := `{
		"id": "cli:api:snapshot",
		"result": {
			"type": "session_snapshot",
			"snapshot": {
				"protocol": 19,
				"version": "0.8.0",
				"focused_pane_id": "w1:p1",
				"focused_tab_id": "w1:t1",
				"focused_workspace_id": "w1",
				"workspaces": [
					{
						"workspace_id": "w1",
						"label": "mcp-agent",
						"number": 1,
						"focused": true,
						"active_tab_id": "w1:t1",
						"pane_count": 2,
						"tab_count": 1
					}
				],
				"tabs": [
					{
						"tab_id": "w1:t1",
						"workspace_id": "w1",
						"label": "1",
						"number": 1,
						"focused": true,
						"pane_count": 2
					}
				],
				"panes": [
					{
						"pane_id": "w1:p1",
						"workspace_id": "w1",
						"tab_id": "w1:t1",
						"cwd": "/home/user",
						"focused": true,
						"terminal_title": "bash"
					},
					{
						"pane_id": "w1:p2",
						"workspace_id": "w1",
						"tab_id": "w1:t1",
						"cwd": "/home/user/app",
						"focused": false,
						"terminal_title": "htop"
					}
				],
				"agents": []
			}
		}
	}`

	var resp SnapshotResponse
	if err := json.Unmarshal([]byte(rawJSON), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Result.Snapshot.FocusedPaneID != "w1:p1" {
		t.Errorf("Expected focused_pane_id w1:p1, got %s", resp.Result.Snapshot.FocusedPaneID)
	}
	if len(resp.Result.Snapshot.Panes) != 2 {
		t.Errorf("Expected 2 panes, got %d", len(resp.Result.Snapshot.Panes))
	}
}

func TestPaneSplitParsing(t *testing.T) {
	rawJSON := `{
		"id": "cli:pane:split",
		"result": {
			"type": "pane_split",
			"pane": {
				"pane_id": "w1:p2",
				"workspace_id": "w1",
				"tab_id": "w1:t1",
				"cwd": "/home/user",
				"focused": false
			}
		}
	}`

	var resp PaneSplitResponse
	if err := json.Unmarshal([]byte(rawJSON), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.Result.Pane.PaneID != "w1:p2" {
		t.Errorf("Expected pane ID w1:p2, got %s", resp.Result.Pane.PaneID)
	}
}

func TestManagerHerdrCmdFormatting(t *testing.T) {
	mgr := NewManager(func(cmd string) (string, string, int, error) {
		return "", "", 0, nil
	}, "test-session", true, true)

	cmd := mgr.herdrCmd("pane", "run", "w1:p1", "ls -la")
	if !strings.Contains(cmd, `herdr --session "test-session"`) {
		t.Errorf("Expected session flag in command, got %s", cmd)
	}
	if !strings.Contains(cmd, `"w1:p1"`) {
		t.Errorf("Expected pane ID in command, got %s", cmd)
	}
}
