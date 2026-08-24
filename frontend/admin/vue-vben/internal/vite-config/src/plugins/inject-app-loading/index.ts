import fs from 'node:fs';
import fsp from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { readPackageJSON } from '@vben/node-utils';

import { type PluginOption } from 'vite';

const THEME_INIT_FILENAME = 'theme-init.js';

/**
 * 用于生成将loading样式注入到项目中
 * 为多app提供loading样式，无需在每个 app -> index.html单独引入
 */
async function viteInjectAppLoadingPlugin(
  isBuild: boolean,
  env: Record<string, any> = {},
  loadingTemplate = 'loading.html',
): Promise<PluginOption | undefined> {
  const loadingHtml = await getLoadingRawByHtmlTemplate(loadingTemplate);
  const { version } = await readPackageJSON(process.cwd());
  const envRaw = isBuild ? 'prod' : 'dev';
  const cacheName = `'${env.VITE_APP_NAMESPACE}-${version}-${envRaw}-preferences-theme'`;

  // 主题探测脚本：刷新时在首屏前为 <html> 应用 dark 类，避免暗黑模式闪烁（FOUC）。
  // 缓存键在构建期确定，脚本内容固定。开发期无 CSP，沿用内联避免额外网络请求。
  // 生产期需配合 CSP script-src 'self'（阻断内联 <script>），故外联为同源
  // theme-init.js，由本插件在 generateBundle 阶段随产物输出。
  const themeInitSource = `var theme = localStorage.getItem(${cacheName});document.documentElement.classList.toggle('dark', /dark/.test(theme));`;

  if (!loadingHtml) {
    return;
  }

  // 构建期为外联脚本的 src 前缀；configResolved 在 transformIndexHtml 之前执行。
  let publicPath = '/';

  return {
    enforce: 'pre',
    name: 'vite:inject-app-loading',
    configResolved(config) {
      publicPath = ensureTrailingSlash(config.base);
    },
    async generateBundle() {
      if (!isBuild) {
        return;
      }
      this.emitFile({
        fileName: THEME_INIT_FILENAME,
        source: themeInitSource,
        type: 'asset',
      });
    },
    transformIndexHtml: {
      handler(html) {
        const re = /<body\s*>/;
        // 构建期注入外联 <script src>（同源，script-src 'self' 放行）；
        // 开发期注入内联 <script>，与历史行为一致。
        const scriptTag = isBuild
          ? `<script data-app-loading="inject-js" src="${publicPath}${THEME_INIT_FILENAME}"></script>`
          : `<script data-app-loading="inject-js">${themeInitSource}</script>`;
        html = html.replace(re, `<body>${scriptTag}${loadingHtml}`);
        return html;
      },
      order: 'pre',
    },
  };
}

/**
 * 用于获取loading的html模板
 */
async function getLoadingRawByHtmlTemplate(loadingTemplate: string) {
  // 支持在app内自定义loading模板，模版参考default-loading.html即可
  let appLoadingPath = join(process.cwd(), loadingTemplate);

  if (!fs.existsSync(appLoadingPath)) {
    const __dirname = fileURLToPath(new URL('.', import.meta.url));
    appLoadingPath = join(__dirname, './default-loading.html');
  }

  return await fsp.readFile(appLoadingPath, 'utf8');
}

function ensureTrailingSlash(path: string) {
  return path.endsWith('/') ? path : `${path}/`;
}

export { viteInjectAppLoadingPlugin };
