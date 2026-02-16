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

export function movePage(id, newPath) {
  return request(`/pages/${encodeURIComponent(id)}/move`, {
    method: 'POST',
    body: JSON.stringify({ new_path: newPath }),
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

export function createDir(path) {
  return request('/dirs', {
    method: 'POST',
    body: JSON.stringify({ path }),
  })
}

export function renameDir(path, newName) {
  return request(`/dirs/${encodeURIComponent(path)}`, {
    method: 'PUT',
    body: JSON.stringify({ name: newName }),
  })
}

export function deleteDir(path, { force = false } = {}) {
  const q = force ? '?force=true' : ''
  return request(`/dirs/${encodeURIComponent(path)}${q}`, {
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

// Encode sync path for URL — "." must be sent as "_root" to avoid
// router path normalization issues.
function syncPath(path) {
  return path === '.' ? '_root' : encodeURIComponent(path)
}

export function getSyncStatus() {
  return request('/sync/status')
}

export function getSyncDirStatus(path) {
  return request(`/sync/status/${syncPath(path)}`)
}

export function syncNow() {
  return request('/sync/now', { method: 'POST' })
}

export function syncNowDir(path) {
  return request(`/sync/now/${syncPath(path)}`, { method: 'POST' })
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
  return request(`/sync/remotes/${syncPath(path)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function removeRemote(path) {
  return request(`/sync/remotes/${syncPath(path)}`, {
    method: 'DELETE',
  })
}

export function getSyncLog(path, n = 20) {
  return request(`/sync/log/${syncPath(path)}?n=${n}`)
}

export function syncCommit(path, message) {
  return request(`/sync/commit/${syncPath(path)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message }),
  })
}

export function listExcludedDirs() {
  return request('/sync/excluded')
}

export function excludeDir(dirName) {
  return request(`/sync/exclude/${encodeURIComponent(dirName)}`, { method: 'POST' })
}

export function includeDir(dirName) {
  return request(`/sync/include/${encodeURIComponent(dirName)}`, { method: 'POST' })
}
