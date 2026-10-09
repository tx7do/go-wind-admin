import { defineOverridesPreferences } from '@vben/preferences';

/**
 * @description 项目配置文件
 * 只需要覆盖项目中的一部分配置，不需要的配置不用覆盖，会自动使用默认配置
 * !!! 更改配置后请清空缓存，否则可能不生效
 */
export const overridesPreferences = defineOverridesPreferences({
  // overrides
  app: {
    name: import.meta.env.VITE_APP_TITLE,
    accessMode: import.meta.env.VITE_ROUTER_ACCESS_MODE,
    // 三端统一默认头像（橘猫，react/ele 同图；覆盖 vben 框架默认的 default-avatar.webp）
    defaultAvatar: '/default-avatar.png',
  },
});
