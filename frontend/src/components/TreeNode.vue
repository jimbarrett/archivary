<template>
  <div class="tree-node">
    <div
      v-if="node.is_dir"
      class="tree-dir"
      :class="{ 'drag-over': isDragOver }"
      @dragenter="onDragEnter"
      @dragover.prevent="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
    >
      <span class="tree-arrow" :class="{ expanded }" @click.stop="toggleExpand">&#9654;</span>
      <span class="tree-label" @click="navigateToDir">{{ node.name }}</span>
      <span v-if="dirSyncStatus" class="sync-badge" :class="syncClass" :title="syncTitle">
        <svg v-if="dirSyncStatus.error" width="10" height="10" viewBox="0 0 16 16" fill="currentColor">
          <circle cx="8" cy="8" r="7"/>
        </svg>
        <svg v-else width="10" height="10" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M2 2v4h4" /><path d="M14 14v-4h-4" />
          <path d="M3.5 10a5 5 0 0 0 8.5 1" /><path d="M12.5 6A5 5 0 0 0 4 5" />
        </svg>
      </span>
    </div>

    <router-link
      v-else
      :to="{ name: 'page', params: { id: node.page_id } }"
      class="tree-file"
      active-class="active"
      draggable="true"
      @dragstart="onDragStart"
      @dragend="onDragEnd"
    >
      <span class="tree-icon">&#9643;</span>
      <span class="tree-label">{{ node.name }}</span>
    </router-link>

    <div v-if="node.is_dir && expanded && node.children" class="tree-children">
      <TreeNode
        v-for="child in sortedChildren"
        :key="child.path"
        :node="child"
        :sync-status="syncStatus"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { movePage } from '../lib/api.js'
import { refreshSidebarTree } from '../lib/events.js'

const router = useRouter()

const props = defineProps({
  node: { type: Object, required: true },
  syncStatus: { type: Object, default: () => ({}) },
})

const expanded = ref(true)
const isDragOver = ref(false)
let dragCounter = 0

const sortedChildren = computed(() => {
  if (!props.node.children) return []
  return [...props.node.children].sort((a, b) => {
    // Directories first, then alphabetical
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
})

// Get sync status for this directory (only top-level synced dirs have entries)
const dirSyncStatus = computed(() => {
  if (!props.node.is_dir || !props.node.path) return null
  return props.syncStatus[props.node.path] || null
})

const syncClass = computed(() => {
  const s = dirSyncStatus.value
  if (!s) return ''
  if (s.error) return 'sync-error'
  if (!s.clean) return 'sync-pending'
  return 'sync-clean'
})

const syncTitle = computed(() => {
  const s = dirSyncStatus.value
  if (!s) return ''
  if (s.error) return `Sync error: ${s.error}`
  if (!s.clean) return 'Uncommitted changes'
  if (s.ahead > 0) return `${s.ahead} commit(s) to push`
  return 'Synced'
})

function toggleExpand() {
  expanded.value = !expanded.value
}

function navigateToDir() {
  if (props.node.path) {
    router.push({ name: 'directory', params: { path: props.node.path } })
  }
}

// Drag and drop handlers
function onDragStart(event) {
  if (!props.node.page_id) return
  
  // Store the dragged page info
  const dragData = {
    pageId: props.node.page_id,
    currentPath: props.node.path,
    fileName: props.node.path.split('/').pop() // Get just the filename
  }
  event.dataTransfer.effectAllowed = 'move'
  event.dataTransfer.setData('application/json', JSON.stringify(dragData))
  
  // Add visual feedback
  event.target.style.opacity = '0.5'
}

function onDragEnd(event) {
  event.target.style.opacity = '1'
}

function onDragEnter(event) {
  if (!props.node.is_dir) return
  const types = event.dataTransfer.types
  if (!types.includes('application/json')) return

  dragCounter++
  isDragOver.value = true
  event.dataTransfer.dropEffect = 'move'
}

function onDragOver(event) {
  if (!props.node.is_dir) return
  const types = event.dataTransfer.types
  if (!types.includes('application/json')) return

  event.preventDefault()
  event.dataTransfer.dropEffect = 'move'
}

function onDragLeave() {
  dragCounter--
  if (dragCounter <= 0) {
    dragCounter = 0
    isDragOver.value = false
  }
}

async function onDrop(event) {
  event.preventDefault()
  dragCounter = 0
  isDragOver.value = false
  
  if (!props.node.is_dir) return
  
  try {
    const dragData = JSON.parse(event.dataTransfer.getData('application/json'))
    const { pageId, currentPath, fileName } = dragData
    
    // Build new path: directory path + filename
    const newPath = props.node.path ? `${props.node.path}/${fileName}` : fileName
    
    // Don't move if it's already in this directory
    if (currentPath === newPath) return
    
    // Call the API to move the page
    await movePage(pageId, newPath)
    
    // Notify the tree to refresh
    refreshSidebarTree()
    
  } catch (error) {
    console.error('Failed to move page:', error)
    alert(`Failed to move file: ${error.message}`)
  }
}
</script>

<style scoped>
.tree-node {
  user-select: none;
}

.tree-dir {
  display: flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.tree-dir:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.tree-dir.drag-over {
  background: rgba(122, 162, 247, 0.15);
  outline: 2px dashed var(--accent);
  outline-offset: -2px;
  color: var(--accent);
}

.tree-arrow {
  font-size: 0.55rem;
  margin-right: 0.4rem;
  transition: transform 0.15s;
  display: inline-block;
  width: 0.8rem;
  text-align: center;
  cursor: pointer;
  padding: 0.15rem 0;
  border-radius: 2px;
}

.tree-arrow:hover {
  color: var(--accent);
}

.tree-dir > .tree-label {
  cursor: pointer;
}

.tree-dir > .tree-label:hover {
  color: var(--accent);
}

.tree-arrow.expanded {
  transform: rotate(90deg);
}

.tree-file {
  display: flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-decoration: none;
  border-left: 2px solid transparent;
}

.tree-file:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  text-decoration: none;
}

.tree-file.active {
  background: var(--bg-active);
  color: var(--accent);
  border-left-color: var(--accent);
}

.tree-icon {
  margin-right: 0.4rem;
  font-size: 0.75rem;
  width: 0.8rem;
  text-align: center;
}

.tree-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tree-children {
  padding-left: 0.75rem;
}

.sync-badge {
  margin-left: auto;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding-left: 0.35rem;
}

.sync-clean {
  color: var(--success);
}

.sync-pending {
  color: var(--warning);
}

.sync-error {
  color: var(--error);
}
</style>
