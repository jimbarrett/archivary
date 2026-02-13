<template>
  <div class="sync-view">
    <div class="sync-header">
      <h1>Git Sync</h1>
      <div class="header-actions">
        <button
          v-if="remotes.length"
          class="btn btn-secondary"
          @click="onSyncAll"
          :disabled="syncing"
        >
          {{ syncing ? 'Syncing...' : 'Sync All' }}
        </button>
        <button class="btn btn-primary" @click="showAddDialog = true">
          Add Remote
        </button>
      </div>
    </div>

    <p class="description">
      Sync workspace directories to git remotes. Each synced directory is an independent git repository.
    </p>

    <div v-if="!remotes.length" class="empty-state">
      <p>No synced directories configured.</p>
      <p class="hint">Click "Add Remote" to sync a directory to a git repository, or clone an existing one.</p>
    </div>

    <div v-else class="remote-list">
      <div v-for="remote in remotes" :key="remote.path" class="remote-card">
        <div class="remote-header">
          <div class="remote-info">
            <h2 class="remote-path">{{ remote.path }}</h2>
            <span class="remote-url" v-if="remote.url">{{ remote.url }}</span>
            <span class="remote-url local" v-else>Local only (no remote)</span>
          </div>
          <div class="remote-status">
            <span class="status-badge" :class="statusClass(remote.path)">
              {{ statusLabel(remote.path) }}
            </span>
          </div>
        </div>

        <div class="remote-details">
          <div class="detail-row">
            <span class="detail-label">Branch</span>
            <span class="detail-value">{{ remote.branch || 'main' }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Auto-commit</span>
            <span class="detail-value">{{ remote.auto_commit ? 'On' : 'Off' }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Auto-push</span>
            <span class="detail-value">
              {{ remote.auto_push ? `Every ${remote.push_interval_minutes} min` : 'Off' }}
            </span>
          </div>
        </div>

        <div v-if="statusError(remote.path)" class="error-banner">
          {{ statusError(remote.path) }}
        </div>

        <div class="remote-actions">
          <button class="btn btn-small" @click="onSyncOne(remote.path)" :disabled="syncing">
            Sync Now
          </button>
          <button class="btn btn-small" @click="onShowLog(remote.path)">
            View Log
          </button>
          <button class="btn btn-small" @click="editRemote = remote">
            Settings
          </button>
          <button class="btn btn-small btn-danger-text" @click="onRemove(remote)">
            Unsync
          </button>
        </div>

        <div v-if="logPath === remote.path && logEntries.length" class="log-section">
          <h3 class="log-title">Recent Commits</h3>
          <div v-for="entry in logEntries" :key="entry.hash" class="log-entry">
            <span class="log-hash">{{ entry.hash.substring(0, 7) }}</span>
            <span class="log-message">{{ entry.message }}</span>
            <span class="log-meta">{{ entry.author }} &middot; {{ formatTime(entry.time) }}</span>
          </div>
        </div>
        <div v-if="logPath === remote.path && !logEntries.length && !logLoading" class="log-section">
          <p class="log-empty">No commits yet.</p>
        </div>
      </div>
    </div>

    <SyncSettingsDialog
      v-if="showAddDialog"
      @close="showAddDialog = false"
      @saved="onRemoteSaved"
    />

    <SyncSettingsDialog
      v-if="editRemote"
      :remote="editRemote"
      @close="editRemote = null"
      @saved="onRemoteSaved"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getSyncLog, removeRemote } from '../lib/api.js'
import { syncStatus, syncRemotes, syncing, refreshSyncStatus, triggerSyncAll, triggerSyncDir } from '../lib/sync.js'
import { refreshSidebarTree } from '../lib/events.js'
import SyncSettingsDialog from '../components/SyncSettingsDialog.vue'

const remotes = syncRemotes
const showAddDialog = ref(false)
const editRemote = ref(null)
const logPath = ref(null)
const logEntries = ref([])
const logLoading = ref(false)

onMounted(() => {
  refreshSyncStatus()
})

function statusClass(path) {
  const s = syncStatus.value[path]
  if (!s) return 'status-unknown'
  if (s.error) return 'status-error'
  if (!s.clean) return 'status-pending'
  if (s.ahead > 0) return 'status-ahead'
  return 'status-clean'
}

function statusLabel(path) {
  const s = syncStatus.value[path]
  if (!s) return 'Unknown'
  if (s.error) return 'Error'
  if (!s.clean) return 'Changes'
  if (s.ahead > 0) return `${s.ahead} to push`
  if (s.behind > 0) return `${s.behind} to pull`
  return 'Clean'
}

function statusError(path) {
  const s = syncStatus.value[path]
  return s?.error || ''
}

async function onSyncAll() {
  try {
    await triggerSyncAll()
    refreshSidebarTree()
  } catch (e) {
    alert('Sync failed: ' + e.message)
  }
}

async function onSyncOne(path) {
  try {
    await triggerSyncDir(path)
    refreshSidebarTree()
    // Refresh log if it's open
    if (logPath.value === path) {
      await loadLog(path)
    }
  } catch (e) {
    alert('Sync failed: ' + e.message)
  }
}

async function onShowLog(path) {
  if (logPath.value === path) {
    logPath.value = null
    logEntries.value = []
    return
  }
  await loadLog(path)
}

async function loadLog(path) {
  logPath.value = path
  logLoading.value = true
  try {
    logEntries.value = await getSyncLog(path, 15)
  } catch (e) {
    logEntries.value = []
  } finally {
    logLoading.value = false
  }
}

async function onRemove(remote) {
  if (!confirm(`Unsync "${remote.path}"? The .git directory will be removed but your files will remain.`)) return
  try {
    await removeRemote(remote.path)
    await refreshSyncStatus()
    refreshSidebarTree()
    if (logPath.value === remote.path) {
      logPath.value = null
      logEntries.value = []
    }
  } catch (e) {
    alert('Remove failed: ' + e.message)
  }
}

async function onRemoteSaved() {
  await refreshSyncStatus()
  refreshSidebarTree()
}

function formatTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const diff = now - d
  if (diff < 60000) return 'just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  return d.toLocaleDateString()
}
</script>

<style scoped>
.sync-view {
  max-width: 800px;
}

.sync-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.sync-header h1 {
  font-size: 1.5rem;
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.description {
  color: var(--text-secondary);
  font-size: 0.9rem;
  margin-bottom: 1.5rem;
}

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--text-secondary);
}

.empty-state .hint {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin-top: 0.5rem;
}

.remote-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.remote-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1rem;
}

.remote-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.75rem;
}

.remote-path {
  font-size: 1.1rem;
  font-weight: 600;
  margin: 0;
}

.remote-url {
  font-size: 0.8rem;
  color: var(--text-muted);
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
}

.remote-url.local {
  font-family: inherit;
  font-style: italic;
}

.status-badge {
  font-size: 0.72rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 10px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.status-clean {
  background: rgba(158, 206, 106, 0.15);
  color: var(--success);
}

.status-pending {
  background: rgba(224, 175, 104, 0.15);
  color: var(--warning);
}

.status-ahead {
  background: rgba(122, 162, 247, 0.15);
  color: var(--accent);
}

.status-error {
  background: rgba(247, 118, 142, 0.15);
  color: var(--error);
}

.status-unknown {
  background: var(--bg-hover);
  color: var(--text-muted);
}

.remote-details {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.detail-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-muted);
  display: block;
}

.detail-value {
  font-size: 0.85rem;
  color: var(--text-secondary);
  display: block;
}

.error-banner {
  background: rgba(255, 80, 80, 0.1);
  border: 1px solid var(--error);
  border-radius: 4px;
  padding: 0.4rem 0.6rem;
  color: var(--error);
  font-size: 0.8rem;
  margin-bottom: 0.75rem;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
}

.remote-actions {
  display: flex;
  gap: 0.4rem;
}

.btn {
  padding: 0.3rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 0.8rem;
  cursor: pointer;
  text-decoration: none;
  display: inline-block;
  background: var(--bg-input);
  color: var(--text-primary);
}

.btn:hover {
  background: var(--bg-hover);
}

.btn-primary {
  background: var(--accent-dim);
  border-color: var(--accent);
}

.btn-primary:hover {
  background: var(--accent);
  color: var(--bg-primary);
}

.btn-secondary {
  background: var(--bg-input);
  color: var(--text-primary);
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-small {
  padding: 0.2rem 0.6rem;
  font-size: 0.75rem;
}

.btn-danger-text {
  color: var(--error);
  border-color: transparent;
  background: none;
}

.btn-danger-text:hover {
  background: rgba(247, 118, 142, 0.1);
}

.log-section {
  margin-top: 0.75rem;
  border-top: 1px solid var(--border);
  padding-top: 0.75rem;
}

.log-title {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
}

.log-entry {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  padding: 0.2rem 0;
  font-size: 0.8rem;
}

.log-hash {
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  color: var(--accent);
  flex-shrink: 0;
}

.log-message {
  color: var(--text-primary);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-meta {
  color: var(--text-muted);
  font-size: 0.72rem;
  flex-shrink: 0;
}

.log-empty {
  font-size: 0.8rem;
  color: var(--text-muted);
  font-style: italic;
}
</style>
