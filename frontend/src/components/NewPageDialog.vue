<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <div class="dialog">
      <div class="mode-tabs">
        <button :class="{ active: mode === 'page' }" @click="mode = 'page'">New Page</button>
        <button :class="{ active: mode === 'dir' }" @click="mode = 'dir'">New Directory</button>
      </div>

      <template v-if="mode === 'page'">
        <label class="field-label">Filename</label>
        <input
          ref="nameInput"
          v-model="filename"
          type="text"
          placeholder="my-page.md"
          class="field-input"
          @keydown.enter="submit"
          @keydown.escape="$emit('close')"
        />

        <label class="field-label">Folder</label>
        <select v-model="selectedFolder" class="field-input" @keydown.enter="submit">
          <option value="">/ (root)</option>
          <option v-for="dir in dirs" :key="dir" :value="dir">{{ dir }}</option>
          <option value="__new__">+ New directory...</option>
        </select>

        <div v-if="selectedFolder === '__new__'" class="new-folder-wrapper">
          <input
            ref="newFolderInput"
            v-model="newFolder"
            type="text"
            placeholder="path/to/folder"
            class="field-input"
            @keydown.enter="submit"
          />
        </div>

        <div v-if="conflict" class="conflict-warning">
          <p>A file already exists at <strong>{{ conflictPath }}</strong>.</p>
          <div class="conflict-actions">
            <button class="btn btn-secondary" @click="focusFilename">Change name</button>
            <button class="btn btn-danger" @click="forceCreate">Overwrite</button>
          </div>
        </div>
      </template>

      <template v-else>
        <label class="field-label">Directory Name</label>
        <input
          ref="dirNameInput"
          v-model="dirName"
          type="text"
          placeholder="my-directory"
          class="field-input"
          @keydown.enter="submit"
          @keydown.escape="$emit('close')"
        />

        <label class="field-label">Parent Folder</label>
        <select v-model="dirParent" class="field-input" @keydown.enter="submit">
          <option value="">/ (root)</option>
          <option v-for="dir in dirs" :key="dir" :value="dir">{{ dir }}</option>
        </select>
      </template>

      <div v-if="error" class="error-msg">{{ error }}</div>

      <div class="dialog-actions">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-primary" @click="submit" :disabled="!isValid || submitting">
          {{ submitting ? 'Creating...' : 'Create' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { getTree, checkPath, createPage, createDir } from '../lib/api.js'
import { refreshSidebarTree } from '../lib/events.js'
import { useRouter } from 'vue-router'

const router = useRouter()

const emit = defineEmits(['close', 'created'])

const mode = ref('page')
const filename = ref('')
const selectedFolder = ref('')
const newFolder = ref('')
const dirName = ref('')
const dirParent = ref('')
const dirs = ref([])
const nameInput = ref(null)
const dirNameInput = ref(null)
const newFolderInput = ref(null)
const conflict = ref(false)
const conflictPath = ref('')
const error = ref('')
const submitting = ref(false)

const isValid = computed(() => {
  if (mode.value === 'page') {
    const name = filename.value.trim()
    if (!name.length) return false
    if (selectedFolder.value === '__new__' && !newFolder.value.trim().length) return false
    return true
  } else {
    return dirName.value.trim().length > 0
  }
})

watch(mode, (val) => {
  error.value = ''
  conflict.value = false
  nextTick(() => {
    if (val === 'page' && nameInput.value) nameInput.value.focus()
    if (val === 'dir' && dirNameInput.value) dirNameInput.value.focus()
  })
})

watch(selectedFolder, (val) => {
  conflict.value = false
  if (val === '__new__') {
    nextTick(() => {
      if (newFolderInput.value) newFolderInput.value.focus()
    })
  }
})

watch(filename, () => {
  conflict.value = false
})

onMounted(async () => {
  try {
    const tree = await getTree()
    dirs.value = extractDirs(tree, '')
  } catch (e) {
    console.error('Failed to load tree:', e)
  }
  requestAnimationFrame(() => {
    if (nameInput.value) nameInput.value.focus()
  })
})

function extractDirs(node, prefix) {
  const result = []
  if (!node.children) return result
  for (const child of node.children) {
    if (child.is_dir) {
      const path = prefix ? `${prefix}/${child.name}` : child.name
      result.push(path)
      result.push(...extractDirs(child, path))
    }
  }
  return result
}

function buildPagePath() {
  let name = filename.value.trim()
  if (!name.endsWith('.md')) {
    name += '.md'
  }
  let dir = ''
  if (selectedFolder.value === '__new__') {
    dir = newFolder.value.trim().replace(/^\/+|\/+$/g, '')
  } else {
    dir = selectedFolder.value
  }
  return dir ? `${dir}/${name}` : name
}

function buildDirPath() {
  const name = dirName.value.trim()
  const parent = dirParent.value
  return parent ? `${parent}/${name}` : name
}

async function submit() {
  if (!isValid.value || submitting.value) return
  error.value = ''
  submitting.value = true

  try {
    if (mode.value === 'dir') {
      await submitDir()
    } else {
      await submitPage()
    }
  } finally {
    submitting.value = false
  }
}

async function submitDir() {
  const path = buildDirPath()
  try {
    await createDir(path)
    refreshSidebarTree()
    emit('created')
    emit('close')
    router.push({ name: 'directory', params: { path } })
  } catch (e) {
    error.value = e.message
  }
}

async function submitPage() {
  const path = buildPagePath()

  // If we already showed the conflict and user clicked Create again, ignore
  if (conflict.value) return

  try {
    const result = await checkPath(path)
    if (result.exists) {
      conflict.value = true
      conflictPath.value = path
      return
    }
  } catch (e) {
    // If check fails, proceed anyway
  }

  try {
    const result = await createPage({ content: '', path })
    refreshSidebarTree()
    emit('created')
    emit('close')
    router.push({ name: 'edit', params: { id: result.id } })
  } catch (e) {
    error.value = e.message
  }
}

async function forceCreate() {
  submitting.value = true
  error.value = ''
  try {
    const result = await createPage({ content: '', path: conflictPath.value })
    refreshSidebarTree()
    emit('created')
    emit('close')
    router.push({ name: 'edit', params: { id: result.id } })
  } catch (e) {
    error.value = e.message
  } finally {
    submitting.value = false
  }
}

function focusFilename() {
  conflict.value = false
  conflictPath.value = ''
  nextTick(() => {
    if (nameInput.value) {
      nameInput.value.focus()
      nameInput.value.select()
    }
  })
}
</script>

<style scoped>
.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.dialog {
  background: var(--bg-sidebar);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1.25rem;
  width: 360px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.mode-tabs {
  display: flex;
  gap: 0;
  margin-bottom: 1rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  overflow: hidden;
}

.mode-tabs button {
  flex: 1;
  padding: 0.4rem 0;
  font-size: 0.8rem;
  font-weight: 600;
  background: var(--bg-input);
  color: var(--text-secondary);
  border: none;
  cursor: pointer;
}

.mode-tabs button.active {
  background: var(--accent-dim);
  color: var(--text-primary);
}

.mode-tabs button:not(.active):hover {
  background: var(--bg-hover);
}

.field-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
  margin-bottom: 0.25rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.field-input {
  width: 100%;
  padding: 0.4rem 0.6rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.85rem;
  outline: none;
  margin-bottom: 0.75rem;
}

.field-input:focus {
  border-color: var(--accent-dim);
}

select.field-input {
  cursor: pointer;
  appearance: auto;
}

.new-folder-wrapper {
  margin-bottom: 0.75rem;
}

.new-folder-wrapper .field-input {
  margin-bottom: 0;
}

.conflict-warning {
  background: rgba(255, 80, 80, 0.1);
  border: 1px solid var(--error);
  border-radius: 4px;
  padding: 0.6rem 0.75rem;
  margin-bottom: 0.75rem;
}

.conflict-warning p {
  font-size: 0.8rem;
  color: var(--error);
  margin: 0 0 0.5rem;
}

.conflict-actions {
  display: flex;
  gap: 0.5rem;
}

.error-msg {
  background: rgba(255, 80, 80, 0.1);
  border: 1px solid var(--error);
  border-radius: 4px;
  padding: 0.5rem 0.75rem;
  color: var(--error);
  font-size: 0.8rem;
  margin-bottom: 0.75rem;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.btn {
  padding: 0.35rem 0.85rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 0.85rem;
  cursor: pointer;
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

.btn-danger {
  background: none;
  color: var(--error);
  border-color: var(--error);
}

.btn-danger:hover {
  background: var(--error);
  color: var(--bg-primary);
}
</style>
