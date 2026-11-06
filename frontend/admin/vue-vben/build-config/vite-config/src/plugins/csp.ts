import {createHash} from "node:crypto";

import type {PluginOption} from "vite";

/**
 * 生产构建期向 index.html 注入 Content-Security-Policy meta。
 *
 * 仅在 build 时注入：dev 下 vite 会注入 HMR / React Refresh 内联脚本，
 * `script-src "self"` 会拦截这些内联脚本导致白屏，故 dev 必须跳过。
 *
 * 通过 meta 而非响应头：react 项目仓库内无独立部署 header 层（仅 vben 有 nginx 配置），
 * meta CSP 是规范支持、部署无关的途径，能阻断 XSS 注入的内联脚本执行。
 * 限制：meta CSP 不支持 frame-ancestors / report-uri 等指令，故点击劫持防护仍需部署侧 header。
 *
 * index.html 中自带的内联脚本（主题预绘防闪烁）会被 `script-src "self"` 拦截，
 * 故构建期对其内容逐个计算 sha256 并以 `"sha256-..."` 白名单放行：
 * 内容任一字符变动哈希即失效，外来注入的内联脚本无法借道通过。
 */
function collectInlineScriptHashes(html: string): string[] {
    const hashes: string[] = [];
    const scriptPattern = /<script(?![^>]*\bsrc=)[^>]*>([\s\S]*?)<\/script>/gi;
    let match: RegExpExecArray | null;
    while ((match = scriptPattern.exec(html)) !== null) {
        const content = match[1];
        if (content.trim().length === 0) {
            continue;
        }
        const digest = createHash("sha256").update(content, "utf8").digest("base64");
        hashes.push(`'sha256-${digest}'`);
    }
    return hashes;
}

export function cspMetaPlugin(): PluginOption {
    let isBuild = false;
    return {
        name: "gowind-csp-meta",
        configResolved(config) {
            isBuild = config.command === "build";
        },
        transformIndexHtml: {
            order: "post",
            handler(html) {
                if (!isBuild) {
                    return;
                }
                const scriptSrc = ["script-src 'self'", ...collectInlineScriptHashes(html)].join(
                    " ",
                );
                return {
                    tags: [
                        {
                            tag: "meta",
                            attrs: {
                                "http-equiv": "Content-Security-Policy",
                                content: `${scriptSrc}; base-uri 'self'; object-src 'none'`,
                            },
                            injectTo: "head-prepend",
                        },
                    ],
                };
            },
        },
    };
}
