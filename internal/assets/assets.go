package assets

import "embed"

//go:embed web/*
var WebFS embed.FS

//go:embed ComfyUI-ComfySwap/__init__.py
//go:embed ComfyUI-ComfySwap/js/*
//go:embed ComfyUI-ComfySwap/README.md
//go:embed ComfyUI-ComfySwap/LICENSE
//go:embed ComfyUI-ComfySwap/pyproject.toml
var PluginFS embed.FS
