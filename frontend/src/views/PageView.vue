<template>
  <div class="page-view" v-if="page">
    <div class="page-actions">
      <router-link :to="{ name: 'edit', params: { id: page.id } }" class="btn btn-secondary">
        Edit
      </router-link>
      <button class="btn btn-danger" @click="confirmDelete">Delete</button>
    </div>
    <article class="page-content" v-html="renderedContent" @click="handleClick"></article>

    <div v-if="backlinks.length" class="backlinks">
      <h3>Linked from</h3>
      <ul>
        <li v-for="bl in backlinks" :key="bl.id">
          <router-link :to="{ name: 'page', params: { id: bl.id } }">
            {{ bl.title }}
          </router-link>
        </li>
      </ul>
    </div>
  </div>

  <div v-else-if="error" class="error-state">
    <h2>Something went wrong</h2>
    <p>{{ error }}</p>
    <router-link to="/" class="home-link">Go home</router-link>
  </div>

  <div v-else class="loading">
    Loading...
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getPage, getBacklinks, getPages, deletePage } from '../lib/api.js'
import { renderMarkdown } from '../lib/markdown.js'
import { refreshSidebarTree } from '../lib/events.js'

const router = useRouter()

const props = defineProps({
  id: { type: String, required: true },
})

const page = ref(null)
const backlinks = ref([])
const allPages = ref([])
const error = ref(null)

const renderedContent = computed(() => {
  if (!page.value) return ''
  return renderMarkdown(page.value.content, allPages.value)
})

async function loadPage(id) {
  page.value = null
  error.value = null
  backlinks.value = []

  try {
    const [pageData, blData, pagesData] = await Promise.all([
      getPage(id),
      getBacklinks(id),
      allPages.value.length ? Promise.resolve(allPages.value) : getPages(),
    ])
    page.value = pageData
    backlinks.value = blData
    allPages.value = pagesData
  } catch (e) {
    error.value = e.message
  }
}

watch(() => props.id, loadPage, { immediate: true })

async function confirmDelete() {
  if (!confirm(`Delete "${page.value.title}"? This cannot be undone.`)) return
  try {
    await deletePage(props.id)
    refreshSidebarTree()
    router.push({ name: 'home' })
  } catch (e) {
    alert('Delete failed: ' + e.message)
  }
}

// Intercept clicks on wiki-links to use Vue Router navigation
function handleClick(e) {
  const link = e.target.closest('a.wiki-link')
  if (link) {
    e.preventDefault()
    const pageId = link.dataset.pageId
    if (pageId) {
      router.push({ name: 'page', params: { id: pageId } })
    }
  }
}
</script>

<style scoped>
.page-view {
  max-width: 800px;
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.btn {
  padding: 0.3rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 0.8rem;
  cursor: pointer;
  text-decoration: none;
  display: inline-block;
}

.btn-secondary {
  background: var(--bg-input);
  color: var(--text-primary);
}

.btn-secondary:hover {
  background: var(--bg-hover);
  text-decoration: none;
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

.page-content :deep(h1) {
  font-size: 1.8rem;
  font-weight: 600;
  margin-bottom: 1rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--border);
}

.page-content :deep(h2) {
  font-size: 1.4rem;
  font-weight: 600;
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
}

.page-content :deep(h3) {
  font-size: 1.15rem;
  font-weight: 600;
  margin-top: 1.25rem;
  margin-bottom: 0.5rem;
}

.page-content :deep(p) {
  margin-bottom: 0.75rem;
}

.page-content :deep(ul),
.page-content :deep(ol) {
  margin-bottom: 0.75rem;
  padding-left: 1.5rem;
}

.page-content :deep(li) {
  margin-bottom: 0.25rem;
}

.page-content :deep(code) {
  background: var(--bg-input);
  padding: 0.15rem 0.35rem;
  border-radius: 3px;
  font-size: 0.85em;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
}

.page-content :deep(pre) {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 1rem;
  margin-bottom: 0.75rem;
  overflow-x: auto;
}

.page-content :deep(pre code) {
  background: none;
  padding: 0;
}

.page-content :deep(blockquote) {
  border-left: 3px solid var(--accent-dim);
  padding-left: 1rem;
  color: var(--text-secondary);
  margin-bottom: 0.75rem;
}

.page-content :deep(a) {
  color: var(--accent);
  text-decoration: none;
}

.page-content :deep(a:hover) {
  text-decoration: underline;
}

.page-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border);
  margin: 1.5rem 0;
}

.page-content :deep(.wiki-link) {
  color: var(--accent);
  cursor: pointer;
  border-bottom: 1px dashed var(--accent-dim);
}

.page-content :deep(.wiki-link.broken) {
  color: var(--error);
  border-bottom-color: var(--error);
}

.backlinks {
  margin-top: 2.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border);
}

.backlinks h3 {
  font-size: 0.85rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
}

.backlinks ul {
  list-style: none;
  padding: 0;
}

.backlinks li {
  margin-bottom: 0.25rem;
}

.backlinks a {
  font-size: 0.9rem;
}

.loading {
  color: var(--text-muted);
  padding: 2rem;
}

.error-state {
  padding: 2rem;
  text-align: center;
}

.error-state h2 {
  font-size: 1.3rem;
  margin-bottom: 0.5rem;
}

.error-state p {
  color: var(--error);
  margin-bottom: 1rem;
}

.home-link {
  color: var(--accent);
  font-size: 0.9rem;
}
</style>
