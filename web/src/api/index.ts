import { useAppNotifyStore } from '@/store/appNotify';
import axios, { AxiosError, AxiosPromise, AxiosResponse } from 'axios';
import { getHostAPIUrl } from '@/hooks/useHostAPI';

let appNotifyStore = null;

const notifyConfig: { type: 'danger'; duration: number } = {
  type: 'danger',
  duration: 2500,
};

const service = axios.create({
  baseURL: getHostAPIUrl(),
  timeout: 50000,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true, // 携带 session cookie
});

service.interceptors.response.use(
  (response: AxiosResponse<SucceedResponse>): AxiosPromise<SucceedResponse> => {
    return Promise.resolve(response);
  },
  (e: AxiosError<ErrorResponse>): AxiosPromise<ErrorResponse | undefined> => {
    if (e.config?.url?.startsWith('/api/sub/flow') || e.config?.url?.startsWith('https://api.github.com/'))
      return Promise.resolve(e.response);

    if (!appNotifyStore) appNotifyStore = useAppNotifyStore();

    if (appNotifyStore) {
      if (!e.response || e.response.status === 0) {
        appNotifyStore.showNotify({
          title: '网络错误或后端异常，无法连接后端服务\n',
          content: 'code: ' + (e.response?.status ?? 0) + ' msg: ' + e.message,
          ...notifyConfig,
        });
        return Promise.reject(e.response);
      } else {
        let content = 'type: ' + e.response.data?.error?.type;
        if (e.response.data?.error?.details)
          content += '\n' + e.response.data.error.details;
        appNotifyStore.showNotify({
          title: e.response.data?.error?.message,
          content,
          ...notifyConfig,
        });
        return Promise.resolve(e.response);
      }
    }
  }
);

export default service;
