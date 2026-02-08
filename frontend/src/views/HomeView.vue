<template>
  <div class="home">
    <h1>Welcome to Archivary</h1>
    <p>Select a page from the sidebar, or use search to find something.</p>
  </div>
</template>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 60vh;
  text-align: center;
}

.home h1 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.home p {
  color: var(--text-secondary);
  font-size: 0.95rem;
}
</style>
