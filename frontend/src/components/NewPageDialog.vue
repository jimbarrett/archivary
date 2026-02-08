<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <div class="dialog">
      <h3>New Page</h3>

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

      <div class="dialog-actions">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-primary" @click="submit" :disabled="!isValid">Create</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { getTree } from '../lib/api.js'

const emit = defineEmits(['create', 'close'])

const filename = ref('')
const selectedFolder = ref('')
const newFolder = ref('')
const dirs = ref([])
const nameInput = ref(null)
const newFolderInput = ref(null)

const isValid = computed(() => {
  const name = filename.value.trim()
  if (!name.length) return false
  if (selectedFolder.value === '__new__' && !newFolder.value.trim().length) return false
  return true
})

watch(selectedFolder, (val) => {
  if (val === '__new__') {
    nextTick(() => {
      if (newFolderInput.value) newFolderInput.value.focus()
    })
  }
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

function submit() {
  if (!isValid.value) return
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
  const path = dir ? `${dir}/${name}` : name
  emit('create', path)
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

.dialog h3 {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
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
</style>
