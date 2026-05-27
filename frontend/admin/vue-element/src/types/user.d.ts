declare global {
  /**
   * 用户基本信息（从 identityservicev1_User 适配）
   * 保留此接口作为前端 UI 层的稳定类型，
   * 避免后端 proto 类型变更直接影响组件代码
   */
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
