import request from '@/api';
import { AxiosPromise } from 'axios';

export function useEnvApi() {
  return {
    // 返回模拟的 env，让前端认为后端已连接
    getEnv: (): AxiosPromise<MyAxiosRes> => {
      return Promise.resolve({
        data: {
          status: 'success',
          data: {
            backend: 'Node',
            version: '2.16.21',
            feature: { share: true },
          },
        },
      } as any);
    },
    refreshCache: (): AxiosPromise<MyAxiosRes> => {
      return Promise.resolve({
        data: { status: 'success', data: {} },
      } as any);
    },
  };
}
