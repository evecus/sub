<template>
  <NavBar />
  <main class="page-body">
    <router-view />
  </main>
  <MagicPathDialog
    v-model="showMagicPathDialog"
    :url-api-error="urlApiError"
    :url-api-value="urlApiValue"
    :connection-cycle="connectionCheckCycle"
  />
</template>

<script setup lang="ts">
import NavBar from "@/components/NavBar.vue";
import MagicPathDialog from "@/components/MagicPathDialog.vue";
import { useThemes } from "@/hooks/useThemes";
import { useGlobalStore } from "@/store/global";
import { useSubsStore } from "@/store/subs";
import { getFlowsUrlList } from "@/utils/getFlowsUrlList";
import { initStores } from "@/utils/initApp";
import { storeToRefs } from "pinia";
import { ref, watchEffect, onMounted } from "vue";
import { useHostAPI } from "@/hooks/useHostAPI";
import { useRoute, useRouter } from "vue-router";

const subsStore = useSubsStore();
const globalStore = useGlobalStore();
const route = useRoute();
const router = useRouter();
const { subs, flows } = storeToRefs(subsStore);
const allLength = ref(null);

const showMagicPathDialog = ref(false);
const isBackendCheckInProgress = ref(true);
const connectionCheckCycle = ref(Date.now());

type NavigatorExtend = Navigator & { standalone?: boolean };
const navigator: NavigatorExtend = window.navigator;

function isLegacyDevices() {
  const w = window.screen.width, h = window.screen.height;
  return (w === 375 && h === 667) || (w === 414 && h === 736);
}
globalStore.setBottomSafeArea(navigator.standalone && !isLegacyDevices() ? 18 : 0);

const { handleUrlQuery } = useHostAPI();
const urlApiConfigSuccess = ref(false);
const urlApiError = ref('');
const urlApiValue = ref('');

const processUrlApiConfig = async () => {
  isBackendCheckInProgress.value = true;
  connectionCheckCycle.value = Date.now();

  const query = window.location.search;
  let hasUrlParams = false;

  if (query) {
    const hasApiParam = query.slice(1).split('&').map(i => i.split('=')).find(i => i[0] === 'api');
    const hasMagicPathParam = query.slice(1).split('&').map(i => i.split('=')).find(i => i[0] === 'magicpath');

    if (hasApiParam) {
      urlApiValue.value = decodeURIComponent(hasApiParam[1]).replace(/\/$/, '');
      urlApiError.value = '通过 URL 参数指定的 API 地址连接失败，请检查地址是否正确';
      hasUrlParams = true;
    } else if (hasMagicPathParam) {
      const magicPath = decodeURIComponent(hasMagicPathParam[1]);
      urlApiValue.value = `${window.location.origin}/${magicPath.replace(/^\/+/, '')}`;
      urlApiError.value = '通过 URL 参数指定的 magicpath 连接失败，请检查路径是否正确';
      hasUrlParams = true;
    }
  }

  const result = await handleUrlQuery({
    errorCb: async () => {
      try {
        await initStores(true, true, false);
        const hasBackendEnv = Object.keys(globalStore.env).length > 0 && globalStore.env.backend;
        if (hasBackendEnv) {
          showMagicPathDialog.value = false;
          localStorage.setItem('backendConfigured', 'true');
          globalStore.setFetchResult(true);
        } else {
          globalStore.setFetchResult(false);
          const skippedCycle = parseInt(sessionStorage.getItem('skippedConnectionCycle') || '0');
          if (route.path === '/subs' && skippedCycle !== connectionCheckCycle.value) {
            showMagicPathDialog.value = true;
          }
        }
      } catch (e) {
        globalStore.setFetchResult(false);
        const skippedCycle = parseInt(sessionStorage.getItem('skippedConnectionCycle') || '0');
        if (route.path === '/subs' && skippedCycle !== connectionCheckCycle.value) {
          showMagicPathDialog.value = true;
        }
      }
    },
  });

  if (result) {
    urlApiConfigSuccess.value = true;
    const fetchResult = globalStore.fetchResult;
    const skippedCycle = parseInt(sessionStorage.getItem('skippedConnectionCycle') || '0');
    if (fetchResult) {
      urlApiError.value = '';
      showMagicPathDialog.value = false;
    } else if (hasUrlParams && skippedCycle !== connectionCheckCycle.value) {
      if (route.path === '/subs') showMagicPathDialog.value = true;
    }
    await initStores(false, true, false);
  }

  isBackendCheckInProgress.value = false;
};

processUrlApiConfig();
useThemes();

watchEffect(() => {
  allLength.value = getFlowsUrlList(subs.value).length;
  const currentLength = Object.keys(flows.value).length;
  globalStore.setFlowFetching(allLength.value !== currentLength);
});

onMounted(() => {
  router.afterEach((to) => {
    if (to.path === '/subs') checkAndShowMagicPathDialog();
  });
  checkAndShowMagicPathDialog();
});

function checkAndShowMagicPathDialog() {
  if (isBackendCheckInProgress.value || showMagicPathDialog.value) return;
  const skippedCycle = parseInt(sessionStorage.getItem('skippedConnectionCycle') || '0');
  if (skippedCycle === connectionCheckCycle.value) return;
  if (!globalStore.fetchResult && !globalStore.isLoading && route.path === '/subs') {
    showMagicPathDialog.value = true;
  }
}
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
</style>
