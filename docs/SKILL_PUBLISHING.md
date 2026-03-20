# Skill Publishing Guide

How to publish the `comfy-swap` skill to various skill registries.

## 1. skills.sh (OpenSkills Registry)

[skills.sh](https://skills.sh) is the primary registry for agent skills.

### Prerequisites

```bash
npm install -g openskills
```

### Publishing

```bash
# Login (first time)
npx openskills login

# Publish from skills directory
cd skills/comfy-swap
npx openskills publish

# Or publish with version bump
npx openskills publish --bump patch  # 1.0.0 -> 1.0.1
npx openskills publish --bump minor  # 1.0.0 -> 1.1.0
npx openskills publish --bump major  # 1.0.0 -> 2.0.0
```

### Verify Publication

```bash
# Search for your skill
npx openskills search comfy-swap

# View skill info
npx openskills info comfy-swap
```

### Update Published Skill

```bash
# Edit skills/comfy-swap/SKILL.md and plugin.json
# Then republish
cd skills/comfy-swap
npx openskills publish --bump patch
```

## 2. GitHub-based Distribution

Skills can also be installed directly from GitHub:

### For Users

```bash
# Install from GitHub
npx openskills install github:kamjin3086/comfy-swap/skills/comfy-swap

# Or clone and use locally
git clone https://github.com/kamjin3086/comfy-swap.git
# Copy skills/comfy-swap to your skills directory
```

### Making Your Repo Installable

The skill is already properly structured in `skills/comfy-swap/`:

```
skills/comfy-swap/
├── SKILL.md              # Main skill file (required)
├── plugin.json           # Metadata (recommended)
├── README.md             # Documentation
└── references/           # Additional docs
    ├── cli-reference.md
    ├── setup.md
    └── workflow-management.md
```

## 3. SkillsHub / Other Registries

### Anthropic SkillsHub

If Anthropic launches an official skills hub:

1. Check their submission guidelines
2. Ensure skill follows their format requirements
3. Submit via their web interface or CLI

### Custom Enterprise Registry

For internal enterprise use:

```bash
# Set custom registry
export OPENSKILLS_REGISTRY=https://skills.yourcompany.com

# Publish to custom registry
npx openskills publish
```

## 4. Skill Quality Checklist

Before publishing, verify:

- [ ] `SKILL.md` has proper frontmatter (name, description)
- [ ] `plugin.json` has correct version and metadata
- [ ] All references/ files are included
- [ ] README.md explains installation and basic usage
- [ ] Tested with real AI agent (Claude, GPT, etc.)
- [ ] No hardcoded paths or secrets
- [ ] License is specified

## 5. Version Management

Keep versions in sync:

| File | Version Field |
|------|---------------|
| `plugin.json` | `"version": "1.0.0"` |
| GitHub Release | Tag: `v1.0.0` |
| CHANGELOG.md | Document changes |

### Semantic Versioning

- **PATCH** (1.0.x): Bug fixes, doc updates
- **MINOR** (1.x.0): New features, backward compatible
- **MAJOR** (x.0.0): Breaking changes

## 6. Useful Commands

```bash
# Test skill locally before publishing
npx openskills read comfy-swap

# Validate skill structure
npx openskills validate skills/comfy-swap

# Check what would be published
npx openskills pack skills/comfy-swap --dry-run

# Unpublish (use with caution)
npx openskills unpublish comfy-swap
```

## 7. Troubleshooting

### "Skill already exists"

```bash
# Check ownership
npx openskills info comfy-swap

# If you own it, bump version and republish
npx openskills publish --bump patch
```

### "Invalid skill format"

```bash
# Validate structure
npx openskills validate skills/comfy-swap

# Common issues:
# - Missing SKILL.md frontmatter
# - Invalid plugin.json JSON
# - Missing required fields
```

### "Authentication failed"

```bash
# Re-login
npx openskills logout
npx openskills login
```
