import type {RouteObject} from "react-router-dom";
import React, {type ComponentType} from "react";
import type {ReactNode} from "react";

export type IconName = string; // 图标名，如 "UserOutlined"
export type IconType = IconName | ComponentType<{ className?: string; style?: React.CSSProperties }>;

export type BadgeVariant = 'default' | 'primary' | 'success' | 'warning' | 'destructive' | string;
export type BadgeType = 'dot' | 'normal';

export type AccessModeType = 'frontend' | 'backend';

/**
 * 后端返回的路由对象（组件路径为字符串）
 */
export interface BackendRoute extends Omit<AppRouteObject, 'element' | 'children'> {
    component?: string;
    children?: BackendRoute[];
}

export interface GenerateRoutesOptions {
    // 必填
    staticRoutes: AppRouteObject[];
    mode: AccessModeType;

    // 权限
    permissions?: string[];
    roles?: string[];

    // 后端模式
    fetchMenuListAsync?: () => Promise<BackendRoute[]>;
    layoutMap?: ComponentRecord;
    pageMap?: ComponentRecord;

    // 前端模式
    forbiddenElement?: React.ReactNode;

    // 可选增强
    autoInjectRedirect?: boolean;
    autoSort?: boolean;
    filterHidden?: boolean;
    onRoutesGenerated?: (routes: AppRouteObject[]) => void;
    onMenusGenerated?: (menus: AppMenu[]) => void;
}

export interface GenerateRoutesResult {
    routes: AppRouteObject[];
    menus: AppMenu[];
}

export interface RouteMeta {
    // 基础信息
    title?: string;                    // 必填：菜单/标签页标题
    icon?: IconType;                  // 菜单/标签页图标
    activeIcon?: IconType;            // 激活时图标

    // 权限控制
    authority?: string[];             // 权限码数组（包含权限码和角色码，前端/后端路由统一使用）
    ignoreAccess?: boolean;           // 忽略权限，公开访问

    // 菜单/标签页显示控制
    hideInMenu?: boolean;             // 菜单中隐藏
    hideInTab?: boolean;              // 标签页中隐藏
    hideInBreadcrumb?: boolean;       // 面包屑中隐藏
    hideChildrenInMenu?: boolean;     // 子路由不在菜单显示

    // 特殊行为
    menuVisibleWithForbidden?: boolean; // 菜单可见，但访问返回 403
    activePath?: string;              // 手动指定激活的菜单路径
    affixTab?: boolean;               // 固定标签页
    affixTabOrder?: number;           // 固定标签页排序
    keepAlive?: boolean;              // 页面缓存
    loaded?: boolean;                 // 路由是否已加载（内部使用）

    // 外部链接/iframe
    link?: string;                    // 外链跳转 URL
    iframeSrc?: string;               // iframe 嵌入地址
    openInNewWindow?: boolean;        // 新窗口打开

    // 其他
    order?: number;                   // 菜单排序
    query?: Record<string, any>;      // 路由携带参数
    maxNumOfOpenTab?: number;         // 标签页最大打开数（-1 无限制）

    // 菜单徽标
    badge?: string;                   // 徽标内容
    badgeType?: 'dot' | 'normal';     // 徽标类型
    badgeVariants?: BadgeVariant;     // 徽标颜色

    // 其他扩展（保留灵活性）
    [key: string]: any;
}

export interface AppRoute extends Omit<RouteObject, 'meta' | 'children' | 'element'> {
    // 必填字段
    name: string;     // 路由唯一标识（权限匹配关键）
    path: string;     // 路由路径（修正 any → string）

    // 可选字段
    redirect?: string;
    index?: boolean;

    // 元数据
    meta?: RouteMeta;

    // 子路由（递归）
    children?: AppRoute[];

    // React Router v6+ 核心：用 element 替代 component
    element?: ReactNode;

    // 扩展：后端返回的组件路径字符串（用于动态加载）
    componentPath?: string;

    // 面包屑/菜单辅助字段
    parent?: string;
    parents?: string[];
}


/**
 * AppRouteObject = RouteObject + meta + 业务字段
 * 用于替代原生的 RouteObject，携带业务元数据
 */
export interface AppRouteObject extends Omit<RouteObject, 'children'> {
    // 路由唯一标识（权限匹配关键）
    // 注：index 路由（index: true）不需要 name 和 path，所以改为可选
    name?: string;

    path?: string;

    // 元数据（核心扩展）
    meta?: RouteMeta;

    redirect?: string;

    icon?: IconType;
    activeIcon?: IconType;

    disabled?: boolean;
    show?: boolean;

    order?: number;

    // 子路由（递归使用 AppRouteObject）
    children?: AppRouteObject[];

    // 业务辅助字段
    parent?: string;          // 父级路径
    parents?: string[];       // 所有父级路径（面包屑用）

    componentPath?: string;   // 后端返回的组件路径字符串（动态加载用）

    element?: ReactNode;
}

export interface AppMenuBadge {
    badge?: string;
    badgeType?: BadgeType;
    badgeVariants?: BadgeVariant;
}

export interface AppMenu extends AppMenuBadge {
    // ========== 必填 ==========
    name: string;     // 唯一标识（与路由 name 对应）
    path: string;     // 点击跳转路径
    label: string;    // 菜单显示名称

    // ========== 可选 ==========
    icon?: IconType;
    activeIcon?: IconType;

    order?: number;   // 排序值（越小越前）

    badge?: string;
    badgeType?: 'dot' | 'normal';
    badgeVariants?: BadgeVariant;

    disabled?: boolean;
    show?: boolean;   // 控制菜单渲染（默认 true）

    // 父子关系（用于面包屑/高亮）
    parent?: string;
    parents?: string[];

    // 子菜单
    children?: AppMenu[];

    // 权限控制
    authority?: string[];
    menuVisibleWithForbidden?: boolean;

    // 扩展：保留原始元数据
    meta?: RouteMeta;

    // 其他自定义字段
    [key: string]: any;
}

// React 组件映射类型
export type ComponentRecordType = Record<string, ComponentType<any>>;
export type ComponentRecord = Record<string, ComponentType<any>>;

export interface GenerateMenuAndRoutesOptions {
    fetchMenuListAsync?: () => Promise<BackendRoute[]>;
    forbiddenElement?: React.ReactNode;
    layoutMap?: ComponentRecordType;
    pageMap?: ComponentRecordType;
    permissions?: string[];
    roles?: string[];
    routes: AppRouteObject[];
    autoInjectRedirect?: boolean;
    autoSort?: boolean;
}


/**
 * 路由模块的默认导出类型
 * 每个动态导入的路由文件应默认导出 AppRouteObject[]
 */
export interface RouteModuleType {
    default: AppRouteObject[];
}

/**
 * Vite glob 导入的动态模块类型
 * 兼容 .tsx/.ts 文件 + 默认导出
 */
export type RouteModuleRecord = Record<string, RouteModuleType | AppRouteObject[] | (() => Promise<RouteModuleType>)>;

export const isAppRoute = (route: any): route is AppRouteObject => {
    return route && typeof route.name === 'string' && typeof route.path === 'string';
};


export interface MenuRecordBadgeRaw {
    /**
     * 徽标
     */
    badge?: string;
    /**
     * 徽标类型
     */
    badgeType?: "dot" | "normal";
    /**
     * 徽标颜色
     */
    badgeVariants?: "destructive" | "primary" | string;
}

/**
 * 菜单原始对象
 */
export interface MenuRecordRaw extends MenuRecordBadgeRaw {
    /**
     * 激活时的图标名
     */
    activeIcon?: string;
    /**
     * 子菜单
     */
    children?: MenuRecordRaw[];
    /**
     * 是否禁用菜单
     * @default false
     */
    disabled?: boolean;
    /**
     * 图标名
     */
    icon?: IconType;
    /**
     * 菜单名
     */
    name: string;
    /**
     * 排序号
     */
    order?: number;
    /**
     * 父级路径
     */
    parent?: string;
    /**
     * 所有父级路径
     */
    parents?: string[];
    /**
     * 菜单路径，唯一，可当作key
     */
    path: string;
    /**
     * 是否显示菜单
     * @default true
     */
    show?: boolean;
}
