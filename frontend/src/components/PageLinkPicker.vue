<template>
  <div class="picker-overlay" @click.self="$emit('close')">
    <div class="picker-modal">
      <div class="picker-header">
        <h3>Insert page link</h3>
        <button class="picker-close" @click="$emit('close')">&times;</button>
      </div>

      <input
        ref="searchInput"
        v-model="query"
        type="text"
        placeholder="Search pages..."
        class="picker-search"
        @input="filterPages"
        @keydown.escape="$emit('close')"
        @keydown.enter="selectFirst"
      />

      <div class="picker-results">
        <div
          v-for="page in filtered"
          :key="page.id"
          class="picker-item"
          @click="$emit('select', page)"
        >
          <span class="picker-item-title">{{ page.title }}</span>
          <span class="picker-item-path">{{ page.path }}</span>
        </div>
        <div v-if="!filtered.length && query" class="picker-empty">
          No pages found.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getPages } from '../lib/api.js'

const emit = defineEmits(['select', 'close'])

const query = ref('')
const pages = ref([])
const filtered = ref([])
const searchInput = ref(null)

onMounted(async () => {
  try {
    pages.value = await getPages()
    filtered.value = pages.value
  } catch (e) {
    console.error('Failed to load pages:', e)
  }
  // Auto-focus the search input
  requestAnimationFrame(() => {
    if (searchInput.value) searchInput.value.focus()
  })
})

function filterPages() {
  const q = query.value.toLowerCase()
  if (!q) {
    filtered.value = pages.value
    return
  }
  filtered.value = pages.value.filter(
    (p) => p.title.toLowerCase().includes(q) || p.path.toLowerCase().includes(q)
  )
}

function selectFirst() {
  if (filtered.value.length) {
    emit('select', filtered.value[0])
  }
}
</script>

<style scoped>
.picker-overlay {
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

.picker-modal {
  background: var(--bg-sidebar);
  border: 1px solid var(--border);
  border-radius: 8px;
  width: 420px;
  max-height: 400px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.picker-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border);
}

.picker-header h3 {
  font-size: 0.9rem;
  font-weight: 600;
}

.picker-close {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0 0.25rem;
}

.picker-close:hover {
  color: var(--text-primary);
}

.picker-search {
  margin: 0.75rem;
  padding: 0.4rem 0.6rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.85rem;
  outline: none;
}

.picker-search::placeholder {
  color: var(--text-muted);
}

.picker-search:focus {
  border-color: var(--accent-dim);
}

.picker-results {
  overflow-y: auto;
  flex: 1;
  padding-bottom: 0.5rem;
}

.picker-item {
  padding: 0.4rem 1rem;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.picker-item:hover {
  background: var(--bg-hover);
}

.picker-item-title {
  font-size: 0.85rem;
  color: var(--text-primary);
}

.picker-item-path {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.picker-empty {
  padding: 1rem;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.85rem;
}
</style>
