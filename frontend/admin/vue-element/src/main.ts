/**
 * 应用启动入口
 *
 * @description
 * Vue3 应用初始化，包括样式、插件、配置的加载
 */

/**
 * 应用初始化完成之后再进行页面加载渲染
 */
async function initApplication() {
  // name用于指定项目唯一标识
  // 用于区分不同项目的偏好设置以及存储数据的key前缀以及其他一些需要隔离的数据。
  // 不拼 VITE_APP_VERSION：版本进前缀会让每次发版都把用户偏好/页签静默清空
  // （此前 VITE_APP_VERSION 未定义还产出过 "gowind-admin-undefined-dev" 这种键）。
  const env = import.meta.env.PROD ? "prod" : "dev";
  const namespace = `${import.meta.env.VITE_APP_NAMESPACE}-${env}`;

  // 存储键迁移：历史前缀拼过 VITE_APP_VERSION（element 未定义它，产出
  // gowind-admin-undefined-dev），统一收敛到 `${ns}-${env}`。只搬能匹配旧形态
  // （版本段+环境段）的本应用键；现役键与新前缀同名键不动。同 vben main.ts。
  const ns = import.meta.env.VITE_APP_NAMESPACE;
  const legacyRe = new RegExp(`^${ns}-.+?-${env}(?=:|-)`);
  const currentRe = new RegExp(`^${ns}-${env}(?=:|-)`);
  for (let i = localStorage.length - 1; i >= 0; i--) {
    const key = localStorage.key(i);
    if (!key || key.startsWith(`${ns}-`) === false || currentRe.test(key)) continue;
    if (!legacyRe.test(key)) continue;
    const nextKey = key.replace(legacyRe, `${ns}-${env}`);
    if (localStorage.getItem(nextKey) === null) {
      localStorage.setItem(nextKey, localStorage.getItem(key)!);
    }
    localStorage.removeItem(key);
  }

  // 启动应用并挂载
  // vue应用主要逻辑及视图
  const { bootstrap } = await import("./bootstrap");
  await bootstrap(namespace);
}

initApplication();
