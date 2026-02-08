<template>
  <div class="not-found">
    <h1>Page not found</h1>
    <p>The page you're looking for doesn't exist.</p>
    <router-link to="/" class="home-link">Go home</router-link>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 60vh;
  text-align: center;
}

.not-found h1 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.not-found p {
  color: var(--text-secondary);
  margin-bottom: 1rem;
}

.home-link {
  color: var(--accent);
  font-size: 0.9rem;
}
</style>
