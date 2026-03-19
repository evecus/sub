import { defineStore } from 'pinia';
import { useEnvApi } from '@/api/env';
import { initStores } from '@/utils/initApp';
import service from '@/api';
import { getHostAPIUrl } from '@/hooks/useHostAPI';

const envApi = useEnvApi();

export const useGlobalStore = defineStore('globalStore', {
  state: (): GlobalStoreState => {
    return {
      subProgressStyle: localStorage.getItem('subProgressStyle') || 'hidden',
      gistUpload: localStorage.getItem('gistUpload') || 'base64',
      isLoading: true,
      isFlowFetching: true,
      fetchResult: false,
      bottomSafeArea: 0,
      isDefaultIcon: localStorage.getItem('isDefaultIcon') === '1',
      isDarkMode: false,
      env: {},
      isDockerDeployment: false,
      isSimpleMode: localStorage.getItem('isSimpleMode') === '1',
      isLeftRight: localStorage.getItem('isLr') === '1',
      isIconColor: localStorage.getItem('iconColor') === '1',
      isEditorCommon: localStorage.getItem('iseditorCommon') !== '1',
      isSimpleReicon: localStorage.getItem('isSimpleReicon') === '1',
      showFloatingRefreshButton: localStorage.getItem('showFloatingRefreshButton') === '1',
      istabBar: localStorage.getItem('istabBar') === '1',
      istabBar2: localStorage.getItem('istabBar2') === '1',
      ishostApi: getHostAPIUrl(),
      savedPositions: {},
      defaultIconCollection: localStorage.getItem('defaultIconCollection') || '',
      defaultIconCollections: [
        { text: "Koolson/QureColor", value: "https://raw.githubusercontent.com/Koolson/Qure/master/Other/QureColor.json" },
        { text: "Koolson/QureColor-All", value: "https://raw.githubusercontent.com/Koolson/Qure/master/Other/QureColor-All.json" },
        { text: "Orz-3/mini", value: "https://raw.githubusercontent.com/Orz-3/mini/master/mini.json" },
        { text: "Orz-3/miniColor", value: "https://raw.githubusercontent.com/Orz-3/mini/master/miniColor.json" },
      ],
      customIconCollections: localStorage.getItem("customIconCollections")
        ? JSON.parse(localStorage.getItem("customIconCollections"))
        : [],
    };
  },
  getters: {},
  actions: {
    setSubProgressStyle(style: string) {
      if (style && style !== 'hidden') localStorage.setItem('subProgressStyle', style);
      else localStorage.removeItem('subProgressStyle');
      this.subProgressStyle = style;
    },
    setGistUpload(style: string) {
      if (style && style !== 'base64') localStorage.setItem('gistUpload', style);
      else localStorage.removeItem('gistUpload');
      this.gistUpload = style;
    },
    setBottomSafeArea(height: number) {
      this.bottomSafeArea = height;
    },
    setLoading(isLoading: boolean) {
      this.isLoading = isLoading;
    },
    setFlowFetching(isFlowFetching: boolean) {
      this.isFlowFetching = isFlowFetching;
    },
    setFetchResult(fetchResult: boolean) {
      this.fetchResult = fetchResult;
    },
    setIsDarkMode(isDarkMode: boolean) {
      this.isDarkMode = isDarkMode;
    },
    async setHostAPI(hostApi: string) {
      this.ishostApi = hostApi;
      service.defaults.baseURL = hostApi;
      await initStores(true, true, true);
    },
    async setEnv() {
      const res = await envApi.getEnv();
      if (res?.data?.status === 'success') {
        this.env = res.data.data;
        if (this.env?.meta?.node?.env?.SUB_STORE_DOCKER === 'true') {
          this.isDockerDeployment = true;
        }
      }
    },
    setDockerDeployment(isDockerDeployment: boolean) {
      this.isDockerDeployment = isDockerDeployment;
    },
    setSavedPositions(key: string, value: any) {
      this.savedPositions[key] = value;
    },
    setDefaultIconCollection(defaultIconCollection: string) {
      if (defaultIconCollection) localStorage.setItem('defaultIconCollection', defaultIconCollection);
      else localStorage.removeItem('defaultIconCollection');
      this.defaultIconCollection = defaultIconCollection;
    },
    setCustomIconCollections(collections: any[]) {
      if (collections && collections.length > 0) {
        const list = Array.from(new Set([...this.customIconCollections, ...collections]));
        localStorage.setItem('customIconCollections', JSON.stringify(list));
        this.customIconCollections = list;
      }
    },
    setSimpleMode(isSimpleMode: boolean) {
      if (isSimpleMode) localStorage.setItem('isSimpleMode', '1');
      else localStorage.removeItem('isSimpleMode');
      this.isSimpleMode = isSimpleMode;
    },
    setLeftRight(isLeftRight: boolean) {
      if (isLeftRight) localStorage.setItem('isLr', '1');
      else localStorage.removeItem('isLr');
      this.isLeftRight = isLeftRight;
    },
    setIconColor(isIconColor: boolean) {
      if (isIconColor) localStorage.setItem('iconColor', '1');
      else localStorage.removeItem('iconColor');
      this.isIconColor = isIconColor;
    },
    setIsDefaultIcon(isDefaultIcon: boolean) {
      if (isDefaultIcon) localStorage.setItem('isDefaultIcon', '1');
      else localStorage.removeItem('isDefaultIcon');
      this.isDefaultIcon = isDefaultIcon;
    },
    setEditorCommon(isEditorCommon: boolean) {
      if (!isEditorCommon) localStorage.setItem('iseditorCommon', '1');
      else localStorage.removeItem('iseditorCommon');
      this.isEditorCommon = isEditorCommon;
    },
    setSimpleReicon(isSimpleReicon: boolean) {
      if (isSimpleReicon) localStorage.setItem('isSimpleReicon', '1');
      else localStorage.removeItem('isSimpleReicon');
      this.isSimpleReicon = isSimpleReicon;
    },
    setShowFloatingRefreshButton(show: boolean) {
      if (show) localStorage.setItem('showFloatingRefreshButton', '1');
      else localStorage.removeItem('showFloatingRefreshButton');
      this.showFloatingRefreshButton = show;
    },
    settabBar(istabBar: boolean) {
      if (istabBar) localStorage.setItem('istabBar', '1');
      else localStorage.removeItem('istabBar');
      this.istabBar = istabBar;
    },
    settabBar2(istabBar2: boolean) {
      if (istabBar2) localStorage.setItem('istabBar2', '1');
      else localStorage.removeItem('istabBar2');
      this.istabBar2 = istabBar2;
    },
  },
});
