/**
 * API Hooks 索引文件
 * 导出所有业务模块的 hooks 及其枚举工具函数
 */

// 通用枚举与工具函数
export * from "./shared";

// 认证相关
export * from "./auth";
export * from "./access_key";

// MFA 相关
export * from "./mfa";

// 管理门户相关
export * from "./admin-portal";

// 用户相关
export * from "./user";

// 用户个人资料
export * from "./user-profile";

// 租户管理
export * from "./tenant";
export * from "./plan";

// 组织人员管理 (OPM)
export * from "./org-unit";
export * from "./position";

// 权限管理
export * from "./permission";
export * from "./permission-group";
export * from "./role";

// 系统管理
export * from "./menu";
export * from "./api";
export * from "./dict";
export * from "./file";
export * from "./file-transfer";
export * from "./task";
export * from "./login-policy";
export * from "./language";
export * from "./config";

// 日志审计
export * from "./login-audit-log";
export * from "./api-audit-log";
export * from "./operation-audit-log";
export * from "./data-access-audit-log";
export * from "./permission-audit-log";
export * from "./policy-evaluation-log";

// 缓存监控
export * from "./redis-cache-monitor";

// 在线会话
export * from "./online-session";

// 服务监控
export * from "./server-monitor";

// 通知渠道
export * from "./notification-channel";

// 通知投递台账（只读）
export * from "./notification-delivery";

// 通知路由规则（事件类型 → 渠道 + 派发方式）
export * from "./notification-rule";

// 首页分析概览
export * from "./dashboard";

// 内部消息
export * from "./internal-message";

// 脚本管理
export * from "./script";
