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

func registerAgentTools(s *server.MCPServer, client *sshclient.Client) {
	// 1. remote_agent_start
	agentStartTool := mcp.NewTool("remote_agent_start",
		mcp.WithDescription("Start a dedicated subagent (e.g. codex, claude) inside an existing or new terminal pane on the remote machine."),
		mcp.WithString("name", mcp.Description("Unique agent name ([a-z][a-z0-9_-]{0,31}, Required, alias: Name).")),
		mcp.WithString("Name", mcp.Description("Alias for name.")),
		mcp.WithString("kind", mcp.Description("Agent kind/type (default 'codex', alias: Kind).")),
		mcp.WithString("Kind", mcp.Description("Alias for kind.")),
		mcp.WithString("paneId", mcp.Description("Target pane ID to host the agent (alias: PaneId, pane).")),
		mcp.WithString("PaneId", mcp.Description("Alias for paneId.")),
		mcp.WithString("pane", mcp.Description("Alias for paneId.")),
		mcp.WithString("args", mcp.Description("Optional extra native agent CLI arguments (alias: Args).")),
		mcp.WithString("Args", mcp.Description("Alias for args.")),
	)

	s.AddTool(agentStartTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := getParamString(request, "name", "Name")
		if name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}
		kind := getParamString(request, "kind", "Kind")
		paneID := getParamString(request, "paneId", "PaneId", "pane", "Pane")
		argsStr := getParamString(request, "args", "Args")
		var args []string
		if argsStr != "" {
			args = strings.Fields(argsStr)
		}

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled"), nil
		}

		res, err := hm.AgentStart(name, kind, paneID, args...)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start agent: %v", err)), nil
		}

		out, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 2. remote_agent_prompt
	agentPromptTool := mcp.NewTool("remote_agent_prompt",
		mcp.WithDescription("Atomically submit a task/prompt to a running remote agent and wait for completion with token-capped response."),
		mcp.WithString("name", mcp.Description("Live agent name (Required, alias: Name).")),
		mcp.WithString("Name", mcp.Description("Alias for name.")),
		mcp.WithString("prompt", mcp.Description("Task description or prompt text (Required, alias: Prompt, task, Task).")),
		mcp.WithString("Prompt", mcp.Description("Alias for prompt.")),
		mcp.WithString("task", mcp.Description("Alias for prompt.")),
		mcp.WithBoolean("wait", mcp.Description("Wait until agent completes task and settles into idle/done/blocked (default true, alias: Wait).")),
		mcp.WithBoolean("Wait", mcp.Description("Alias for wait.")),
		mcp.WithInteger("timeoutMs", mcp.Description("Timeout in milliseconds (default 120000ms / 2min, alias: TimeoutMs, timeout).")),
		mcp.WithInteger("timeout", mcp.Description("Alias for timeoutMs.")),
	)

	s.AddTool(agentPromptTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := getParamString(request, "name", "Name")
		if name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}
		prompt := getParamString(request, "prompt", "Prompt", "task", "Task")
		if prompt == "" {
			return mcp.NewToolResultError("prompt is required"), nil
		}
		wait := getParamBool(request, true, "wait", "Wait")
		timeoutMs := getParamInt(request, 120000, "timeoutMs", "TimeoutMs", "timeout")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled"), nil
		}

		res, err := hm.AgentPrompt(name, prompt, wait, timeoutMs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Agent prompt error: %v", err)), nil
		}

		out, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 3. remote_agent_read
	agentReadTool := mcp.NewTool("remote_agent_read",
		mcp.WithDescription("Read clean unwrapped transcript/output from a live remote agent."),
		mcp.WithString("name", mcp.Description("Target agent name (Required, alias: Name).")),
		mcp.WithString("Name", mcp.Description("Alias for name.")),
		mcp.WithString("source", mcp.Description("Snapshot source: 'recent-unwrapped' (default), 'visible', 'detection' (alias: Source).")),
		mcp.WithString("Source", mcp.Description("Alias for source.")),
		mcp.WithInteger("lines", mcp.Description("Number of output lines to read (default 50, alias: Lines).")),
		mcp.WithInteger("Lines", mcp.Description("Alias for lines.")),
	)

	s.AddTool(agentReadTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := getParamString(request, "name", "Name")
		if name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}
		source := getParamString(request, "source", "Source")
		lines := getParamInt(request, 50, "lines", "Lines")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled"), nil
		}

		content, err := hm.AgentRead(name, source, lines)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read agent: %v", err)), nil
		}

		return mcp.NewToolResultText(content), nil
	})

	// 4. remote_agent_wait
	agentWaitTool := mcp.NewTool("remote_agent_wait",
		mcp.WithDescription("Wait for a remote agent to reach a specific lifecycle state (idle, done, or blocked)."),
		mcp.WithString("name", mcp.Description("Agent name to wait on (Required, alias: Name).")),
		mcp.WithString("Name", mcp.Description("Alias for name.")),
		mcp.WithString("until", mcp.Description("Target state: 'idle', 'done', 'blocked' (alias: Until).")),
		mcp.WithString("Until", mcp.Description("Alias for until.")),
		mcp.WithInteger("timeoutMs", mcp.Description("Timeout in milliseconds (default 60000ms, alias: TimeoutMs, timeout).")),
		mcp.WithInteger("timeout", mcp.Description("Alias for timeoutMs.")),
	)

	s.AddTool(agentWaitTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := getParamString(request, "name", "Name")
		if name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}
		until := getParamString(request, "until", "Until")
		timeoutMs := getParamInt(request, 60000, "timeoutMs", "TimeoutMs", "timeout")

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled"), nil
		}

		res, err := hm.AgentWait(name, until, timeoutMs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Agent wait error: %v", err)), nil
		}

		out, _ := json.MarshalIndent(res, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 5. remote_agent_list
	agentListTool := mcp.NewTool("remote_agent_list",
		mcp.WithDescription("List all active subagents running inside the remote Herdr session with their current states."),
	)

	s.AddTool(agentListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled"), nil
		}

		agents, err := hm.AgentList()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list agents: %v", err)), nil
		}

		out, _ := json.MarshalIndent(agents, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})

	// 6. remote_agent_send_keys
	agentKeysTool := mcp.NewTool("remote_agent_send_keys",
		mcp.WithDescription("Send interactive keys (esc, ctrl+c, enter, etc.) to a remote agent."),
		mcp.WithString("name", mcp.Description("Target agent name (Required, alias: Name).")),
		mcp.WithString("Name", mcp.Description("Alias for name.")),
		mcp.WithString("keys", mcp.Description("Keys to send, e.g. 'esc', 'ctrl+c' (Required, alias: Keys).")),
		mcp.WithString("Keys", mcp.Description("Alias for keys.")),
	)

	s.AddTool(agentKeysTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := getParamString(request, "name", "Name")
		if name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}
		keysStr := getParamString(request, "keys", "Keys")
		if keysStr == "" {
			return mcp.NewToolResultError("keys is required"), nil
		}

		rawKeys := strings.FieldsFunc(keysStr, func(r rune) bool {
			return r == ' ' || r == ','
		})

		hm := client.Herdr()
		if hm == nil || !hm.IsEnabled() {
			return mcp.NewToolResultError("Herdr terminal engine is disabled"), nil
		}

		if err := hm.AgentSendKeys(name, rawKeys...); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send keys to agent: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Sent keys [%s] to agent %s", strings.Join(rawKeys, ", "), name)), nil
	})
}
