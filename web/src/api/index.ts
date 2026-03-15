import { useAppNotifyStore } from '@/store/appNotify';
import axios, { AxiosError, AxiosPromise, AxiosResponse } from 'axios';

let appNotifyStore = null;

const notifyConfig: { type: 'danger'; duration: number } = {
  type: 'danger',
  duration: 2500,
};

// 固定使用相对路径，与 Go 后端同域
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
    // 401 → 跳转登录
    if (e.response?.status === 401) {
      window.location.href = '/login';
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
