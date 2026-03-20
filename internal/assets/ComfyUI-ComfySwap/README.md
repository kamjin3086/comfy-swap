# ComfyUI-ComfySwap

A ComfyUI extension that enables seamless workflow swap to [Comfy-Swap](https://github.com/kamjin3086/comfy-swap), making your workflows callable via REST API and CLI.

## Features

- **One-Click Export**: Export workflows directly from ComfyUI's interface
- **Parameter Mapping**: Automatically detect and configure API parameters
- **Multiple Options**: Save to queue, direct send, copy JSON, or download file
- **Seamless Sync**: Pending workflows are automatically synced to Comfy-Swap

## Installation

### Method 1: Git Clone (Recommended)

```bash
cd ComfyUI/custom_nodes
git clone https://github.com/kamjin3086/ComfyUI-ComfySwap.git
```

### Method 2: Download ZIP

1. Download from Comfy-Swap UI: **Settings** → **Plugin Installation** → **Download**
2. Extract to `ComfyUI/custom_nodes/ComfyUI-ComfySwap`
3. Restart ComfyUI

## Usage

### Export a Workflow

1. Create or open a workflow in ComfyUI
2. Access the export dialog via:
   - **Right-click**: Click on canvas → **Export to ComfySwap**
   - **Menu**: `Workflow` → `Export to ComfySwap`
3. Configure your workflow:
   - Choose **Create New** or **Update Existing**
   - Enter a workflow name
   - Select which parameters to expose as API inputs
   - Optionally merge related parameters
4. Click **Swap** to make it available via API/CLI

### Options

| Option | Description |
|--------|-------------|
| **Swap** | Register workflow with Comfy-Swap (via pending queue) |
| **Direct Send** | Send directly to a specified Comfy-Swap server |
| **Export JSON** | Copy to clipboard for manual import |
| **Export File** | Download .json file for backup or transfer |

## API Endpoints

This plugin exposes the following endpoints on your ComfyUI server:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/comfyswap/status` | GET | Get plugin status and version |
| `/comfyswap/pending` | GET | List pending workflows |
| `/comfyswap/pending` | POST | Add workflow to queue |
| `/comfyswap/pending/{id}` | DELETE | Remove workflow from queue |

## Requirements

- ComfyUI (latest version recommended)
- Comfy-Swap server for full functionality

## Troubleshooting

### Export dialog doesn't appear

- Refresh the browser page
- Check browser console for errors
- Ensure the plugin is properly installed in `custom_nodes`

### Workflows not syncing

- Verify Comfy-Swap server is running and connected
- Check the pending queue via `/comfyswap/pending`
- Use "Direct Send" option as an alternative

## License

MIT License

## Links

- [Comfy-Swap](https://github.com/kamjin3086/comfy-swap) - Main application
- [ComfyUI](https://github.com/comfyanonymous/ComfyUI) - The AI image generation platform
