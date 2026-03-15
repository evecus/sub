import { useAppNotifyStore } from "@/store/appNotify";
import { useArtifactsStore } from "@/store/artifacts";
import { useGlobalStore } from "@/store/global";
import { useSettingsStore } from "@/store/settings";
import { useSubsStore } from "@/store/subs";
import i18n from "@/locales";

export const initStores = async (
  needNotify: boolean,
  needFetchFlow: boolean,
  needRefreshCache: boolean
) => {
  const { showNotify } = useAppNotifyStore();
  const globalStore = useGlobalStore();
  const subsStore = useSubsStore();
  const artifactsStore = useArtifactsStore();
  const settingsStore = useSettingsStore();
  const { t } = i18n.global;

  globalStore.setLoading(true);

  // 直接设为已连接，跳过后端握手
  globalStore.setFetchResult(true);
  await globalStore.setEnv();

  try {
    await subsStore.fetchSubsData();
    await artifactsStore.fetchArtifactsData();
    await settingsStore.syncLocalAppearanceSetting();
    await settingsStore.fetchSettings();
  } catch (e) {
    console.error("initStores error", e);
  }

  globalStore.setLoading(false);

  if (needNotify) {
    showNotify({ title: t("globalNotify.refresh.succeed"), type: "primary" });
  }

  if (needFetchFlow) await subsStore.fetchFlows();
};
