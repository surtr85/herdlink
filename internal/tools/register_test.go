package tools

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/surtr85/mcp-ssh-workspace/internal/config"
	"github.com/surtr85/mcp-ssh-workspace/internal/sshclient"
)

func TestParameterResolvers(t *testing.T) {
	// Test getParamString with camelCase vs PascalCase
	reqCamel := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"commandLine": "echo hello",
				"cwd":         "/tmp",
				"isDaemon":    true,
				"waitMs":      1000,
			},
		},
	}
	if got := getParamString(reqCamel, "commandLine", "CommandLine", "command"); got != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", got)
	}
	if got := getParamString(reqCamel, "cwd", "Cwd"); got != "/tmp" {
		t.Errorf("expected '/tmp', got %q", got)
	}
	if got := getParamBool(reqCamel, false, "isDaemon", "IsDaemon"); got != true {
		t.Errorf("expected true, got %v", got)
	}

	reqPascal := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"CommandLine": "echo world",
				"Cwd":         "/var",
				"IsDaemon":    false,
			},
		},
	}
	if got := getParamString(reqPascal, "commandLine", "CommandLine", "command"); got != "echo world" {
		t.Errorf("expected 'echo world', got %q", got)
	}
	if got := getParamString(reqPascal, "cwd", "Cwd"); got != "/var" {
		t.Errorf("expected '/var', got %q", got)
	}
	if got := getParamBool(reqPascal, true, "isDaemon", "IsDaemon"); got != false {
		t.Errorf("expected false, got %v", got)
	}
}

func TestRegisterToolsWithTerminal(t *testing.T) {
	cfg := &config.Config{
		EnableHerdr:        true,
		AutoBootstrapHerdr: true,
		HerdrSessionName:   "test-session",
	}
	client := sshclient.NewClient(cfg)
	defer client.Close()

	s := server.NewMCPServer("test-server", "1.0.0")
	RegisterTools(s, client)

	// Verify server registered all tools without panic
	tools := s.ListTools()
	foundTools := make(map[string]bool)
	for _, tool := range tools {
		foundTools[tool.Tool.Name] = true
	}

	requiredTools := []string{
		"remote_connect",
		"remote_run_command",
		"remote_terminal_run",
		"remote_terminal_split",
		"remote_terminal_read",
		"remote_terminal_workspace_create",
		"remote_terminal_workspace_list",
		"remote_agent_start",
		"remote_agent_prompt",
		"remote_agent_read",
		"remote_agent_wait",
		"remote_agent_list",
		"local_herdr_status",
		"local_herdr_workspace_create",
		"local_herdr_attach",
	}

	for _, name := range requiredTools {
		if !foundTools[name] {
			t.Errorf("Expected tool %s to be registered", name)
		}
	}
}
