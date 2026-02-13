<template>
  <div class="directory-view" v-if="dirNode">
    <nav class="breadcrumb">
      <router-link to="/" class="breadcrumb-link">Home</router-link>
      <span v-for="(crumb, i) in breadcrumbs" :key="crumb.path">
        <span class="breadcrumb-sep">/</span>
        <router-link
          v-if="i < breadcrumbs.length - 1"
          :to="{ name: 'directory', params: { path: crumb.path } }"
          class="breadcrumb-link"
        >{{ crumb.name }}</router-link>
        <span v-else class="breadcrumb-current">{{ crumb.name }}</span>
      </span>
    </nav>

    <div class="dir-header">
      <div class="dir-header-left">
        <h1 v-if="!renaming">{{ dirNode.name }}</h1>
        <input
          v-else
          ref="renameInput"
          v-model="newName"
          type="text"
          class="rename-input"
          @keydown.enter="submitRename"
          @keydown.escape="cancelRename"
        />
        <span v-if="dirSync" class="sync-indicator" :class="syncClass">
          {{ syncLabel }}
        </span>
      </div>
      <div class="dir-actions">
        <template v-if="dirSync">
          <button class="btn btn-secondary" @click="onSyncDir" :disabled="syncingDir">
            {{ syncingDir ? 'Syncing...' : 'Sync Now' }}
          </button>
          <button class="btn btn-secondary" @click="showSyncEdit = true">Sync Settings</button>
          <button class="btn btn-secondary btn-stop" @click="onStopSync">Unsync</button>
        </template>
        <template v-else-if="isTopLevel">
          <button class="btn btn-secondary" @click="showSyncSetup = true">Setup Sync</button>
        </template>
        <button v-if="!renaming" class="btn btn-secondary" @click="startRename">Rename</button>
        <template v-else>
          <button class="btn btn-primary" @click="submitRename" :disabled="!newName.trim()">Save</button>
          <button class="btn btn-secondary" @click="cancelRename">Cancel</button>
        </template>
        <button class="btn btn-danger" @click="confirmDelete">Delete</button>
      </div>
    </div>

    <div v-if="error" class="error-msg">{{ error }}</div>

    <div v-if="dirSync && dirSync.error" class="error-msg">
      Sync error: {{ dirSync.error }}
    </div>

    <div v-if="dirSync && !dirSync.clean" class="commit-bar">
      <input
        v-model="commitMsg"
        type="text"
        class="commit-input"
        placeholder="Commit message (optional)"
        @keydown.enter="onCommit"
        @keydown.escape="commitMsg = ''"
      />
      <button class="btn btn-primary" @click="onCommit" :disabled="committing">
        {{ committing ? 'Committing...' : 'Commit' }}
      </button>
    </div>

    <div v-if="subdirs.length" class="section">
      <h2 class="section-title">Directories</h2>
      <div class="item-list">
        <router-link
          v-for="dir in subdirs"
          :key="dir.path"
          :to="{ name: 'directory', params: { path: dir.path } }"
          class="item dir-item"
        >
          <span class="item-icon">&#9654;</span>
          <span class="item-name">{{ dir.name }}</span>
        </router-link>
      </div>
    </div>

    <div v-if="pages.length" class="section">
      <h2 class="section-title">Pages</h2>
      <div class="item-list">
        <router-link
          v-for="page in pages"
          :key="page.page_id"
          :to="{ name: 'page', params: { id: page.page_id } }"
          class="item page-item"
        >
          <span class="item-icon">&#9643;</span>
          <span class="item-name">{{ page.name }}</span>
          <span class="item-path">{{ page.path }}</span>
        </router-link>
      </div>
    </div>

    <div v-if="!subdirs.length && !pages.length" class="empty-state">
      This directory is empty.
    </div>

    <div v-if="dirSync && logEntries.length" class="section log-section">
      <h2 class="section-title">Recent Commits</h2>
      <div v-for="entry in logEntries" :key="entry.hash" class="log-entry">
        <span class="log-hash">{{ entry.hash.substring(0, 7) }}</span>
        <span class="log-message">{{ entry.message }}</span>
        <span class="log-meta">{{ formatTime(entry.time) }}</span>
      </div>
    </div>

    <SyncSettingsDialog
      v-if="showSyncSetup"
      :default-path="path"
      @close="showSyncSetup = false"
      @saved="onSyncSettingsSaved"
    />

    <SyncSettingsDialog
      v-if="showSyncEdit"
      :remote="dirSyncRemote"
      @close="showSyncEdit = false"
      @saved="onSyncSettingsSaved"
    />
  </div>

  <div v-else-if="loading" class="loading">Loading...</div>

  <div v-else class="error-state">
    <h2>Directory not found</h2>
    <p>{{ path }}</p>
    <router-link to="/" class="home-link">Go home</router-link>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getTree, renameDir, deleteDir, removeRemote, getSyncLog, syncCommit } from '../lib/api.js'
import { refreshSidebarTree } from '../lib/events.js'
import { syncStatus, syncRemotes, refreshSyncStatus, triggerSyncDir } from '../lib/sync.js'
import SyncSettingsDialog from '../components/SyncSettingsDialog.vue'

const router = useRouter()

const props = defineProps({
  path: { type: String, required: true },
})

const tree = ref(null)
const loading = ref(true)
const error = ref(null)
const renaming = ref(false)
const newName = ref('')
const renameInput = ref(null)
const syncingDir = ref(false)
const showSyncSetup = ref(false)
const showSyncEdit = ref(false)
const logEntries = ref([])
const commitMsg = ref('')
const committing = ref(false)

const dirNode = computed(() => {
  if (!tree.value) return null
  return findDir(tree.value, props.path)
})

const breadcrumbs = computed(() => {
  if (!props.path) return []
  const parts = props.path.split('/')
  return parts.map((name, i) => ({
    name,
    path: parts.slice(0, i + 1).join('/'),
  }))
})

const subdirs = computed(() => {
  if (!dirNode.value || !dirNode.value.children) return []
  return [...dirNode.value.children]
    .filter(c => c.is_dir)
    .sort((a, b) => a.name.localeCompare(b.name))
})

const pages = computed(() => {
  if (!dirNode.value || !dirNode.value.children) return []
  return [...dirNode.value.children]
    .filter(c => !c.is_dir)
    .sort((a, b) => a.name.localeCompare(b.name))
})

// Is this a top-level directory (can be synced)?
const isTopLevel = computed(() => {
  return props.path && !props.path.includes('/')
})

// Sync status for this directory
const dirSync = computed(() => {
  return syncStatus.value[props.path] || null
})

// Full remote config for this directory
const dirSyncRemote = computed(() => {
  return syncRemotes.value.find(r => r.path === props.path) || null
})

const syncClass = computed(() => {
  const s = dirSync.value
  if (!s) return ''
  if (s.error) return 'sync-error'
  if (!s.clean) return 'sync-pending'
  return 'sync-clean'
})

const syncLabel = computed(() => {
  const s = dirSync.value
  if (!s) return ''
  if (s.error) return 'Sync Error'
  if (!s.clean) return 'Uncommitted Changes'
  if (s.ahead > 0) return `${s.ahead} to push`
  return 'Synced'
})

function findDir(node, targetPath) {
  if (node.path === targetPath && node.is_dir) return node
  if (!node.children) return null
  for (const child of node.children) {
    const found = findDir(child, targetPath)
    if (found) return found
  }
  return null
}

async function loadTree() {
  loading.value = true
  error.value = null
  try {
    tree.value = await getTree()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function loadLog() {
  if (!dirSync.value) {
    logEntries.value = []
    return
  }
  try {
    logEntries.value = await getSyncLog(props.path, 5)
  } catch (e) {
    logEntries.value = []
  }
}

watch(() => props.path, async () => {
  renaming.value = false
  error.value = null
  showSyncSetup.value = false
  showSyncEdit.value = false
  await loadTree()
  await loadLog()
}, { immediate: true })

// Reload log when sync status changes
watch(syncStatus, () => {
  if (dirSync.value) loadLog()
})

function startRename() {
  newName.value = dirNode.value.name
  renaming.value = true
  nextTick(() => {
    if (renameInput.value) {
      renameInput.value.focus()
      renameInput.value.select()
    }
  })
}

function cancelRename() {
  renaming.value = false
  newName.value = ''
}

async function submitRename() {
  const name = newName.value.trim()
  if (!name || name === dirNode.value.name) {
    cancelRename()
    return
  }
  error.value = null
  try {
    await renameDir(props.path, name)
    refreshSidebarTree()
    const parentDir = props.path.includes('/') ? props.path.substring(0, props.path.lastIndexOf('/')) : ''
    const newPath = parentDir ? `${parentDir}/${name}` : name
    renaming.value = false
    router.replace({ name: 'directory', params: { path: newPath } })
  } catch (e) {
    error.value = e.message
  }
}

async function confirmDelete() {
  if (!confirm(`Delete directory "${dirNode.value.name}"? This only works for empty directories.`)) return
  error.value = null
  try {
    await deleteDir(props.path)
    refreshSidebarTree()
    const parentDir = props.path.includes('/') ? props.path.substring(0, props.path.lastIndexOf('/')) : ''
    if (parentDir) {
      router.push({ name: 'directory', params: { path: parentDir } })
    } else {
      router.push({ name: 'home' })
    }
  } catch (e) {
    error.value = e.message
  }
}

async function onSyncDir() {
  syncingDir.value = true
  try {
    await triggerSyncDir(props.path)
    refreshSidebarTree()
    await loadLog()
  } catch (e) {
    error.value = 'Sync failed: ' + e.message
  } finally {
    syncingDir.value = false
  }
}

async function onStopSync() {
  if (!confirm(`Unsync "${props.path}"? The .git directory will be removed but your files will remain.`)) return
  try {
    await removeRemote(props.path)
    await refreshSyncStatus()
    refreshSidebarTree()
    logEntries.value = []
    await loadTree()
  } catch (e) {
    error.value = e.message
  }
}

async function onSyncSettingsSaved() {
  await refreshSyncStatus()
  refreshSidebarTree()
  await loadTree()
  await loadLog()
}

async function onCommit() {
  committing.value = true
  error.value = null
  try {
    await syncCommit(props.path, commitMsg.value.trim() || 'manual commit')
    commitMsg.value = ''
    await refreshSyncStatus()
    await loadLog()
  } catch (e) {
    error.value = 'Commit failed: ' + e.message
  } finally {
    committing.value = false
  }
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
.directory-view {
  max-width: 800px;
}

.breadcrumb {
  font-size: 0.8rem;
  margin-bottom: 1rem;
  color: var(--text-muted);
}

.breadcrumb-link {
  color: var(--accent);
  text-decoration: none;
}

.breadcrumb-link:hover {
  text-decoration: underline;
}

.breadcrumb-sep {
  margin: 0 0.35rem;
  color: var(--text-muted);
}

.breadcrumb-current {
  color: var(--text-secondary);
}

.dir-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
  gap: 1rem;
  flex-wrap: wrap;
}

.dir-header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.dir-header h1 {
  font-size: 1.8rem;
  font-weight: 600;
  margin: 0;
}

.rename-input {
  font-size: 1.8rem;
  font-weight: 600;
  background: var(--bg-input);
  border: 1px solid var(--accent-dim);
  border-radius: 4px;
  color: var(--text-primary);
  padding: 0.1rem 0.4rem;
  flex: 1;
  outline: none;
}

.sync-indicator {
  font-size: 0.72rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 10px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
}

.sync-clean {
  background: rgba(158, 206, 106, 0.15);
  color: var(--success);
}

.sync-pending {
  background: rgba(224, 175, 104, 0.15);
  color: var(--warning);
}

.sync-error {
  background: rgba(247, 118, 142, 0.15);
  color: var(--error);
}

.dir-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.error-msg {
  background: rgba(255, 80, 80, 0.1);
  border: 1px solid var(--error);
  border-radius: 4px;
  padding: 0.5rem 0.75rem;
  color: var(--error);
  font-size: 0.85rem;
  margin-bottom: 1rem;
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

.item-list {
  display: flex;
  flex-direction: column;
}

.item {
  display: flex;
  align-items: center;
  padding: 0.5rem 0.75rem;
  text-decoration: none;
  color: var(--text-primary);
  border-radius: 4px;
  gap: 0.5rem;
}

.item:hover {
  background: var(--bg-hover);
  text-decoration: none;
}

.item-icon {
  font-size: 0.75rem;
  color: var(--text-muted);
  width: 1rem;
  text-align: center;
  flex-shrink: 0;
}

.item-name {
  font-size: 0.9rem;
}

.item-path {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-left: auto;
}

.empty-state {
  color: var(--text-muted);
  font-size: 0.9rem;
  padding: 1rem 0;
}

.commit-bar {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
}

.commit-input {
  flex: 1;
  padding: 0.4rem 0.6rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.85rem;
  outline: none;
}

.commit-input:focus {
  border-color: var(--accent-dim);
}

.log-section {
  border-top: 1px solid var(--border);
  padding-top: 1rem;
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

.btn-primary:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-input);
  color: var(--text-primary);
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-stop {
  color: var(--error);
  border-color: var(--error);
  background: none;
}

.btn-stop:hover {
  background: rgba(247, 118, 142, 0.1);
}

.btn-danger {
  background: none;
  color: var(--error);
  border-color: var(--error);
}

.btn-danger:hover {
  background: var(--error);
  color: var(--bg-primary);
}

.loading {
  color: var(--text-muted);
  padding: 2rem;
}

.error-state {
  padding: 2rem;
  text-align: center;
}

.error-state h2 {
  font-size: 1.3rem;
  margin-bottom: 0.5rem;
}

.error-state p {
  color: var(--error);
  margin-bottom: 1rem;
}

.home-link {
  color: var(--accent);
  font-size: 0.9rem;
}
</style>
