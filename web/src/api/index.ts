import { useAppNotifyStore } from '@/store/appNotify';
import axios, { AxiosError, AxiosPromise, AxiosResponse } from 'axios';

let appNotifyStore = null;

const notifyConfig: { type: 'danger'; duration: number } = {
  type: 'danger',
  duration: 2500,
};

// 全局 401 事件总线
export const authBus = {
  listeners: [] as Array<() => void>,
  onUnauthorized(cb: () => void) {
    this.listeners.push(cb);
  },
  emit() {
    this.listeners.forEach(cb => cb());
  },
};

const service = axios.create({
  baseURL: '',
  timeout: 50000,
  headers: { 'Content-Type': 'application/json' },
});

service.interceptors.response.use(
  (response: AxiosResponse<SucceedResponse>): AxiosPromise<SucceedResponse> => {
    return Promise.resolve(response);
  },
  (e: AxiosError<ErrorResponse>): AxiosPromise<ErrorResponse | undefined> => {
    // 401 → 通知 App.vue 显示登录页
    if (e.response?.status === 401) {
      const url = e.config?.url || '';
      if (!url.includes('/auth/')) {
        authBus.emit();
      }
      return Promise.reject(e.response);
    }

    if (e.config?.url?.startsWith('/api/sub/flow') || e.config?.url?.startsWith('https://api.github.com/'))
      return Promise.resolve(e.response);

    if (!appNotifyStore) appNotifyStore = useAppNotifyStore();

    if (appNotifyStore) {
      if (!e.response || e.response.status === 0) {
        appNotifyStore.showNotify({
          title: '网络错误或后端异常',
          content: e.message,
          ...notifyConfig,
        });
        return Promise.reject(e.response);
      } else {
        const content = 'type: ' + (e.response.data?.error?.type || e.response.status);
        appNotifyStore.showNotify({
          title: e.response.data?.error?.message || '请求失败',
          content,
          ...notifyConfig,
        });
        return Promise.resolve(e.response);
      }
    }
  }
);

export default service;
