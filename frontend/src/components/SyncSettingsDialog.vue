<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <div class="dialog">
      <h3>{{ isEditing ? 'Sync Settings' : 'Setup Workspace Sync' }}</h3>

      <label class="field-label">Remote URL</label>
      <input
        ref="urlInput"
        v-model="form.url"
        type="text"
        placeholder="git@github.com:user/repo.git"
        class="field-input"
        @keydown.escape="$emit('close')"
      />
      <p class="field-hint">Leave empty to create a local-only git repo (no remote sync).</p>

      <label class="field-label">Branch</label>
      <input
        v-if="!isEditing"
        v-model="form.branch"
        type="text"
        placeholder="main"
        class="field-input"
        @keydown.escape="$emit('close')"
      />
      <span v-else class="field-value">{{ actualBranch }}</span>

      <div class="toggle-group">
        <label class="toggle-row">
          <input type="checkbox" v-model="form.autoCommit" />
          <span>Auto-commit on save</span>
        </label>
        <label class="toggle-row">
          <input type="checkbox" v-model="form.autoPush" />
          <span>Auto-push on timer</span>
        </label>
      </div>

      <div v-if="form.autoPush" class="interval-row">
        <label class="field-label">Push interval (minutes)</label>
        <input
          v-model.number="form.pushInterval"
          type="number"
          min="1"
          max="60"
          class="field-input interval-input"
        />
      </div>

      <!-- Workspace contents preview (setup mode only) -->
      <div v-if="!isEditing && workspaceEntries.length" class="workspace-preview">
        <label class="field-label">Workspace Contents</label>
        <p class="field-hint">Uncheck items to exclude from sync</p>
        <div class="entry-list">
          <label v-for="entry in workspaceEntries" :key="entry.name" class="entry-row">
            <input
              type="checkbox"
              :checked="!uncheckedEntries.has(entry.name)"
              @change="toggleEntry(entry.name)"
            />
            <span class="entry-name">{{ entry.name }}{{ entry.is_dir ? '/' : '' }}</span>
          </label>
        </div>
      </div>

      <div v-if="error" class="error-msg">{{ error }}</div>

      <div class="dialog-actions">
        <button
          v-if="isEditing"
          class="btn btn-danger unsync-btn"
          @click="onUnsync"
        >Unsync Workspace</button>
        <div class="dialog-actions-right">
          <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
          <button class="btn btn-primary" @click="submit" :disabled="submitting">
            {{ submitting ? 'Saving...' : (isEditing ? 'Save' : 'Setup') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { addRemote, updateRemote, removeRemote, getWorkspaceEntries } from '../lib/api.js'
import { refreshSyncStatus, rootSyncStatus } from '../lib/sync.js'

const props = defineProps({
  remote: { type: Object, default: null },
})

const emit = defineEmits(['close', 'saved'])

const isEditing = computed(() => !!props.remote)
const actualBranch = computed(() => rootSyncStatus.value?.branch || props.remote?.branch || 'main')

const form = ref({
  url: props.remote?.url || '',
  branch: props.remote?.branch || 'main',
  autoCommit: props.remote?.auto_commit ?? true,
  autoPush: props.remote?.auto_push ?? false,
  pushInterval: props.remote?.push_interval_minutes ?? 5,
})

const error = ref('')
const submitting = ref(false)
const urlInput = ref(null)
const workspaceEntries = ref([])
const uncheckedEntries = reactive(new Set())

function toggleEntry(name) {
  if (uncheckedEntries.has(name)) {
    uncheckedEntries.delete(name)
  } else {
    uncheckedEntries.add(name)
  }
}

onMounted(async () => {
  if (!isEditing.value) {
    try {
      workspaceEntries.value = await getWorkspaceEntries()
    } catch (e) {
      console.debug('Failed to load workspace entries:', e)
    }
    nextTick(() => {
      if (urlInput.value) urlInput.value.focus()
    })
  }
})

async function submit() {
  error.value = ''
  submitting.value = true

  try {
    if (isEditing.value) {
      await updateRemote('.', {
        url: form.value.url,
        auto_commit: form.value.autoCommit,
        auto_push: form.value.autoPush,
        push_interval_minutes: form.value.pushInterval,
      })
    } else {
      // Split unchecked items into dirs and files
      const excludedDirs = []
      const excludedFiles = []
      for (const name of uncheckedEntries) {
        const entry = workspaceEntries.value.find(e => e.name === name)
        if (entry?.is_dir) {
          excludedDirs.push(name)
        } else {
          excludedFiles.push(name)
        }
      }

      await addRemote({
        path: '.',
        url: form.value.url.trim(),
        branch: form.value.branch.trim() || 'main',
        auto_commit: form.value.autoCommit,
        auto_push: form.value.autoPush,
        push_interval_minutes: form.value.pushInterval,
        excluded_dirs: excludedDirs,
        excluded_files: excludedFiles,
      })
    }
    await refreshSyncStatus()
    emit('saved')
    emit('close')
  } catch (e) {
    error.value = e.message
  } finally {
    submitting.value = false
  }
}

async function onUnsync() {
  if (!confirm('Unsync the workspace? The .git directory will be removed but your files will remain.')) return
  try {
    await removeRemote('.')
    await refreshSyncStatus()
    emit('saved')
    emit('close')
  } catch (e) {
    error.value = e.message
  }
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
  width: 400px;
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
  margin-bottom: 0.25rem;
}

.field-input:focus {
  border-color: var(--accent-dim);
}

.field-hint {
  font-size: 0.72rem;
  color: var(--text-muted);
  margin-bottom: 0.75rem;
}

.field-value {
  display: block;
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding: 0.4rem 0;
  margin-bottom: 0.25rem;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
}

.toggle-group {
  margin: 0.75rem 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.toggle-row input[type="checkbox"] {
  accent-color: var(--accent);
}

.interval-row {
  margin-bottom: 0.75rem;
}

.interval-input {
  width: 80px;
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
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.dialog-actions-right {
  display: flex;
  gap: 0.5rem;
  margin-left: auto;
}

.unsync-btn {
  margin-right: auto;
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

.btn-secondary {
  background: var(--bg-input);
  color: var(--text-primary);
}

.btn-danger {
  background: none;
  color: var(--error);
  border-color: var(--error);
}

.btn-danger:hover {
  background: rgba(247, 118, 142, 0.1);
}

.workspace-preview {
  margin: 0.75rem 0;
}

.entry-list {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 0.25rem 0;
}

.entry-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.2rem 0.6rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.entry-row:hover {
  background: var(--bg-hover);
}

.entry-row input[type="checkbox"] {
  accent-color: var(--accent);
}

.entry-name {
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 0.8rem;
}
</style>
