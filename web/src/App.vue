<template>
  <div v-if="!authReady" class="auth-loading">
    <div class="spinner"></div>
  </div>
  <template v-else-if="!isLoggedIn">
    <LoginPage @login="onLogin" />
  </template>
  <template v-else>
    <NavBar />
    <main class="page-body">
      <router-view />
    </main>
  </template>
</template>

<script setup lang="ts">
import NavBar from "@/components/NavBar.vue";
import LoginPage from "@/views/Login.vue";
import { useThemes } from "@/hooks/useThemes";
import { useGlobalStore } from "@/store/global";
import { useSubsStore } from "@/store/subs";
import { getFlowsUrlList } from "@/utils/getFlowsUrlList";
import { initStores } from "@/utils/initApp";
import { storeToRefs } from "pinia";
import { ref, watchEffect, onMounted } from "vue";

const subsStore = useSubsStore();
const globalStore = useGlobalStore();
const { subs, flows } = storeToRefs(subsStore);
const allLength = ref(null);

// ── 登录状态 ──────────────────────────────────────────
const authReady = ref(false);
const isLoggedIn = ref(false);

const checkAuth = async () => {
  try {
    const res = await fetch('/api/auth/check');
    isLoggedIn.value = res.ok;
  } catch {
    isLoggedIn.value = false;
  }
  authReady.value = true;
  if (isLoggedIn.value) {
    await boot();
  }
};

const onLogin = async () => {
  isLoggedIn.value = true;
  await boot();
};

// 全局 401 拦截
const origFetch = window.fetch.bind(window);
window.fetch = async (...args) => {
  const res = await origFetch(...args);
  const url = typeof args[0] === 'string' ? args[0] : '';
  if (res.status === 401 && !url.includes('/auth/')) {
    isLoggedIn.value = false;
  }
  return res;
};

const boot = async () => {
  globalStore.setBottomSafeArea(0);
  globalStore.setFetchResult(true);
  await initStores(false, true, false);
  localStorage.setItem('backendConfigured', 'true');
};

checkAuth();

// ── 主题 ──────────────────────────────────────────────
useThemes();

// ── 流量状态 ──────────────────────────────────────────
watchEffect(() => {
  allLength.value = getFlowsUrlList(subs.value).length;
  const currentLength = Object.keys(flows.value).length;
  globalStore.setFlowFetching(allLength.value !== currentLength);
});
</script>

<style lang="scss">
#app {
  font-family: "Roboto", "nutui-iconfont", "Noto Sans", Arial, "PingFang SC",
    "Source Han Sans SC", "Source Han Sans CN", "Microsoft YaHei", "ST Heiti",
    SimHei, sans-serif;
  display: flex;
  align-items: center;
  flex-direction: column;
  position: absolute;
  min-height: 100%;
  width: 100%;
  background: var(--background-color);
  overflow: hidden;

  .page-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: auto;
    width: 100%;
    @include responsive-container-width;
  }

  overflow-y: auto;
}

.auth-loading {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--background-color);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid rgba(0,0,0,0.1);
  border-top-color: var(--primary-color, #478EF2);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }
</style>
