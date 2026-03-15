import { AxiosPromise } from 'axios';

const wrapResponse = (data: any) => ({
  data: { status: 'success', data },
} as any);

// 我们没有文件管理功能，返回空数据保持前端不崩溃
export function useFilesApi() {
  return {
    getFiles: (): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse([])),
    getWholeFiles: (): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse([])),
    getFile: (name: string): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse(null)),
    getWholeFile: (name: string): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse(null)),
    createFile: (data: any): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    editFile: (name: string, data: any): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    deleteFile: (name: string): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
  };
}
