import { ref, computed } from 'vue';

// 固定使用同域，不需要后端配置
const fixedUrl = computed(() => window.location.origin);

export const getHostAPIUrl = (): string => '';
export const useHostAPI = () => {
  return {
    currentName: ref('local'),
    currentUrl: fixedUrl,
    apis: ref([{ name: 'local', url: '' }]),
    setCurrent: () => {},
    addApi: async () => true,
    deleteApi: () => {},
    editApi: () => {},
    handleUrlQuery: async () => true,
    defaultAPI: '',
  };
};
