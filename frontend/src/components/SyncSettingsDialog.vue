<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <div class="dialog">
      <h3>{{ isEditing ? 'Edit Sync Settings' : 'Add Synced Directory' }}</h3>

      <template v-if="!isEditing">
        <label class="field-label">Directory Name</label>
        <input
          ref="pathInput"
          v-model="form.path"
          type="text"
          placeholder="my-notes"
          class="field-input"
          @keydown.escape="$emit('close')"
        />
        <p class="field-hint">A new directory will be created in your workspace.</p>
      </template>

      <label class="field-label">Remote URL</label>
      <input
        v-model="form.url"
        type="text"
        placeholder="git@github.com:user/repo.git"
        class="field-input"
        @keydown.escape="$emit('close')"
      />
      <p class="field-hint">Leave empty to create a local-only git repo (no remote sync).</p>

      <label class="field-label">Branch</label>
      <input
        v-model="form.branch"
        type="text"
        placeholder="main"
        class="field-input"
        @keydown.escape="$emit('close')"
      />

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

      <div v-if="error" class="error-msg">{{ error }}</div>

      <div class="dialog-actions">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-primary" @click="submit" :disabled="!isValid || submitting">
          {{ submitting ? 'Saving...' : (isEditing ? 'Save' : 'Add') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { addRemote, updateRemote } from '../lib/api.js'
import { refreshSyncStatus } from '../lib/sync.js'

const props = defineProps({
  // If editing, pass the existing remote config
  remote: { type: Object, default: null },
  // For "add new" from a directory view — pre-fills the path
  defaultPath: { type: String, default: '' },
})

const emit = defineEmits(['close', 'saved'])

const isEditing = computed(() => !!props.remote)

const form = ref({
  path: props.remote?.path || props.defaultPath || '',
  url: props.remote?.url || '',
  branch: props.remote?.branch || 'main',
  autoCommit: props.remote?.auto_commit ?? true,
  autoPush: props.remote?.auto_push ?? true,
  pushInterval: props.remote?.push_interval_minutes ?? 5,
})

const error = ref('')
const submitting = ref(false)
const pathInput = ref(null)

const isValid = computed(() => {
  if (!isEditing.value && !form.value.path.trim()) return false
  return true
})

onMounted(() => {
  if (!isEditing.value) {
    nextTick(() => {
      if (pathInput.value) pathInput.value.focus()
    })
  }
})

async function submit() {
  if (!isValid.value) return
  error.value = ''
  submitting.value = true

  try {
    if (isEditing.value) {
      await updateRemote(props.remote.path, {
        url: form.value.url,
        branch: form.value.branch,
        auto_commit: form.value.autoCommit,
        auto_push: form.value.autoPush,
        push_interval_minutes: form.value.pushInterval,
      })
    } else {
      await addRemote({
        path: form.value.path.trim(),
        url: form.value.url.trim(),
        branch: form.value.branch.trim() || 'main',
        auto_commit: form.value.autoCommit,
        auto_push: form.value.autoPush,
        push_interval_minutes: form.value.pushInterval,
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
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.75rem;
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
</style>
