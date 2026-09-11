package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/surtr85/mcp-ssh-workspace/internal/sshclient"
)

func registerTerminalTools(s *server.MCPServer, client *sshclient.Client) {
	// 1. remote_terminal_run
	termRunTool := mcp.NewTool("remote_terminal_run",
		mcp.WithDescription("Execute a command inside an intelligent persistent background terminal pane (powered by herdr) with real PTY and full TUI support. Automatically preserves session state across disconnects."),
		mcp.WithString("commandLine", mcp.Description("The exact command line string to execute in the terminal pane (Required, alias: command, CommandLine).")),
		mcp.WithString("CommandLine", mcp.Description("Alias for commandLine.")),
		mcp.WithString("command", mcp.Description("Alias for commandLine.")),
		mcp.WithString("workspaceId", mcp.Description("Target workspace/space ID (e.g. 'w1', 'w2'). Runs in the workspace's root pane (alias: WorkspaceId, space).")),
		mcp.WithString("WorkspaceId", mcp.Description("Alias for workspaceId.")),
		mcp.WithString("space", mcp.Description("Alias for workspaceId.")),
		mcp.WithString("paneId", mcp.Description("Target pane ID (e.g. 'w1:p1'). If omitted, runs in the active/default pane (alias: PaneId, pane).")),
		mcp.WithString("PaneId", mcp.Description("Alias for paneId.")),
		mcp.WithString("pane", mcp.Description("Alias for paneId.")),
		mcp.WithString("cwd", mcp.Description("Optional working directory to change to inside the pane before running (alias: Cwd).")),
		mcp.WithString("Cwd", mcp.Description("Alias for cwd.")),
		mcp.WithString("waitMatch", mcp.Description("Literal text to wait for in terminal output before returning (alias: match, WaitMatch).")),
		mcp.WithString("match", mcp.Description("Alias for waitMatch.")),
		mcp.WithString("waitRegex", mcp.Description("Regex pattern to wait for in terminal output before returning (alias: regex, WaitRegex).")),
		mcp.WithString("regex", mcp.Description("Alias for waitRegex.")),
		mcp.WithInteger("timeoutMs", mcp.Description("Maximum milliseconds to wait for output pattern or initial response (default 5000ms, alias: timeout, TimeoutMs).")),
		mcp.WithInteger("timeout", mcp.Description("Alias for timeoutMs.")),
	)

	s.AddTool(termRunTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cmd := getParamString(request, "commandLine", "CommandLine", "command", "command_line", "cmd")
		if cmd == "" {
			return mcp.NewToolResultError("commandLine is required"), nil
		}
		workspaceID := getParamString(request, "workspaceId", "WorkspaceId", "space", "Space")
		paneID := getParamString(request, "paneId", "PaneId", "pane_id", "pane", "Pane")
		if paneID == "" && workspaceID != "" {
			paneID = workspaceID + ":p1"
		}
		cwd := getParamString(request, "cwd", "Cwd")
		match := getParamString(request, "waitMatch", "WaitMatch", "match", "Match")
		regex := getParamString(request, "waitRegex", "WaitRegex", "regex", "Regex")
		timeoutMs := getParamInt(request, 5000, "timeoutMs", "TimeoutMs", "timeout", "Timeout")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		res, err := hm.RunInPane(paneID, cmd, cwd, match, regex, timeoutMs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Terminal execution error: %v", err)), nil
		}

		return mcp.NewToolResultText(res.Stdout), nil
	})

	// 2. remote_terminal_split
	splitTool := mcp.NewTool("remote_terminal_split",
		mcp.WithDescription("Split an existing terminal pane horizontally ('right') or vertically ('down') to create a concurrent multitasking terminal workspace for parallel commands."),
		mcp.WithString("direction", mcp.Description("Split direction: 'right' (horizontal split) or 'down' (vertical split) (default 'right', alias: Direction).")),
		mcp.WithString("Direction", mcp.Description("Alias for direction.")),
		mcp.WithString("cwd", mcp.Description("Working directory for the newly spawned pane (alias: Cwd).")),
		mcp.WithString("Cwd", mcp.Description("Alias for cwd.")),
		mcp.WithString("targetPaneId", mcp.Description("Existing pane to split from (alias: paneId, target_pane_id, TargetPaneId).")),
		mcp.WithString("paneId", mcp.Description("Alias for targetPaneId.")),
		mcp.WithBoolean("noFocus", mcp.Description("If true, do not switch active focus to the new pane (default false, alias: NoFocus).")),
		mcp.WithBoolean("NoFocus", mcp.Description("Alias for noFocus.")),
	)

	s.AddTool(splitTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		direction := getParamString(request, "direction", "Direction")
		if direction == "" {
			direction = "right"
		}
		cwd := getParamString(request, "cwd", "Cwd")
		targetPaneID := getParamString(request, "targetPaneId", "TargetPaneId", "target_pane_id", "paneId", "PaneId", "pane")
		noFocus := getParamBool(request, false, "noFocus", "NoFocus", "no_focus")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		pane, err := hm.SplitPane(direction, cwd, targetPaneID, noFocus)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Terminal split error: %v", err)), nil
		}

		out, _ := json.MarshalIndent(pane, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 3. remote_terminal_read
	readTool := mcp.NewTool("remote_terminal_read",
		mcp.WithDescription("Read terminal screen buffer or scrollback from a persistent pane with automatic token capping and ANSI cleaning. Supports visible viewport, recent unwrapped logs, or detection snapshot."),
		mcp.WithString("paneId", mcp.Description("Target pane ID (e.g. 'w1:p1'). If omitted, reads from active/default pane (alias: PaneId, pane).")),
		mcp.WithString("PaneId", mcp.Description("Alias for paneId.")),
		mcp.WithString("pane", mcp.Description("Alias for paneId.")),
		mcp.WithString("source", mcp.Description("Terminal snapshot source: 'recent-unwrapped' (clean unwrapped logs), 'visible' (rendered viewport), 'recent' (recent lines), 'detection' (bottom buffer) (default 'recent-unwrapped', alias: Source).")),
		mcp.WithString("Source", mcp.Description("Alias for source.")),
		mcp.WithInteger("lines", mcp.Description("Number of lines to read from scrollback (default 40, alias: Lines, maxLines, MaxLines).")),
		mcp.WithInteger("Lines", mcp.Description("Alias for lines.")),
		mcp.WithInteger("maxLines", mcp.Description("Alias for lines.")),
		mcp.WithInteger("MaxLines", mcp.Description("Alias for lines.")),
		mcp.WithString("format", mcp.Description("Output format: 'text' (plain text with ANSI stripped, default) or 'ansi' (preserved terminal colors) (alias: Format).")),
		mcp.WithString("Format", mcp.Description("Alias for format.")),
	)

	s.AddTool(readTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		paneID := getParamString(request, "paneId", "PaneId", "pane", "Pane")
		source := getParamString(request, "source", "Source")
		lines := getParamInt(request, 40, "lines", "Lines", "maxLines", "MaxLines")
		format := getParamString(request, "format", "Format")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		content, err := hm.ReadPane(paneID, source, lines, format)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Terminal read error: %v", err)), nil
		}

		return mcp.NewToolResultText(content), nil
	})

	// 4. remote_terminal_send_keys
	sendKeysTool := mcp.NewTool("remote_terminal_send_keys",
		mcp.WithDescription("Send logical keystrokes to an interactive CLI or TUI app inside a terminal pane (e.g. 'ctrl+c', 'esc', 'enter', 'up', 'down', 'q')."),
		mcp.WithString("keys", mcp.Description("Space-separated or comma-separated keystrokes to dispatch, e.g. 'ctrl+c', 'esc', 'enter', 'up', 'q' (Required, alias: Keys, key).")),
		mcp.WithString("Keys", mcp.Description("Alias for keys.")),
		mcp.WithString("key", mcp.Description("Alias for keys.")),
		mcp.WithString("paneId", mcp.Description("Target pane ID (alias: PaneId, pane).")),
		mcp.WithString("PaneId", mcp.Description("Alias for paneId.")),
		mcp.WithString("pane", mcp.Description("Alias for paneId.")),
	)

	s.AddTool(sendKeysTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		keysStr := getParamString(request, "keys", "Keys", "key", "Key")
		if keysStr == "" {
			return mcp.NewToolResultError("keys parameter is required"), nil
		}
		paneID := getParamString(request, "paneId", "PaneId", "pane", "Pane")

		// Split on spaces or commas
		rawKeys := strings.FieldsFunc(keysStr, func(r rune) bool {
			return r == ' ' || r == ','
		})

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		if err := hm.SendKeys(paneID, rawKeys...); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send keys: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully sent keys [%s] to pane %s", strings.Join(rawKeys, ", "), paneID)), nil
	})

	// 5. remote_terminal_send_text
	sendTextTool := mcp.NewTool("remote_terminal_send_text",
		mcp.WithDescription("Send literal text or input lines directly into a running terminal pane with bracketed paste mode support."),
		mcp.WithString("text", mcp.Description("Literal text string to send (Required, alias: Text, input, Input).")),
		mcp.WithString("Text", mcp.Description("Alias for text.")),
		mcp.WithString("input", mcp.Description("Alias for text.")),
		mcp.WithString("Input", mcp.Description("Alias for text.")),
		mcp.WithString("paneId", mcp.Description("Target pane ID (alias: PaneId, pane).")),
		mcp.WithString("PaneId", mcp.Description("Alias for paneId.")),
		mcp.WithString("pane", mcp.Description("Alias for paneId.")),
	)

	s.AddTool(sendTextTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text := getParamString(request, "text", "Text", "input", "Input")
		if text == "" {
			return mcp.NewToolResultError("text parameter is required"), nil
		}
		paneID := getParamString(request, "paneId", "PaneId", "pane", "Pane")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		if err := hm.SendText(paneID, text); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send text: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully sent text to pane %s", paneID)), nil
	})

	// 6. remote_terminal_wait
	waitTool := mcp.NewTool("remote_terminal_wait",
		mcp.WithDescription("Wait until terminal output matches a literal string or regular expression, with configurable timeout."),
		mcp.WithString("match", mcp.Description("Literal string to wait for (alias: Match, waitMatch).")),
		mcp.WithString("Match", mcp.Description("Alias for match.")),
		mcp.WithString("regex", mcp.Description("Regex pattern to wait for (alias: Regex, waitRegex).")),
		mcp.WithString("Regex", mcp.Description("Alias for regex.")),
		mcp.WithInteger("timeoutMs", mcp.Description("Timeout in milliseconds (default 30000ms, alias: timeout, TimeoutMs).")),
		mcp.WithInteger("timeout", mcp.Description("Alias for timeoutMs.")),
		mcp.WithString("paneId", mcp.Description("Target pane ID (alias: PaneId, pane).")),
		mcp.WithString("PaneId", mcp.Description("Alias for paneId.")),
		mcp.WithString("pane", mcp.Description("Alias for paneId.")),
	)

	s.AddTool(waitTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		match := getParamString(request, "match", "Match", "waitMatch")
		regex := getParamString(request, "regex", "Regex", "waitRegex")
		if match == "" && regex == "" {
			return mcp.NewToolResultError("either match or regex must be provided"), nil
		}
		timeoutMs := getParamInt(request, 30000, "timeoutMs", "TimeoutMs", "timeout", "Timeout")
		paneID := getParamString(request, "paneId", "PaneId", "pane", "Pane")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		if err := hm.WaitOutput(paneID, match, regex, timeoutMs); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Wait failed: %v", err)), nil
		}

		readOut, _ := hm.ReadPane(paneID, "recent-unwrapped", 50, "text")
		return mcp.NewToolResultText(readOut), nil
	})

	// 7. remote_terminal_list
	listTool := mcp.NewTool("remote_terminal_list",
		mcp.WithDescription("List all active persistent terminal workspaces, tabs, panes, and live agent processes."),
	)

	s.AddTool(listTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		snapshot, err := hm.GetSnapshot()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list terminal state: %v", err)), nil
		}

		out, _ := json.MarshalIndent(snapshot, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 8. remote_terminal_close
	closeTool := mcp.NewTool("remote_terminal_close",
		mcp.WithDescription("Close a terminal pane and cleanly terminate its associated background process."),
		mcp.WithString("paneId", mcp.Description("Target pane ID to close (Required, alias: PaneId, pane).")),
		mcp.WithString("PaneId", mcp.Description("Alias for paneId.")),
		mcp.WithString("pane", mcp.Description("Alias for paneId.")),
	)

	s.AddTool(closeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		paneID := getParamString(request, "paneId", "PaneId", "pane", "Pane")
		if paneID == "" {
			return mcp.NewToolResultError("paneId is required"), nil
		}

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		if err := hm.ClosePane(paneID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to close pane: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Terminal pane %s successfully closed.", paneID)), nil
	})

	// 9. remote_terminal_workspace_create
	wsCreateTool := mcp.NewTool("remote_terminal_workspace_create",
		mcp.WithDescription("Create a new full-screen, isolated terminal workspace (space) for distraction-free human and agent multitasking."),
		mcp.WithString("label", mcp.Description("Descriptive label or name for the workspace (e.g. 'server-logs', 'dev', 'monitor', alias: Label, name).")),
		mcp.WithString("Label", mcp.Description("Alias for label.")),
		mcp.WithString("name", mcp.Description("Alias for label.")),
		mcp.WithString("cwd", mcp.Description("Initial working directory for the workspace (alias: Cwd).")),
		mcp.WithString("Cwd", mcp.Description("Alias for cwd.")),
		mcp.WithBoolean("noFocus", mcp.Description("If true, do not switch active UI focus to the new workspace (default false, alias: NoFocus).")),
		mcp.WithBoolean("NoFocus", mcp.Description("Alias for noFocus.")),
	)

	s.AddTool(wsCreateTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		label := getParamString(request, "label", "Label", "name", "Name")
		cwd := getParamString(request, "cwd", "Cwd")
		noFocus := getParamBool(request, false, "noFocus", "NoFocus")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		ws, err := hm.CreateWorkspace(label, cwd, noFocus)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create workspace: %v", err)), nil
		}

		out, _ := json.MarshalIndent(ws.Result, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 10. remote_terminal_workspace_close
	wsCloseTool := mcp.NewTool("remote_terminal_workspace_close",
		mcp.WithDescription("Close a full workspace (space) and terminate all processes and panes running inside it."),
		mcp.WithString("workspaceId", mcp.Description("Target workspace ID to close (e.g. 'w1', 'w2', Required, alias: WorkspaceId, id).")),
		mcp.WithString("WorkspaceId", mcp.Description("Alias for workspaceId.")),
		mcp.WithString("id", mcp.Description("Alias for workspaceId.")),
	)

	s.AddTool(wsCloseTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wsID := getParamString(request, "workspaceId", "WorkspaceId", "id", "ID")
		if wsID == "" {
			return mcp.NewToolResultError("workspaceId is required"), nil
		}

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		if err := hm.CloseWorkspace(wsID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to close workspace: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Workspace %s successfully closed.", wsID)), nil
	})

	// 11. remote_terminal_workspace_list
	wsListTool := mcp.NewTool("remote_terminal_workspace_list",
		mcp.WithDescription("List all active full-screen terminal spaces (workspaces) with active tab, pane count, and focus status."),
	)

	s.AddTool(wsListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		workspaces, err := hm.ListWorkspaces()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list workspaces: %v", err)), nil
		}

		out, _ := json.MarshalIndent(workspaces, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 12. remote_terminal_workspace_focus
	wsFocusTool := mcp.NewTool("remote_terminal_workspace_focus",
		mcp.WithDescription("Switch user active view/focus to a specific terminal space (workspace)."),
		mcp.WithString("workspaceId", mcp.Description("Target workspace ID to focus (e.g. 'w1', 'w2', Required, alias: WorkspaceId, id).")),
		mcp.WithString("WorkspaceId", mcp.Description("Alias for workspaceId.")),
		mcp.WithString("id", mcp.Description("Alias for workspaceId.")),
	)

	s.AddTool(wsFocusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wsID := getParamString(request, "workspaceId", "WorkspaceId", "id", "ID")
		if wsID == "" {
			return mcp.NewToolResultError("workspaceId is required"), nil
		}

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		if err := hm.FocusWorkspace(wsID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to focus workspace: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully focused workspace %s.", wsID)), nil
	})

	// 13. remote_terminal_session_cleanup
	sessionCleanupTool := mcp.NewTool("remote_terminal_session_cleanup",
		mcp.WithDescription("Cleanly stop and delete the active Herdr session on the remote host to release all system memory, processes, and CPU resources upon task completion."),
	)

	s.AddTool(sessionCleanupTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled in configuration"), nil
		}

		if err := hm.CleanupSession(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to cleanup session: %v", err)), nil
		}

		return mcp.NewToolResultText("Herdr session successfully terminated and cleaned up to release all system resources."), nil
	})
}
