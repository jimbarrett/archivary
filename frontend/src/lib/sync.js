import { ref, readonly } from 'vue'
import { getSyncStatus, listRemotes, syncNow as apiSyncNow, syncNowDir as apiSyncNowDir } from './api.js'

// Reactive sync state — polled periodically
const syncStatus = ref({})   // map of path -> DirSyncStatus
const syncRemotes = ref([])  // array of RemoteConfig
const syncing = ref(false)   // true while a sync operation is in progress

let pollInterval = null

// Start polling sync status every 30 seconds
export function startSyncPolling() {
  refreshSyncStatus()
  pollInterval = setInterval(refreshSyncStatus, 30000)
}

// Stop polling
export function stopSyncPolling() {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

// Refresh sync status and remotes
export async function refreshSyncStatus() {
  try {
    const [status, remotes] = await Promise.all([
      getSyncStatus(),
      listRemotes(),
    ])
    syncStatus.value = status
    syncRemotes.value = remotes
  } catch (e) {
    // Sync may not be configured — that's fine
    console.debug('Sync status fetch failed:', e)
  }
}

// Trigger sync for all repos
export async function triggerSyncAll() {
  syncing.value = true
  try {
    await apiSyncNow()
    await refreshSyncStatus()
  } finally {
    syncing.value = false
  }
}

// Trigger sync for one repo
export async function triggerSyncDir(path) {
  syncing.value = true
  try {
    await apiSyncNowDir(path)
    await refreshSyncStatus()
  } finally {
    syncing.value = false
  }
}

// Check if a path (file or directory) is inside a synced directory
export function isSynced(path) {
  if (!path) return false
  for (const remote of syncRemotes.value) {
    if (path === remote.path || path.startsWith(remote.path + '/')) {
      return true
    }
  }
  return false
}

// Get sync status for a specific directory path
export function getSyncStatusForPath(path) {
  return syncStatus.value[path] || null
}

// Exported readonly refs for components
export { syncStatus, syncRemotes, syncing }
