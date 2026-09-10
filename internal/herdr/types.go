package herdr

type SnapshotResponse struct {
	ID     string         `json:"id"`
	Result SnapshotResult `json:"result"`
	Error  *HerdrError    `json:"error,omitempty"`
}

type SnapshotResult struct {
	Type     string       `json:"type"`
	Snapshot SnapshotData `json:"snapshot"`
}

type SnapshotData struct {
	FocusedPaneID      string      `json:"focused_pane_id"`
	FocusedTabID       string      `json:"focused_tab_id"`
	FocusedWorkspaceID string      `json:"focused_workspace_id"`
	Workspaces         []Workspace `json:"workspaces"`
	Tabs               []Tab       `json:"tabs"`
	Panes              []Pane      `json:"panes"`
	Agents             []Agent     `json:"agents"`
	Protocol           int         `json:"protocol"`
	Version            string      `json:"version"`
}

type Workspace struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
	Number      int    `json:"number"`
	Focused     bool   `json:"focused"`
	ActiveTabID string `json:"active_tab_id"`
	PaneCount   int    `json:"pane_count"`
	TabCount    int    `json:"tab_count"`
	AgentStatus string `json:"agent_status,omitempty"`
}

type Tab struct {
	TabID       string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
	Number      int    `json:"number"`
	Focused     bool   `json:"focused"`
	PaneCount   int    `json:"pane_count"`
	AgentStatus string `json:"agent_status,omitempty"`
}

type Pane struct {
	PaneID                 string `json:"pane_id"`
	WorkspaceID            string `json:"workspace_id"`
	TabID                  string `json:"tab_id"`
	TerminalID             string `json:"terminal_id,omitempty"`
	TerminalTitle          string `json:"terminal_title,omitempty"`
	TerminalTitleStripped  string `json:"terminal_title_stripped,omitempty"`
	Cwd                    string `json:"cwd"`
	Focused                bool   `json:"focused"`
	AgentStatus            string `json:"agent_status,omitempty"`
	Revision               int    `json:"revision,omitempty"`
}

type Agent struct {
	AgentID     string `json:"agent_id,omitempty"`
	Name        string `json:"name"`
	PaneID      string `json:"pane_id"`
	Kind        string `json:"kind"`
	Status      string `json:"status"` // idle, done, blocked, working, unknown
	Task        string `json:"task,omitempty"`
	TerminalID  string `json:"terminal_id,omitempty"`
}

type HerdrError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PaneListResponse struct {
	ID     string `json:"id"`
	Result struct {
		Type  string `json:"type"`
		Panes []Pane `json:"panes"`
	} `json:"result"`
	Error *HerdrError `json:"error,omitempty"`
}

type PaneSplitResponse struct {
	ID     string `json:"id"`
	Result struct {
		Type           string `json:"type"`
		Pane           Pane   `json:"pane"`
		PreviousPaneID string `json:"previous_pane_id,omitempty"`
	} `json:"result"`
	Error *HerdrError `json:"error,omitempty"`
}

type WorkspaceListResponse struct {
	ID     string `json:"id"`
	Result struct {
		Type       string      `json:"type"`
		Workspaces []Workspace `json:"workspaces"`
	} `json:"result"`
	Error *HerdrError `json:"error,omitempty"`
}

type WorkspaceCreateResponse struct {
	ID     string `json:"id"`
	Result struct {
		Type      string    `json:"type"`
		Workspace Workspace `json:"workspace"`
		Tab       Tab       `json:"tab"`
		RootPane  Pane      `json:"root_pane"`
	} `json:"result"`
	Error *HerdrError `json:"error,omitempty"`
}

type TerminalRunResult struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exit_code"`
	PaneID     string `json:"pane_id"`
	Cwd        string `json:"cwd"`
	DurationMs int64  `json:"duration_ms"`
}
