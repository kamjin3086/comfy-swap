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
      <strong>${escapeHtml(wf.name || wf.id)}</strong>
      <br><small>${wf.params_count} params · v${wf.version}</small>
    `;
    li.addEventListener("click", () => {
      document.querySelectorAll(".workflow-list li").forEach(el => el.classList.remove("active"));
      li.classList.add("active");
      loadWorkflowDetail(wf.id);
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
    const targets = (p.targets || []).map((t) => `${t.node_id}.${t.field}`).join(", ");
    const safeName = escapeHtml(p.name || "");
    const safeType = escapeHtml(p.type || "");
    const safeTargets = escapeHtml(targets);

    if (p.type === "image") {
      card.innerHTML = `
        <strong>${safeName}</strong>
        <small>Type: ${safeType}</small>
        <small>Targets: ${safeTargets}</small>
        <input type="file" data-param="${safeName}" accept="image/*" />
      `;
    } else {
      const safeValue = escapeHtml(String(p.default ?? ""));
      card.innerHTML = `
        <strong>${safeName}</strong>
        <small>Type: ${safeType}</small>
        <small>Targets: ${safeTargets}</small>
        <input data-param="${safeName}" value="${safeValue}" placeholder="Enter value..." />
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

async function runPlayground(wait) {
  const workflowId = document.getElementById("playgroundWorkflow").value;
  if (!workflowId) {
    showToast("Please select a workflow", "error");
    return;
  }
  const params = await collectPlaygroundParams();
  renderPlaygroundResult(`
    <div class="running-indicator">
      <div class="loading-spinner"></div>
      <span>Running workflow...</span>
    </div>
  `);

  const runResp = await jsonFetch("/api/prompt", {
    method: "POST",
    body: JSON.stringify({ workflow_id: workflowId, params }),
  });
  playgroundPromptId = runResp.prompt_id || "";

  if (!wait) {
    renderPlaygroundResult(`
      <div class="prompt-id">Prompt ID: ${escapeHtml(playgroundPromptId)}</div>
      <p>Run submitted. Click "Run and Wait" or use CLI/API history to fetch outputs.</p>
      <details>
        <summary>Raw Response</summary>
        <pre>${escapeHtml(JSON.stringify(runResp, null, 2))}</pre>
      </details>
    `);
    showToast("Workflow submitted", "success");
    return;
  }

  const started = Date.now();
  let history = {};
  while (true) {
    history = await jsonFetch(`/api/history/${encodeURIComponent(playgroundPromptId)}`);
    const entry = getHistoryEntry(history, playgroundPromptId);
    if (entry) break;
    if (Date.now() - started > 300000) {
      throw new Error("Timeout while waiting for generation.");
    }
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }

  const entry = getHistoryEntry(history, playgroundPromptId);
  const images = extractHistoryImages(entry);
  const imagesHtml =
    images.length === 0
      ? `<div class="empty-state">
           <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
             <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
             <circle cx="8.5" cy="8.5" r="1.5"/>
             <polyline points="21 15 16 10 5 21"/>
           </svg>
           <p>No output images found</p>
         </div>`
      : `<div class="result-images">${images
          .map(
            (img) => `
          <div>
            <img src="${imageURL(img)}" alt="${escapeHtml(img.filename)}" loading="lazy" />
            <small>${escapeHtml(img.filename)}</small>
          </div>
        `
          )
          .join("")}</div>`;

  renderPlaygroundResult(`
    <div class="prompt-id">Prompt ID: ${escapeHtml(playgroundPromptId)}</div>
    ${imagesHtml}
    <details>
      <summary>Raw History JSON</summary>
      <pre>${escapeHtml(JSON.stringify(history, null, 2))}</pre>
    </details>
  `);
  showToast("Generation completed", "success");
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
  const urlSpan = document.getElementById("installCommandUrl");
  if (urlSpan) {
    urlSpan.textContent = window.location.origin;
  }

  const copyBtn = document.getElementById("copyInstallCommand");
  const commandCode = document.getElementById("installCommand");
  if (copyBtn && commandCode) {
    copyBtn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(commandCode.textContent);
        copyBtn.classList.add("copied");
        copyBtn.innerHTML = `
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        `;
        setTimeout(() => {
          copyBtn.classList.remove("copied");
          copyBtn.innerHTML = `
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
            </svg>
          `;
        }, 2000);
        showToast("Command copied to clipboard", "success");
      } catch (e) {
        showToast("Failed to copy command", "error");
      }
    });
  }

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

  document.getElementById("wizardInstallPlugin")?.addEventListener("click", async () => {
    try {
      await installPluginFromWizard();
    } catch (e) {
      showToast(`Install failed: ${e.message}`, "error");
    }
  });

  document.getElementById("settingsInstallPlugin")?.addEventListener("click", async () => {
    try {
      await installPluginFromSettings();
    } catch (e) {
      showToast(`Install failed: ${e.message}`, "error");
    }
  });

  document.getElementById("restoreBtn")?.addEventListener("click", async () => {
    try {
      await restoreBackup();
    } catch (e) {
      showToast(`Restore failed: ${e.message}`, "error");
    }
  });

  document.getElementById("playgroundWorkflow")?.addEventListener("change", async (e) => {
    try {
      await loadPlaygroundWorkflow(e.target.value);
    } catch (err) {
      showToast(`Failed to load workflow: ${err.message}`, "error");
    }
  });

  document.getElementById("playgroundRun")?.addEventListener("click", async () => {
    try {
      await runPlayground(false);
    } catch (err) {
      showToast(`Run failed: ${err.message}`, "error");
      renderPlaygroundResult(`
        <div class="empty-state">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
          <p>Run failed: ${escapeHtml(err.message)}</p>
        </div>
      `);
    }
  });

  document.getElementById("playgroundRunWait")?.addEventListener("click", async () => {
    try {
      await runPlayground(true);
    } catch (err) {
      showToast(`Run failed: ${err.message}`, "error");
      renderPlaygroundResult(`
        <div class="empty-state">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
          <p>Run failed: ${escapeHtml(err.message)}</p>
        </div>
      `);
    }
  });
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
    await loadWorkflows();
    
    // Start polling plugin status every 10 seconds
    setInterval(async () => {
      await loadPluginStatus();
    }, 10000);
  }
}

function init() {
  initEventListeners();
  boot();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", init);
} else {
  init();
}
