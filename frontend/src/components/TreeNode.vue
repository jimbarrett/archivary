<template>
  <div class="tree-node">
    <div
      v-if="node.is_dir"
      class="tree-dir"
      @click="onDirClick"
    >
      <span class="tree-arrow" :class="{ expanded }">&#9654;</span>
      <span class="tree-label">{{ node.name }}</span>
    </div>

    <router-link
      v-else
      :to="{ name: 'page', params: { id: node.page_id } }"
      class="tree-file"
      active-class="active"
    >
      <span class="tree-icon">&#9643;</span>
      <span class="tree-label">{{ node.name }}</span>
    </router-link>

    <div v-if="node.is_dir && expanded && node.children" class="tree-children">
      <TreeNode
        v-for="child in sortedChildren"
        :key="child.path"
        :node="child"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const props = defineProps({
  node: { type: Object, required: true },
})

const expanded = ref(true)

const sortedChildren = computed(() => {
  if (!props.node.children) return []
  return [...props.node.children].sort((a, b) => {
    // Directories first, then alphabetical
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
})

function onDirClick() {
  expanded.value = !expanded.value
  if (props.node.path) {
    router.push({ name: 'directory', params: { path: props.node.path } })
  }
}
</script>

<style scoped>
.tree-node {
  user-select: none;
}

.tree-dir {
  display: flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  cursor: pointer;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.tree-dir:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.tree-arrow {
  font-size: 0.55rem;
  margin-right: 0.4rem;
  transition: transform 0.15s;
  display: inline-block;
  width: 0.8rem;
  text-align: center;
}

.tree-arrow.expanded {
  transform: rotate(90deg);
}

.tree-file {
  display: flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-decoration: none;
  border-left: 2px solid transparent;
}

.tree-file:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  text-decoration: none;
}

.tree-file.active {
  background: var(--bg-active);
  color: var(--accent);
  border-left-color: var(--accent);
}

.tree-icon {
  margin-right: 0.4rem;
  font-size: 0.75rem;
  width: 0.8rem;
  text-align: center;
}

.tree-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tree-children {
  padding-left: 0.75rem;
}
</style>
