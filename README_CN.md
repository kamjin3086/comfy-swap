# Comfy-Swap

[English](README.md) | [中文](README_CN.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Comfy-Swap** 将 ComfyUI 工作流暴露为稳定、可用于生产的 API。它填补了 ComfyUI 可视化工作流设计与实际应用集成之间的空白。

## 它做什么

Comfy-Swap 将复杂的 ComfyUI 工作流转换为**简单、统一的 API 端点**：

- **AI Agent 友好** — 干净的 JSON 接口，可预测的参数，便于 LLM 和自动化工具调用
- **开发者友好** — 同一工作流，同一参数，无论使用 REST API 还是 CLI
- **生产就绪** — 稳定的 API 契约，即使更新工作流内部结构也不会破坏集成

### REST API

```bash
curl -X POST 'http://localhost:8189/api/prompt' \
  -H 'Content-Type: application/json' \
  -d '{
    "workflow_id": "portrait-gen",
    "params": {
      "prompt": "专业头像照，摄影棚灯光",
      "seed": 42
    }
  }'
```

### CLI

```bash
comfy-swap run portrait-gen -p prompt="专业头像照" -p seed="42" --wait --save ./output/
```

**同一工作流，同一参数，自由选择接口。**

## 为什么需要 Comfy-Swap？

| 问题 | 解决方案 |
|------|----------|
| ComfyUI 工作流是复杂的 JSON，包含节点 ID | Comfy-Swap 提供命名参数如 `prompt`、`seed`、`image` |
| 修改工作流内部会破坏集成 | 参数映射保持 API 稳定 |
| 难以从脚本、Agent 或自动化工具调用 | 简单的 REST/CLI，接口一致 |
| 没有统一的方式追踪 API 使用 | 内置请求日志，支持筛选 |

## 核心功能

- **工作流 → API**：将任何 ComfyUI 工作流导出为可调用的 API 端点
- **参数映射**：将友好名称映射到内部节点字段（一个参数 → 多个节点）
- **双接口**：REST API 和 CLI 共享相同参数
- **图片 I/O**：上传图片作为输入，下载生成的输出
- **请求日志**：追踪所有 API 调用，支持按工作流和时间筛选
- **Web 测试台**：集成前交互式测试工作流
- **备份/恢复**：导出所有配置便于迁移

## 它不做什么

Comfy-Swap 专注于**工作流暴露和 API 集成**。它故意不做：

- ❌ 管理 ComfyUI 安装或更新
- ❌ 安装或管理自定义节点
- ❌ 提供工作流编辑 UI
- ❌ 替代 ComfyUI 的执行引擎

这些由 ComfyUI 本身或 [ComfyUI Manager](https://github.com/ltdrdata/ComfyUI-Manager) 处理。

---

## 快速开始

### 1. 下载 & 运行

从 [**Releases**](https://github.com/your-repo/comfy-swap/releases) 下载适合你平台的最新版本。

```bash
# Windows
.\comfy-swap.exe serve --port 8189 --data-dir ./data

# macOS / Linux
./comfy-swap serve --port 8189 --data-dir ./data
```

打开 `http://localhost:8189` 完成设置。

> **从源码构建：** `git clone` + `go build -o comfy-swap .`（需要 Go 1.21+）

### 2. 安装 ComfyUI 插件

**方式 A：Git Clone（推荐）**

```bash
cd /path/to/ComfyUI/custom_nodes
git clone https://github.com/your-repo/ComfyUI-ComfySwap.git
```

安装后重启 ComfyUI。

<details>
<summary><b>方式 B：下载 ZIP</b></summary>

1. 打开 Comfy-Swap 网页界面 (`http://localhost:8189`)
2. 进入 **Settings** → **Plugin Installation** → **Download**
3. 点击 **Download Plugin ZIP**
4. 解压到 `ComfyUI/custom_nodes/ComfyUI-ComfySwap`
5. 重启 ComfyUI

</details>

### 3. 导出你的工作流

在 ComfyUI 中：右键画布 → **Export to ComfySwap** → 配置参数 → Swap

这会让你的工作流通过 Comfy-Swap 的统一 API 和 CLI 接口可用。

### 4. 调用 API

```bash
# REST API
curl -X POST 'http://localhost:8189/api/prompt' \
  -H 'Content-Type: application/json' \
  -d '{"workflow_id": "my-workflow", "params": {"prompt": "一只猫"}}'

# CLI
./comfy-swap run my-workflow -p prompt="一只猫" --wait
```

## CLI 命令

```bash
# 查看所有命令和选项
./comfy-swap --help
./comfy-swap run --help
```

| 命令 | 说明 |
|------|------|
| `serve` | 启动 HTTP 服务 |
| `run <id> -p key=value` | 执行工作流 |
| `list` | 列出所有工作流 |
| `info <id>` | 查看工作流详情 |
| `status <prompt_id>` | 检查执行状态 |
| `result <prompt_id>` | 获取结果（配合 `--save`） |
| `health` | 服务健康检查 |

**全局参数：** `-s, --server`（服务地址）、`-q, --quiet`、`--json`、`--pretty`

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/prompt` | POST | 执行工作流 |
| `/api/workflows` | GET | 列出工作流 |
| `/api/workflows/{id}` | GET | 获取工作流详情 |
| `/api/history/{prompt_id}` | GET | 执行历史 |
| `/api/logs` | GET | 请求日志 |
| `/api/upload` | POST | 上传图片 |
| `/api/view` | GET | 下载输出 |

## 参数映射

```json
{
  "name": "seed",
  "type": "integer",
  "default": -1,
  "description": "随机种子，-1 表示随机",
  "targets": [
    { "node_id": "3", "field": "seed" },
    { "node_id": "9", "field": "seed" }
  ]
}
```

一个 `seed` 参数自动更新多个节点。你的 API 保持简洁。

## 许可证

[MIT](LICENSE)
