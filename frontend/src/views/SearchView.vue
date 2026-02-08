<template>
  <div class="search-view">
    <h1>Search results</h1>
    <p v-if="query" class="query-info">
      Showing results for <strong>"{{ query }}"</strong>
    </p>

    <div v-if="results.length" class="results">
      <router-link
        v-for="r in results"
        :key="r.id"
        :to="{ name: 'page', params: { id: r.id } }"
        class="result-card"
      >
        <div class="result-title">{{ r.title }}</div>
        <div class="result-path">{{ r.path }}</div>
        <div class="result-snippet" v-html="r.snippet"></div>
      </router-link>
    </div>

    <p v-else-if="query && !loading" class="no-results">
      No results found.
    </p>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { searchPages } from '../lib/api.js'

const route = useRoute()
const query = ref('')
const results = ref([])
const loading = ref(false)

async function doSearch(q) {
  query.value = q
  if (!q) {
    results.value = []
    return
  }
  loading.value = true
  try {
    results.value = await searchPages(q)
  } catch (e) {
    console.error('Search failed:', e)
  } finally {
    loading.value = false
  }
}

watch(() => route.query.q, (q) => doSearch(q || ''), { immediate: true })
</script>

<style scoped>
.search-view {
  max-width: 700px;
}

h1 {
  font-size: 1.3rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.query-info {
  color: var(--text-secondary);
  margin-bottom: 1.5rem;
  font-size: 0.9rem;
}

.result-card {
  display: block;
  padding: 0.75rem 1rem;
  margin-bottom: 0.5rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  text-decoration: none;
  color: var(--text-primary);
  transition: background 0.15s;
}

.result-card:hover {
  background: var(--bg-hover);
  text-decoration: none;
}

.result-title {
  font-weight: 600;
  font-size: 0.95rem;
  margin-bottom: 0.15rem;
}

.result-path {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-bottom: 0.35rem;
}

.result-snippet {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.no-results {
  color: var(--text-muted);
  font-size: 0.9rem;
}
</style>
