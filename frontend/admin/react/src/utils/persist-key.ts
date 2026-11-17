/**
 * 持久化存储键命名空间（宿主侧）。
 *
 * 历史 store 键（app-tabs / lock-storage 等）是全局裸键——同一 origin:port
 * 上先后跑两个应用会互相覆盖（实测有其他项目在 localhost:15888 留下的
 * "GoWind Quant Admin" 偏好污染）。本助手读取 index.html 内联脚本写入的
 * `document.documentElement.dataset.gowindStorageNs`（vite %ENV% 注入），
 * 为键加 `<ns>:` 前缀；未设置时回退裸键。
 *
 * 注意：admin-react 包内的 store 键（app-preferences 等）同名能力随包版本
 * 发布（persistKey 已导出）；本包内实现发布后可收敛为从包导入。
 */

/** 读宿主注入的存储命名空间（index.html 内联脚本写入 data-gowind-storage-ns）。 */
export function getStorageNamespace(): string {
  if (typeof document === 'undefined') return '';
  return document.documentElement.dataset.gowindStorageNs ?? '';
}

/** 为 persist store 生成带命名空间的 localStorage 键（含旧裸键一次性迁移）。 */
export function persistKey(name: string): string {
  const ns = getStorageNamespace();
  if (!ns) return name;

  const key = `${ns}:${name}`;
  if (typeof localStorage !== 'undefined') {
    if (localStorage.getItem(key) === null && localStorage.getItem(name) !== null) {
      localStorage.setItem(key, localStorage.getItem(name)!);
      localStorage.removeItem(name);
    }
  }
  return key;
}
