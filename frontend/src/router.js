import { createRouter, createWebHistory } from 'vue-router'
import PageView from './views/PageView.vue'
import EditView from './views/EditView.vue'
import SearchView from './views/SearchView.vue'
import HomeView from './views/HomeView.vue'
import DirectoryView from './views/DirectoryView.vue'
import NotFoundView from './views/NotFoundView.vue'

const routes = [
  { path: '/', name: 'home', component: HomeView },
  { path: '/dir/:path(.*)', name: 'directory', component: DirectoryView, props: true },
  { path: '/page/:id', name: 'page', component: PageView, props: true },
  { path: '/page/:id/edit', name: 'edit', component: EditView, props: true },
  { path: '/new', name: 'new', component: EditView },
  { path: '/search', name: 'search', component: SearchView },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
