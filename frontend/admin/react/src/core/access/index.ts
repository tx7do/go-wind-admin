// 权限管理模块 (Access Control)
// 对齐 Vue-Vben 版 core/access，提供完整的权限管理解决方案

// Hook
export { useAccess, getAccessStatic, checkAccessByRoles, checkAccessByCodes, checkAccessByAuthority } from './use-access';

// 组件
export { AccessControl } from './access-control';
export type { AccessControlProps } from './access-control';
