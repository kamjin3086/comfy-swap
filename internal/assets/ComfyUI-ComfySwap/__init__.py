"""
ComfyUI-ComfySwap: A ComfyUI extension for Comfy-Swap workflow synchronization.

This plugin enables seamless workflow export from ComfyUI to Comfy-Swap,
allowing you to create API endpoints from your ComfyUI workflows.
"""

import server
from aiohttp import web

__version__ = "1.0.0"

WEB_DIRECTORY = "./js"
NODE_CLASS_MAPPINGS = {}
NODE_DISPLAY_NAME_MAPPINGS = {}

# Pending workflows queue (in-memory storage for workflows awaiting sync)
pending_workflows = []


@server.PromptServer.instance.routes.get("/comfyswap/status")
async def get_status(request):
    """Get plugin status and version information."""
    return web.json_response({
        "installed": True,
        "version": __version__,
        "pending_count": len(pending_workflows)
    })


@server.PromptServer.instance.routes.get("/comfyswap/pending")
async def get_pending(request):
    """Get the list of pending workflows awaiting synchronization."""
    return web.json_response({
        "workflows": pending_workflows
    })


@server.PromptServer.instance.routes.post("/comfyswap/pending")
async def add_pending(request):
    """Add a workflow to the pending queue. Updates if workflow ID already exists."""
    try:
        data = await request.json()
        if not data.get("id") or not data.get("name"):
            return web.json_response({"error": "Missing id or name"}, status=400)
        
        # Check if workflow already exists, update if so
        for i, wf in enumerate(pending_workflows):
            if wf.get("id") == data["id"]:
                pending_workflows[i] = data
                return web.json_response({"status": "updated", "id": data["id"]})
        
        pending_workflows.append(data)
        return web.json_response({"status": "added", "id": data["id"]})
    except Exception as e:
        return web.json_response({"error": str(e)}, status=400)


@server.PromptServer.instance.routes.delete("/comfyswap/pending/{workflow_id}")
async def remove_pending(request):
    """Remove a specific workflow from the pending queue after successful sync."""
    workflow_id = request.match_info.get("workflow_id")
    global pending_workflows
    pending_workflows = [wf for wf in pending_workflows if wf.get("id") != workflow_id]
    return web.json_response({"status": "removed", "id": workflow_id})


@server.PromptServer.instance.routes.delete("/comfyswap/pending")
async def clear_pending(request):
    """Clear all pending workflows from the queue."""
    global pending_workflows
    pending_workflows = []
    return web.json_response({"status": "cleared"})


__all__ = ["WEB_DIRECTORY", "NODE_CLASS_MAPPINGS", "NODE_DISPLAY_NAME_MAPPINGS"]
