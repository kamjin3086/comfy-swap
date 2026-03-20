# comfy-swap

A skill for running ComfyUI workflows as stable APIs via CLI and REST.

## What it does

This skill enables AI agents to:
- Run ComfyUI image generation workflows
- Manage workflow parameters without re-exporting
- Generate images with custom prompts, seeds, and settings
- Track execution status and download outputs

## Installation

### 1. Install the CLI

```bash
# Download from releases
# https://github.com/kamjin3086/comfy-swap/releases

# Or build from source
git clone https://github.com/kamjin3086/comfy-swap.git
cd comfy-swap && go build -o comfy-swap .
```

### 2. Add to PATH

```bash
# Linux/macOS
sudo mv comfy-swap /usr/local/bin/

# Windows (PowerShell)
Copy-Item comfy-swap.exe -Destination "$env:LOCALAPPDATA\Microsoft\WindowsApps\"
```

### 3. Start server and configure

```bash
comfy-swap serve &
comfy-swap config set --comfyui-url http://localhost:8188
```

## Usage

```bash
# List workflows
comfy-swap list

# Generate image
comfy-swap run my-workflow prompt="a cat" --wait --save ./output/
```

## Requirements

- ComfyUI instance (local or remote)
- Go 1.21+ (for building from source)

## Links

- [GitHub Repository](https://github.com/kamjin3086/comfy-swap)
- [Full Documentation](https://github.com/kamjin3086/comfy-swap#readme)
