import { useEnvApi } from "@/api/env";
import i18n from "@/locales";
import { useAppNotifyStore } from "@/store/appNotify";
import { useGlobalStore } from "@/store/global";
import { useSettingsStore } from "@/store/settings";
import { useSubsStore } from "@/store/subs";

export const initStores = async (
  needNotify: boolean,
  needFetchFlow: boolean,
  needRefreshCache: boolean
) => {
  const { showNotify } = useAppNotifyStore();
  const globalStore = useGlobalStore();
  const subsStore = useSubsStore();
  const settingsStore = useSettingsStore();
  const { t } = i18n.global;

  let isSucceed = true;

  if (needRefreshCache) {
    showNotify({ title: t("globalNotify.refresh.loading"), type: "primary" });
  }

  globalStore.setLoading(true);
  globalStore.setFetchResult(true);

  try {
    localStorage.removeItem("envCache");

    // 获取后端环境信息
    const timeoutPromise = new Promise((_, reject) =>
      setTimeout(() => reject(new Error('timeout')),
        localStorage.getItem('timeout') ? parseInt(localStorage.getItem('timeout'), 10) : 3000
      )
    );
    await Promise.race([globalStore.setEnv(), timeoutPromise]);

    const hasBackendEnv = Object.keys(globalStore.env).length > 0 && globalStore.env.backend;
    if (!hasBackendEnv) {
      globalStore.setFetchResult(false);
      isSucceed = false;
      throw new Error('Failed to get backend env');
    }

    await subsStore.fetchSubsData();
    await settingsStore.syncLocalAppearanceSetting();
    await settingsStore.fetchSettings();

    if (needRefreshCache) {
      const { data } = await useEnvApi().refreshCache();
      if (data.status !== "success") {
        globalStore.setFetchResult(false);
        isSucceed = false;
      }
    }
  } catch (e) {
    console.error('initStores error', e);
    globalStore.setFetchResult(false);
    subsStore.subs = [];
    subsStore.collections = [];
    isSucceed = false;
  }

  if (isSucceed && needNotify) {
    showNotify({ title: t("globalNotify.refresh.succeed"), type: "primary" });
  }

  globalStore.setLoading(false);
  if (needFetchFlow) await subsStore.fetchFlows();
};
