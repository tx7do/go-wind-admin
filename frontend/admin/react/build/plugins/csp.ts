import type {PluginOption} from 'vite';

/**
 * 生产构建期向 index.html 注入 Content-Security-Policy meta。
 *
 * 仅在 build 时注入：dev 下 vite 会注入 HMR / React Refresh 内联脚本，
 * `script-src 'self'` 会拦截这些内联脚本导致白屏，故 dev 必须跳过。
 *
 * 通过 meta 而非响应头：react 项目仓库内无独立部署 header 层（仅 vben 有 nginx 配置），
 * meta CSP 是规范支持、部署无关的途径，能阻断 XSS 注入的内联脚本执行。
 * 限制：meta CSP 不支持 frame-ancestors / report-uri 等指令，故点击劫持防护仍需部署侧 header。
 */
export function cspMetaPlugin(): PluginOption {
    let isBuild = false;
    return {
        name: 'gowind-csp-meta',
        configResolved(config) {
            isBuild = config.command === 'build';
        },
        transformIndexHtml: {
            order: 'post',
            handler() {
                if (!isBuild) {
                    return;
                }
                return {
                    tags: [
                        {
                            tag: 'meta',
                            attrs: {
                                'http-equiv': 'Content-Security-Policy',
                                content:
                                    "script-src 'self'; base-uri 'self'; object-src 'none'",
                            },
                            injectTo: 'head-prepend',
                        },
                    ],
                };
            },
        },
    };
}
