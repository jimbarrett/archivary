const BASE = '/api'

async function request(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

export function getPages(dir = '') {
  const params = dir ? `?dir=${encodeURIComponent(dir)}` : ''
  return request(`/pages${params}`)
}

export function getPage(id) {
  return request(`/pages/${encodeURIComponent(id)}`)
}

export function createPage(data) {
  return request('/pages', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updatePage(id, data) {
  return request(`/pages/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function deletePage(id) {
  return request(`/pages/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function searchPages(query) {
  return request(`/search?q=${encodeURIComponent(query)}`)
}

export function getTree() {
  return request('/tree')
}

export function getBacklinks(id) {
  return request(`/pages/${encodeURIComponent(id)}/backlinks`)
}
