let currentWorkflowId = "";
let currentWorkflow = null;
let playgroundWorkflow = null;
let playgroundPromptId = "";

async function jsonFetch(url, opts = {}) {
  const r = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  if (!r.ok) {
    const text = await r.text();
    throw new Error(text || `HTTP ${r.status}`);
  }
  return r.json();
}

function showToast(message, type = "info") {
  const existing = document.querySelector(".toast");
  if (existing) existing.remove();

  const toast = document.createElement("div");
  toast.className = `toast ${type}`;
  toast.textContent = message;
  document.body.appendChild(toast);

  setTimeout(() => toast.remove(), 3000);
}

function updateStatus(serverStatus, comfyReachable) {
  const badge = document.getElementById("status");
  if (!badge) return;
  
  const textEl = badge.querySelector(".status-text");
  if (!textEl) return;
  
  badge.classList.remove("connected", "disconnected");
  
  if (comfyReachable) {
    badge.classList.add("connected");
    textEl.textContent = "ComfyUI Connected";
  } else {
    badge.classList.add("disconnected");
    textEl.textContent = "ComfyUI Disconnected";
  }
}

async function loadHealth() {
  try {
    const h = await jsonFetch("/api/health");
    const reach = h.comfyui?.reachable ?? false;
    updateStatus(h.status, reach);
    
    // Also check plugin status
    if (reach) {
      await loadPluginStatus();
    }
  } catch (e) {
    updateStatus("error", false);
  }
}

async function loadPluginStatus() {
  const badge = document.getElementById("pluginStatus");
  const syncBtn = document.getElementById("syncBtn");
  const pendingCount = document.getElementById("pendingCount");
  
  if (!badge) return;
  
  const textEl = badge.querySelector(".status-text");
  if (!textEl) return;
  
  try {
    const status = await jsonFetch("/api/plugin-status");
    
    badge.classList.remove("not-installed", "connected");
    
    if (status.installed) {
      badge.classList.add("connected");
      textEl.textContent = `Plugin: v${status.version || "?"}`;
      
      // Show sync button if there are pending workflows
      const pending = status.pending_count || 0;
      if (pending > 0 && syncBtn && pendingCount) {
        syncBtn.classList.remove("hidden");
        pendingCount.textContent = pending;
      } else if (syncBtn) {
        syncBtn.classList.add("hidden");
      }
    } else {
      badge.classList.add("not-installed");
      textEl.textContent = "Plugin: Not Installed";
      if (syncBtn) syncBtn.classList.add("hidden");
    }
  } catch (e) {
    badge.classList.add("not-installed");
    textEl.textContent = "Plugin: Not Detected";
    if (syncBtn) syncBtn.classList.add("hidden");
  }
}

async function syncPendingWorkflows() {
  try {
    const result = await jsonFetch("/api/sync-pending", { method: "POST" });
    if (result.synced > 0) {
      showToast(`Synced ${result.synced} workflow(s)`, "success");
      await loadWorkflows();
    } else {
      showToast("No new workflows to sync", "info");
    }
    if (result.errors && result.errors.length > 0) {
      console.warn("Sync errors:", result.errors);
    }
    await loadPluginStatus();
  } catch (e) {
    showToast(`Sync failed: ${e.message}`, "error");
  }
}

async function loadSettings() {
  try {
    const s = await jsonFetch("/api/settings");
    document.getElementById("comfyUrl").value = s.comfyui_url || "";
    document.getElementById("wizardComfyUrl").value = s.comfyui_url || "http://127.0.0.1:8188";
    const logDaysEl = document.getElementById("logRetentionDays");
    if (logDaysEl) logDaysEl.value = s.log_retention_days || 7;
    return true;
  } catch (_) {}
  return false;
}

async function saveSettings() {
  const comfyui_url = document.getElementById("comfyUrl").value.trim();
  await jsonFetch("/api/settings", {
    method: "PUT",
    body: JSON.stringify({ comfyui_url }),
  });
  await loadHealth();
}

async function saveWizardSettings() {
  const comfyui_url = document.getElementById("wizardComfyUrl").value.trim();
  await jsonFetch("/api/settings", {
    method: "PUT",
    body: JSON.stringify({ comfyui_url }),
  });
  await loadHealth();
}

async function loadWorkflows() {
  const list = await jsonFetch("/api/workflows");
  const ul = document.getElementById("workflowList");
  const select = document.getElementById("playgroundWorkflow");
  ul.innerHTML = "";
  select.innerHTML = "";

  if (list.length === 0) {
    const li = document.createElement("li");
    li.className = "workflow-empty";
    li.textContent = "No workflows found";
    ul.appendChild(li);

    const opt = document.createElement("option");
    opt.value = "";
    opt.textContent = "No workflows available";
    select.appendChild(opt);
    renderPlaygroundParams(null);
    return;
  }

  for (const wf of list) {
    const li = document.createElement("li");
    li.dataset.id = wf.id;
    li.innerHTML = `
      <div class="wf-info">
        <strong>${escapeHtml(wf.name || wf.id)}</strong>
        <small>${wf.params_count} params · v${wf.version}</small>
      </div>
      <div class="wf-actions">
        <div class="wf-copy-dropdown">
          <button class="wf-action-btn wf-copy-btn" title="Copy command">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
            </svg>
          </button>
          <div class="wf-dropdown-menu">
            <button class="wf-dropdown-item" data-type="curl" data-wf-id="${escapeHtml(wf.id)}">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="16 18 22 12 16 6"/>
                <polyline points="8 6 2 12 8 18"/>
              </svg>
              Copy cURL
            </button>
            <button class="wf-dropdown-item" data-type="cli" data-wf-id="${escapeHtml(wf.id)}">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="4 17 10 11 4 5"/>
                <line x1="12" y1="19" x2="20" y2="19"/>
              </svg>
              Copy CLI
            </button>
          </div>
        </div>
      </div>
    `;
    li.querySelector(".wf-info").addEventListener("click", () => {
      document.querySelectorAll(".workflow-list li").forEach(el => el.classList.remove("active"));
      li.classList.add("active");
      loadWorkflowDetail(wf.id);
    });
    li.querySelector(".wf-copy-btn").addEventListener("click", (e) => {
      e.stopPropagation();
      const dropdown = li.querySelector(".wf-dropdown-menu");
      document.querySelectorAll(".wf-dropdown-menu.show").forEach(d => {
        if (d !== dropdown) d.classList.remove("show");
      });
      dropdown.classList.toggle("show");
    });
    li.querySelectorAll(".wf-dropdown-item").forEach(item => {
      item.addEventListener("click", async (e) => {
        e.stopPropagation();
        const type = item.dataset.type;
        const wfId = item.dataset.wfId;
        li.querySelector(".wf-dropdown-menu").classList.remove("show");
        await copyWorkflowCommand(wfId, type);
      });
    });
    ul.appendChild(li);

    const opt = document.createElement("option");
    opt.value = wf.id;
    opt.textContent = `${wf.name} (${wf.id})`;
    select.appendChild(opt);
  }

  if (list.length > 0) {
    await loadPlaygroundWorkflow(select.value || list[0].id);
  }
}

function renderWorkflowDetail(wf) {
  const container = document.getElementById("workflowDetail");
  const rows = (wf.param_mapping || [])
    .map(
      (p, idx) => `
      <tr>
        <td><input data-k="name" data-i="${idx}" value="${escapeHtml(p.name || "")}" /></td>
        <td><span class="type-badge">${escapeHtml(p.type || "")}</span></td>
        <td><input data-k="default" data-i="${idx}" value="${escapeHtml(String(p.default ?? ""))}" /></td>
        <td><input data-k="description" data-i="${idx}" value="${escapeHtml(p.description || "")}" /></td>
        <td><code>${(p.targets || []).map((t) => `${escapeHtml(t.node_id)}.${escapeHtml(t.field)}`).join("<br/>")}</code></td>
      </tr>
    `
    )
    .join("");

  container.innerHTML = `
    <h3>${escapeHtml(wf.name)} <small>(${escapeHtml(wf.id)})</small></h3>
    <p class="workflow-meta">Version: ${wf.version}</p>
    <table>
      <thead>
        <tr>
          <th>Param Name</th>
          <th>Type</th>
          <th>Default</th>
          <th>Description</th>
          <th>Targets</th>
        </tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>
    <div class="workflow-actions">
      <button id="saveMappingBtn" class="btn btn-primary">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
          <polyline points="17 21 17 13 7 13 7 21"/>
          <polyline points="7 3 7 8 15 8"/>
        </svg>
        Save Changes
      </button>
      <button id="deleteWorkflowBtn" class="btn btn-danger">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6"/>
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
        </svg>
        Delete
      </button>
    </div>
  `;

  document.getElementById("saveMappingBtn").addEventListener("click", saveMappingPatch);
  document.getElementById("deleteWorkflowBtn").addEventListener("click", deleteCurrentWorkflow);
}

async function loadWorkflowDetail(id) {
  currentWorkflowId = id;
  currentWorkflow = await jsonFetch(`/api/workflows/${encodeURIComponent(id)}`);
  renderWorkflowDetail(currentWorkflow);
}

async function saveMappingPatch() {
  if (!currentWorkflow) return;
  const rows = Array.from(document.querySelectorAll("#workflowDetail tbody tr"));
  const mapping = currentWorkflow.param_mapping.map((item, idx) => {
    const row = rows[idx];
    const name = row.querySelector(`input[data-k="name"]`).value.trim();
    const def = row.querySelector(`input[data-k="default"]`).value;
    const description = row.querySelector(`input[data-k="description"]`).value.trim();
    return { ...item, name, default: def, description };
  });
  await jsonFetch(`/api/workflows/${encodeURIComponent(currentWorkflowId)}/mapping`, {
    method: "PATCH",
    body: JSON.stringify({ param_mapping: mapping }),
  });
  showToast("Mapping updated successfully", "success");
  await loadWorkflowDetail(currentWorkflowId);
  await loadWorkflows();
}

async function deleteCurrentWorkflow() {
  if (!currentWorkflowId) return;
  if (!window.confirm(`Delete workflow "${currentWorkflowId}"?`)) return;
  await jsonFetch(`/api/workflows/${encodeURIComponent(currentWorkflowId)}`, { method: "DELETE" });
  currentWorkflowId = "";
  currentWorkflow = null;
  document.getElementById("workflowDetail").innerHTML = `
    <div class="empty-state">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
        <polyline points="14 2 14 8 20 8"/>
      </svg>
      <p>Select a workflow to view and edit its details</p>
    </div>
  `;
  showToast("Workflow deleted", "success");
  await loadWorkflows();
}

async function restoreBackup() {
  const input = document.getElementById("restoreFile");
  if (!input.files || !input.files.length) {
    showToast("Please select a ZIP file", "error");
    return;
  }
  const fd = new FormData();
  fd.append("file", input.files[0]);
  const r = await fetch("/api/restore", { method: "POST", body: fd });
  if (!r.ok) {
    throw new Error(await r.text());
  }
  showToast("Restore completed successfully", "success");
  await boot();
}

async function installPlugin(customNodesPath) {
  if (!customNodesPath) {
    showToast("Please enter the custom_nodes path", "error");
    return;
  }
  await jsonFetch("/api/install-plugin", {
    method: "POST",
    body: JSON.stringify({ custom_nodes_path: customNodesPath }),
  });
  showToast("Plugin installed! Please refresh ComfyUI.", "success");
}

async function installPluginFromWizard() {
  const custom_nodes_path = document.getElementById("wizardCustomNodesPath").value.trim();
  await installPlugin(custom_nodes_path);
}

async function installPluginFromSettings() {
  const custom_nodes_path = document.getElementById("settingsCustomNodesPath").value.trim();
  await installPlugin(custom_nodes_path);
}

function escapeHtml(str) {
  return String(str)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function renderPlaygroundParams(wf) {
  playgroundWorkflow = wf;
  const container = document.getElementById("playgroundParams");
  container.innerHTML = "";
  if (!wf) return;

  for (const p of wf.param_mapping || []) {
    const card = document.createElement("div");
    card.className = "param-card";
    const safeName = escapeHtml(p.name || "");
    const safeType = escapeHtml(p.type || "");

    if (p.type === "image") {
      card.innerHTML = `
        <strong>${safeName}</strong>
        <span class="param-meta">${safeType}</span>
        <input type="file" data-param="${safeName}" accept="image/*" />
      `;
    } else {
      const safeValue = escapeHtml(String(p.default ?? ""));
      card.innerHTML = `
        <strong>${safeName}</strong>
        <span class="param-meta">${safeType}</span>
        <input data-param="${safeName}" value="${safeValue}" placeholder="${safeName}..." />
      `;
    }
    container.appendChild(card);
  }
}

async function loadPlaygroundWorkflow(id) {
  if (!id) {
    renderPlaygroundParams(null);
    return;
  }
  const wf = await jsonFetch(`/api/workflows/${encodeURIComponent(id)}`);
  renderPlaygroundParams(wf);
  renderPlaygroundOutput();
}

function renderPlaygroundResult(contentHtml) {
  const el = document.getElementById("playgroundResult");
  el.innerHTML = contentHtml;
}

function getHistoryEntry(history, promptId) {
  return history?.[promptId] || null;
}

function extractHistoryImages(entry) {
  const outputs = entry?.outputs || {};
  const images = [];
  for (const nodeOutput of Object.values(outputs)) {
    const nodeImages = nodeOutput?.images || [];
    for (const img of nodeImages) {
      images.push({
        filename: img.filename || "",
        subfolder: img.subfolder || "",
        type: img.type || "output",
      });
    }
  }
  return images;
}

function imageURL(img) {
  const q = new URLSearchParams({
    filename: img.filename,
    subfolder: img.subfolder || "",
    type: img.type || "output",
  });
  return `/api/view?${q.toString()}`;
}

async function uploadImageFile(file) {
  const fd = new FormData();
  fd.append("image", file);
  const r = await fetch("/api/upload", { method: "POST", body: fd });
  if (!r.ok) {
    throw new Error(await r.text());
  }
  return r.json();
}

async function collectPlaygroundParams() {
  if (!playgroundWorkflow) {
    throw new Error("No workflow selected");
  }
  const params = {};
  for (const p of playgroundWorkflow.param_mapping || []) {
    const selector = `[data-param="${CSS.escape(p.name)}"]`;
    const el = document.querySelector(selector);
    if (!el) continue;
    if (p.type === "image") {
      const file = el.files && el.files[0];
      if (file) {
        const uploaded = await uploadImageFile(file);
        params[p.name] = uploaded.filename;
      } else if (p.default) {
        params[p.name] = p.default;
      }
      continue;
    }
    const raw = (el.value ?? "").trim();
    if (raw === "" && p.default !== undefined) {
      params[p.name] = p.default;
      continue;
    }
    params[p.name] = raw;
  }
  return params;
}

function updateResultStatus(status, text) {
  const header = document.querySelector(".result-header");
  if (!header) return;
  
  let statusEl = header.querySelector(".result-status");
  if (!statusEl) {
    statusEl = document.createElement("div");
    statusEl.className = "result-status";
    header.appendChild(statusEl);
  }
  
  statusEl.className = `result-status ${status}`;
  if (status === "running") {
    statusEl.innerHTML = `<div class="loading-spinner" style="width:14px;height:14px;"></div><span>${text}</span>`;
  } else if (status === "completed") {
    statusEl.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg><span>${text}</span>`;
  } else {
    statusEl.innerHTML = text ? `<span>${text}</span>` : "";
  }
}

function generateCurlCommand(workflowId, params) {
  const baseUrl = window.location.origin;
  const payload = { workflow_id: workflowId, params };
  const jsonStr = JSON.stringify(payload, null, 2);
  return `curl -X POST '${baseUrl}/api/prompt' \\
  -H 'Content-Type: application/json' \\
  -d '${jsonStr.replace(/'/g, "'\\''")}'`;
}

function generateCliCommand(workflowId, params) {
  const args = [];
  for (const [key, value] of Object.entries(params || {})) {
    const strVal = String(value).replace(/"/g, '\\"');
    args.push(`-p ${key}="${strVal}"`);
  }
  return `comfy-swap run ${workflowId} ${args.join(' ')}`.trim();
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).then(() => {
    showToast("Copied to clipboard", "success");
  }).catch(() => {
    showToast("Failed to copy", "error");
  });
}

async function copyWorkflowCommand(workflowId, type = "curl") {
  try {
    const wf = await jsonFetch(`/api/workflows/${encodeURIComponent(workflowId)}`);
    const params = {};
    for (const p of wf.param_mapping || []) {
      params[p.name] = p.default ?? (p.type === "integer" ? 0 : p.type === "float" ? 0.0 : "");
    }
    const cmd = type === "cli" 
      ? generateCliCommand(workflowId, params)
      : generateCurlCommand(workflowId, params);
    copyToClipboard(cmd);
  } catch (e) {
    showToast(`Failed to generate ${type}: ` + e.message, "error");
  }
}

function showImageLightbox(src, alt) {
  const existing = document.querySelector(".lightbox-overlay");
  if (existing) existing.remove();
  
  const overlay = document.createElement("div");
  overlay.className = "lightbox-overlay";
  overlay.innerHTML = `
    <div class="lightbox-content">
      <button class="lightbox-close">&times;</button>
      <img src="${src}" alt="${escapeHtml(alt || '')}" />
      <div class="lightbox-caption">${escapeHtml(alt || '')}</div>
    </div>
  `;
  document.body.appendChild(overlay);
  
  overlay.addEventListener("click", (e) => {
    if (e.target === overlay || e.target.classList.contains("lightbox-close")) {
      overlay.remove();
    }
  });
  document.addEventListener("keydown", function handler(e) {
    if (e.key === "Escape") {
      overlay.remove();
      document.removeEventListener("keydown", handler);
    }
  });
}

let playgroundLogs = [];

async function runPlayground() {
  const workflowId = document.getElementById("playgroundWorkflow").value;
  if (!workflowId) {
    showToast("Please select a workflow", "error");
    return;
  }
  
  const params = await collectPlaygroundParams();
  const curlCmd = generateCurlCommand(workflowId, params);
  const cliCmd = generateCliCommand(workflowId, params);
  const timestamp = new Date().toLocaleTimeString();
  
  const logEntry = {
    time: timestamp,
    workflowId,
    params,
    curl: curlCmd,
    cli: cliCmd,
    status: "running",
    promptId: "",
    images: [],
    elapsed: 0
  };
  playgroundLogs.unshift(logEntry);
  
  renderPlaygroundOutput();
  updateResultStatus("running", "Submitting...");

  const runResp = await jsonFetch("/api/prompt", {
    method: "POST",
    body: JSON.stringify({ workflow_id: workflowId, params }),
  });
  playgroundPromptId = runResp.prompt_id || "";
  logEntry.promptId = playgroundPromptId;
  
  renderPlaygroundOutput();
  updateResultStatus("running", "Generating...");

  const started = Date.now();
  let history = {};
  let pollCount = 0;
  
  while (true) {
    try {
      history = await jsonFetch(`/api/history/${encodeURIComponent(playgroundPromptId)}`);
      const entry = getHistoryEntry(history, playgroundPromptId);
      if (entry) break;
    } catch (e) {
      console.warn("Poll error:", e);
    }
    
    pollCount++;
    const elapsed = Math.round((Date.now() - started) / 1000);
    logEntry.elapsed = elapsed;
    updateResultStatus("running", `Generating... ${elapsed}s`);
    
    if (Date.now() - started > 300000) {
      logEntry.status = "error";
      logEntry.error = "Timeout (5 min)";
      renderPlaygroundOutput();
      throw new Error("Timeout while waiting for generation (5 min).");
    }
    
    const delay = pollCount < 5 ? 1000 : pollCount < 15 ? 2000 : 3000;
    await new Promise((resolve) => setTimeout(resolve, delay));
  }

  const entry = getHistoryEntry(history, playgroundPromptId);
  const images = extractHistoryImages(entry);
  const elapsed = Math.round((Date.now() - started) / 1000);
  
  logEntry.status = "completed";
  logEntry.elapsed = elapsed;
  logEntry.images = images;
  
  renderPlaygroundOutput();
  updateResultStatus("completed", `Done in ${elapsed}s`);
  showToast(`Generation completed (${images.length} image${images.length !== 1 ? 's' : ''})`, "success");
}

function renderPlaygroundOutput() {
  const container = document.querySelector(".result-content");
  if (!container) return;
  
  if (playgroundLogs.length === 0) {
    container.innerHTML = `
      <div class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        <p>Select a workflow and click Generate to test</p>
      </div>
    `;
    return;
  }
  
  const logsHtml = playgroundLogs.map((log, idx) => {
    const statusIcon = log.status === "running" 
      ? '<div class="loading-spinner" style="width:12px;height:12px;"></div>'
      : log.status === "completed"
      ? '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--success)" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>'
      : '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--danger)" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>';
    
    const statusText = log.status === "running" 
      ? `Generating... ${log.elapsed}s`
      : log.status === "completed"
      ? `Done in ${log.elapsed}s`
      : `Error: ${log.error || "Unknown"}`;
    
    const imagesHtml = log.images.length > 0 ? `
      <div class="log-images">
        ${log.images.map(img => `
          <div class="log-image" onclick="showImageLightbox('${imageURL(img)}', '${escapeHtml(img.filename)}')">
            <img src="${imageURL(img)}" alt="${escapeHtml(img.filename)}" loading="lazy" />
          </div>
        `).join('')}
      </div>
    ` : '';
    
    return `
      <div class="log-entry ${log.status}">
        <div class="log-header">
          <div class="log-status">${statusIcon}<span>${statusText}</span></div>
          <span class="log-time">${log.time}</span>
        </div>
        ${log.promptId ? `<div class="log-prompt-id">Prompt: <code>${escapeHtml(log.promptId)}</code></div>` : ''}
        ${imagesHtml}
        <details class="log-details">
          <summary>Request Details</summary>
          <div class="log-commands">
            <div class="log-cmd-block">
              <div class="log-cmd-header">
                <span>cURL</span>
                <button class="btn-copy" onclick="copyToClipboard(\`${log.curl.replace(/`/g, '\\`').replace(/\\/g, '\\\\')}\`)">Copy</button>
              </div>
              <pre>${escapeHtml(log.curl)}</pre>
            </div>
            <div class="log-cmd-block">
              <div class="log-cmd-header">
                <span>CLI</span>
                <button class="btn-copy" onclick="copyToClipboard(\`${(log.cli || '').replace(/`/g, '\\`').replace(/\\/g, '\\\\')}\`)">Copy</button>
              </div>
              <pre>${escapeHtml(log.cli || '')}</pre>
            </div>
          </div>
        </details>
      </div>
    `;
  }).join('');
  
  container.innerHTML = `
    <div class="log-list">
      ${logsHtml}
    </div>
  `;
}

function initTabs() {
  const tabBtns = document.querySelectorAll(".tab-btn");
  const tabContents = document.querySelectorAll(".tab-content");

  tabBtns.forEach((btn) => {
    btn.addEventListener("click", () => {
      const targetTab = btn.dataset.tab;

      tabBtns.forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");

      tabContents.forEach((content) => {
        content.classList.remove("active");
        if (content.id === `tab-${targetTab}`) {
          content.classList.add("active");
        }
      });
      
      if (targetTab === "logs") {
        fetchLogsTab();
      }
    });
  });
}

function initFileUploadLabels() {
  const fileInputs = document.querySelectorAll('input[type="file"]');
  fileInputs.forEach((input) => {
    input.addEventListener("change", () => {
      const label = input.parentElement.querySelector(".file-upload-label");
      if (label && input.files && input.files.length > 0) {
        label.innerHTML = `
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
          ${input.files[0].name}
        `;
      }
    });
  });
}

function initInstallCommand() {
  // Generic copy command buttons (used in multiple places)
  document.querySelectorAll(".copy-cmd").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const cmd = btn.dataset.cmd;
      if (!cmd) return;
      
      try {
        await navigator.clipboard.writeText(cmd);
        btn.classList.add("copied");
        const originalHTML = btn.innerHTML;
        btn.innerHTML = `
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        `;
        setTimeout(() => {
          btn.classList.remove("copied");
          btn.innerHTML = originalHTML;
        }, 2000);
        showToast("Copied to clipboard", "success");
      } catch (e) {
        showToast("Failed to copy", "error");
      }
    });
  });

  // Mini tabs for install options
  const miniTabs = document.querySelectorAll(".mini-tab");
  miniTabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const targetPanel = tab.dataset.installTab;
      
      miniTabs.forEach((t) => t.classList.remove("active"));
      tab.classList.add("active");

      document.querySelectorAll(".install-panel").forEach((panel) => {
        panel.classList.remove("active");
      });
      document.getElementById(`install-${targetPanel}`)?.classList.add("active");
    });
  });
}

function initImportModal() {
  const importBtn = document.getElementById("importWorkflowBtn");
  const importModal = document.getElementById("importModal");
  const closeBtn = document.getElementById("closeImportModal");
  
  if (!importBtn || !importModal) return;

  // Open modal
  importBtn.addEventListener("click", () => {
    importModal.classList.remove("hidden");
  });

  // Close modal
  closeBtn?.addEventListener("click", () => {
    importModal.classList.add("hidden");
  });

  importModal.addEventListener("click", (e) => {
    if (e.target === importModal) {
      importModal.classList.add("hidden");
    }
  });

  // Import tabs
  const importTabs = document.querySelectorAll(".import-tab");
  importTabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      const targetPanel = tab.dataset.importTab;
      
      importTabs.forEach((t) => t.classList.remove("active"));
      tab.classList.add("active");

      document.querySelectorAll(".import-panel").forEach((panel) => {
        panel.classList.remove("active");
      });
      document.getElementById(`import-${targetPanel}`)?.classList.add("active");
    });
  });

  // Import from paste
  document.getElementById("importFromPaste")?.addEventListener("click", async () => {
    const textarea = document.getElementById("importJsonText");
    const jsonText = textarea?.value?.trim();
    
    if (!jsonText) {
      showToast("Please paste JSON content", "error");
      return;
    }

    try {
      const payload = JSON.parse(jsonText);
      await importWorkflow(payload);
      importModal.classList.add("hidden");
      textarea.value = "";
    } catch (e) {
      showToast(`Import failed: ${e.message}`, "error");
    }
  });

  // Import from file
  const fileInput = document.getElementById("importJsonFile");
  fileInput?.addEventListener("change", () => {
    const label = fileInput.parentElement?.querySelector(".file-upload-label");
    if (label && fileInput.files && fileInput.files.length > 0) {
      label.innerHTML = `
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
          <polyline points="22 4 12 14.01 9 11.01"/>
        </svg>
        ${fileInput.files[0].name}
      `;
    }
  });

  document.getElementById("importFromFile")?.addEventListener("click", async () => {
    const fileInput = document.getElementById("importJsonFile");
    if (!fileInput?.files || !fileInput.files.length) {
      showToast("Please select a JSON file", "error");
      return;
    }

    try {
      const file = fileInput.files[0];
      const text = await file.text();
      const payload = JSON.parse(text);
      await importWorkflow(payload);
      importModal.classList.add("hidden");
      fileInput.value = "";
      const label = fileInput.parentElement?.querySelector(".file-upload-label");
      if (label) {
        label.innerHTML = `
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="17 8 12 3 7 8"/>
            <line x1="12" y1="3" x2="12" y2="15"/>
          </svg>
          Choose JSON file...
        `;
      }
    } catch (e) {
      showToast(`Import failed: ${e.message}`, "error");
    }
  });
}

async function importWorkflow(payload) {
  if (!payload.id || !payload.name || !payload.comfyui_workflow) {
    throw new Error("Invalid workflow JSON: missing id, name, or comfyui_workflow");
  }

  // Try POST first (create)
  let r = await fetch("/api/workflows", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  // If conflict, try PUT (update)
  if (r.status === 409) {
    r = await fetch(`/api/workflows/${encodeURIComponent(payload.id)}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  }

  if (!r.ok) {
    const t = await r.text();
    throw new Error(t);
  }

  showToast(`Workflow "${payload.name}" imported successfully`, "success");
  await loadWorkflows();
}

function initEventListeners() {
  // Sync button (header)
  document.getElementById("syncBtn")?.addEventListener("click", async () => {
    await syncPendingWorkflows();
  });

  // Refresh button (workflow list sidebar)
  document.getElementById("refreshWorkflowBtn")?.addEventListener("click", async (e) => {
    const btn = e.currentTarget;
    if (btn.classList.contains("spinning")) return;
    btn.classList.add("spinning");
    try {
      await syncPendingWorkflows();
    } finally {
      btn.classList.remove("spinning");
    }
  });

  document.getElementById("saveSettings")?.addEventListener("click", async () => {
    try {
      await saveSettings();
      showToast("Settings saved successfully", "success");
    } catch (e) {
      showToast(`Save failed: ${e.message}`, "error");
    }
  });

  document.getElementById("wizardSaveSettings")?.addEventListener("click", async () => {
    try {
      await saveWizardSettings();
      await boot();
      showToast("Settings saved successfully", "success");
    } catch (e) {
      showToast(`Save failed: ${e.message}`, "error");
    }
  });

  // Note: Local plugin installation removed - use git clone instead
  // See Settings > Plugin Installation for instructions

  document.getElementById("restoreBtn")?.addEventListener("click", async () => {
    try {
      await restoreBackup();
    } catch (e) {
      showToast(`Restore failed: ${e.message}`, "error");
    }
  });

  document.getElementById("saveLogSettings")?.addEventListener("click", async () => {
    try {
      const days = parseInt(document.getElementById("logRetentionDays").value) || 7;
      const current = await jsonFetch("/api/settings");
      await jsonFetch("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ ...current, log_retention_days: days }),
      });
      showToast("Log settings saved", "success");
    } catch (e) {
      showToast(`Save failed: ${e.message}`, "error");
    }
  });

  document.getElementById("cleanupLogs")?.addEventListener("click", async () => {
    try {
      const result = await jsonFetch("/api/logs/cleanup", { method: "DELETE" });
      showToast(`Logs cleaned up (retention: ${result.retention_days} days)`, "success");
    } catch (e) {
      showToast(`Cleanup failed: ${e.message}`, "error");
    }
  });

  document.getElementById("playgroundWorkflow")?.addEventListener("change", async (e) => {
    try {
      await loadPlaygroundWorkflow(e.target.value);
    } catch (err) {
      showToast(`Failed to load workflow: ${err.message}`, "error");
    }
  });

  const handleRunClick = async () => {
    try {
      await runPlayground();
    } catch (err) {
      showToast(`Run failed: ${err.message}`, "error");
      updateResultStatus("error", "Failed");
      if (playgroundLogs.length > 0 && playgroundLogs[0].status === "running") {
        playgroundLogs[0].status = "error";
        playgroundLogs[0].error = err.message;
        renderPlaygroundOutput();
      }
    }
  };

  document.getElementById("playgroundRun")?.addEventListener("click", handleRunClick);
  document.getElementById("playgroundRunWait")?.addEventListener("click", handleRunClick);
}

async function boot() {
  await loadHealth();
  const initialized = await loadSettings();
  
  const wizard = document.getElementById("wizard");
  const dashboard = document.getElementById("dashboard");
  
  if (wizard) wizard.classList.toggle("hidden", initialized);
  if (dashboard) dashboard.classList.toggle("hidden", !initialized);
  
  if (initialized) {
    initTabs();
    initFileUploadLabels();
    initInstallCommand();
    initImportModal();
    initLogsTab();
    await loadWorkflows();
    await populateLogsWorkflowFilter();
    
    // Start polling plugin status every 10 seconds
    setInterval(async () => {
      await loadPluginStatus();
    }, 10000);
  }
}

function init() {
  initEventListeners();
  boot();
  
  document.addEventListener("click", (e) => {
    if (!e.target.closest(".wf-copy-dropdown")) {
      document.querySelectorAll(".wf-dropdown-menu.show").forEach(d => d.classList.remove("show"));
    }
  });
}

// ========== Logs Tab ==========

let logsTabState = {
  workflowId: "",
  offset: 0,
  limit: 30,
  startDate: "",
  endDate: "",
  entries: [],
  total: 0,
  hasMore: false,
  loading: false
};

async function initLogsTab() {
  const applyBtn = document.getElementById("logsApplyFilter");
  const refreshBtn = document.getElementById("logsRefresh");
  const prevBtn = document.getElementById("logsPrev");
  const nextBtn = document.getElementById("logsNext");
  const workflowFilter = document.getElementById("logsWorkflowFilter");
  
  if (!applyBtn) return;
  
  applyBtn.addEventListener("click", () => {
    logsTabState.workflowId = document.getElementById("logsWorkflowFilter").value;
    logsTabState.startDate = document.getElementById("logsStartDate").value;
    logsTabState.endDate = document.getElementById("logsEndDate").value;
    logsTabState.offset = 0;
    fetchLogsTab();
  });
  
  refreshBtn.addEventListener("click", () => {
    logsTabState.offset = 0;
    fetchLogsTab();
  });
  
  prevBtn.addEventListener("click", () => {
    if (logsTabState.offset >= logsTabState.limit) {
      logsTabState.offset -= logsTabState.limit;
      fetchLogsTab();
    }
  });
  
  nextBtn.addEventListener("click", () => {
    if (logsTabState.hasMore) {
      logsTabState.offset += logsTabState.limit;
      fetchLogsTab();
    }
  });
  
  workflowFilter.addEventListener("change", () => {
    logsTabState.workflowId = workflowFilter.value;
    logsTabState.offset = 0;
    fetchLogsTab();
  });
}

async function populateLogsWorkflowFilter() {
  const select = document.getElementById("logsWorkflowFilter");
  if (!select) return;
  
  try {
    const list = await jsonFetch("/api/workflows");
    select.innerHTML = '<option value="">All Workflows</option>';
    for (const wf of list) {
      const opt = document.createElement("option");
      opt.value = wf.id;
      opt.textContent = wf.name || wf.id;
      select.appendChild(opt);
    }
  } catch (e) {
    console.warn("Failed to load workflows for filter:", e);
  }
}

async function fetchLogsTab() {
  if (logsTabState.loading) return;
  logsTabState.loading = true;
  
  const content = document.getElementById("logsContent");
  content.innerHTML = `
    <div class="logs-loading">
      <div class="loading-spinner"></div>
      <span>Loading logs...</span>
    </div>
  `;
  
  try {
    const params = new URLSearchParams();
    params.set("limit", logsTabState.limit);
    params.set("offset", logsTabState.offset);
    if (logsTabState.startDate) {
      params.set("start", new Date(logsTabState.startDate).toISOString());
    }
    if (logsTabState.endDate) {
      const end = new Date(logsTabState.endDate);
      end.setHours(23, 59, 59, 999);
      params.set("end", end.toISOString());
    }
    
    const url = logsTabState.workflowId 
      ? `/api/logs/${encodeURIComponent(logsTabState.workflowId)}?${params}`
      : `/api/logs?${params}`;
    
    const result = await jsonFetch(url);
    logsTabState.entries = result.entries || [];
    logsTabState.total = result.total || 0;
    logsTabState.hasMore = result.has_more || false;
    
    renderLogsTab();
  } catch (e) {
    content.innerHTML = `<div class="logs-error">Failed to load logs: ${escapeHtml(e.message)}</div>`;
  } finally {
    logsTabState.loading = false;
  }
}

function renderLogsTab() {
  const content = document.getElementById("logsContent");
  const stats = document.getElementById("logsStats");
  const prevBtn = document.getElementById("logsPrev");
  const nextBtn = document.getElementById("logsNext");
  
  const start = logsTabState.offset + 1;
  const end = Math.min(logsTabState.offset + logsTabState.entries.length, logsTabState.total);
  stats.textContent = logsTabState.total > 0 ? `${start}-${end} of ${logsTabState.total}` : "No logs";
  
  prevBtn.disabled = logsTabState.offset === 0;
  nextBtn.disabled = !logsTabState.hasMore;
  
  if (logsTabState.entries.length === 0) {
    content.innerHTML = `
      <div class="logs-empty">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
        </svg>
        <p>No logs found</p>
      </div>
    `;
    return;
  }
  
  const logsHtml = logsTabState.entries.map(entry => {
    const time = new Date(entry.timestamp).toLocaleString();
    const statusClass = entry.status === "success" ? "success" : "error";
    const statusIcon = entry.status === "success"
      ? '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>'
      : '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>';
    
    const sourceLabel = entry.source === "playground" ? "Playground" : "API";
    const paramsJson = entry.params ? JSON.stringify(entry.params, null, 2) : "{}";
    const curlCmd = generateCurlCommand(entry.workflow_id, entry.params || {});
    const cliCmd = generateCliCommand(entry.workflow_id, entry.params || {});
    
    return `
      <div class="log-row ${statusClass}">
        <div class="log-row-main">
          <div class="log-row-status ${statusClass}">${statusIcon}</div>
          <div class="log-row-content">
            <div class="log-row-top">
              <span class="log-row-workflow">${escapeHtml(entry.workflow_id)}</span>
              <span class="log-row-source">${sourceLabel}</span>
              ${entry.prompt_id ? `<code class="log-row-prompt">${escapeHtml(entry.prompt_id)}</code>` : ''}
            </div>
            <div class="log-row-bottom">
              <span class="log-row-time">${escapeHtml(time)}</span>
              <span class="log-row-duration">${entry.duration_ms || 0}ms</span>
            </div>
          </div>
        </div>
        ${entry.error ? `<div class="log-row-error">${escapeHtml(entry.error)}</div>` : ''}
        <details class="log-row-details">
          <summary>Request Details</summary>
          <div class="log-row-commands">
            <div class="log-cmd-block">
              <div class="log-cmd-header"><span>cURL</span><button class="btn-copy" onclick="copyToClipboard(\`${curlCmd.replace(/`/g, '\\`').replace(/\\/g, '\\\\')}\`)">Copy</button></div>
              <pre>${escapeHtml(curlCmd)}</pre>
            </div>
            <div class="log-cmd-block">
              <div class="log-cmd-header"><span>CLI</span><button class="btn-copy" onclick="copyToClipboard(\`${cliCmd.replace(/`/g, '\\`').replace(/\\/g, '\\\\')}\`)">Copy</button></div>
              <pre>${escapeHtml(cliCmd)}</pre>
            </div>
            <div class="log-cmd-block">
              <div class="log-cmd-header"><span>Parameters</span></div>
              <pre>${escapeHtml(paramsJson)}</pre>
            </div>
          </div>
        </details>
      </div>
    `;
  }).join('');
  
  content.innerHTML = `<div class="logs-list">${logsHtml}</div>`;
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", init);
} else {
  init();
}
