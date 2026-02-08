<template>
  <div class="app">
    <Sidebar />
    <main class="content">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import Sidebar from './components/Sidebar.vue'
</script>

<style>
:root {
  --bg-primary: #1a1b26;
  --bg-secondary: #16161e;
  --bg-sidebar: #1f2029;
  --bg-hover: #292a37;
  --bg-active: #33354a;
  --bg-input: #24253a;
  --text-primary: #c0caf5;
  --text-secondary: #787c99;
  --text-muted: #565a7e;
  --border: #2e2f3e;
  --accent: #7aa2f7;
  --accent-hover: #89b4fa;
  --accent-dim: #3d59a1;
  --success: #9ece6a;
  --warning: #e0af68;
  --error: #f7768e;
  --mark-bg: #3d59a1;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto,
    'Helvetica Neue', sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  line-height: 1.6;
}

.app {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.content {
  flex: 1;
  overflow-y: auto;
  padding: 2rem 3rem;
}

a {
  color: var(--accent);
  text-decoration: none;
}

a:hover {
  color: var(--accent-hover);
  text-decoration: underline;
}

mark {
  background: var(--mark-bg);
  color: var(--text-primary);
  padding: 0 2px;
  border-radius: 2px;
}

::-webkit-scrollbar {
  width: 8px;
}

::-webkit-scrollbar-track {
  background: var(--bg-primary);
}

::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}
</style>
