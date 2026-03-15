import { ref, computed } from 'vue';

// 固定使用同域，不需要后端配置
const fixedUrl = computed(() => window.location.origin);

export const getHostAPIUrl = (): string => '';
export const useHostAPI = () => {
  return {
    currentName: ref('local'),
    currentUrl: fixedUrl,
    apis: ref([{ name: 'local', url: '' }]),
    setCurrent: (_name: string) => {},
    addApi: async (_api: { name: string; url: string }, _skip?: boolean): Promise<boolean> => true,
    deleteApi: (_name: string) => {},
    editApi: (_api: { name: string; url: string }) => {},
    handleUrlQuery: async (_opts?: { errorCb?: () => Promise<void> }): Promise<boolean> => true,
    defaultAPI: '',
  };
};
