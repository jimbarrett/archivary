<template>
  <div class="sync-view">
    <div class="sync-header">
      <h1>Git Sync</h1>
    </div>

    <!-- Not synced: empty state -->
    <template v-if="!isRootSynced">
      <p class="description">
        Sync your workspace to a git remote. All directories and pages will be tracked unless you explicitly exclude them.
      </p>
      <div class="empty-state">
        <p>Workspace sync is not configured.</p>
        <button class="btn btn-primary" @click="showSetup = true">Setup Workspace Sync</button>
      </div>
    </template>

    <!-- Synced: workspace card -->
    <template v-else>
      <div class="remote-card">
        <div class="remote-header">
          <div class="remote-info">
            <h2 class="remote-path">Workspace</h2>
            <span class="remote-url" v-if="rootRemote?.url">{{ rootRemote.url }}</span>
            <span class="remote-url local" v-else>Local only (no remote)</span>
          </div>
          <div class="remote-status">
            <span class="status-badge" :class="statusClass">
              {{ statusLabel }}
            </span>
          </div>
        </div>

        <div class="remote-details">
          <div class="detail-row">
            <span class="detail-label">Branch</span>
            <span class="detail-value">{{ rootRemote?.branch || 'main' }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Auto-commit</span>
            <span class="detail-value">{{ rootRemote?.auto_commit ? 'On' : 'Off' }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Auto-push</span>
            <span class="detail-value">
              {{ rootRemote?.auto_push ? `Every ${rootRemote.push_interval_minutes} min` : 'Off' }}
            </span>
          </div>
        </div>

        <div v-if="statusError" class="error-banner">
          {{ statusError }}
        </div>

        <div class="remote-actions">
          <button class="btn btn-small" @click="onSyncNow" :disabled="syncing">
            {{ syncing ? 'Syncing...' : 'Sync Now' }}
          </button>
          <button class="btn btn-small" @click="showSettings = true">
            Settings
          </button>
          <button class="btn btn-small btn-danger-text" @click="onUnsync">
            Unsync
          </button>
        </div>
      </div>

      <!-- Recent commits -->
      <div v-if="logEntries.length" class="section">
        <h2 class="section-title">Recent Commits</h2>
        <div v-for="entry in logEntries" :key="entry.hash" class="log-entry">
          <span class="log-hash">{{ entry.hash.substring(0, 7) }}</span>
          <span class="log-message">{{ entry.message }}</span>
          <span class="log-meta">{{ entry.author }} &middot; {{ formatTime(entry.time) }}</span>
        </div>
      </div>

      <!-- Excluded directories -->
      <div v-if="excludedDirs.length" class="section">
        <h2 class="section-title">Excluded Directories</h2>
        <div class="excluded-list">
          <div v-for="dir in excludedDirs" :key="dir" class="excluded-item">
            <span class="excluded-name">{{ dir }}/</span>
            <button class="btn btn-small" @click="onInclude(dir)">Include</button>
          </div>
        </div>
      </div>
    </template>

    <SyncSettingsDialog
      v-if="showSetup"
      @close="showSetup = false"
      @saved="onSaved"
    />

    <SyncSettingsDialog
      v-if="showSettings && rootRemote"
      :remote="rootRemote"
      @close="showSettings = false"
      @saved="onSaved"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { getSyncLog, removeRemote, includeDir } from '../lib/api.js'
import { syncStatus, syncing, refreshSyncStatus, triggerSyncAll, isRootSynced, rootRemote, excludedDirs } from '../lib/sync.js'
import { refreshSidebarTree } from '../lib/events.js'
import SyncSettingsDialog from '../components/SyncSettingsDialog.vue'

const showSetup = ref(false)
const showSettings = ref(false)
const logEntries = ref([])

const statusClass = computed(() => {
  const s = syncStatus.value['.']
  if (!s) return 'status-unknown'
  if (s.error) return 'status-error'
  if (!s.clean) return 'status-pending'
  if (s.ahead > 0) return 'status-ahead'
  return 'status-clean'
})

const statusLabel = computed(() => {
  const s = syncStatus.value['.']
  if (!s) return 'Unknown'
  if (s.error) return 'Error'
  if (!s.clean) return 'Changes'
  if (s.ahead > 0) return `${s.ahead} to push`
  if (s.behind > 0) return `${s.behind} to pull`
  return 'Clean'
})

const statusError = computed(() => {
  const s = syncStatus.value['.']
  return s?.error || ''
})

onMounted(async () => {
  await refreshSyncStatus()
  if (isRootSynced.value) {
    await loadLog()
  }
})

watch(isRootSynced, async (val) => {
  if (val) await loadLog()
  else logEntries.value = []
})

async function loadLog() {
  try {
    logEntries.value = await getSyncLog('.', 15)
  } catch (e) {
    logEntries.value = []
  }
}

async function onSyncNow() {
  try {
    await triggerSyncAll()
    refreshSidebarTree()
    await loadLog()
  } catch (e) {
    alert('Sync failed: ' + e.message)
  }
}

async function onUnsync() {
  if (!confirm('Unsync the workspace? The .git directory will be removed but your files will remain.')) return
  try {
    await removeRemote('.')
    await refreshSyncStatus()
    refreshSidebarTree()
    logEntries.value = []
  } catch (e) {
    alert('Unsync failed: ' + e.message)
  }
}

async function onInclude(dir) {
  try {
    await includeDir(dir)
    await refreshSyncStatus()
    await loadLog()
  } catch (e) {
    alert('Include failed: ' + e.message)
  }
}

async function onSaved() {
  await refreshSyncStatus()
  refreshSidebarTree()
  await loadLog()
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
  margin-bottom: 0.5rem;
}

.sync-header h1 {
  font-size: 1.5rem;
  font-weight: 600;
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

.empty-state p {
  margin-bottom: 1rem;
}

.remote-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1.5rem;
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

.section {
  margin-bottom: 1.5rem;
}

.section-title {
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

.excluded-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.excluded-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.4rem 0.6rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 4px;
}

.excluded-name {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
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
</style>
