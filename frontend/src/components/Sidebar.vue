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
        <button class="new-page-btn" @click="showNewPage = true" title="New page">+</button>
      </div>
      <div v-if="tree && tree.children" class="tree">
        <TreeNode
          v-for="node in sortedRootChildren"
          :key="node.path"
          :node="node"
        />
      </div>
      <div v-else class="empty-state">
        No pages yet.
        <button class="link-btn" @click="showNewPage = true">Create one</button>
      </div>
    </div>

    <NewPageDialog
      v-if="showNewPage"
      @create="onCreatePage"
      @close="showNewPage = false"
    />
  </aside>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getTree, searchPages } from '../lib/api.js'
import { treeVersion } from '../lib/events.js'
import TreeNode from './TreeNode.vue'
import NewPageDialog from './NewPageDialog.vue'

const router = useRouter()
const tree = ref(null)
const searchQuery = ref('')
const searchResults = ref([])
const showNewPage = ref(false)

let searchTimeout = null

const sortedRootChildren = computed(() => {
  if (!tree.value || !tree.value.children) return []
  return [...tree.value.children].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
})

onMounted(async () => {
  await refreshTree()
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

function onCreatePage(path) {
  showNewPage.value = false
  // Navigate to the editor for a new page with the chosen path
  router.push({ name: 'new', query: { path } })
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

.new-page-btn {
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

.new-page-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  border-color: var(--accent-dim);
}

.tree-section {
  flex: 1;
  overflow-y: auto;
}

.tree {
  padding-bottom: 1rem;
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
</style>
