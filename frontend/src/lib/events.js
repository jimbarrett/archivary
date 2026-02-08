import { ref } from 'vue'

// Simple reactive counter that triggers sidebar tree refresh when incremented
export const treeVersion = ref(0)

export function refreshSidebarTree() {
  treeVersion.value++
}
