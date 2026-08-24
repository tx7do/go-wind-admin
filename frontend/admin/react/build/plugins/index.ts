import type {PluginOption} from 'vite';
import {visualizer} from 'rollup-plugin-visualizer';
import {timePlugin} from './time';
import {nojekyllPlugin} from './nojekyll';
import {autoImportPlugin} from './autoImport';
import {versionUpdatePlugin} from './version';
import {cspMetaPlugin} from './csp';
import react from '@vitejs/plugin-react-swc';
import tailwindcss from '@tailwindcss/vite';
import viteCompression from 'vite-plugin-compression';

export function createVitePlugins() {
    const isDev = process.env.NODE_ENV === 'development';

    // 插件参数
    const vitePlugins: PluginOption[] = [
        // React SWC 插件，配置以支持 React 19
        react({
            // 确保使用正确的 JSX 运行时
            tsDecorators: false,
        }),
        tailwindcss(),
        // 自动导入
        autoImportPlugin(),
    ];

    // 生产环境才启用这些插件
    if (!isDev) {
        vitePlugins.push(
            // 版本控制
            versionUpdatePlugin(),
            // 生成 .nojekyll 空文件
            nojekyllPlugin(),
            // 打包时间
            timePlugin(),
            // 生产期注入 CSP meta（阻断 XSS 内联脚本执行）
            cspMetaPlugin(),
        );

        // 按需启用：Gzip 压缩（设置 ANALYZE=true 启用包分析时一起启用）
        const enableAnalyze = process.env.ANALYZE === 'true';
        if (enableAnalyze) {
            vitePlugins.push(
                // 压缩包
                viteCompression(),
                // 包分析
                visualizer({
                    gzipSize: true,
                    brotliSize: true,
                }),
            );
        }
    }

    return vitePlugins;
}
