package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/surtr85/mcp-ssh-workspace/internal/herdr"
	"github.com/surtr85/mcp-ssh-workspace/internal/sshclient"
)

func registerLocalHerdrTools(s *server.MCPServer, client *sshclient.Client) {
	// 1. local_herdr_status
	localStatusTool := mcp.NewTool("local_herdr_status",
		mcp.WithDescription("Check if Herdr terminal manager is installed and running on the local host machine."),
	)

	s.AddTool(localStatusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		status, err := herdr.LocalHerdrStatus()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Local Herdr check failed: %v", err)), nil
		}
		return mcp.NewToolResultText(status), nil
	})

	// 2. local_herdr_workspace_create
	localWsCreateTool := mcp.NewTool("local_herdr_workspace_create",
		mcp.WithDescription("Create a new persistent workspace tab on the local developer machine with an optional command."),
		mcp.WithString("label", mcp.Description("Descriptive label for local workspace (Required, alias: Label, name).")),
		mcp.WithString("Label", mcp.Description("Alias for label.")),
		mcp.WithString("cwd", mcp.Description("Working directory for local workspace (alias: Cwd).")),
		mcp.WithString("Cwd", mcp.Description("Alias for cwd.")),
		mcp.WithString("command", mcp.Description("Optional initial command to run in root pane (e.g. 'ssh host', alias: Command, cmd).")),
		mcp.WithString("Command", mcp.Description("Alias for command.")),
		mcp.WithBoolean("noFocus", mcp.Description("If true, do not switch human focus to new workspace (alias: NoFocus).")),
		mcp.WithBoolean("NoFocus", mcp.Description("Alias for noFocus.")),
	)

	s.AddTool(localWsCreateTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		label := getParamString(request, "label", "Label", "name", "Name")
		if label == "" {
			return mcp.NewToolResultError("label is required"), nil
		}
		cwd := getParamString(request, "cwd", "Cwd")
		command := getParamString(request, "command", "Command", "cmd")
		noFocus := getParamBool(request, false, "noFocus", "NoFocus")

		wsID, paneID, err := herdr.LocalHerdrWorkspaceCreate(label, cwd, command, noFocus)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create local workspace: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Local workspace %s (%s) created with pane %s", label, wsID, paneID)), nil
	})

	// 3. local_herdr_attach
	localAttachTool := mcp.NewTool("local_herdr_attach",
		mcp.WithDescription("Attach local Herdr workspace to a remote server so the human developer can observe or interact live."),
		mcp.WithString("host", mcp.Description("Remote SSH host or alias to attach to (Required, alias: Host).")),
		mcp.WithString("Host", mcp.Description("Alias for host.")),
		mcp.WithString("label", mcp.Description("Workspace label (alias: Label).")),
		mcp.WithString("Label", mcp.Description("Alias for label.")),
	)

	s.AddTool(localAttachTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		host := getParamString(request, "host", "Host")
		if host == "" {
			return mcp.NewToolResultError("host is required"), nil
		}
		label := getParamString(request, "label", "Label")
		if label == "" {
			label = host
		}

		// Check if already open
		if wsID, exists := herdr.LocalHerdrFindWorkspaceByLabel(label); exists {
			return mcp.NewToolResultText(fmt.Sprintf("Local workspace '%s' (%s) is already open and attached.", label, wsID)), nil
		}

		target := host
		if client != nil && client.User() != "" && !strings.Contains(host, "@") {
			target = fmt.Sprintf("%s@%s", client.User(), host)
		}
		herdrRemoteCmd := fmt.Sprintf("env -u HERDR_ENV -u HERDR_SOCKET_PATH -u HERDR_PANE_ID -u HERDR_TAB_ID -u HERDR_WORKSPACE_ID herdr --remote %s", target)
		wsID, paneID, err := herdr.LocalHerdrWorkspaceCreate(label, "", herdrRemoteCmd, false)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to attach local Herdr: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Attached local Herdr workspace '%s' (%s, pane %s) via %s", label, wsID, paneID, herdrRemoteCmd)), nil
	})
}
