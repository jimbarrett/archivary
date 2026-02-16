import { ref, computed } from 'vue'
import { getSyncStatus, listRemotes, listExcludedDirs, syncNow as apiSyncNow } from './api.js'

// Reactive sync state — polled periodically
const syncStatus = ref({})   // map of path -> DirSyncStatus
const syncRemotes = ref([])  // array of RemoteConfig
const syncing = ref(false)   // true while a sync operation is in progress
const excludedDirs = ref([]) // array of excluded directory names

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

// Refresh sync status, remotes, and excluded dirs
export async function refreshSyncStatus() {
  try {
    const [status, remotes, excluded] = await Promise.all([
      getSyncStatus(),
      listRemotes(),
      listExcludedDirs(),
    ])
    syncStatus.value = status
    syncRemotes.value = remotes
    excludedDirs.value = excluded
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

// Computed: is the root workspace synced?
export const isRootSynced = computed(() =>
  syncRemotes.value.some(r => r.path === '.')
)

// Computed: sync status for the root repo
export const rootSyncStatus = computed(() =>
  syncStatus.value['.'] || null
)

// Computed: remote config for the root repo
export const rootRemote = computed(() =>
  syncRemotes.value.find(r => r.path === '.') || null
)

// Computed: does any sync remote exist?
export const hasSyncRemotes = computed(() => syncRemotes.value.length > 0)

// Check if a directory name is excluded from sync
export function isDirExcluded(dirName) {
  return excludedDirs.value.includes(dirName)
}

// Check if a path is synced: root is active and top-level dir not excluded
export function isSynced(path) {
  if (!path || !isRootSynced.value) return false
  const topDir = path.includes('/') ? path.split('/')[0] : path
  return !isDirExcluded(topDir)
}

// Get sync status for a specific directory path
export function getSyncStatusForPath(path) {
  return syncStatus.value[path] || null
}

// Exported refs for components
export { syncStatus, syncRemotes, syncing, excludedDirs }
