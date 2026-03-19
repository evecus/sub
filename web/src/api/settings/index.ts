import { AxiosPromise } from 'axios';

const wrapResponse = (data: any) => ({ data: { status: 'success', data } } as any);

export function useSettingsApi() {
  return {
    getSettings: (): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    setSettings: (data: any): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    syncSettings: (query: any, options?: any): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    restoreSettings: (data: any): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
  };
}
