import {create} from 'zustand';
import {persist} from 'zustand/middleware';

// ========== 状态定义 ==========
export interface UserState {
    // 核心状态
    userInfo: BasicUserInfo | null;
    userRoles: string[];         // 角色码（来自 userInfo.roles）
    accessCodes: string[];       // 权限码（来自 GetMyPermissionCode）

    // 计算属性（函数形式，React 中不需要 computed）
    tenantId: number | null;
    isLoggedIn: boolean;

    // 动作
    setUserInfo: (info: BasicUserInfo | null) => void;
    setUserRoles: (roles: string[]) => void;
    setAccessCodes: (codes: string[]) => void;
    isTenantUser: () => boolean;
    $reset: () => void;
}

// ========== Store 实现 ==========
export const useUserStore = create<UserState>()(
    persist(
        (set, get) => ({
            // 初始状态
            userInfo: null,
            userRoles: [],
            accessCodes: [],

            // 计算属性（函数形式，每次调用时计算）
            get tenantId() {
                return get().userInfo?.tenantId ?? null;
            },

            get isLoggedIn() {
                return get().userInfo !== null;
            },

            // 设置用户信息（同时更新角色）
            setUserInfo: (info) => {
                set({
                    userInfo: info,
                    userRoles: info?.roles ?? [],
                });
            },

            // 单独设置角色
            setUserRoles: (roles) => {
                set({userRoles: roles});
            },

            // 单独设置权限码
            setAccessCodes: (codes) => {
                set({accessCodes: codes});
            },

            // 判断是否为租户用户
            isTenantUser: () => {
                const {tenantId} = get();
                return tenantId !== null && tenantId !== undefined && tenantId > 0;
            },

            // 重置状态
            $reset: () => {
                set({
                    userInfo: null,
                    userRoles: [],
                    accessCodes: [],
                });
            },
        }),
        {
            name: 'user-storage', // localStorage key
            // ✅ 可选：只持久化用户信息（角色可从用户信息派生）
            partialize: (state) => ({
                userInfo: state.userInfo,
                // userRoles 可从 userInfo?.roles 派生，不持久化避免不一致
            }),
        }
    )
);
