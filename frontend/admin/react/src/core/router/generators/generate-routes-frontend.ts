import type {ReactNode} from "react";

import {filterTree, mapTree} from "@/utils";
import type {AppRouteObject, RouteMeta} from "@/core/router/types";

/**
 * 动态生成路由 - 前端权限过滤方式
 * @param routes - 静态路由表（AppRouteObject[]，含 meta）
 * @param permissions - 用户权限码列表
 * @param forbiddenElement - 无权限时渲染的元素（<Forbidden />）
 */
export async function generateRoutesByFrontend(
    routes: AppRouteObject[],
    permissions: string[],
    forbiddenElement?: ReactNode
): Promise<AppRouteObject[]> {
    // 1. 根据权限过滤路由树
    const filteredRoutes = filterTree(routes, (route) => {
        return hasPermission(route, permissions);
    });

    // 2. 如果没有提供 403 元素，直接返回
    if (!forbiddenElement) {
        return filteredRoutes;
    }

    // 3. 处理"菜单可见但访问返回 403"的节点
    return mapTree(filteredRoutes, (route) => {
        if (shouldRenderForbidden(route, permissions)) {
            return {
                ...route,
                element: forbiddenElement,
            };
        }
        return route;
    });
}

/**
 * 判断路由是否有权限访问
 * @param route - 路由对象
 * @param permissions - 用户权限码列表
 * @returns 是否允许访问
 */
function hasPermission(route: AppRouteObject, permissions: string[]): boolean {
    const meta = route.meta as RouteMeta | undefined;

    // 情况 1: 无权限要求 = 公开路由（登录页、404 等）
    if (!meta?.authority?.length) {
        return true;
    }

    // 情况 2: 配置了 ignoreAccess = 忽略权限检查
    if (meta?.ignoreAccess) {
        return true;
    }

    // 情况 3: 检查权限码数组（authority 中任一元素匹配用户权限即可）
    if (meta.authority.some((code) => permissions.includes(code))) {
        return true;
    }

    // 情况 4: 无权限，但可能是"菜单可见但禁止访问"，交给上层处理
    return false;
}

/**
 * 判断是否渲染 403 页面（菜单可见但无权限访问）
 * @param route - 路由对象
 * @param permissions - 用户权限码列表
 * @returns 是否应渲染禁止访问页面
 */
function shouldRenderForbidden(route: AppRouteObject, permissions: string[]): boolean {
    const meta = route.meta as RouteMeta | undefined;

    // 必须同时满足：
    // 1. 有权限要求
    // 2. 当前用户无此权限
    // 3. 配置了 menuVisibleWithForbidden = true
    return (
        !!meta?.authority?.length &&
        !hasPermission(route, permissions) &&
        meta?.menuVisibleWithForbidden === true
    );
}

export {hasPermission, shouldRenderForbidden};
