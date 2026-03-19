import { ref, watch, computed, onUnmounted } from 'vue';
import { useGlobalStore } from '@/store/global';
import { useAppNotifyStore } from '@/store/appNotify';
import service from '@/api';
import axios from 'axios';
import { initStores } from '@/utils/initApp';

const lsKey = 'hostAPI';

const getHostAPI = (): HostAPIStorage => {
  const item = localStorage.getItem(lsKey);
  if (item) {
    return JSON.parse(item);
  } else {
    setHostAPI({ current: '', apis: [] });
    return getHostAPI();
  }
};

export const getHostAPIUrl = (): string => {
  const { current, apis } = getHostAPI();
  return (
    apis.find(api => api.name === current)?.url ||
    import.meta.env.VITE_API_URL ||
    ''
  ).replace(/\/$/, '');
};

const setHostAPI = (hostAPI: HostAPIStorage) => {
  localStorage.setItem(lsKey, JSON.stringify(hostAPI));
};

export const useHostAPI = () => {
  const defaultAPI = import.meta.env.VITE_API_URL || '';
  const { showNotify } = useAppNotifyStore();
  const apis = ref(getHostAPI().apis);
  const currentName = ref(getHostAPI().current);
  const currentUrl = computed(() => {
    const url = apis.value.find(api => api.name === currentName.value)?.url ?? defaultAPI;
    return url.startsWith('/') ? `${window.location.origin}${url}` : url;
  });

  const stopWatchCurrent = watch(currentName, async (newName, oldName) => {
    if (newName !== oldName) {
      setHostAPI({ ...getHostAPI(), current: newName });
      const url = apis.value.find(api => api.name === newName)?.url ?? defaultAPI;
      const globalStore = useGlobalStore();
      globalStore.setFetchResult(false);
      await globalStore.setHostAPI(url);
    }
  });

  const stopWatchApis = watch(apis, newApis => {
    setHostAPI({ ...getHostAPI(), apis: newApis });
  }, { deep: true });

  onUnmounted(() => {
    stopWatchCurrent();
    stopWatchApis();
  });

  const setCurrent = (name: string) => {
    if (name !== '' && !apis.value.find(api => api.name === name)) return;
    currentName.value = name;
  };

  const addApi = async ({ name, url }: HostAPI, skipConnectionCheck = false): Promise<boolean> => {
    if (apis.value.find(api => api.name === name)) {
      showNotify({ title: 'API 名称重复', type: 'danger' });
      return false;
    }
    if (skipConnectionCheck) {
      apis.value.push({ name, url });
      return true;
    }
    try {
      const res = await axios.get<{ status: 'success' | 'failed' }>(url + '/api/utils/env');
      if (res?.data?.status === 'success') {
        apis.value.push({ name, url });
        return true;
      } else {
        showNotify({ title: '无效的 API 地址，请检查', type: 'danger' });
        return false;
      }
    } catch {
      showNotify({ title: '无效的 API 地址，请检查', type: 'danger' });
      return false;
    }
  };

  const deleteApi = (name: string) => {
    const index = apis.value.findIndex(api => api.name === name);
    if (index === -1) return;
    apis.value.splice(index, 1);
    if (currentName.value === name) {
      currentName.value = apis.value[0]?.name || '';
    }
  };

  const editApi = ({ name, url }: HostAPI) => {
    const index = apis.value.findIndex(api => api.name === name);
    if (index === -1) return;
    apis.value[index].url = url;
  };

  const handleUrlQuery = async ({ errorCb }: { errorCb?: () => Promise<void> } = {}): Promise<boolean> => {
    const query = window.location.search;
    if (!query) {
      await errorCb?.();
      return false;
    }

    const apiUrl = query.slice(1).split('&').map(i => i.split('=')).find(i => i[0] === 'api');
    const magicPathParam = query.slice(1).split('&').map(i => i.split('=')).find(i => i[0] === 'magicpath');
    const globalStore = useGlobalStore();

    if (apiUrl) {
      const url = decodeURIComponent(apiUrl[1]).replace(/\/$/, '');
      if (!url) return await errorCb?.().then(() => false) ?? false;
      const isExist = apis.value.find(api => api.url === url);
      if (isExist) {
        setCurrent(isExist.name);
        localStorage.setItem('backendConfigured', 'true');
        return true;
      }
      const name = url.slice(0, 10) + (Math.random() * 100).toFixed(0);
      await addApi({ name, url }, true);
      setCurrent(name);
      globalStore.setFetchResult(false);
      try {
        const res = await axios.get<{ status: 'success' | 'failed' }>(url + '/api/utils/env');
        if (res?.data?.status === 'success') {
          localStorage.setItem('backendConfigured', 'true');
          globalStore.setFetchResult(true);
        } else {
          globalStore.setFetchResult(false);
        }
      } catch {
        globalStore.setFetchResult(false);
      }
      return true;
    }

    if (magicPathParam) {
      const magicPath = decodeURIComponent(magicPathParam[1]);
      const apiUrl = `${window.location.origin}/${magicPath.replace(/^\/+/, '')}`;
      const isExist = apis.value.find(api => api.url === apiUrl);
      if (isExist) {
        setCurrent(isExist.name);
        localStorage.setItem('backendConfigured', 'true');
        return true;
      }
      const name = `Custom_${new Date().getTime()}`;
      await addApi({ name, url: apiUrl }, true);
      setCurrent(name);
      localStorage.setItem('backendConfigured', 'true');
      return true;
    }

    await errorCb?.();
    return false;
  };

  return {
    currentName,
    currentUrl,
    apis,
    setCurrent,
    addApi,
    deleteApi,
    editApi,
    handleUrlQuery,
    defaultAPI,
  };
};
