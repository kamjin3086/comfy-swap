# Comfy-Swap Setup

Complete setup guide for AI agents.

## 0. Installation (if not installed)

### Check if already installed
```bash
comfy-swap version
# If command works → skip to Step 1
# If "command not found" → install below
```

### Download or Build

**Option A: Download Release**
- Get latest binary from [Releases](https://github.com/kamjin3086/comfy-swap/releases)
- Extract to a directory (e.g., `D:\tools\comfy-swap` or `/opt/comfy-swap`)

**Option B: Build from Source**
```bash
git clone https://github.com/kamjin3086/comfy-swap.git
cd comfy-swap
go build -o comfy-swap .      # Linux/macOS
go build -o comfy-swap.exe .  # Windows
```

### Add to PATH (Recommended)

Adding to PATH enables `comfy-swap` command from any directory.

**Windows (PowerShell - run as Administrator):**
```powershell
# Quick: Copy to existing PATH directory
Copy-Item .\comfy-swap.exe -Destination "$env:LOCALAPPDATA\Microsoft\WindowsApps\"

# Or: Add custom directory to User PATH permanently
$binDir = "D:\tools\comfy-swap"  # adjust to your path
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$binDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$binDir", "User")
    Write-Host "Added to PATH. Restart terminal to apply."
}
```

**Linux / macOS:**
```bash
# Quick: Move to standard bin directory
sudo mv comfy-swap /usr/local/bin/
sudo chmod +x /usr/local/bin/comfy-swap

# Or: Add to shell profile
COMFY_SWAP_DIR="/path/to/comfy-swap-dir"
echo "export PATH=\"\$PATH:$COMFY_SWAP_DIR\"" >> ~/.bashrc  # or ~/.zshrc
source ~/.bashrc
```

**Verify:**
```bash
# Open new terminal, then:
comfy-swap version
```

## 1. Start Server

```bash
comfy-swap serve
# Server runs on http://localhost:8189
# Keep this running in a dedicated terminal or as a service
```

**Production: Run as Background Service**

*Linux (systemd):*
```bash
sudo tee /etc/systemd/system/comfy-swap.service > /dev/null <<EOF
[Unit]
Description=Comfy-Swap API Server
After=network.target

[Service]
ExecStart=/usr/local/bin/comfy-swap serve
Restart=always
User=$USER

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now comfy-swap
sudo systemctl status comfy-swap
```

*Windows (background process):*
```powershell
# Run in background (simple)
Start-Process -WindowStyle Hidden comfy-swap -ArgumentList "serve"

# Or create a scheduled task to start on login
```

## 2. Configure ComfyUI URL

Check if config already exists before overwriting:
```bash
comfy-swap config get
# If settings exist and correct → skip this step
# If settings exist but wrong URL → proceed to update below
# If "not initialized" → proceed to set below
```

Set or update:
```bash
comfy-swap config set --comfyui-url http://localhost:8188
```

Verify:
```bash
comfy-swap health
# Should show comfyui.reachable: true
```

## 3. Install ComfyUI Plugin

Check current status:
```bash
comfy-swap plugin-status
# connected    → skip to Step 4
# not_installed → continue below
```

### 3a. Local ComfyUI (same machine) — Auto-Install

Find the ComfyUI `custom_nodes` directory, then install directly:

**Windows — search common locations:**
```powershell
# Check common paths in order
$candidates = @(
    "$env:USERPROFILE\ComfyUI\custom_nodes",
    "C:\ComfyUI\custom_nodes",
    "D:\ComfyUI\custom_nodes",
    "$env:USERPROFILE\Desktop\ComfyUI\custom_nodes",
    "$env:USERPROFILE\Documents\ComfyUI\custom_nodes"
)
# Also search for running ComfyUI process to find actual path
Get-Process python* -ErrorAction SilentlyContinue |
    Select-Object -ExpandProperty Path |
    ForEach-Object { Split-Path $_ -Parent }
```

**Linux/macOS — search common locations:**
```bash
# Find custom_nodes directories
find ~ /opt /usr/local -type d -name "custom_nodes" -path "*/ComfyUI/*" 2>/dev/null
# Also check running processes
ps aux | grep -i "[m]ain.py" | grep -i comfy
```

**Install once path is found:**
```bash
comfy-swap install-plugin /path/to/ComfyUI/custom_nodes
```

After installing, refresh or restart ComfyUI, then verify:
```bash
comfy-swap plugin-status
# Should show status: connected
```

### 3b. Remote ComfyUI (different machine)

When ComfyUI runs on a remote server, the AI agent cannot directly access its filesystem. Options:

1. **Ask the user** to run the install command on the remote machine:
   ```bash
   # On the remote machine where ComfyUI is installed:
   comfy-swap install-plugin /path/to/ComfyUI/custom_nodes
   ```

2. **Download plugin ZIP** and have user place it manually:
   ```bash
   # Download from comfy-swap server
   curl -o comfy-swap-plugin.zip http://localhost:8189/api/download-plugin
   # Transfer to remote machine and extract to ComfyUI/custom_nodes/
   ```

3. **If user has SSH access**, the AI agent can install remotely:
   ```bash
   # Copy binary to remote, then run install-plugin
   scp comfy-swap user@remote:/tmp/
   ssh user@remote "/tmp/comfy-swap install-plugin /path/to/ComfyUI/custom_nodes"
   ```

## 4. Import Workflows

**Option A: Sync from ComfyUI** (user exports first via right-click → Export to ComfySwap)
```bash
comfy-swap import --sync
# Verify: output shows synced count > 0
```

**Option B: Import from JSON file**
```bash
comfy-swap import workflow.json
# Verify: output shows "imported" with workflow_id
```

## 5. Final Verification

```bash
comfy-swap list           # Should show imported workflows
comfy-swap health         # comfyui.reachable: true
comfy-swap plugin-status  # status: connected
```

## Environment Variables (Optional)

| Variable | Description | Default |
|----------|-------------|---------|
| `COMFY_SWAP_URL` | Server URL | `http://localhost:8189` |
| `COMFY_SWAP_DATA` | Data directory | OS-specific |

These variables contain no secrets. If using custom values, set them in your shell profile rather than passing inline to avoid shell history clutter. Do not commit `.env` files containing custom paths to version control.
