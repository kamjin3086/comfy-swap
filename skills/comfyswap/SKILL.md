---
name: comfyswap
description: Use Comfy-Swap CLI to run ComfyUI workflows, manage workflow parameters, and generate images. Trigger when user asks to run ComfyUI workflows, generate AI images, list workflows, modify workflow API parameters, inspect workflow nodes, adjust parameter mappings, check generation status, view request logs, or automate image generation tasks. Also trigger when user mentions comfy-swap, ComfyUI API, or image generation pipeline.
---

# Comfy-Swap

CLI tool for running ComfyUI workflows as stable APIs. Respond in the user's language.

## Quick Check

```bash
comfy-swap health
```

| Output | Action |
|--------|--------|
| `status: ok` + `comfyui.reachable: true` | Ready, check plugin status below |
| Command not found | [Install binary](#installation) |
| Connection refused | Start server: `comfy-swap serve` |
| `comfyui.reachable: false` | Configure: `comfy-swap config set --comfyui-url <url>` |

## Installation

### Step 1: Get Binary

**Option A: Download Release** (recommended)
- Download from [Releases](https://github.com/kamjin3086/comfy-swap/releases)

**Option B: Build from Source**
```bash
git clone https://github.com/kamjin3086/comfy-swap.git
cd comfy-swap
go build -o comfy-swap .   # Linux/macOS
go build -o comfy-swap.exe .   # Windows
```

### Step 2: Add to PATH (recommended for global access)

**Windows (PowerShell as Admin):**
```powershell
# Option 1: Copy to existing PATH directory (simplest)
Copy-Item comfy-swap.exe -Destination "$env:LOCALAPPDATA\Microsoft\WindowsApps\"

# Option 2: Add custom directory to PATH permanently
$binPath = "D:\path\to\comfy-swap-dir"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$binPath", "User")
# Restart terminal to apply
```

**Linux / macOS:**
```bash
# Option 1: Move to standard PATH directory
sudo mv comfy-swap /usr/local/bin/
sudo chmod +x /usr/local/bin/comfy-swap

# Option 2: Add to shell profile
echo 'export PATH="$PATH:/path/to/comfy-swap-dir"' >> ~/.bashrc  # or ~/.zshrc
source ~/.bashrc
```

**Verify installation:**
```bash
comfy-swap version
comfy-swap health
```

### Step 3: Start API Server

The server must be running for CLI commands and API calls to work:

```bash
comfy-swap serve
# Server runs on http://localhost:8189
# Data stored in OS standard location automatically
```

**Run as background service (production):**
```bash
# Linux (systemd)
sudo cp comfy-swap /usr/local/bin/
# Create /etc/systemd/system/comfy-swap.service, then:
sudo systemctl enable --now comfy-swap

# Windows (as service or scheduled task)
# Or simply run in a dedicated terminal/tmux session
```

### Step 4: Configure ComfyUI Connection

```bash
comfy-swap config set --comfyui-url http://localhost:8188
comfy-swap health  # Verify: comfyui.reachable: true
```

## Plugin Setup

Check if ComfyUI plugin is installed:
```bash
comfy-swap plugin-status
```

| Status | Action |
|--------|--------|
| `connected` | Ready to export workflows |
| `not_installed` | Install plugin below |
| `not_configured` | Configure ComfyUI URL first |

### Install Plugin (Local ComfyUI)

```bash
# Find ComfyUI custom_nodes directory, then:
comfy-swap install-plugin /path/to/ComfyUI/custom_nodes

# Restart ComfyUI, then verify:
comfy-swap plugin-status
```

Common paths:
- Windows: `C:\ComfyUI\custom_nodes`, `D:\ComfyUI\custom_nodes`
- Linux/macOS: `~/ComfyUI/custom_nodes`

For **remote ComfyUI**, see `references/setup.md` (Step 3).

## Running Workflows

```bash
# 1. List available workflows
comfy-swap list
# If empty → export workflow in ComfyUI (right-click → Export to ComfySwap), then:
#   comfy-swap import --sync

# 2. Check parameters
comfy-swap info <workflow_id>

# 3. Run and save output
comfy-swap run <workflow_id> prompt="a cat" seed=42 --wait --save ./output/
```

**After run completes**, verify and report:
- Exit code 0 → success; non-zero → check stderr
- Output JSON includes `prompt_id`, `status`, and `files` (saved paths)
- Confirm files exist: `ls ./output/`
- On failure: check `comfy-swap logs --workflow <id> --limit 1` for error details

## When to Read Additional References

| Task | Read |
|------|------|
| First-time setup / no workflows | `references/setup.md` |
| Remote ComfyUI plugin installation | `references/setup.md` (Step 3) |
| Modify workflow API parameters | `references/workflow-management.md` |
| Full CLI reference | `references/cli-reference.md` |
| Troubleshooting errors | `references/cli-reference.md` (Error Handling section) |

## Core Commands

| Command | Purpose |
|---------|---------|
| `health` | Check server + ComfyUI status |
| `plugin-status` | Check if ComfyUI plugin is installed |
| `install-plugin <path>` | Install plugin to ComfyUI custom_nodes |
| `list` | List workflows |
| `info <id>` | Show workflow parameters |
| `run <id> key=value --wait --save ./` | Execute workflow |
| `workflow nodes/params/add-param/...` | Manage API parameters |
| `logs` | View request history |

## Parameter Format

```bash
prompt="text value"    # string
seed=42                # integer  
cfg=7.5                # float
image=@./input.png     # image file (@ prefix)
```
