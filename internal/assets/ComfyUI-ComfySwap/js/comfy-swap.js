import { app } from "../../scripts/app.js";

console.log("[ComfySwap] Plugin loading...");

// ============================================================
// Utility Functions
// ============================================================

function slugify(value) {
  return String(value || "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9_]+/g, "-")  // 保留下划线
    .replace(/^-+|-+$/g, "")
    .slice(0, 64) || "workflow";
}

function generateWorkflowName(promptObj) {
  const parts = [];
  let mainModel = "";
  let sampler = "";
  let resolution = "";
  let hasImg2Img = false;
  let hasControlNet = false;
  let hasUpscale = false;
  let hasInpaint = false;
  
  for (const node of Object.values(promptObj || {})) {
    const classType = (node.class_type || "").toLowerCase();
    const inputs = node.inputs || {};
    
    if (classType.includes("checkpointloader") && inputs.ckpt_name) {
      const ckpt = String(inputs.ckpt_name).replace(/\.(safetensors|ckpt|pt)$/i, "");
      const shortName = ckpt.split(/[\/\\]/).pop().slice(0, 20);
      mainModel = shortName;
    }
    if (classType.includes("ksampler") && inputs.sampler_name) {
      sampler = inputs.sampler_name;
    }
    if (classType.includes("emptylatent") && inputs.width && inputs.height) {
      resolution = `${inputs.width}x${inputs.height}`;
    }
    if (classType.includes("loadimage")) hasImg2Img = true;
    if (classType.includes("controlnet")) hasControlNet = true;
    if (classType.includes("upscale")) hasUpscale = true;
    if (classType.includes("inpaint")) hasInpaint = true;
  }
  
  if (mainModel) parts.push(mainModel);
  
  const tags = [];
  if (hasImg2Img) tags.push("i2i");
  if (hasControlNet) tags.push("cn");
  if (hasUpscale) tags.push("up");
  if (hasInpaint) tags.push("inp");
  
  if (tags.length) parts.push(tags.join("-"));
  if (sampler && sampler !== "euler") parts.push(sampler.slice(0, 8));
  if (resolution) parts.push(resolution);
  
  if (parts.length === 0) {
    const date = new Date();
    const ts = `${(date.getMonth()+1).toString().padStart(2,'0')}${date.getDate().toString().padStart(2,'0')}`;
    return `workflow-${ts}`;
  }
  
  return slugify(parts.join("-")).slice(0, 50);
}

function detectCandidateParams(prompt) {
  const list = [];
  for (const [nodeId, node] of Object.entries(prompt || {})) {
    const classType = node.class_type || "";
    const inputs = node.inputs || {};
    for (const key of Object.keys(inputs)) {
      const value = inputs[key];
      if (Array.isArray(value)) continue;
      
      if (["text", "seed", "steps", "cfg", "scheduler", "sampler_name", "denoise", "width", "height", "image", "positive", "negative", "ckpt_name", "vae_name", "clip_skip", "batch_size"].includes(key)) {
        let type = "string";
        if (["seed", "steps", "width", "height", "batch_size", "clip_skip"].includes(key)) type = "integer";
        else if (["denoise", "cfg"].includes(key)) type = "float";
        else if (key === "image") type = "image";
        
        list.push({
          name: key,
          type,
          node_id: String(nodeId),
          field: key,
          class_type: classType,
          default: value,
        });
      }
    }
  }
  return list;
}

function mergeCandidates(candidates) {
  const map = new Map();
  for (const c of candidates) {
    const key = `${c.name}:${c.type}`;
    if (!map.has(key)) {
      map.set(key, {
        name: c.name,
        type: c.type,
        default: c.default ?? "",
        description: getParamDescription(c.name, c.type),
        targets: [],
      });
    }
    map.get(key).targets.push({ node_id: c.node_id, field: c.field });
  }
  return Array.from(map.values());
}

async function fetchExistingWorkflows() {
  const url = localStorage.getItem("comfy_swap_url") || "http://localhost:8189";
  try {
    const r = await fetch(`${url}/api/workflows`, { method: "GET" });
    if (!r.ok) return [];
    const list = await r.json();
    return Array.isArray(list) ? list : [];
  } catch (e) {
    console.warn("[ComfySwap] Failed to fetch workflows:", e);
    return [];
  }
}

function createEl(tag, attrs = {}, text = "") {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") el.className = v;
    else if (k === "style") el.style.cssText = v;
    else el.setAttribute(k, v);
  }
  if (text) el.textContent = text;
  return el;
}

// ============================================================
// Styles
// ============================================================

const modalStyle = `
  .cs-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    z-index: 99999;
    display: flex;
    align-items: center;
    justify-content: center;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  }
  .cs-modal {
    width: min(840px, 95vw);
    max-height: 90vh;
    overflow: auto;
    background: #1a1a1a;
    border-radius: 6px;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
    border: 1px solid #333;
    color: #e0e0e0;
  }
  .cs-header {
    padding: 16px 20px;
    background: #252525;
    border-bottom: 1px solid #333;
  }
  .cs-header h3 {
    margin: 0 0 4px 0;
    font-size: 16px;
    font-weight: 600;
    color: #fff;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .cs-header h3 svg { opacity: 0.7; }
  .cs-header-desc {
    font-size: 12px;
    color: #888;
    margin: 0;
  }
  .cs-body {
    padding: 16px 20px;
  }
  .cs-form-row {
    margin-bottom: 14px;
  }
  .cs-form-row label {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
    font-size: 12px;
    font-weight: 500;
    color: #aaa;
  }
  .cs-form-row label .cs-label-hint {
    font-weight: 400;
    color: #666;
    font-size: 11px;
  }
  .cs-form-row input {
    width: 100%;
    padding: 8px 10px;
    border: 1px solid #444;
    border-radius: 4px;
    font-size: 13px;
    color: #fff;
    background: #2a2a2a;
    transition: border-color 0.2s;
  }
  .cs-form-row input:focus {
    outline: none;
    border-color: #0af;
  }
  .cs-name-preview {
    margin-top: 6px;
    font-size: 11px;
    color: #666;
  }
  .cs-name-preview code {
    background: #333;
    padding: 2px 6px;
    border-radius: 3px;
    font-family: monospace;
    color: #0af;
  }
  .cs-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }
  .cs-section-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 500;
    color: #aaa;
  }
  .cs-section-title svg { color: #888; }
  .cs-param-count {
    font-size: 11px;
    color: #666;
    background: #333;
    padding: 2px 8px;
    border-radius: 3px;
  }
  .cs-info-box {
    background: #252525;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 10px 12px;
    margin-bottom: 14px;
    font-size: 11px;
    color: #888;
    display: flex;
    align-items: flex-start;
    gap: 8px;
  }
  .cs-info-box svg { flex-shrink: 0; margin-top: 1px; color: #666; }
  .cs-info-box p { margin: 0; line-height: 1.5; }
  .cs-table-container {
    border: 1px solid #333;
    border-radius: 4px;
    overflow: hidden;
    margin-bottom: 14px;
  }
  .cs-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
  }
  .cs-table th {
    background: #252525;
    padding: 8px 10px;
    text-align: left;
    font-weight: 500;
    color: #888;
    border-bottom: 1px solid #333;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }
  .cs-table td {
    padding: 8px 10px;
    border-bottom: 1px solid #2a2a2a;
    color: #ccc;
    vertical-align: middle;
  }
  .cs-table tr:last-child td { border-bottom: none; }
  .cs-table tr:hover { background: #222; }
  .cs-table tr.cs-row-disabled { opacity: 0.4; }
  .cs-table input[type="text"] {
    width: 100%;
    padding: 5px 7px;
    border: 1px solid #444;
    border-radius: 3px;
    font-size: 12px;
    background: #2a2a2a;
    color: #fff;
  }
  .cs-table input[type="text"]:focus {
    outline: none;
    border-color: #0af;
  }
  .cs-table input[type="checkbox"] {
    width: 14px;
    height: 14px;
    accent-color: #0af;
    cursor: pointer;
  }
  .cs-type {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 10px;
    font-weight: 500;
  }
  .cs-type.integer { background: #1e3a5f; color: #5cb3ff; }
  .cs-type.float { background: #3d3012; color: #f5b642; }
  .cs-type.string { background: #1a3d2e; color: #4ade80; }
  .cs-type.image { background: #3d1f3d; color: #f472b6; }
  .cs-type-icon { font-size: 9px; }
  .cs-node { 
    font-size: 10px; 
    color: #666; 
    font-family: monospace;
    background: #2a2a2a;
    padding: 2px 5px;
    border-radius: 2px;
  }
  .cs-empty { text-align: center; padding: 24px; color: #666; font-size: 12px; }
  .cs-actions {
    display: flex;
    gap: 6px;
    margin-bottom: 10px;
  }
  .cs-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 6px 12px;
    border: none;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }
  .cs-btn-sm {
    padding: 4px 8px;
    font-size: 11px;
  }
  .cs-btn-outline {
    background: #2a2a2a;
    color: #ccc;
    border: 1px solid #444;
  }
  .cs-btn-outline:hover { 
    background: #333;
    border-color: #555;
  }
  .cs-btn-primary {
    background: #0af;
    color: #000;
  }
  .cs-btn-primary:hover { 
    background: #0cf;
  }
  .cs-footer {
    padding: 12px 20px;
    background: #252525;
    border-top: 1px solid #333;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .cs-toast {
    position: fixed;
    bottom: 24px;
    right: 24px;
    padding: 10px 16px;
    background: #333;
    color: #fff;
    border-radius: 4px;
    font-size: 13px;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.3);
    z-index: 100000;
    animation: cs-slideIn 0.2s ease;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .cs-toast.success { background: #059669; }
  .cs-toast.error { background: #dc2626; }
  @keyframes cs-slideIn {
    from { transform: translateY(16px); opacity: 0; }
    to { transform: translateY(0); opacity: 1; }
  }
  .cs-footer-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .cs-btn-link {
    background: none;
    color: #666;
    font-size: 11px;
    padding: 4px 8px;
  }
  .cs-btn-link:hover { color: #888; }
  .cs-more-panel {
    display: none;
    position: absolute;
    bottom: 50px;
    right: 20px;
    width: 320px;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 4px;
    box-shadow: 0 10px 30px rgba(0,0,0,0.4);
    z-index: 10;
  }
  .cs-more-panel.show { display: block; }
  .cs-more-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    border-bottom: 1px solid #333;
    font-size: 12px;
    font-weight: 500;
    color: #ccc;
  }
  .cs-more-close {
    background: none;
    border: none;
    font-size: 16px;
    color: #666;
    cursor: pointer;
    padding: 0 4px;
  }
  .cs-more-close:hover { color: #888; }
  .cs-more-body { padding: 6px; }
  .cs-more-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    padding: 10px;
    border-radius: 4px;
    transition: background 0.15s;
  }
  .cs-more-item:hover { background: #252525; }
  .cs-more-item-info {
    flex: 1;
    min-width: 0;
  }
  .cs-more-item-info strong {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: #ccc;
    font-weight: 500;
  }
  .cs-more-item-info span {
    font-size: 10px;
    color: #666;
    margin-top: 2px;
    display: block;
  }
  .cs-more-item-action {
    display: flex;
    gap: 5px;
  }
  .cs-more-item-action input {
    width: 120px;
    padding: 5px 7px;
    border: 1px solid #444;
    border-radius: 3px;
    font-size: 11px;
    background: #2a2a2a;
    color: #fff;
  }
  .cs-stats {
    display: flex;
    gap: 16px;
    padding: 10px 14px;
    background: #252525;
    border-radius: 4px;
    margin-bottom: 14px;
    border: 1px solid #333;
  }
  .cs-stat {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #888;
  }
  .cs-stat-value {
    font-weight: 600;
    color: #fff;
  }
  .cs-stat-icon { color: #666; }
  .cs-mode-section {
    margin-bottom: 14px;
  }
  .cs-mode-tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 12px;
  }
  .cs-mode-tab {
    flex: 1;
    padding: 8px 12px;
    background: #2a2a2a;
    border: 1px solid #444;
    border-radius: 4px;
    color: #888;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }
  .cs-mode-tab:hover { background: #333; color: #aaa; }
  .cs-mode-tab.active {
    background: #0af;
    border-color: #0af;
    color: #000;
  }
  .cs-mode-panel.hidden { display: none; }
  .cs-name-preview {
    margin-top: 6px;
    font-size: 11px;
    color: #666;
  }
  .cs-name-preview code {
    background: #333;
    padding: 2px 6px;
    border-radius: 3px;
    font-family: monospace;
    color: #0af;
  }
  .cs-form-row select {
    width: 100%;
    padding: 8px 10px;
    border: 1px solid #444;
    border-radius: 4px;
    font-size: 13px;
    color: #fff;
    background: #2a2a2a;
  }
  .cs-form-row select:focus {
    outline: none;
    border-color: #0af;
  }
  .cs-metadata {
    margin-bottom: 14px;
  }
  .cs-metadata-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 10px;
    background: #252525;
    border: 1px solid #333;
    border-radius: 4px;
    cursor: pointer;
    font-size: 11px;
    color: #888;
    width: 100%;
    text-align: left;
  }
  .cs-metadata-toggle:hover { background: #2a2a2a; }
  .cs-metadata-toggle svg { transition: transform 0.2s; }
  .cs-metadata-toggle.open svg { transform: rotate(90deg); }
  .cs-metadata-content {
    display: none;
    margin-top: 8px;
    padding: 10px;
    background: #1e1e1e;
    border: 1px solid #333;
    border-radius: 4px;
    max-height: 300px;
    overflow: auto;
  }
  .cs-metadata-content.show { display: block; }
  .cs-metadata-content pre {
    margin: 0;
    font-size: 10px;
    line-height: 1.4;
    color: #aaa;
    white-space: pre-wrap;
    word-break: break-all;
    font-family: 'Consolas', 'Monaco', monospace;
  }
`;

// ============================================================
// Main Functions
// ============================================================

async function openComfySwapExport() {
  let exported;
  try {
    exported = await app.graphToPrompt();
  } catch (e) {
    alert(`Swap failed: ${e.message}`);
    return;
  }
  const promptObj = exported?.output || exported;
  if (!promptObj || Object.keys(promptObj).length === 0) {
    alert("No workflow to swap. Please create a workflow first.");
    return;
  }
  const candidates = detectCandidateParams(promptObj);
  const initialMapping = mergeCandidates(candidates);
  openMappingPanel(promptObj, initialMapping);
}

function showToast(message, type = "") {
  const existing = document.querySelector(".cs-toast");
  if (existing) existing.remove();
  const toast = createEl("div", { class: `cs-toast ${type}` });
  const icon = type === "success" 
    ? '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>'
    : type === "error"
    ? '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>'
    : '';
  toast.innerHTML = icon + message;
  document.body.appendChild(toast);
  setTimeout(() => toast.remove(), 3000);
}

function getTypeIcon(type) {
  const icons = {
    integer: "123",
    float: "1.5",
    string: "Aa",
    image: "🖼"
  };
  return icons[type] || "?";
}

function getParamDescription(name, type) {
  const descriptions = {
    text: "Main prompt describing generated content",
    positive: "Positive prompt",
    negative: "Negative prompt, content to avoid",
    seed: "Random seed, -1 for random",
    steps: "Sampling steps, higher is more refined",
    cfg: "Prompt guidance strength",
    denoise: "Denoise strength (0-1)",
    width: "Image width (px)",
    height: "Image height (px)",
    batch_size: "Batch generation count",
    sampler_name: "Sampler algorithm",
    scheduler: "Scheduler type",
    ckpt_name: "Model checkpoint",
    vae_name: "VAE encoder",
    clip_skip: "CLIP skip layers",
    image: "Input image file",
  };
  return descriptions[name] || "";
}

function renderRows(state) {
  if (state.mapping.length === 0) {
    return `<tr><td colspan="6" class="cs-empty">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="margin-bottom:8px;opacity:0.5;">
        <circle cx="12" cy="12" r="10"/>
        <line x1="12" y1="8" x2="12" y2="12"/>
        <line x1="12" y1="16" x2="12.01" y2="16"/>
      </svg>
      <br/>No mappable parameters detected<br/>
      <span style="font-size:11px;color:#94a3b8;">Click "+ Add All" to add all input parameters</span>
    </td></tr>`;
  }
  return state.mapping.map((p, i) => {
    const nodeIds = (p.targets || []).map(t => t.node_id).join(", ");
    const rowClass = p.selected === false ? "cs-row-disabled" : "";
    const descValue = p.description || "";
    return `
    <tr class="${rowClass}">
      <td style="width:28px;text-align:center;">
        <input type="checkbox" data-k="sel" data-i="${i}" ${p.selected !== false ? "checked" : ""} title="Select to include in API"/>
      </td>
      <td style="width:100px;">
        <input type="text" data-k="name" data-i="${i}" value="${p.name}" placeholder="name"/>
      </td>
      <td style="width:60px;">
        <span class="cs-type ${p.type}">
          <span class="cs-type-icon">${getTypeIcon(p.type)}</span>
          ${p.type}
        </span>
      </td>
      <td style="width:120px;">
        <input type="text" data-k="default" data-i="${i}" value="${String(p.default ?? "")}" placeholder="default"/>
      </td>
      <td style="width:140px;">
        <input type="text" data-k="desc" data-i="${i}" value="${descValue}" placeholder="description"/>
      </td>
      <td style="width:70px;">
        <span class="cs-node" title="Mapped ComfyUI node">Node ${nodeIds}</span>
      </td>
    </tr>`;
  }).join("");
}

function buildPayload(state, promptObj) {
  const selectedParams = state.mapping.filter(m => m.selected !== false);
  const id = state.mode === "update" ? state.existingId : slugify(state.name);
  return {
    id,
    name: state.name,
    comfyui_workflow: promptObj,
    param_mapping: selectedParams.map(m => ({
      name: m.name,
      type: m.type,
      default: m.default,
      description: m.description || "",
      targets: m.targets,
    })),
  };
}

async function openMappingPanel(promptObj, initialMapping) {
  if (!document.getElementById("cs-style")) {
    const style = document.createElement("style");
    style.id = "cs-style";
    style.textContent = modalStyle;
    document.head.appendChild(style);
  }

  const existingWorkflows = await fetchExistingWorkflows();
  const autoName = generateWorkflowName(promptObj);
  
  const state = {
    mode: "create",
    name: autoName,
    existingId: "",
    mapping: initialMapping.map(m => ({ ...m, selected: true })),
  };

  const nodeCount = Object.keys(promptObj || {}).length;
  const paramCount = initialMapping.length;

  const overlay = createEl("div", { class: "cs-overlay" });
  const modal = createEl("div", { class: "cs-modal" });
  modal.innerHTML = `
    <div class="cs-header">
      <h3>
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="17 8 12 3 7 8"/>
          <line x1="12" y1="3" x2="12" y2="15"/>
        </svg>
        Export to Comfy-Swap
      </h3>
      <p class="cs-header-desc">Make this workflow callable via API and CLI</p>
    </div>
    <div class="cs-body">
      <div class="cs-stats">
        <div class="cs-stat">
          <svg class="cs-stat-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          </svg>
          <span>Nodes: <span class="cs-stat-value">${nodeCount}</span></span>
        </div>
        <div class="cs-stat">
          <svg class="cs-stat-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06"/>
          </svg>
          <span>Detected Params: <span class="cs-stat-value">${paramCount}</span></span>
        </div>
      </div>
      
      <div class="cs-mode-section">
        <div class="cs-mode-tabs">
          <button class="cs-mode-tab active" data-mode="create">Create New</button>
          <button class="cs-mode-tab" data-mode="update">Update Existing</button>
        </div>
        
        <div class="cs-mode-panel" id="cs-mode-create">
          <div class="cs-form-row">
            <label>Workflow Name</label>
            <input id="cs-name" value="${state.name}" placeholder="e.g. portrait-flux-1024" />
            <div class="cs-name-preview">API ID: <code id="cs-id-preview">${slugify(state.name)}</code></div>
          </div>
        </div>
        
        <div class="cs-mode-panel hidden" id="cs-mode-update">
          <div class="cs-form-row">
            <label>Select Workflow to Update</label>
            <select id="cs-existing-select">
              ${existingWorkflows.length === 0 
                ? '<option value="">No workflows found on server</option>'
                : existingWorkflows.map(w => `<option value="${w.id}">${w.name} (${w.id})</option>`).join('')}
            </select>
          </div>
          <div class="cs-form-row">
            <label>New Name <span class="cs-label-hint">(optional, leave empty to keep current)</span></label>
            <input id="cs-update-name" value="" placeholder="Leave empty to keep existing name" />
          </div>
        </div>
      </div>
      
      <div class="cs-section-header">
        <div class="cs-section-title">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="4" y1="21" x2="4" y2="14"/>
            <line x1="4" y1="10" x2="4" y2="3"/>
            <line x1="12" y1="21" x2="12" y2="12"/>
            <line x1="12" y1="8" x2="12" y2="3"/>
            <line x1="20" y1="21" x2="20" y2="16"/>
            <line x1="20" y1="12" x2="20" y2="3"/>
            <line x1="1" y1="14" x2="7" y2="14"/>
            <line x1="9" y1="8" x2="15" y2="8"/>
            <line x1="17" y1="16" x2="23" y2="16"/>
          </svg>
          API Parameters
        </div>
        <span class="cs-param-count" id="cs-selected-count">${paramCount} selected</span>
      </div>
      
      <div class="cs-actions">
        <button class="cs-btn cs-btn-outline cs-btn-sm" id="cs-add-all" title="Add all configurable input parameters from workflow">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Add All
        </button>
        <button class="cs-btn cs-btn-outline cs-btn-sm" id="cs-merge" title="Merge selected parameters into one (changes sync together)">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="16 3 21 3 21 8"/>
            <line x1="4" y1="20" x2="21" y2="3"/>
            <polyline points="21 16 21 21 16 21"/>
            <line x1="15" y1="15" x2="21" y2="21"/>
            <line x1="4" y1="4" x2="9" y2="9"/>
          </svg>
          Merge Selected
        </button>
      </div>
      <div class="cs-table-container">
        <table class="cs-table">
          <thead>
            <tr>
              <th style="width:28px;text-align:center;">
                <input type="checkbox" id="cs-select-all" checked title="Select/Deselect All"/>
              </th>
              <th>Name</th>
              <th>Type</th>
              <th>Default</th>
              <th>Description</th>
              <th>Source</th>
            </tr>
          </thead>
          <tbody id="cs-body"></tbody>
        </table>
      </div>
      
      <div class="cs-metadata">
        <button class="cs-metadata-toggle" id="cs-metadata-toggle">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
          View Raw Workflow JSON (for debugging)
        </button>
        <div class="cs-metadata-content" id="cs-metadata-content">
          <pre id="cs-metadata-json"></pre>
        </div>
      </div>
    </div>
    <div class="cs-footer">
      <button class="cs-btn cs-btn-outline" id="cs-cancel">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
        Cancel
      </button>
      <div class="cs-footer-right">
        <button class="cs-btn cs-btn-link" id="cs-more" title="More options">More ▾</button>
        <button class="cs-btn cs-btn-primary" id="cs-save">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="17 1 21 5 17 9"/>
            <path d="M3 11V9a4 4 0 0 1 4-4h14"/>
            <polyline points="7 23 3 19 7 15"/>
            <path d="M21 13v2a4 4 0 0 1-4 4H3"/>
          </svg>
          Swap
        </button>
      </div>
    </div>
    <div class="cs-more-panel" id="cs-more-panel">
      <div class="cs-more-header">
        <span>Export Options</span>
        <button class="cs-more-close" id="cs-more-close">×</button>
      </div>
      <div class="cs-more-body">
        <div class="cs-more-item">
          <div class="cs-more-item-info">
            <strong>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="22" y1="2" x2="11" y2="13"/>
                <polygon points="22 2 15 22 11 13 2 9 22 2"/>
              </svg>
              Direct Send
            </strong>
            <span>Send to specified Comfy-Swap server</span>
          </div>
          <div class="cs-more-item-action">
            <input id="cs-url" value="" placeholder="http://localhost:8189" />
            <button class="cs-btn cs-btn-outline cs-btn-sm" id="cs-send">Send</button>
          </div>
        </div>
        <div class="cs-more-item">
          <div class="cs-more-item-info">
            <strong>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
              </svg>
              Export JSON
            </strong>
            <span>Copy to clipboard for manual import</span>
          </div>
          <button class="cs-btn cs-btn-outline cs-btn-sm" id="cs-copy">Copy</button>
        </div>
        <div class="cs-more-item">
          <div class="cs-more-item-info">
            <strong>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                <polyline points="7 10 12 15 17 10"/>
                <line x1="12" y1="15" x2="12" y2="3"/>
              </svg>
              Export File
            </strong>
            <span>Download .json file for backup or transfer</span>
          </div>
          <button class="cs-btn cs-btn-outline cs-btn-sm" id="cs-download">Export</button>
        </div>
      </div>
    </div>
  `;
  overlay.appendChild(modal);
  document.body.appendChild(overlay);

  const body = modal.querySelector("#cs-body");
  const selectedCount = modal.querySelector("#cs-selected-count");
  const selectAllCheckbox = modal.querySelector("#cs-select-all");
  const nameInput = modal.querySelector("#cs-name");
  const idPreview = modal.querySelector("#cs-id-preview");
  const existingSelect = modal.querySelector("#cs-existing-select");
  const updateNameInput = modal.querySelector("#cs-update-name");
  const modeTabs = modal.querySelectorAll(".cs-mode-tab");
  const modeCreatePanel = modal.querySelector("#cs-mode-create");
  const modeUpdatePanel = modal.querySelector("#cs-mode-update");
  
  const updateSelectedCount = () => {
    const count = state.mapping.filter(m => m.selected !== false).length;
    selectedCount.textContent = `${count} selected`;
  };
  
  const refresh = () => { 
    body.innerHTML = renderRows(state); 
    updateSelectedCount();
  };
  refresh();

  modeTabs.forEach(tab => {
    tab.addEventListener("click", () => {
      modeTabs.forEach(t => t.classList.remove("active"));
      tab.classList.add("active");
      state.mode = tab.dataset.mode;
      
      if (state.mode === "create") {
        modeCreatePanel.classList.remove("hidden");
        modeUpdatePanel.classList.add("hidden");
      } else {
        modeCreatePanel.classList.add("hidden");
        modeUpdatePanel.classList.remove("hidden");
        if (existingSelect.value && !state.existingId) {
          state.existingId = existingSelect.value;
        }
      }
    });
  });

  nameInput.addEventListener("input", () => {
    state.name = nameInput.value.trim();
    idPreview.textContent = slugify(state.name);
  });

  existingSelect.addEventListener("change", () => {
    state.existingId = existingSelect.value;
    const selected = existingWorkflows.find(w => w.id === existingSelect.value);
    if (selected && !updateNameInput.value) {
      updateNameInput.placeholder = `Current: ${selected.name}`;
    }
  });

  updateNameInput.addEventListener("input", () => {
    state.name = updateNameInput.value.trim();
  });

  selectAllCheckbox.addEventListener("change", () => {
    const checked = selectAllCheckbox.checked;
    state.mapping.forEach(m => m.selected = checked);
    refresh();
  });

  overlay.addEventListener("click", e => { if (e.target === overlay) overlay.remove(); });
  modal.querySelector("#cs-cancel").addEventListener("click", () => overlay.remove());

  const metadataToggle = modal.querySelector("#cs-metadata-toggle");
  const metadataContent = modal.querySelector("#cs-metadata-content");
  const metadataJson = modal.querySelector("#cs-metadata-json");
  metadataJson.textContent = JSON.stringify(promptObj, null, 2);
  
  metadataToggle.addEventListener("click", () => {
    metadataToggle.classList.toggle("open");
    metadataContent.classList.toggle("show");
  });

  body.addEventListener("click", e => {
    const btn = e.target;
    if (btn.dataset?.k === "split") {
      const i = Number(btn.dataset.i);
      const item = state.mapping[i];
      if (!item?.targets || item.targets.length <= 1) return;
      const first = { ...item, targets: [item.targets[0]] };
      const rest = item.targets.slice(1).map((t, idx) => ({
        ...item,
        name: `${item.name}_${idx + 2}`,
        targets: [t],
      }));
      state.mapping.splice(i, 1, first, ...rest);
      refresh();
    }
  });

  body.addEventListener("input", e => {
    const input = e.target;
    if (!(input instanceof HTMLInputElement)) return;
    const i = Number(input.dataset.i);
    const item = state.mapping[i];
    if (!item) return;
    if (input.dataset.k === "name") item.name = input.value.trim();
    if (input.dataset.k === "default") item.default = input.value;
    if (input.dataset.k === "desc") item.description = input.value;
  });

  body.addEventListener("change", e => {
    const input = e.target;
    if (!(input instanceof HTMLInputElement)) return;
    const i = Number(input.dataset.i);
    const item = state.mapping[i];
    if (item && input.dataset.k === "sel") {
      item.selected = input.checked;
      updateSelectedCount();
      input.closest("tr")?.classList.toggle("cs-row-disabled", !input.checked);
    }
  });

  modal.querySelector("#cs-merge").addEventListener("click", () => {
    const sel = state.mapping.map((m, i) => [m, i]).filter(([m]) => m.selected).map(([, i]) => i);
    if (sel.length < 2) { alert("Select at least 2 parameters to merge."); return; }
    const base = state.mapping[sel[0]];
    for (let k = sel.length - 1; k >= 1; k--) {
      base.targets = [...base.targets, ...(state.mapping[sel[k]].targets || [])];
      state.mapping.splice(sel[k], 1);
    }
    refresh();
  });

  modal.querySelector("#cs-add-all").addEventListener("click", () => {
    let added = 0;
    for (const [nodeId, node] of Object.entries(promptObj || {})) {
      for (const [key, value] of Object.entries(node.inputs || {})) {
        if (Array.isArray(value)) continue;
        if (state.mapping.some(m => m.targets.some(t => t.node_id === String(nodeId) && t.field === key))) continue;
        let type = "string";
        if (typeof value === "number") type = Number.isInteger(value) ? "integer" : "float";
        state.mapping.push({
          name: `${key}_${nodeId}`,
          type,
          default: value,
          targets: [{ node_id: String(nodeId), field: key }],
          selected: false,
        });
        added++;
      }
    }
    refresh();
    if (added) showToast(`Added ${added} inputs`, "success");
  });

  // Validate helper
  function validate() {
    if (state.mode === "create") {
      state.name = modal.querySelector("#cs-name").value.trim();
      if (!state.name) { alert("Please enter a workflow name."); return null; }
      const newId = slugify(state.name);
      const conflict = existingWorkflows.find(w => w.id === newId);
      if (conflict) {
        alert(`A workflow with ID "${newId}" already exists.\nUse "Update Existing" to modify it, or choose a different name.`);
        return null;
      }
    } else {
      state.existingId = modal.querySelector("#cs-existing-select").value;
      if (!state.existingId) { alert("Please select a workflow to update."); return null; }
      const newName = modal.querySelector("#cs-update-name").value.trim();
      if (newName) state.name = newName;
      else {
        const existing = existingWorkflows.find(w => w.id === state.existingId);
        state.name = existing?.name || state.existingId;
      }
    }
    const selected = state.mapping.filter(m => m.selected !== false);
    if (!selected.length) { alert("Please select at least one parameter."); return null; }
    return buildPayload(state, promptObj);
  }

  // Swap (main action) - directly send to Comfy-Swap server
  modal.querySelector("#cs-save").addEventListener("click", async () => {
    const payload = validate();
    if (!payload) return;
    
    const url = (localStorage.getItem("comfy_swap_url") || "http://localhost:8189").trim();
    
    try {
      let r;
      if (state.mode === "create") {
        // Create mode: POST directly
        r = await fetch(`${url}/api/workflows`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
      } else {
        // Update mode: PUT to existing workflow
        r = await fetch(`${url}/api/workflows/${encodeURIComponent(payload.id)}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
      }
      
      if (!r.ok) {
        const errText = await r.text();
        let errMsg = errText;
        try {
          const errJson = JSON.parse(errText);
          errMsg = errJson.error || errJson.message || errText;
        } catch (_) {}
        throw new Error(errMsg);
      }
      
      const action = state.mode === "create" ? "created" : "updated";
      showToast(`"${state.name}" ${action}!`, "success");
      overlay.remove();
    } catch (e) {
      if (e.message.includes("fetch") || e.message.includes("network") || e.message.includes("Failed")) {
        alert(`Cannot connect to Comfy-Swap server.\n\nMake sure it's running at: ${url}`);
      } else {
        alert(`Swap failed: ${e.message}`);
      }
    }
  });

  // More panel toggle
  const morePanel = modal.querySelector("#cs-more-panel");
  const urlInput = modal.querySelector("#cs-url");
  urlInput.value = localStorage.getItem("comfy_swap_url") || "http://localhost:8189";
  
  modal.querySelector("#cs-more").addEventListener("click", () => {
    morePanel.classList.toggle("show");
  });
  modal.querySelector("#cs-more-close").addEventListener("click", () => {
    morePanel.classList.remove("show");
  });

  // Direct send
  modal.querySelector("#cs-send").addEventListener("click", async () => {
    const payload = validate();
    if (!payload) return;
    const url = urlInput.value.trim();
    if (!url) { alert("Please enter Comfy-Swap URL."); return; }
    try {
      let r = await fetch(`${url}/api/workflows`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (r.status === 409) {
        r = await fetch(`${url}/api/workflows/${encodeURIComponent(payload.id)}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
      }
      if (!r.ok) throw new Error(await r.text());
      localStorage.setItem("comfy_swap_url", url);
      showToast(`Sent to ${url}`, "success");
      morePanel.classList.remove("show");
    } catch (e) {
      alert(`Send failed: ${e.message}`);
    }
  });

  // Export JSON (copy)
  modal.querySelector("#cs-copy").addEventListener("click", async () => {
    const payload = validate();
    if (!payload) return;
    try {
      await navigator.clipboard.writeText(JSON.stringify(payload, null, 2));
      showToast("Exported to clipboard!", "success");
      morePanel.classList.remove("show");
    } catch (e) {
      alert(`Export failed: ${e.message}`);
    }
  });

  // Export File (download)
  modal.querySelector("#cs-download").addEventListener("click", () => {
    const payload = validate();
    if (!payload) return;
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `${payload.id}.json`;
    a.click();
    URL.revokeObjectURL(a.href);
    showToast(`Exported ${payload.id}.json`, "success");
    morePanel.classList.remove("show");
  });
}

// ============================================================
// Register Extension
// ============================================================

app.registerExtension({
  name: "ComfySwap",
  commands: [{ id: "comfyswap.export", label: "Export to ComfySwap", function: openComfySwapExport }],
  menuCommands: [{ path: ["Workflow"], commands: ["comfyswap.export"] }],
  getCanvasMenuItems() {
    return [null, { content: "Export to ComfySwap", callback: openComfySwapExport }];
  },
});

console.log("[ComfySwap] Plugin loaded. Access: Workflow menu or right-click canvas.");
