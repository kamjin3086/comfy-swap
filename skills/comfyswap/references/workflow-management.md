# Workflow Management

Modify workflow API parameters without re-exporting from ComfyUI.

## View Workflow Structure

### List All Nodes
```bash
comfy-swap workflow nodes <workflow_id>
```

Output shows configurable fields per node:
```json
{"nodes": [
  {"node_id": "3", "class_type": "KSampler", "fields": [
    {"name": "seed", "type": "integer"},
    {"name": "steps", "type": "integer"}
  ]},
  {"node_id": "6", "class_type": "CLIPTextEncode", "fields": [
    {"name": "text", "type": "string"}
  ]}
]}
```

### View Current Parameters
```bash
comfy-swap workflow params <workflow_id>
```

## Modify Parameters

### Update Existing Parameter
```bash
# Rename
comfy-swap workflow update-param <id> --name old_name --rename new_name

# Change default
comfy-swap workflow update-param <id> --name seed --default 42

# Update description
comfy-swap workflow update-param <id> --name prompt --desc "Main prompt"

# Change type
comfy-swap workflow update-param <id> --name cfg --type float
```

### Add New Parameter
```bash
comfy-swap workflow add-param <id> \
  --name steps \
  --type integer \
  --node 3 \
  --field steps \
  --default 20 \
  --desc "Sampling steps"
```

### Remove Parameter
```bash
comfy-swap workflow remove-param <id> --name unwanted_param
```

## Multi-Node Control

One parameter can control multiple nodes:

```bash
# Add target (seed controls both node 3 and 9)
comfy-swap workflow add-target <id> --name seed --node 9 --field seed

# Remove target
comfy-swap workflow remove-target <id> --name seed --node 9
```

## Delete Workflow

```bash
comfy-swap workflow delete <id> --force
```

## Common Tasks

### "Add CFG scale control"
```bash
comfy-swap workflow nodes my-wf           # Find KSampler node
comfy-swap workflow add-param my-wf --name cfg --type float --node 3 --field cfg --default 7.5
```

### "Simplify API to only prompt and seed"
```bash
comfy-swap workflow params my-wf -q       # List all params
comfy-swap workflow remove-param my-wf --name steps
comfy-swap workflow remove-param my-wf --name sampler
# ... remove unwanted params
```

### "Sync all samplers to same seed"
```bash
comfy-swap workflow add-target my-wf --name seed --node 9 --field seed
```
