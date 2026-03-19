# comfy-swap

`comfy-swap` turns ComfyUI workflows into stable, callable APIs and CLI commands.

It is inspired by `llama-swap`, but tailored for ComfyUI workflow exposure, mapping, and maintenance.

## Highlights

- Convert ComfyUI workflows into reusable APIs via a ComfyUI plugin.
- Map one API parameter to multiple workflow targets (`targets[]`).
- Keep API contracts stable while workflows evolve (`re-connect` + update).
- Use either HTTP API or CLI (same binary, same behavior).
- Support image inputs (`image=@file.png`) and output download.
- Built-in web Playground to test params and generation quickly.
- Backup and restore all runtime data.
- Optional one-command plugin installation.

## Architecture

- **Server mode**: `comfy-swap serve`
- **CLI mode**: all other commands (`run`, `list`, `info`, etc.)
- **Storage**: local filesystem under `data/`
  - `data/settings.json`
  - `data/workflows/*.json`

## Quick Start

### 1) Build

```bash
# Linux / macOS
go build -o comfy-swap .

# Windows (PowerShell)
go build -o comfy-swap.exe .
```

### 2) Start server

```bash
# Linux / macOS
./comfy-swap serve --port 8189 --data-dir ./data

# Windows (PowerShell)
.\comfy-swap.exe serve --port 8189 --data-dir ./data
```

Open `http://localhost:8189` and complete the first-run wizard.

### 3) Install ComfyUI plugin

```bash
# Linux / macOS
./comfy-swap install-plugin /path/to/ComfyUI/custom_nodes

# Windows (PowerShell)
.\comfy-swap.exe install-plugin D:\ComfyUI\custom_nodes
```

Then refresh the ComfyUI page.

### 4) Connect a workflow from ComfyUI

In ComfyUI:

- `File` -> `Export` -> `Connect to Comfy Swap`
- Adjust parameter mapping in the popup
- Click `Connect`

The workflow will be available from both API and CLI.

### 5) Validate in Playground

Open the `Playground` section in the web UI:

- Select a workflow
- Fill params (including image upload params)
- Click `Run` or `Run and Wait`
- Inspect generated images and raw history payload

## CLI Usage

### Health

```bash
# Linux / macOS
./comfy-swap health

# Windows (PowerShell)
.\comfy-swap.exe health
```

### List workflows

```bash
# Linux / macOS
./comfy-swap list
./comfy-swap list -q

# Windows (PowerShell)
.\comfy-swap.exe list
.\comfy-swap.exe list -q
```

### Show workflow details

```bash
# Linux / macOS
./comfy-swap info txt2img

# Windows (PowerShell)
.\comfy-swap.exe info txt2img
```

### Run workflow

```bash
# Async (default)
./comfy-swap run txt2img prompt="a cat" seed=42

# Wait until complete
./comfy-swap run txt2img prompt="a cat" --wait

# Wait + save outputs
./comfy-swap run txt2img prompt="a cat" --wait --save ./output/

# Quiet mode (stdout is only prompt_id or file path)
./comfy-swap run txt2img prompt="a cat" --wait --save ./output/ -q

# Windows (PowerShell) example
.\comfy-swap.exe run txt2img prompt="a cat" --wait --save .\output\ -q
```

### `@file` support

```bash
# Long text prompt from file
./comfy-swap run txt2img prompt=@prompt.txt --wait --save ./out/

# Image upload for image-type params
./comfy-swap run img2img image=@input.png prompt="watercolor style" --wait --save ./out/

# Windows (PowerShell) examples
.\comfy-swap.exe run txt2img prompt=@prompt.txt --wait --save .\out\
.\comfy-swap.exe run img2img image=@input.png prompt="watercolor style" --wait --save .\out\
```

### Status and result

```bash
# Linux / macOS
./comfy-swap status <prompt_id>
./comfy-swap result <prompt_id>
./comfy-swap result <prompt_id> --save ./output/

# Windows (PowerShell)
.\comfy-swap.exe status <prompt_id>
.\comfy-swap.exe result <prompt_id>
.\comfy-swap.exe result <prompt_id> --save .\output\
```

## HTTP API (summary)

- `GET /api/health`
- `GET /api/settings`
- `PUT /api/settings`
- `GET /api/settings/status`
- `POST /api/workflows`
- `GET /api/workflows`
- `GET /api/workflows/{id}`
- `PUT /api/workflows/{id}`
- `PATCH /api/workflows/{id}/mapping`
- `DELETE /api/workflows/{id}`
- `POST /api/upload`
- `POST /api/prompt`
- `GET /api/history/{prompt_id}`
- `GET /api/view`
- `GET /api/backup`
- `POST /api/restore`
- `POST /api/install-plugin`

## Parameter Mapping Model

Each workflow file (`data/workflows/{id}.json`) includes:

- `comfyui_workflow`: API-format ComfyUI graph
- `param_mapping[]`:
  - `name`
  - `type`: `string | integer | float | boolean | image`
  - `default`
  - `targets[]`: one-to-many mapping (`node_id`, `field`)

Example:

```json
{
  "name": "seed",
  "type": "integer",
  "default": 0,
  "targets": [
    { "node_id": "3", "field": "seed" },
    { "node_id": "9", "field": "seed" }
  ]
}
```

## Development

### Run tests

```bash
go test ./...
```

### Build

```bash
go build ./...
```

### CI Workflow

GitHub Actions workflow is included at `.github/workflows/ci.yml` and runs:

- `go test ./...`
- `go build ./...`

### Lint/format

```bash
gofmt -w .
```

## Notes

- `comfy-swap` does not replace ComfyUI execution; it orchestrates and proxies it.
- For remote ComfyUI, skip wizard auto-install and run `install-plugin` on that machine.
- API compatibility is preserved as long as mapped API parameter names remain stable.
