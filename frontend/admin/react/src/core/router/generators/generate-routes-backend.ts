import {type ComponentType, createElement, Suspense} from 'react';
import {Navigate} from 'react-router-dom';

import {mapTree} from '@/utils';
import Loading from '@/components/common/Loading';
import type {AppRouteObject, BackendRoute, ComponentRecord, GenerateRoutesOptions} from "@/core/router/types"; // 全局加载组件

/**
 * 动态生成路由 - 后端方式（React 版本）
 * @param options - 配置选项
 * @returns 转换后的路由树（AppRouteObject[]），可直接传入 createBrowserRouter
 */
export async function generateRoutesByBackend(
    options: GenerateRoutesOptions
): Promise<AppRouteObject[]> {
    const {fetchMenuListAsync, layoutMap = {}, pageMap = {}} = options;

    try {
        // 1. 从后端获取菜单路由（组件路径为字符串）
        const menuRoutes = await fetchMenuListAsync?.();
        if (!menuRoutes?.length) {
            return [];
        }

        // 2. 预处理 pageMap：标准化路径键名（支持多种写法）
        const normalizedPageMap: ComponentRecord = {};
        for (const [key, value] of Object.entries(pageMap)) {
            const normalizedKey = normalizeViewPath(key);
            normalizedPageMap[normalizedKey] = value;
            // 兼容：如果传入的是带后缀的，也注册不带后缀的映射
            if (normalizedKey.endsWith('.tsx')) {
                normalizedPageMap[normalizedKey.replace(/\.tsx$/, '')] = value;
            }
        }

        // 3. 归位嵌套绝对子路径 + 补目录组缺省重定向，再递归转换路由树
        rebaseBackendTree(menuRoutes);
        return convertRoutes(menuRoutes, layoutMap, normalizedPageMap);
    } catch (error) {
        console.error('Failed to generate routes from backend:', error);
        return [];
    }
}

/**
 * 递归转换后端路由为 React 路由
 */
function convertRoutes(
    routes: BackendRoute[],
    layoutMap: ComponentRecord,
    pageMap: ComponentRecord
): AppRouteObject[] {
    // 注意：不手动递归 children，由 mapTree 自动处理递归
    // children 必须保留在返回对象中，mapTree 才能自动递归
    return mapTree(routes, (node) => {
        const {component: componentPath, name, ...rest} = node;

        if (!name) {
            console.error('Route name is required', node);
        }

        const route: AppRouteObject = {
            ...rest,
            name,
        };

        if (componentPath) {
            let Component: ComponentType<any> | undefined;

            // 1. 优先匹配布局组件
            if (layoutMap[componentPath]) {
                Component = layoutMap[componentPath];
            }
            // 2. 匹配页面组件（标准化路径后查找）
            else {
                const normalizedPath = normalizeViewPath(componentPath);
                // 尝试多种后缀组合
                const candidates = [
                    normalizedPath,
                    normalizedPath.endsWith('.tsx') ? normalizedPath : `${normalizedPath}.tsx`,
                    normalizedPath.endsWith('/index') ? normalizedPath : `${normalizedPath}/index`,
                    `${normalizedPath}/index.tsx`,
                ];

                for (const candidate of candidates) {
                    if (pageMap[candidate]) {
                        Component = pageMap[candidate];
                        break;
                    }
                }
            }

            // 3. 渲染元素：必须用 Suspense 包裹 lazy 组件
            if (Component) {
                route.element = createElement(
                    Suspense,
                    {fallback: createElement(Loading)},
                    createElement(Component)
                );
            } else {
                // 开发环境警告，生产环境降级处理
                if (import.meta.env.DEV) {
                    console.warn(`Component not found for path: ${componentPath}`);
                }
                route.element = createElement('div', null, `404 - Component "${componentPath}" not found`);
            }
        }

        // 处理 redirect：将后端返回的 redirect 转换为 React Router 可识别的导航
        // 1. 有 redirect + 有 children → 注入 index 子路由进行重定向
        // 2. 有 redirect + 无 component → 纯重定向路由，element 设为 <Navigate>
        if (rest.redirect) {
            if (rest.children?.length) {
                // 在 children 最前面注入 index 路由（匹配父路径本身）
                const indexRoute: AppRouteObject = {
                    index: true,
                    element: createElement(Navigate, {to: rest.redirect, replace: true}),
                };
                route.children = [indexRoute, ...(rest.children as any[])];
            } else if (!componentPath) {
                // 纯重定向路由（无组件、无子路由）
                route.element = createElement(Navigate, {to: rest.redirect, replace: true});
            }
        }

        return route;
    });
}

/**
 * 归位后端菜单树中的嵌套绝对子路径，并为缺省目录组注入重定向。
 *
 * 后端菜单树是 vue-router 风格：目录组内子项常带绝对路径（/dashboard →
 * /analytics、/profile → /profile）。vue-router 允许嵌套绝对子路径，而
 * react-router 禁止（除非以父路径全路径为前缀），createBrowserRouter 会直接
 * 抛错卡死路由创建。这里把绝对子路径归位为父目录组内的相对路径：
 * - 前缀与父路径匹配者剥掉父前缀（/profile → 空路径 index）；
 * - 其余剥掉开头斜杠（/analytics → analytics），URL 随父路径延伸为
 *   /dashboard/analytics，与前端模式静态路由的 URL 空间保持一致。
 */
function rebaseBackendTree(nodes: BackendRoute[], parentPath = ''): void {
    for (const node of nodes) {
        const fullPath = node.path ? resolveRoutePath(parentPath, node.path) : parentPath;

        if (parentPath && node.path?.startsWith('/')) {
            node.path = node.path === parentPath || node.path.startsWith(`${parentPath}/`)
                ? node.path.slice(parentPath.length).replace(/^\/+/, '')
                : node.path.replace(/^\/+/, '');
        }

        // 目录组缺省重定向：convertRoutes 会将其转成 index <Navigate>，否则
        // 直访目录组路径（/dashboard）只渲染出空布局。首个子路径为空（归位成
        // index 的 /profile 组）或重定向目标等于自身（自指死循环）时不注入。
        const firstChild = node.children?.[0];
        if (parentPath && node.children?.length && !node.redirect && firstChild?.path) {
            const target = resolveRoutePath(fullPath, firstChild.path);
            if (target !== fullPath) {
                node.redirect = target;
            }
        }

        if (node.children?.length) {
            rebaseBackendTree(node.children, fullPath);
        }
    }
}

function resolveRoutePath(parentPath: string, childPath: string): string {
    if (!childPath) return parentPath;
    if (childPath.startsWith('/')) return childPath;
    return `${parentPath.replace(/\/+$/, '')}/${childPath}`;
}

/**
 * 标准化视图路径（适配 React 目录结构）
 * @param path - 后端返回的组件路径，如 "./views/system/user" 或 "/views/dashboard"
 */
export function normalizeViewPath(path: string): string {
    // 1. 去除相对路径前缀
    const normalized = path.replace(/^(\.\/|\.\.\/)+/, '');

    // 2. 确保以 / 开头
    const withSlash = normalized.startsWith('/') ? normalized : `/${normalized}`;

    // 3. 去除目录前缀（适配 React 项目结构）
    // 支持多种常见目录：/views, /pages, /src/pages 等；后端组件路径带 app/ 段
    // （如 app/opm/user/list/index.vue），而 React pageMap 键不含 app/，一并剥掉；
    // 后端种子用下划线目录（vue-element 端约定，如 internal_message），React
    // 页面目录统一横线，归一化时转换
    return withSlash
        .replace(/^\/(src\/)?(views|pages)\//i, '/')  // 去除 /views/ 或 /pages/
        .replace(/^\/app\//, '/')                     // 去除 /app/ 段
        .replace(/\.(vue|tsx|jsx|ts)$/i, '')          // 去除组件文件扩展名（后端下发的是 vue 端路径 xxx/index.vue）
        .replace(/_/g, '-')                           // 下划线 → 横线（React 页面目录约定）
        .replace(/\/+$/, '');                          // 去除末尾斜杠
}
