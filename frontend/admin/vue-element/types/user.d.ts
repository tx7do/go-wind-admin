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
