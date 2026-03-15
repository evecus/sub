import { AxiosPromise } from 'axios';

const wrapResponse = (data: any) => ({ data: { status: 'success', data } } as any);

export function useArtifactsApi() {
  return {
    getArtifacts: (): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse([])),
    getOneArtifact: (name: string): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse(null)),
    syncOneArtifact: (name: string): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    createArtifact: (data: any): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    editArtifact: (name: string, data: any): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    deleteArtifact: (name: string): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    syncAllArtifact: (): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
    restoreArtifacts: (): AxiosPromise<MyAxiosRes> => Promise.resolve(wrapResponse({})),
  };
}
