import type { BasicUserInfo } from '@vben-core/typings';

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
   * 租户id（后端 uint64 → proto3 JSON 为字符串）
   */
  tenantId: string;
}

export type { UserInfo };
