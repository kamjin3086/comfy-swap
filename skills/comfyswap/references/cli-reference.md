# CLI Reference

Complete command reference for Comfy-Swap.

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--server` | `-s` | Server URL (default: http://localhost:8189) |
| `--quiet` | `-q` | Minimal output for scripting |
| `--json` | | JSON output (default) |
| `--pretty` | | Pretty-print JSON |

## Server & Plugin

```bash
comfy-swap serve [--port 8189]
comfy-swap health
comfy-swap plugin-status                   # Check if ComfyUI plugin is installed
comfy-swap install-plugin <custom_nodes_path>
```

## Configuration

```bash
comfy-swap config get
comfy-swap config get -q                    # Just URL
comfy-swap config set --comfyui-url URL [--log-retention-days N]
```

## Workflows

```bash
comfy-swap list
comfy-swap list -q                          # Just IDs
comfy-swap info <workflow_id>
comfy-swap import <file.json>
comfy-swap import -                         # From stdin
comfy-swap import --sync                    # Sync pending from ComfyUI
```

## Workflow Management

```bash
comfy-swap workflow nodes <id>
comfy-swap workflow params <id>
comfy-swap workflow update-param <id> --name <name> [--rename|--type|--default|--desc]
comfy-swap workflow add-param <id> --name <n> --type <t> --node <id> --field <f> [--default|--desc]
comfy-swap workflow remove-param <id> --name <name>
comfy-swap workflow add-target <id> --name <name> --node <id> --field <field>
comfy-swap workflow remove-target <id> --name <name> --node <id>
comfy-swap workflow delete <id> [--force]
```

## Execution

```bash
# Basic
comfy-swap run <workflow_id> key=value key2=value2

# Wait for completion
comfy-swap run <id> prompt="test" --wait

# Save output
comfy-swap run <id> prompt="test" --wait --save ./output/

# Custom timeout (default 300s)
comfy-swap run <id> prompt="test" --wait --timeout 600

# Image input
comfy-swap run <id> image=@./input.png --wait --save ./out/

# JSON params
comfy-swap run <id> --params '{"prompt":"test"}' --wait

# Quiet mode (returns file paths)
comfy-swap run <id> prompt="test" --wait --save ./out/ -q
```

## Status & Results

```bash
comfy-swap status <prompt_id>
comfy-swap status <prompt_id> -q            # Just "running" or "completed"
comfy-swap result <prompt_id>
comfy-swap result <prompt_id> --save ./downloads/
```

## Logs

```bash
comfy-swap logs
comfy-swap logs --workflow <id>
comfy-swap logs --limit 50 --offset 100
comfy-swap logs -q                          # Just total count
```

## Parameter Types

| Type | CLI Format | Example |
|------|------------|---------|
| string | `key="value"` | `prompt="a cat"` |
| integer | `key=123` | `seed=42` |
| float | `key=1.5` | `cfg=7.5` |
| boolean | `key=true` | `hires=true` |
| image | `key=@path` | `image=@./input.png` |

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| `command not found` | Not in PATH | Install binary, add to PATH |
| `connection refused` | Server not running | `comfy-swap serve` |
| `comfyui_url not configured` | No config | `comfy-swap config set --comfyui-url ...` |
| `workflow not found` | Not imported | `comfy-swap list`, import if missing |
| `parameter not found` | Wrong name | `comfy-swap workflow params <id>` |
| `plugin not_installed` | Plugin missing | `comfy-swap install-plugin <custom_nodes>` |
| `ComfyUI unreachable` | ComfyUI down | Check ComfyUI is running |
| `timeout` | Generation slow | Increase `--timeout` |
| `no pending workflows` | Nothing to sync | User must export from ComfyUI first |

## REST API Mapping

| CLI | API |
|-----|-----|
| `health` | `GET /api/health` |
| `plugin-status` | `GET /api/plugin-status` |
| `install-plugin` | `POST /api/install-plugin` |
| `config get` | `GET /api/settings` |
| `config set` | `PUT /api/settings` |
| `list` | `GET /api/workflows` |
| `info <id>` | `GET /api/workflows/{id}` |
| `import --sync` | `POST /api/sync-pending` |
| `workflow ...` | `PATCH /api/workflows/{id}/mapping` |
| `run` | `POST /api/prompt` |
| `status` | `GET /api/history/{id}` |
| `logs` | `GET /api/logs` |
