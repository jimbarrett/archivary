<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <h1 class="logo">Archivary</h1>
    </div>

    <div class="search-box">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search pages..."
        @input="onSearch"
        @keydown.enter="goToSearch"
      />
    </div>

    <div v-if="searchResults.length" class="search-results">
      <div class="section-label">Results</div>
      <router-link
        v-for="result in searchResults"
        :key="result.id"
        :to="{ name: 'page', params: { id: result.id } }"
        class="search-result"
        @click="clearSearch"
      >
        <span class="result-title">{{ result.title }}</span>
        <span class="result-path">{{ result.path }}</span>
      </router-link>
    </div>

    <div v-else class="tree-section">
      <div class="section-header">
        <span class="section-label">Pages</span>
        <div class="section-actions">
          <button
            v-if="hasSyncRemotes"
            class="icon-btn"
            @click="onSyncNow"
            :disabled="syncing"
            title="Sync all"
          >
            <svg
              width="14" height="14" viewBox="0 0 16 16" fill="none"
              stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"
              :class="{ spinning: syncing }"
            >
              <path d="M1 1v5h5" /><path d="M15 15v-5h-5" />
              <path d="M2.3 10a6 6 0 0 0 10.3 1.5" /><path d="M13.7 6A6 6 0 0 0 3.4 4.5" />
            </svg>
          </button>
          <button class="icon-btn" @click="runReindex" :disabled="reindexing" title="Reindex">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M8 1v6l4 2" /><circle cx="8" cy="8" r="7" />
            </svg>
          </button>
          <button class="icon-btn" @click="showNewPage = true" title="New page">+</button>
        </div>
      </div>
      <div
        v-if="tree && tree.children"
        class="tree"
        :class="{ 'drag-over-root': isDragOverRoot }"
        @dragenter="onRootDragEnter"
        @dragover.prevent="onRootDragOver"
        @dragleave="onRootDragLeave"
        @drop="onRootDrop"
      >
        <TreeNode
          v-for="node in sortedRootChildren"
          :key="node.path"
          :node="node"
          :sync-status="syncStatus"
        />
      </div>
      <div v-else class="empty-state">
        No pages yet.
        <button class="link-btn" @click="showNewPage = true">Create one</button>
      </div>

      <div class="sidebar-footer">
        <router-link to="/sync" class="footer-link" title="Sync settings">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M1 1v5h5" /><path d="M15 15v-5h-5" />
            <path d="M2.3 10a6 6 0 0 0 10.3 1.5" /><path d="M13.7 6A6 6 0 0 0 3.4 4.5" />
          </svg>
          <span>Git Sync</span>
          <span v-if="remoteCount" class="badge">{{ remoteCount }}</span>
        </router-link>
      </div>
    </div>

    <NewPageDialog
      v-if="showNewPage"
      @created="onCreated"
      @close="showNewPage = false"
    />
  </aside>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getTree, searchPages, reindex, movePage } from '../lib/api.js'
import { treeVersion } from '../lib/events.js'
import { syncStatus, syncRemotes, syncing, startSyncPolling, stopSyncPolling, triggerSyncAll } from '../lib/sync.js'
import TreeNode from './TreeNode.vue'
import NewPageDialog from './NewPageDialog.vue'

const router = useRouter()
const tree = ref(null)
const searchQuery = ref('')
const searchResults = ref([])
const showNewPage = ref(false)
const reindexing = ref(false)
const isDragOverRoot = ref(false)
let rootDragCounter = 0

let searchTimeout = null

const sortedRootChildren = computed(() => {
  if (!tree.value || !tree.value.children) return []
  return [...tree.value.children].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
})

const hasSyncRemotes = computed(() => syncRemotes.value.length > 0)
const remoteCount = computed(() => syncRemotes.value.length)

onMounted(async () => {
  await refreshTree()
  startSyncPolling()
})

onUnmounted(() => {
  stopSyncPolling()
})

// Refresh tree when other components signal changes
watch(treeVersion, () => refreshTree())

function onSearch() {
  clearTimeout(searchTimeout)
  if (!searchQuery.value.trim()) {
    searchResults.value = []
    return
  }
  searchTimeout = setTimeout(async () => {
    try {
      searchResults.value = await searchPages(searchQuery.value)
    } catch (e) {
      console.error('Search failed:', e)
    }
  }, 200)
}

function goToSearch() {
  if (searchQuery.value.trim()) {
    router.push({ name: 'search', query: { q: searchQuery.value } })
  }
}

function clearSearch() {
  searchQuery.value = ''
  searchResults.value = []
}

async function refreshTree() {
  try {
    tree.value = await getTree()
  } catch (e) {
    console.error('Failed to load tree:', e)
  }
}

function onCreated() {
  showNewPage.value = false
  refreshTree()
}

async function runReindex() {
  reindexing.value = true
  try {
    await reindex()
    await refreshTree()
  } catch (e) {
    console.error('Reindex failed:', e)
  } finally {
    reindexing.value = false
  }
}

async function onSyncNow() {
  try {
    await triggerSyncAll()
    await refreshTree()
  } catch (e) {
    console.error('Sync failed:', e)
  }
}

// Root directory drag and drop
function onRootDragEnter(event) {
  const types = event.dataTransfer.types
  if (!types.includes('application/json')) return

  rootDragCounter++
  isDragOverRoot.value = true
}

function onRootDragOver(event) {
  const types = event.dataTransfer.types
  if (!types.includes('application/json')) return

  event.preventDefault()
  event.dataTransfer.dropEffect = 'move'
}

function onRootDragLeave() {
  rootDragCounter--
  if (rootDragCounter <= 0) {
    rootDragCounter = 0
    isDragOverRoot.value = false
  }
}

async function onRootDrop(event) {
  event.preventDefault()
  rootDragCounter = 0
  isDragOverRoot.value = false
  
  try {
    const dragData = JSON.parse(event.dataTransfer.getData('application/json'))
    const { pageId, currentPath, fileName } = dragData
    
    // If the file is already in root, don't do anything
    if (!currentPath.includes('/')) return
    
    // Move to root (no directory prefix)
    await movePage(pageId, fileName)
    
    // Refresh the tree
    await refreshTree()
    
  } catch (error) {
    console.error('Failed to move page:', error)
    alert(`Failed to move file: ${error.message}`)
  }
}

defineExpose({ refreshTree })
</script>

<style scoped>
.sidebar {
  width: 260px;
  min-width: 260px;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.sidebar-header {
  padding: 1.25rem 1rem 0.75rem;
  border-bottom: 1px solid var(--border);
}

.logo {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.02em;
}

.search-box {
  padding: 0.75rem 0.75rem 0.5rem;
}

.search-box input {
  width: 100%;
  padding: 0.4rem 0.6rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.85rem;
  outline: none;
}

.search-box input::placeholder {
  color: var(--text-muted);
}

.search-box input:focus {
  border-color: var(--accent-dim);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem 0.25rem 1rem;
}

.section-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.section-actions {
  display: flex;
  gap: 0.25rem;
}

.icon-btn {
  background: none;
  border: 1px solid var(--border);
  border-radius: 3px;
  color: var(--text-muted);
  font-size: 0.9rem;
  width: 1.5rem;
  height: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  line-height: 1;
}

.icon-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text-primary);
  border-color: var(--accent-dim);
}

.icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.tree-section {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.tree {
  flex: 1;
  padding-bottom: 1rem;
}

.tree.drag-over-root {
  background: rgba(122, 162, 247, 0.1);
  outline: 2px dashed var(--accent);
  outline-offset: -4px;
}

.search-results {
  flex: 1;
  overflow-y: auto;
}

.search-results .section-label {
  padding: 0.5rem 1rem 0.25rem;
}

.search-result {
  display: block;
  padding: 0.4rem 1rem;
  text-decoration: none;
  color: var(--text-primary);
  border-left: 2px solid transparent;
}

.search-result:hover {
  background: var(--bg-hover);
  text-decoration: none;
}

.result-title {
  display: block;
  font-size: 0.85rem;
}

.result-path {
  display: block;
  font-size: 0.7rem;
  color: var(--text-muted);
}

.empty-state {
  padding: 1rem;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.link-btn {
  background: none;
  border: none;
  color: var(--accent);
  cursor: pointer;
  font-size: 0.85rem;
  padding: 0;
}

.link-btn:hover {
  text-decoration: underline;
}

.sidebar-footer {
  padding: 0.5rem 0.75rem;
  border-top: 1px solid var(--border);
  margin-top: auto;
}

.footer-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  color: var(--text-secondary);
  text-decoration: none;
}

.footer-link:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  text-decoration: none;
}

.badge {
  margin-left: auto;
  background: var(--accent-dim);
  color: var(--accent);
  font-size: 0.65rem;
  font-weight: 600;
  padding: 0.1rem 0.35rem;
  border-radius: 8px;
  min-width: 1.1rem;
  text-align: center;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.spinning {
  animation: spin 1s linear infinite;
}
</style>
