declare global {
    type ClassType = Array<object | string> | object | string;

    /**
     * React Router 路由 handle 元数据类型
     * 用于 useMatches() 获取路由附加信息
     */
    interface RouteHandle {
        title?: string;
        icon?: string;
        authority?: string[];
        [key: string]: any;
    }

    interface BasicOption {
        label: string;
        value: string;
    }

    type SelectOption = BasicOption;

    type TabOption = BasicOption;

    interface BasicUserInfo {
        [key: string]: any;

        /**
         * 头像
         */
        avatar: string;
        /**
         * 用户id
         */
        id: number;

        /**
         * 用户昵称
         */
        nickname: string;
        /**
         * 用户实名
         */
        realname: string;
        /**
         * 用户角色
         */
        roles?: string[];
        /**
         * 租户id
         */
        tenantId: number;
        /**
         * 用户名
         */
        username: string;
    }

    /** 用户信息 */
    interface UserInfo extends BasicUserInfo {
        /**
         * 用户描述
         */
        description: string;

        /**
         * 首页地址
         */
        homePath: string;

        /**
         * accessToken
         */
        token: string;

        /**
         * 租户id
         */
        tenantId: number;
    }
}

export {};
