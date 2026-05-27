import {filterTree, mapTree} from '@/utils';
import type {AppMenu, AppRouteObject, RouteMeta} from "@/core/router/types";

/**
 * 根据路由配置生成菜单列表
 * @param routes - 路由配置数组（AppRouteObject[]，含 meta）
 * @param getAllRoutes - 获取完整路由树的函数（用于解析动态路径）
 * @returns 菜单树数组（AppMenu[]），可直接传给 ProLayout
 */
export async function generateMenus(
    routes: AppRouteObject[],
    getAllRoutes: () => AppRouteObject[]
): Promise<AppMenu[]> {
    // 1. 构建路由映射：name → 最终 path（用于处理动态路由/重定向）
    const allRoutes = getAllRoutes();
    const routePathMap = buildRoutePathMap(allRoutes);

    // 2. 递归转换路由 → 菜单
    let menus = mapTree<AppRouteObject, AppMenu>(routes, (route): AppMenu => {
        return convertRouteToMenu(route, routePathMap);
    });

    // 3. 递归排序（按 meta.order）
    menus = sortMenus(menus);

    // 4. 过滤隐藏菜单（meta.hideInMenu）
    menus = filterTree<AppMenu>(menus, (menu) => menu.show !== false);

    return menus;
}

/**
 * 构建路由映射：name → 最终 path
 * 用于处理动态路由、重定向等场景
 */
function buildRoutePathMap(routes: AppRouteObject[]): Record<string, string> {
    const map: Record<string, string> = {};

    const traverse = (routeList: AppRouteObject[], parentPath = '') => {
        routeList.forEach((route) => {
            // 计算完整路径（处理相对路径）
            const fullPath = route.path?.startsWith('/')
                ? route.path
                : `${parentPath}/${route.path}`.replace(/\/+/g, '/');

            // 优先使用 name 作为 key，其次使用 path
            const key = route.name || route.path || '';
            if (key) {
                map[key] = fullPath;
            }

            // 递归处理子路由
            if (route.children?.length) {
                traverse(route.children, fullPath);
            }
        });
    };

    traverse(routes);
    return map;
}

/**
 * 单个路由 → 菜单项转换
 */
function convertRouteToMenu(
    route: AppRouteObject,
    routePathMap: Record<string, string>
): AppMenu {
    const meta = (route.meta || {}) as RouteMeta;

    // 解析路径：优先使用路由映射中的最终 path
    const resolvedPath = route.name && routePathMap[route.name]
        ? routePathMap[route.name]
        : route.path || '';

    // 提取菜单相关元数据（带默认值）
    const {
        title = '',
        icon,
        activeIcon,
        order,
        hideInMenu = false,
        hideChildrenInMenu = false,
        badge,
        badgeType,
        badgeVariants,
        link,
        redirect,
        menuVisibleWithForbidden,
    } = meta;

    // 菜单名称：title > name > path
    const name = title || route.name || resolvedPath || 'Unnamed';

    // 处理子菜单
    let children: AppMenu[] = [];
    if (route.children?.length && !hideChildrenInMenu) {
        children = route.children as unknown as AppMenu[]; // 递归时由 mapTree 处理
    }

    // 注入父子关系（用于面包屑/高亮）
    const parents = route.parents || [];
    const currentParents = [...parents, resolvedPath].filter(Boolean);

    // 决定菜单点击跳转路径
    // 优先级：link(外链) > redirect(重定向) > resolvedPath
    const menuPath = link || redirect || resolvedPath;

    return {
        // 必填字段
        name: route.name || resolvedPath,  // 唯一标识
        path: menuPath,                    // 跳转路径
        label: name,                       // 显示名称

        // 可选字段
        icon,
        activeIcon,
        order,
        badge,
        badgeType,
        badgeVariants,
        disabled: meta.disabled,
        show: !hideInMenu,

        // 父子关系
        parent: route.parent || parents[parents.length - 1],
        parents: currentParents,

        // 子菜单（递归）
        children,

        // 权限/行为
        authority: meta.authority,
        menuVisibleWithForbidden,

        // 扩展字段
        meta, // 保留完整元数据供后续使用
    };
}

/**
 * 递归排序菜单树（按 order 字段）
 */
function sortMenus(menuList: AppMenu[]): AppMenu[] {
    return menuList
        .map((menu) => ({
            ...menu,
            // 递归排序子菜单
            children: menu.children?.length ? sortMenus(menu.children) : [],
        }))
        .sort((a, b) => {
            // order 越小越靠前，undefined 排最后
            const orderA = a.order ?? Number.MAX_SAFE_INTEGER;
            const orderB = b.order ?? Number.MAX_SAFE_INTEGER;
            return orderA - orderB;
        });
}

/**
 * 工具：判断菜单是否应该显示（权限 + 配置）
 * 可用于二次过滤
 */
export const shouldShowMenu = (menu: AppMenu, permissions: string[]): boolean => {
    // 1. 配置隐藏
    if (menu.show === false) return false;

    // 2. 无权限要求 = 显示
    if (!menu.authority?.length) return true;

    // 3. 检查权限
    if (menu.authority.some(code => permissions.includes(code))) return true;

    // 4. 无权限但配置了可见 = 显示但禁用（由前端控制）
    return menu.menuVisibleWithForbidden === true;
};
