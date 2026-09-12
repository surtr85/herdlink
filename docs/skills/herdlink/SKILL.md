---
name: herdlink
description: Master control plane for AI coding agents connecting to remote servers over SSH via the Herdr terminal multiplexer. Use to execute interactive PTY commands, manage multitasking spaces, supervise remote subagents, and perform surgical SFTP file edits with ultra-low token consumption.
---

# Herdlink (`herdlink`) — Agent Control Plane Reference

Herdlink provides AI coding agents with native, local-like operating capabilities across remote SSH machines. It pairs persistent multiplexed SSH/SFTP tunnels with the **Herdr** terminal engine to enable real PTY emulation, spatial multitasking, subagent orchestration, and token-capped execution.

---

## 1. Core Architecture & Mental Model

```
Agent (Antigravity/Claude) <--> Herdlink MCP Engine <--> SSH / Herdr Multiplexer <--> Remote Server
                                       |
                                       +--> Local Herdr Client (Live Human GUI)
```

1. **Remote Herdr Headless Engine**: Runs a persistent daemon (default shared session or named session) on the remote server. Survives network disconnects. Hosts workspaces (`w1`, `w2`), tabs, and panes (`w1:p1`).
2. **Local Herdr Native Remote Bridge**: Syncs directly with the user's local Herdr app via native `herdr --remote <host>` so both the developer and AI agent share the exact same live workspace in real time.
3. **SFTP Subsystem**: Delivers binary-safe, line-sliced file reading and atomic search-and-replace chunk editing directly over SFTP without invoking shell commands.

---

## 2. The 1-Step Teleport Protocol (`remote_connect`)

**Never run multi-step discovery probes.** When asked to connect to a host:

```json
{
  "host": "192.168.1.69",
  "user": "amadeus",
  "syncLocalHerdr": true
}
```

Herdlink executes the following automatically:

- Resolves `~/.ssh/config` host aliases or direct IPs.
- Establishes persistent SSH2 & SFTP connection pool.
- Verifies / auto-bootstraps remote Herdr daemon and initializes root pane.
- Opens an interactive workspace in the user's local Herdr UI (`syncLocalHerdr: true`) via native `herdr --remote <host>` so the user can observe live.
- Probes kernel, OS, hostname, and sudo privileges in a single subshell.

### Returned Status (<50 Tokens):

```text
✓ Connected to amadeus@Home (Debian GNU/Linux 13, Linux 6.12.95)
✓ Herdr Remote: Ready (session: "default", root_pane: w1:p1)
✓ Local Herdr: Synced (workspace: HomeServer [w37], pane: w37:p1)
✓ Sudo: Configured | CWD: /home/amadeus
```

**Do NOT follow up with `remote_session_info`, `uname`, or `whoami`.** All necessary context is already returned.

---

## 3. Token Conservation Protocol (The 7 Cardinal Rules)

Every token spent on terminal noise is wasted budget. Strictly enforce these rules:

1. **Auto ANSI Cleansing**: Leave `format: "text"` (default). Herdlink automatically strips ANSI escape codes and terminal redraw sequences, saving 30–50% tokens. Only use `format: "ansi"` when analyzing terminal colors.
2. **Smart Head+Tail Truncation**: Reading output defaults to `lines: 40` (or `maxLines: 40`). Herdlink preserves the command invocation and exit/error traces while folding the middle with a single indicator:
   `... [85 lines truncated for token efficiency] ...`
3. **Zero-Polling Pattern Waits**: Never write polling loops. Use `waitMatch` or `waitRegex` inside `remote_terminal_run` or `remote_terminal_wait` to block until the expected token appears.
4. **Surgical Chunk Editing**: Never rewrite entire remote files. Use `remote_replace_file_content` with concise `TargetContent` and `ReplacementContent`.
5. **Token-Capped Sliced Reads**: When inspecting remote source code or logs, use `remote_view_file` with explicit `StartLine` and `EndLine` ranges (max 100 lines per call).
6. **Task Isolation over Monolithic Panes**: When launching a background server or watcher (`npm run dev`, `docker compose up`), use `remote_terminal_workspace_create` to prevent stream spam from contaminating the primary terminal buffer.
7. **Resource Hygiene**: On task completion, always call `remote_terminal_session_cleanup` to tear down lingering background sessions and release system RAM.

---

## 4. Execution & Terminal Control Primitives

### Interactive PTY Commands (`remote_terminal_run`)

Execute commands inside an intelligent Herdr PTY pane:

```json
{
  "commandLine": "docker ps --format 'table {{.Names}}\t{{.Status}}'",
  "timeoutMs": 5000
}
```

- **Interactive TUIs & Prompts**: For commands asking for confirmation (`[Y/n]`), use `waitMatch: "?"` or send keys.
- **Key Dispatches**: Use `remote_terminal_send_keys(keys: "ctrl+c")` to interrupt, or `remote_terminal_send_keys(keys: "enter")` to submit.

### Spatial Multitasking (Spaces & Panes)

- **New Isolated Space**:
  ```json
  // remote_terminal_workspace_create
  { "label": "containers", "cwd": "/home/amadeus/Containers", "noFocus": true }
  ```
- **Split Existing Pane**:
  ```json
  // remote_terminal_split
  { "direction": "right", "cwd": "/home/amadeus/Projects" }
  ```
- **List Spaces & Panes**:
  ```json
  // remote_terminal_workspace_list
  ```

---

## 5. Remote Subagent Delegation (`remote_agent_*`)

Herdr features native multi-agent supervision. You can delegate tasks to CLI subagents running directly on the remote machine:

1. **Start Remote Agent**:
   ```json
   // remote_agent_start
   { "name": "indexer", "kind": "codex", "paneId": "w1:p2" }
   ```
2. **Prompt Remote Agent**:
   ```json
   // remote_agent_prompt
   {
     "name": "indexer",
     "prompt": "Analyze /srv/docker logs and report any container restart loops.",
     "wait": true,
     "timeoutMs": 60000
   }
   ```
3. **Read Findings**:
   ```json
   // remote_agent_read
   { "name": "indexer", "lines": 40 }
   ```

---

## 6. File Operations & Surgical SFTP Editing

- **Line-Sliced Viewing**:
  ```json
  // remote_view_file
  { "absolutePath": "/etc/fstab", "startLine": 1, "endLine": 30 }
  ```
- **Atomic Chunk Replacement**:
  ```json
  // remote_replace_file_content
  {
    "targetFile": "/srv/docker/docker-compose.yml",
    "targetContent": "image: nginx:1.24",
    "replacementContent": "image: nginx:1.26-alpine"
  }
  ```
- **Directory Trees**:
  ```json
  // remote_list_dir
  { "directoryPath": "/home/amadeus/Containers/enabled" }
  ```

---

## 7. Session Cleanup Protocol

When your remote work is finished:

```json
// remote_terminal_session_cleanup
{}
```

This releases remote PTYs, closes open background sockets, and frees host memory.
