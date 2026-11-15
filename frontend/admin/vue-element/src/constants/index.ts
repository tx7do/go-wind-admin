/**
 * 应用常量定义
 *
 * @description
 * 包含应用中所有的常量定义，包括角色、存储键名等。
 * 机制层常量（LOGIN_PATH/DEFAULT_HOME_PATH/DRAWER_WIDTH）已迁入
 * @go-wind/admin-element，此处 re-export 保持消费点不变。
 */

// 机制层常量（唯一真相在包内）
export { LOGIN_PATH, DEFAULT_HOME_PATH, DRAWER_WIDTH } from "@go-wind/admin-element";

/**
 * 应用存储前缀
 */
export const APP_PREFIX = "gowind";

/**
 * 超级管理员角色标识
 *
 * @description
 * 拥有系统最高权限，可以访问所有资源
 */
export const ROLE_ROOT = "ROOT";

/**
 * 平台租户ID
 *
 * @description
 * 用于前端识别平台租户（不参与套餐/菜单配置）
 */
export const PLATFORM_TENANT_ID = 0;
