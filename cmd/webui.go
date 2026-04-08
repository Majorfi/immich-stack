package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const settingsHTML = `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>immich-front-back — Settings</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script>tailwind.config = { darkMode: 'class' }</script>
  <style>
    body { background: #1a1a2e; }
    .card { background: #16213e; border: 1px solid #0f3460; }
    input, select { background: #0f3460; border: 1px solid #1a4a8a; color: #e2e8f0; }
    input:focus, select:focus { border-color: #4f8ef7; outline: none; }
    .btn-primary { background: #4f8ef7; }
    .btn-primary:hover { background: #3a7de0; }
    .btn-success { background: #22c55e; }
    .btn-success:hover { background: #16a34a; }
    .label { color: #94a3b8; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; }
    .toast { position: fixed; bottom: 1.5rem; right: 1.5rem; padding: 0.75rem 1.5rem; border-radius: 0.5rem; font-weight: 600; transition: opacity 0.3s; }
  </style>
</head>
<body class="text-slate-200 min-h-screen p-6">
  <div class="max-w-3xl mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="text-2xl font-bold text-white">immich-front-back</h1>
        <p class="text-slate-400 text-sm mt-1">Photo stack settings</p>
      </div>
      <div class="flex gap-3">
        <button onclick="runNow()" class="btn-success text-white font-semibold px-4 py-2 rounded-lg text-sm">
          &#9654; Run Now
        </button>
        <button onclick="saveSettings()" class="btn-primary text-white font-semibold px-4 py-2 rounded-lg text-sm">
          Save Settings
        </button>
      </div>
    </div>

    <!-- Status bar -->
    <div id="status-bar" class="card rounded-lg p-4 mb-6 text-sm flex gap-6">
      <div><span class="label">Last run</span><br><span id="last-run" class="text-white">&#8212;</span></div>
      <div><span class="label">Next run</span><br><span id="next-run" class="text-white">&#8212;</span></div>
      <div><span class="label">Mode</span><br><span id="run-mode" class="text-white">&#8212;</span></div>
      <div><span class="label">API Keys</span><br><span id="api-keys" class="text-white">&#8212;</span></div>
    </div>

    <!-- Connection (read-only) -->
    <div class="card rounded-lg p-5 mb-4">
      <h2 class="font-semibold text-slate-300 mb-4">Connection <span class="text-xs text-slate-500 font-normal">(set via env vars &#8212; restart required to change)</span></h2>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <div class="label mb-1">API URL</div>
          <div id="ro-api-url" class="text-slate-400 text-sm font-mono bg-slate-800 rounded px-3 py-2">&#8212;</div>
        </div>
        <div>
          <div class="label mb-1">API Keys</div>
          <div id="ro-api-keys" class="text-slate-400 text-sm font-mono bg-slate-800 rounded px-3 py-2">&#8212;</div>
        </div>
      </div>
    </div>

    <!-- Stacking -->
    <div class="card rounded-lg p-5 mb-4">
      <h2 class="font-semibold text-slate-300 mb-4">Stacking</h2>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="label block mb-1">Time Window (ms)</label>
          <input id="deltaMs" type="number" min="0" step="100" class="w-full rounded px-3 py-2 text-sm">
          <p class="text-slate-500 text-xs mt-1">Group photos taken within this window. Default: 5000</p>
        </div>
        <div>
          <label class="label block mb-1">Cron Interval (seconds)</label>
          <input id="cronInterval" type="number" min="60" class="w-full rounded px-3 py-2 text-sm">
          <p class="text-slate-500 text-xs mt-1">Seconds between auto-runs. Default: 3600</p>
        </div>
        <div>
          <label class="label block mb-1">Parent Filename Priority</label>
          <input id="parentFilenamePromote" type="text" class="w-full rounded px-3 py-2 text-sm">
          <p class="text-slate-500 text-xs mt-1">Comma-separated suffixes. Bare name first = stack cover</p>
        </div>
        <div>
          <label class="label block mb-1">Parent Extension Priority</label>
          <input id="parentExtPromote" type="text" class="w-full rounded px-3 py-2 text-sm">
          <p class="text-slate-500 text-xs mt-1">Comma-separated extensions in priority order</p>
        </div>
        <div class="col-span-2">
          <label class="label block mb-1">Criteria Override (JSON) <span class="text-slate-500 font-normal normal-case">&#8212; overrides time window if set</span></label>
          <textarea id="criteria" rows="3" class="w-full rounded px-3 py-2 text-sm font-mono bg-[#0f3460] border border-[#1a4a8a] text-slate-200 focus:border-[#4f8ef7] focus:outline-none resize-y"></textarea>
          <p class="text-slate-500 text-xs mt-1">Leave empty to use Time Window setting above</p>
        </div>
      </div>
    </div>

    <!-- Behaviour -->
    <div class="card rounded-lg p-5 mb-4">
      <h2 class="font-semibold text-slate-300 mb-4">Behaviour</h2>
      <div class="grid grid-cols-2 gap-3">
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="dryRun" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Dry Run</div><div class="text-slate-500 text-xs">Preview changes without applying</div></div>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="replaceStacks" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Replace Stacks</div><div class="text-slate-500 text-xs">Update stacks when group changes</div></div>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="withArchived" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Include Archived</div><div class="text-slate-500 text-xs">Process archived assets</div></div>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="withDeleted" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Include Deleted</div><div class="text-slate-500 text-xs">Process trashed assets</div></div>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="removeSingleAssetStacks" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Remove Single-asset Stacks</div><div class="text-slate-500 text-xs">Delete stacks with only one photo</div></div>
        </label>
      </div>
    </div>

    <!-- Filters -->
    <div class="card rounded-lg p-5 mb-4">
      <h2 class="font-semibold text-slate-300 mb-4">Filters</h2>
      <div class="grid grid-cols-2 gap-4">
        <div class="col-span-2">
          <label class="label block mb-1">Album IDs / Names</label>
          <input id="filterAlbumIDs" type="text" placeholder="album-id-1, Album Name, ..." class="w-full rounded px-3 py-2 text-sm">
        </div>
        <div>
          <label class="label block mb-1">Taken After (ISO 8601)</label>
          <input id="filterTakenAfter" type="text" placeholder="2024-01-01" class="w-full rounded px-3 py-2 text-sm">
        </div>
        <div>
          <label class="label block mb-1">Taken Before (ISO 8601)</label>
          <input id="filterTakenBefore" type="text" placeholder="2024-12-31" class="w-full rounded px-3 py-2 text-sm">
        </div>
      </div>
    </div>

    <!-- Logging -->
    <div class="card rounded-lg p-5 mb-4">
      <h2 class="font-semibold text-slate-300 mb-4">Logging</h2>
      <div class="w-40">
        <label class="label block mb-1">Log Level</label>
        <select id="logLevel" class="w-full rounded px-3 py-2 text-sm">
          <option value="debug">debug</option>
          <option value="info">info</option>
          <option value="warn">warn</option>
          <option value="error">error</option>
        </select>
      </div>
    </div>

    <!-- Metadata Sync -->
    <div class="card rounded-lg p-5 mb-4">
      <h2 class="font-semibold text-slate-300 mb-4">Metadata Sync <span class="text-xs text-slate-500 font-normal">&#8212; syncs primary&#8217;s metadata to sub-assets after each stacking run</span></h2>
      <div class="grid grid-cols-2 gap-3">
        <label class="flex items-center gap-3 cursor-pointer col-span-2">
          <input id="syncMetadataEnabled" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Enable Metadata Sync</div><div class="text-slate-500 text-xs">Master switch &#8212; must be on for any sync to run</div></div>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="syncDate" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Sync Date</div><div class="text-slate-500 text-xs">Apply primary&#8217;s date to sub-assets (preserves sub&#8217;s time)</div></div>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="syncTags" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Sync Tags</div><div class="text-slate-500 text-xs">Copy all primary tags to sub-assets</div></div>
        </label>
        <label class="flex items-center gap-3 cursor-pointer">
          <input id="syncPeople" type="checkbox" class="w-4 h-4 rounded accent-blue-500">
          <div><div class="text-sm font-medium">Sync People (best-effort)</div><div class="text-slate-500 text-xs">Assign unmatched faces using primary&#8217;s recognized people</div></div>
        </label>
      </div>
    </div>
  </div>

  <!-- Toast notification -->
  <div id="toast" class="toast opacity-0 pointer-events-none text-white text-sm"></div>

  <script>
    let statusTimer = null;

    async function loadSettings() {
      const res = await fetch('/api/settings');
      const d = await res.json();

      // Read-only status
      document.getElementById('ro-api-url').textContent = d.apiURL || '—';
      document.getElementById('ro-api-keys').textContent = d.apiKeyCount ? d.apiKeyCount + ' key(s) configured' : '—';
      document.getElementById('run-mode').textContent = d.runMode || '—';
      document.getElementById('api-keys').textContent = d.apiKeyCount ? d.apiKeyCount + ' configured' : '—';
      updateNextRun(d.nextRunIn);
      document.getElementById('last-run').textContent = d.lastRun || 'Never';

      // Editable fields
      document.getElementById('deltaMs').value = d.deltaMs != null ? d.deltaMs : 5000;
      document.getElementById('cronInterval').value = d.cronInterval != null ? d.cronInterval : 3600;
      document.getElementById('parentFilenamePromote').value = d.parentFilenamePromote != null ? d.parentFilenamePromote : ',a,b';
      document.getElementById('parentExtPromote').value = d.parentExtPromote != null ? d.parentExtPromote : '.jpg,.png,.jpeg,.heic,.dng';
      document.getElementById('criteria').value = d.criteria != null ? d.criteria : '';
      document.getElementById('dryRun').checked = !!d.dryRun;
      document.getElementById('replaceStacks').checked = d.replaceStacks !== false;
      document.getElementById('withArchived').checked = !!d.withArchived;
      document.getElementById('withDeleted').checked = !!d.withDeleted;
      document.getElementById('removeSingleAssetStacks').checked = !!d.removeSingleAssetStacks;
      document.getElementById('filterAlbumIDs').value = d.filterAlbumIDs != null ? d.filterAlbumIDs : '';
      document.getElementById('filterTakenAfter').value = d.filterTakenAfter != null ? d.filterTakenAfter : '';
      document.getElementById('filterTakenBefore').value = d.filterTakenBefore != null ? d.filterTakenBefore : '';
      document.getElementById('logLevel').value = d.logLevel != null ? d.logLevel : 'info';
      document.getElementById('syncMetadataEnabled').checked = !!d.syncMetadataEnabled;
      document.getElementById('syncDate').checked = !!d.syncDate;
      document.getElementById('syncTags').checked = !!d.syncTags;
      document.getElementById('syncPeople').checked = !!d.syncPeople;
    }

    function updateNextRun(seconds) {
      if (seconds == null || seconds < 0) {
        document.getElementById('next-run').textContent = '—';
        return;
      }
      if (seconds === 0) {
        document.getElementById('next-run').textContent = 'Running…';
        return;
      }
      const h = Math.floor(seconds / 3600);
      const m = Math.floor((seconds % 3600) / 60);
      const s = seconds % 60;
      const parts = [];
      if (h > 0) parts.push(h + 'h');
      if (m > 0 || h > 0) parts.push(m + 'm');
      parts.push(s + 's');
      document.getElementById('next-run').textContent = 'in ' + parts.join(' ');
    }

    async function saveSettings() {
      const payload = {
        deltaMs: parseInt(document.getElementById('deltaMs').value) || 5000,
        cronInterval: parseInt(document.getElementById('cronInterval').value) || 3600,
        parentFilenamePromote: document.getElementById('parentFilenamePromote').value,
        parentExtPromote: document.getElementById('parentExtPromote').value,
        criteria: document.getElementById('criteria').value,
        dryRun: document.getElementById('dryRun').checked,
        replaceStacks: document.getElementById('replaceStacks').checked,
        withArchived: document.getElementById('withArchived').checked,
        withDeleted: document.getElementById('withDeleted').checked,
        removeSingleAssetStacks: document.getElementById('removeSingleAssetStacks').checked,
        filterAlbumIDs: document.getElementById('filterAlbumIDs').value,
        filterTakenAfter: document.getElementById('filterTakenAfter').value,
        filterTakenBefore: document.getElementById('filterTakenBefore').value,
        logLevel: document.getElementById('logLevel').value,
        syncMetadataEnabled: document.getElementById('syncMetadataEnabled').checked,
        syncDate: document.getElementById('syncDate').checked,
        syncTags: document.getElementById('syncTags').checked,
        syncPeople: document.getElementById('syncPeople').checked,
      };
      const res = await fetch('/api/settings', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(payload) });
      if (res.ok) showToast('Settings saved ✓', '#22c55e');
      else showToast('Save failed ✗', '#ef4444');
    }

    async function runNow() {
      const res = await fetch('/api/run', { method: 'POST' });
      if (res.ok) showToast('Run triggered ✓', '#4f8ef7');
      else showToast('Failed to trigger run', '#ef4444');
    }

    function showToast(msg, color) {
      const t = document.getElementById('toast');
      t.textContent = msg;
      t.style.background = color;
      t.style.opacity = '1';
      setTimeout(function() { t.style.opacity = '0'; }, 2500);
    }

    // Poll status every 5 seconds
    async function pollStatus() {
      try {
        const res = await fetch('/api/status');
        const d = await res.json();
        updateNextRun(d.nextRunIn);
        if (d.lastRun) document.getElementById('last-run').textContent = d.lastRun;
      } catch(e) {}
    }

    loadSettings();
    setInterval(pollStatus, 5000);
  </script>
</body>
</html>`

// maskedAPIURL returns just the host portion of the URL to avoid leaking path details.
func maskedAPIURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}

// humanDuration returns a human-readable "X ago" string for a past time.
func humanDuration(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		return t.Format("2006-01-02 15:04")
	}
}

func startWebUI(port int) {
	mux := http.NewServeMux()

	// CORS middleware wrapper
	withCORS := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			h(w, r)
		}
	}

	// GET / — serve HTML UI
	mux.HandleFunc("/", withCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, settingsHTML)
	}))

	// GET /api/settings — return current settings + read-only fields
	mux.HandleFunc("/api/settings", withCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			s := loadAppSettings()

			// Count API keys
			keys := strings.Split(os.Getenv("API_KEY"), ",")
			keyCount := 0
			for _, k := range keys {
				if strings.TrimSpace(k) != "" {
					keyCount++
				}
			}

			statusMu.RLock()
			nrt := nextRunTime
			lrt := lastRunTime
			statusMu.RUnlock()

			var nextRunIn *int
			if !nrt.IsZero() {
				secs := int(time.Until(nrt).Seconds())
				if secs < 0 {
					secs = 0
				}
				nextRunIn = &secs
			}

			resp := map[string]interface{}{
				// editable settings
				"deltaMs":               s.DeltaMs,
				"parentFilenamePromote": s.ParentFilenamePromote,
				"parentExtPromote":      s.ParentExtPromote,
				"criteria":              s.Criteria,
				"cronInterval":          s.CronInterval,
				"dryRun":                s.DryRun,
				"replaceStacks":         s.ReplaceStacks,
				"removeSingleAssetStacks": s.RemoveSingleAssetStacks,
				"withArchived":          s.WithArchived,
				"withDeleted":           s.WithDeleted,
				"filterAlbumIDs":        s.FilterAlbumIDs,
				"filterTakenAfter":      s.FilterTakenAfter,
				"filterTakenBefore":     s.FilterTakenBefore,
				"logLevel":              s.LogLevel,
				"syncMetadataEnabled":   s.SyncMetadataEnabled,
				"syncDate":              s.SyncDate,
				"syncTags":              s.SyncTags,
				"syncPeople":            s.SyncPeople,
				// read-only
				"apiURL":      maskedAPIURL(os.Getenv("API_URL")),
				"apiKeyCount": keyCount,
				"runMode":     os.Getenv("RUN_MODE"),
				"lastRun":     humanDuration(lrt),
				"nextRunIn":   nextRunIn,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.Method == http.MethodPost {
			var s AppSettings
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := saveAppSettings(s); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"ok":true}`)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))

	// POST /api/run — trigger immediate run
	mux.HandleFunc("/api/run", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		select {
		case manualRunCh <- struct{}{}:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"ok":true}`)
		default:
			// Channel full — run already queued
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"ok":true,"queued":true}`)
		}
	}))

	// GET /api/status — return status snapshot
	mux.HandleFunc("/api/status", withCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		statusMu.RLock()
		nrt := nextRunTime
		lrt := lastRunTime
		statusMu.RUnlock()

		var nextRunIn *int
		if !nrt.IsZero() {
			secs := int(time.Until(nrt).Seconds())
			if secs < 0 {
				secs = 0
			}
			nextRunIn = &secs
		}

		resp := map[string]interface{}{
			"lastRun":   humanDuration(lrt),
			"nextRunIn": nextRunIn,
		}
		json.NewEncoder(w).Encode(resp)
	}))

	addr := fmt.Sprintf(":%d", port)
	http.ListenAndServe(addr, mux)
}
