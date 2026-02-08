<template>
  <div class="edit-view">
    <div class="edit-header">
      <div class="edit-title">
        <span v-if="page">Editing: {{ page.title || page.path }}</span>
        <span v-else-if="isNew">New Page</span>
      </div>
      <div class="edit-actions">
        <button class="btn btn-secondary" @click="cancel">Cancel</button>
        <button class="btn btn-primary" @click="save" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save' }}
        </button>
      </div>
    </div>

    <div class="toolbar">
      <button @click="insertFormat('heading')" title="Heading (H2)">H</button>
      <button @click="insertFormat('bold')" title="Bold">B</button>
      <button @click="insertFormat('italic')" title="Italic"><em>I</em></button>
      <button @click="insertFormat('code')" title="Inline code">&lt;/&gt;</button>
      <button @click="insertFormat('codeblock')" title="Code block">{ }</button>
      <button @click="insertFormat('list')" title="Bullet list">&#8226;</button>
      <button @click="insertFormat('olist')" title="Numbered list">1.</button>
      <button @click="insertFormat('quote')" title="Blockquote">&gt;</button>
      <button @click="insertFormat('hr')" title="Horizontal rule">&#8212;</button>
      <button @click="showLinkPicker = true" title="Insert page link">&#128279;</button>
    </div>

    <div class="editor-area">
      <div class="editor-pane">
        <textarea
          ref="textarea"
          v-model="content"
          placeholder="Write your markdown here..."
          @keydown="onKeydown"
          spellcheck="true"
        ></textarea>
      </div>
      <div class="preview-pane">
        <div class="preview-label">Preview</div>
        <div class="preview-content" v-html="renderedPreview"></div>
      </div>
    </div>

    <PageLinkPicker
      v-if="showLinkPicker"
      @select="insertPageLink"
      @close="showLinkPicker = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getPage, updatePage, createPage, getPages } from '../lib/api.js'
import { refreshSidebarTree } from '../lib/events.js'
import { renderMarkdown } from '../lib/markdown.js'
import PageLinkPicker from '../components/PageLinkPicker.vue'

const props = defineProps({
  id: { type: String, default: null },
})

const router = useRouter()
const route = useRoute()

const page = ref(null)
const content = ref('')
const saving = ref(false)
const allPages = ref([])
const showLinkPicker = ref(false)
const textarea = ref(null)

const isNew = computed(() => !props.id)

// For new pages, path comes from query param
const newPath = computed(() => route.query.path || '')

const renderedPreview = computed(() => {
  return renderMarkdown(content.value, allPages.value)
})

onMounted(async () => {
  try {
    allPages.value = await getPages()
  } catch (e) {
    console.error('Failed to load pages for link resolution:', e)
  }

  if (props.id) {
    try {
      page.value = await getPage(props.id)
      content.value = page.value.content
    } catch (e) {
      console.error('Failed to load page:', e)
    }
  }
})

async function save() {
  saving.value = true
  try {
    if (isNew.value) {
      const result = await createPage({
        content: content.value,
        path: newPath.value,
      })
      refreshSidebarTree()
      router.push({ name: 'page', params: { id: result.id } })
    } else {
      await updatePage(props.id, { content: content.value })
      refreshSidebarTree()
      router.push({ name: 'page', params: { id: props.id } })
    }
  } catch (e) {
    alert('Save failed: ' + e.message)
  } finally {
    saving.value = false
  }
}

function cancel() {
  if (props.id) {
    router.push({ name: 'page', params: { id: props.id } })
  } else {
    router.push({ name: 'home' })
  }
}

function onKeydown(e) {
  // Ctrl+S / Cmd+S to save
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault()
    save()
    return
  }

  // Tab inserts spaces instead of changing focus
  if (e.key === 'Tab') {
    e.preventDefault()
    insertAtCursor('  ')
  }
}

function insertAtCursor(text) {
  const el = textarea.value
  if (!el) return
  const start = el.selectionStart
  const end = el.selectionEnd
  content.value = content.value.substring(0, start) + text + content.value.substring(end)
  // Restore cursor position after Vue re-renders
  requestAnimationFrame(() => {
    el.selectionStart = el.selectionEnd = start + text.length
    el.focus()
  })
}

function wrapSelection(before, after) {
  const el = textarea.value
  if (!el) return
  const start = el.selectionStart
  const end = el.selectionEnd
  const selected = content.value.substring(start, end)
  const replacement = before + (selected || 'text') + after
  content.value = content.value.substring(0, start) + replacement + content.value.substring(end)
  requestAnimationFrame(() => {
    if (selected) {
      el.selectionStart = start + before.length
      el.selectionEnd = start + before.length + selected.length
    } else {
      el.selectionStart = start + before.length
      el.selectionEnd = start + before.length + 4 // select "text"
    }
    el.focus()
  })
}

function insertLinePrefix(prefix) {
  const el = textarea.value
  if (!el) return
  const start = el.selectionStart
  // Find the start of the current line
  const lineStart = content.value.lastIndexOf('\n', start - 1) + 1
  content.value = content.value.substring(0, lineStart) + prefix + content.value.substring(lineStart)
  requestAnimationFrame(() => {
    el.selectionStart = el.selectionEnd = start + prefix.length
    el.focus()
  })
}

function insertFormat(type) {
  switch (type) {
    case 'heading':
      insertLinePrefix('## ')
      break
    case 'bold':
      wrapSelection('**', '**')
      break
    case 'italic':
      wrapSelection('*', '*')
      break
    case 'code':
      wrapSelection('`', '`')
      break
    case 'codeblock':
      wrapSelection('\n```\n', '\n```\n')
      break
    case 'list':
      insertLinePrefix('- ')
      break
    case 'olist':
      insertLinePrefix('1. ')
      break
    case 'quote':
      insertLinePrefix('> ')
      break
    case 'hr':
      insertAtCursor('\n---\n')
      break
  }
}

function insertPageLink(page) {
  showLinkPicker.value = false
  insertAtCursor(`[[${page.id}]]`)
}
</script>

<style scoped>
.edit-view {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 4rem);
}

.edit-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.edit-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.edit-actions {
  display: flex;
  gap: 0.5rem;
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
  color: var(--text-primary);
}

.btn-primary:hover {
  background: var(--accent);
  color: var(--bg-primary);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toolbar {
  display: flex;
  gap: 2px;
  padding: 0.35rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 4px 4px 0 0;
  border-bottom: none;
}

.toolbar button {
  padding: 0.25rem 0.5rem;
  background: none;
  border: 1px solid transparent;
  border-radius: 3px;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 0.8rem;
  min-width: 1.8rem;
  text-align: center;
}

.toolbar button:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  border-color: var(--border);
}

.editor-area {
  display: flex;
  flex: 1;
  min-height: 0;
  border: 1px solid var(--border);
  border-radius: 0 0 4px 4px;
  overflow: hidden;
}

.editor-pane {
  flex: 1;
  display: flex;
  border-right: 1px solid var(--border);
}

.editor-pane textarea {
  flex: 1;
  padding: 1rem;
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: none;
  outline: none;
  resize: none;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 0.85rem;
  line-height: 1.6;
  tab-size: 2;
}

.editor-pane textarea::placeholder {
  color: var(--text-muted);
}

.preview-pane {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  background: var(--bg-primary);
}

.preview-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  margin-bottom: 0.75rem;
}

.preview-content :deep(h1) {
  font-size: 1.6rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
  padding-bottom: 0.4rem;
  border-bottom: 1px solid var(--border);
}

.preview-content :deep(h2) {
  font-size: 1.25rem;
  font-weight: 600;
  margin-top: 1.25rem;
  margin-bottom: 0.5rem;
}

.preview-content :deep(p) {
  margin-bottom: 0.6rem;
}

.preview-content :deep(ul),
.preview-content :deep(ol) {
  margin-bottom: 0.6rem;
  padding-left: 1.5rem;
}

.preview-content :deep(code) {
  background: var(--bg-input);
  padding: 0.15rem 0.35rem;
  border-radius: 3px;
  font-size: 0.85em;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
}

.preview-content :deep(pre) {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0.75rem;
  margin-bottom: 0.6rem;
  overflow-x: auto;
}

.preview-content :deep(pre code) {
  background: none;
  padding: 0;
}

.preview-content :deep(blockquote) {
  border-left: 3px solid var(--accent-dim);
  padding-left: 1rem;
  color: var(--text-secondary);
  margin-bottom: 0.6rem;
}

.preview-content :deep(a),
.preview-content :deep(.wiki-link) {
  color: var(--accent);
}

.preview-content :deep(.wiki-link.broken) {
  color: var(--error);
}
</style>
