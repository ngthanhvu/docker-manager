import { createRouter, createWebHistory } from 'vue-router';
import MainShell from './views/MainShell.vue';
import { loadStoredString } from './ui/viewState';

const validTabs = new Set(['dashboard', 'containers', 'images', 'volumes', 'networks', 'compose', 'settings']);

const getDefaultTab = () => {
  const stored = loadStoredString('dock-manager.active-tab', 'dashboard');
  return validTabs.has(stored) ? stored : 'dashboard';
};

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/app/:tab?',
      name: 'app',
      component: MainShell,
      props: true,
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: () => ({ name: 'app', params: { tab: getDefaultTab() } }),
    },
  ],
});

export default router;
