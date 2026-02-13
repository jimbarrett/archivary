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

export function renameDir(path, newName) {
  return request(`/dirs/${encodeURIComponent(path)}`, {
    method: 'PUT',
    body: JSON.stringify({ name: newName }),
  })
}

export function deleteDir(path) {
  return request(`/dirs/${encodeURIComponent(path)}`, {
    method: 'DELETE',
  })
}

export function checkPath(path) {
  return request(`/pages/check-path?path=${encodeURIComponent(path)}`)
}

export function reindex() {
  return request('/reindex', { method: 'POST' })
}

// --- Sync API ---

export function getSyncStatus() {
  return request('/sync/status')
}

export function getSyncDirStatus(path) {
  return request(`/sync/status/${encodeURIComponent(path)}`)
}

export function syncNow() {
  return request('/sync/now', { method: 'POST' })
}

export function syncNowDir(path) {
  return request(`/sync/now/${encodeURIComponent(path)}`, { method: 'POST' })
}

export function listRemotes() {
  return request('/sync/remotes')
}

export function addRemote(data) {
  return request('/sync/remotes', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateRemote(path, data) {
  return request(`/sync/remotes/${encodeURIComponent(path)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function removeRemote(path) {
  return request(`/sync/remotes/${encodeURIComponent(path)}`, {
    method: 'DELETE',
  })
}

export function getSyncLog(path, n = 20) {
  return request(`/sync/log/${encodeURIComponent(path)}?n=${n}`)
}

export function syncCommit(path, message) {
  return request(`/sync/commit/${encodeURIComponent(path)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message }),
  })
}
