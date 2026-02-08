<template>
  <div class="home">
    <h1>Welcome to Archivary</h1>
    <p>Select a page from the sidebar, or use search to find something.</p>

    <button class="reindex-btn" @click="runReindex" :disabled="reindexing">
      {{ reindexing ? 'Reindexing...' : 'Reindex' }}
    </button>
    <p v-if="message" class="reindex-msg" :class="{ error: isError }">{{ message }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { reindex } from '../lib/api.js'
import { refreshSidebarTree } from '../lib/events.js'

const reindexing = ref(false)
const message = ref('')
const isError = ref(false)

async function runReindex() {
  reindexing.value = true
  message.value = ''
  isError.value = false
  try {
    await reindex()
    refreshSidebarTree()
    message.value = 'Reindex complete.'
  } catch (e) {
    message.value = e.message
    isError.value = true
  } finally {
    reindexing.value = false
  }
}
</script>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 60vh;
  text-align: center;
}

.home h1 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.home p {
  color: var(--text-secondary);
  font-size: 0.95rem;
}

.reindex-btn {
  margin-top: 1.5rem;
  padding: 0.4rem 1rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
}

.reindex-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text-primary);
  border-color: var(--accent-dim);
}

.reindex-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.reindex-msg {
  margin-top: 0.5rem;
  font-size: 0.8rem;
}

.reindex-msg.error {
  color: var(--error);
}
</style>
