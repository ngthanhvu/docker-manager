import { createRouter, createWebHistory } from 'vue-router';
import AuthView from './views/AuthView.vue';
import MainShell from './views/MainShell.vue';
import { authState, ensureAuthBootstrap } from './ui/auth';
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
      path: '/auth',
      name: 'auth',
      component: AuthView,
    },
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

router.beforeEach(async (to) => {
  await ensureAuthBootstrap();

  if (authState.user) {
    if (to.name === 'auth') {
      return { name: 'app', params: { tab: getDefaultTab() } };
    }

    if (to.name === 'app') {
      const tab = typeof to.params.tab === 'string' ? to.params.tab : getDefaultTab();
      if (!validTabs.has(tab)) {
        return { name: 'app', params: { tab: getDefaultTab() } };
      }
    }

    return true;
  }

  if (to.name !== 'auth') {
    return { name: 'auth' };
  }

  return true;
});

export default router;
