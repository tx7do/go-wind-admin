import zhCn from "element-plus/es/locale/lang/zh-cn";
import en from "element-plus/es/locale/lang/en";

import { DeviceEnum, SidebarStatus } from "@/enums";
import { STORAGE_KEYS } from "@/constants";
import { defaultPreferences } from "@/settings";
import { loadLocaleMessages } from "@/i18n/setup";
import type { SupportedLanguagesType } from "@/i18n/types";

export const useAppStore = defineStore("app", () => {
  const device = useStorage(STORAGE_KEYS.DEVICE, DeviceEnum.DESKTOP);
  const size = useStorage(STORAGE_KEYS.SIZE, defaultPreferences.size);
  const language = useStorage(STORAGE_KEYS.LANGUAGE, defaultPreferences.language);
  const sidebarStatus = useStorage(STORAGE_KEYS.SIDEBAR_STATUS, SidebarStatus.CLOSED);
  const sidebar = reactive({
    opened: sidebarStatus.value === SidebarStatus.OPENED,
    withoutAnimation: false,
  });
  const activeTopMenuPath = useStorage(STORAGE_KEYS.ACTIVE_TOP_MENU_PATH, "");

  const locale = computed(() => {
    // 根据语言值返回对应的 Element Plus 语言包
    return language?.value === "en" ? en : zhCn;
  });

  function toggleSidebar() {
    sidebar.opened = !sidebar.opened;
    sidebarStatus.value = sidebar.opened ? SidebarStatus.OPENED : SidebarStatus.CLOSED;
  }

  function closeSideBar() {
    sidebar.opened = false;
    sidebarStatus.value = SidebarStatus.CLOSED;
  }

  function openSideBar() {
    sidebar.opened = true;
    sidebarStatus.value = SidebarStatus.OPENED;
  }

  function toggleDevice(val: string) {
    device.value = val;
  }

  function changeSize(val: string) {
    size.value = val;
  }

  async function changeLanguage(val: SupportedLanguagesType) {
    language.value = val;
    // 加载对应的语言包
    await loadLocaleMessages(val);
  }

  function activeTopMenu(val: string) {
    activeTopMenuPath.value = val;
  }

  return {
    device,
    sidebar,
    language,
    locale,
    size,
    activeTopMenu,
    toggleDevice,
    changeSize,
    changeLanguage,
    toggleSidebar,
    closeSideBar,
    openSideBar,
    activeTopMenuPath,
  };
});
