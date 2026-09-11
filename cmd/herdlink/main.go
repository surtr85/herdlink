package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/surtr85/mcp-ssh-workspace/internal/config"
	"github.com/surtr85/mcp-ssh-workspace/internal/sshclient"
	"github.com/surtr85/mcp-ssh-workspace/internal/tools"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Herdlink config error: %v\n", err)
		os.Exit(1)
	}

	client := sshclient.NewClient(cfg)
	defer client.Close()

	s := server.NewMCPServer(
		"herdlink",
		"2.0.0",
		server.WithToolCapabilities(true),
		server.WithDescription("Herdlink: Ultra-fast AI agent terminal control plane and SSH multiplexer bridge powered by Herdr"),
	)

	tools.RegisterTools(s, client)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Herdlink server stopped: %v\n", err)
		os.Exit(1)
	}
}
