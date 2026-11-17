import { addCollection } from '@iconify/react';
import { icons as lucideCollection } from '@iconify-json/lucide';

/**
 * lucide 图标集本地注册：任意 iconify 名称的渲染（侧边栏/菜单列表/面包屑）与
 * IconPicker 选择网格都直接吃本地数据，内网/离线部署不再依赖
 * api.iconify.design 在线拉取。名字清单含别名——种子数据里的旧名
 * （如 area-chart）在新版 lucide 里就是别名，不并进去会搜不到、高亮不了。
 */
addCollection(lucideCollection);

export const lucideIconNames: string[] = [
  ...new Set([
    ...Object.keys(lucideCollection.icons),
    ...Object.keys(lucideCollection.aliases ?? {}),
  ]),
].sort();
