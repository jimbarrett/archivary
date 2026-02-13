import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
})

/**
 * Render markdown to HTML, resolving [[uuid]] wiki-links to page links.
 * @param {string} content - raw markdown
 * @param {Array} pages - list of all pages (used to resolve IDs to titles)
 * @returns {string} HTML
 */
export function renderMarkdown(content, pages = []) {
  // Build a lookup map of page ID -> page info
  const pageMap = {}
  for (const p of pages) {
    pageMap[p.id] = p
  }

  // Replace [[uuid]] and [[uuid|label]] with HTML links before rendering
  const resolved = content.replace(
    /\[\[([^\]|]+)(?:\|([^\]]+))?\]\]/g,
    (match, id, label) => {
      const target = pageMap[id]
      if (target) {
        const title = label || target.title
        return `<a href="/page/${encodeURIComponent(id)}" class="wiki-link" data-page-id="${id}">${escapeHtml(title)}</a>`
      }
      // Broken link — target page doesn't exist
      const display = label || id
      return `<span class="wiki-link broken" title="Page not found: ${escapeHtml(id)}">${escapeHtml(display)}</span>`
    }
  )

  return md.render(resolved)
}

function escapeHtml(str) {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
