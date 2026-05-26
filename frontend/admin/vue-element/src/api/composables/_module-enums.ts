/**
 * 模块专属枚举与工具函数
 * 从 stores/modules/api 迁移而来
 */
import { computed } from "vue";
import { i18n } from "@/i18n";

const t = i18n.global.t;

// ==============================
// 菜单 (menu)
// ==============================
import type { resourceservicev1_Menu as Menu, resourceservicev1_Menu_Type as Menu_Type } from "@/api/generated/admin/service/v1";

export const menuTypeList = computed(() => [
  { value: "CATALOG", label: t("enum.menu.type.CATALOG") },
  { value: "MENU", label: t("enum.menu.type.MENU") },
  { value: "BUTTON", label: t("enum.menu.type.BUTTON") },
  { value: "EMBEDDED", label: t("enum.menu.type.EMBEDDED") },
  { value: "LINK", label: t("enum.menu.type.LINK") },
]);

export function menuTypeToName(menuType: any): string {
  const values = menuTypeList.value;
  const matchedItem = values.find((item) => item.value === menuType);
  return matchedItem ? matchedItem.label : "";
}

export function menuTypeToColor(menuType: Menu_Type) {
  switch (menuType) {
    case "BUTTON": return "#F56C6C";
    case "CATALOG": return "#27AE60";
    case "EMBEDDED": return "#4096FF";
    case "LINK": return "#9B59B6";
    case "MENU": return "#165DFF";
    default: return "#86909C";
  }
}

export const isCatalog = (type: string) => type === "CATALOG";
export const isMenu = (type: string) => type === "MENU";
export const isButton = (type: string) => type === "BUTTON";
export const isEmbedded = (type: string) => type === "EMBEDDED";
export const isLink = (type: string) => type === "LINK";

export function travelMenuChild(nodes: Menu[] | undefined, parent: Menu): boolean {
  if (nodes === undefined) return false;
  if (parent.parentId === 0 || parent.parentId === undefined) {
    if (parent?.meta?.title) parent.meta.title = t(parent?.meta?.title ?? "");
    nodes.push(parent);
    return true;
  }
  for (const node of nodes) {
    if (node === undefined) continue;
    if (node.id === parent.parentId) {
      if (parent?.meta?.title) parent.meta.title = t(parent?.meta?.title ?? "");
      if (node.children !== undefined) node.children.push(parent);
      return true;
    }
    if (travelMenuChild(node.children, parent)) return true;
  }
  return false;
}

export function buildMenuTree(menus: Menu[]): Menu[] {
  const tree: Menu[] = [];
  for (const menu of menus) {
    if (!menu) continue;
    if (menu.parentId !== 0 && menu.parentId !== undefined) continue;
    if (menu?.meta?.title) menu.meta.title = t(menu?.meta?.title ?? "");
    tree.push(menu);
  }
  for (const menu of menus) {
    if (!menu) continue;
    if (menu.parentId === 0 || menu.parentId === undefined) continue;
    if (travelMenuChild(tree, menu)) continue;
    if (menu?.meta?.title) menu.meta.title = t(menu?.meta?.title ?? "");
    tree.push(menu);
  }
  return tree;
}

// ==============================
// 用户 (user)
// ==============================
import type { identityservicev1_User_Gender as User_Gender, identityservicev1_User_Status as User_Status } from "@/api/generated/admin/service/v1";

export const userStatusList = computed(() => [
  { value: "NORMAL", label: t("enum.user.status.NORMAL") },
  { value: "DISABLED", label: t("enum.user.status.DISABLED") },
  { value: "PENDING", label: t("enum.user.status.PENDING") },
  { value: "LOCKED", label: t("enum.user.status.LOCKED") },
  { value: "EXPIRED", label: t("enum.user.status.EXPIRED") },
  { value: "CLOSED", label: t("enum.user.status.CLOSED") },
]);

const USER_STATUS_COLOR_MAP: Record<string, string> = {
  NORMAL: "#4096FF",
  DISABLED: "#909399",
  PENDING: "#FF9A2E",
  LOCKED: "#F56C6C",
  TERMINATED: "#F53F3F",
  EXPIRED: "#C9CDD4",
  CLOSED: "#86909C",
  DEFAULT: "#86909C",
};

export function userStatusToColor(status: User_Status) {
  return USER_STATUS_COLOR_MAP[status as string] || USER_STATUS_COLOR_MAP.DEFAULT;
}

export function userStatusToName(status?: User_Status) {
  const values = userStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

export const genderList = computed(() => [
  { value: "SECRET", label: t("enum.gender.SECRET") },
  { value: "MALE", label: t("enum.gender.MALE") },
  { value: "FEMALE", label: t("enum.gender.FEMALE") },
]);

export function genderToName(gender?: User_Gender) {
  const values = genderList.value;
  const matchedItem = values.find((item) => item.value === gender);
  return matchedItem ? matchedItem.label : "";
}

export function genderToColor(gender?: User_Gender) {
  switch (gender) {
    case "FEMALE": return "#F77272";
    case "MALE": return "#4096FF";
    case "SECRET": return "#86909C";
    default: return "#C9CDD4";
  }
}

// ==============================
// 组织单位 (org-unit)
// ==============================
import type { identityservicev1_OrgUnit as OrgUnit, identityservicev1_OrgUnit_Status as OrgUnit_Status, identityservicev1_OrgUnit_Type as OrgUnit_Type } from "@/api/generated/admin/service/v1";

export const orgUnitStatusList = computed(() => [
  { value: "ON", label: t("enum.status.ON") },
  { value: "OFF", label: t("enum.status.OFF") },
]);

export function orgUnitStatusToName(status: OrgUnit_Status) {
  const values = orgUnitStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

export function orgUnitStatusToColor(status: OrgUnit_Status) {
  switch (status) {
    case "OFF": return "#8C8C8C";
    case "ON": return "#52C41A";
    default: return "#C9CDD4";
  }
}

export const orgUnitTypeList = computed(() => {
  const typeOrder: OrgUnit_Type[] = [
    "COMPANY", "DIVISION", "DEPARTMENT", "TEAM", "PROJECT",
    "COMMITTEE", "REGION", "SUBSIDIARY", "BRANCH", "OTHER",
  ];
  return typeOrder.map((type) => ({ value: type, label: t(`enum.orgUnit.type.${type}`) }));
});

export const orgUnitTypeListForQuery = computed(() => {
  const queryAllowTypes: OrgUnit_Type[] = [
    "BRANCH", "COMMITTEE", "COMPANY", "DEPARTMENT", "DIVISION",
    "OTHER", "PROJECT", "REGION", "SUBSIDIARY", "TEAM",
  ];
  const allowTypeSet = new Set(queryAllowTypes);
  return orgUnitTypeList.value.filter((item) => allowTypeSet.has(item.value));
});

export function orgUnitTypeToName(orgUnitType: OrgUnit_Type) {
  const values = orgUnitTypeList.value;
  const matchedItem = values.find((item) => item.value === orgUnitType);
  return matchedItem ? matchedItem.label : "";
}

const ORG_UNIT_COLOR_MAP: Record<string, string> = {
  BRANCH: "#4096FF", COMMITTEE: "#00B42A", COMPANY: "#165DFF",
  DEPARTMENT: "#722ED1", DIVISION: "#FF7D00", OTHER: "#86909C",
  PROJECT: "#F53F3F", REGION: "#14C9C9", SUBSIDIARY: "#6B778C",
  TEAM: "#FFC53D", DEFAULT: "#C9CDD4",
};

export function orgUnitTypeToColor(orgUnitType: OrgUnit_Type) {
  return ORG_UNIT_COLOR_MAP[orgUnitType as string] || ORG_UNIT_COLOR_MAP.DEFAULT;
}

export const findOrgUnit = (list: OrgUnit[], id: number): null | OrgUnit | undefined => {
  for (const item of list) {
    // eslint-disable-next-line eqeqeq
    if (item.id == id) return item;
    if (item.children && item.children.length > 0) {
      const found = findOrgUnit(item.children, id);
      if (found) return found;
    }
  }
  return null;
};

// ==============================
// 职位 (position)
// ==============================
import type { identityservicev1_Position_Status as Position_Status, identityservicev1_Position_Type as Position_Type } from "@/api/generated/admin/service/v1";

export const membershipPositionStatusList = computed(() => [
  { value: "PROBATION", label: t("enum.membershipPosition.status.PROBATION") },
  { value: "ACTIVE", label: t("enum.membershipPosition.status.ACTIVE") },
  { value: "LEAVE", label: t("enum.membershipPosition.status.LEAVE") },
  { value: "RESIGNED", label: t("enum.membershipPosition.status.RESIGNED") },
  { value: "TERMINATED", label: t("enum.membershipPosition.status.TERMINATED") },
  { value: "EXPIRED", label: t("enum.membershipPosition.status.EXPIRED") },
]);

export function membershipPositionStatusToName(status: any) {
  const values = membershipPositionStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

const MEMBERSHIP_POSITION_STATUS_COLOR_MAP: Record<string, string> = {
  PROBATION: "#4096FF", ACTIVE: "#00B42A", LEAVE: "#FF9A2E",
  RESIGNED: "#F56C6C", TERMINATED: "#F53F3F", EXPIRED: "#909399", DEFAULT: "#C9CDD4",
};

export function membershipPositionStatusToColor(status: Position_Status) {
  return MEMBERSHIP_POSITION_STATUS_COLOR_MAP[status as string] || MEMBERSHIP_POSITION_STATUS_COLOR_MAP.DEFAULT;
}

export const positionTypeList = computed(() => [
  { value: "REGULAR", label: t("enum.position.type.REGULAR") },
  { value: "LEADER", label: t("enum.position.type.LEADER") },
  { value: "MANAGER", label: t("enum.position.type.MANAGER") },
  { value: "INTERN", label: t("enum.position.type.INTERN") },
  { value: "CONTRACT", label: t("enum.position.type.CONTRACT") },
  { value: "OTHER", label: t("enum.position.type.OTHER") },
]);

export function positionTypeToName(status: Position_Status) {
  const values = positionTypeList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

const POSITION_TYPE_COLOR_THEME: Record<string, Record<string, string>> = {
  light: {
    REGULAR: "#165DFF", LEADER: "#722ED1", MANAGER: "#FF7D00",
    INTERN: "#52C41A", CONTRACT: "#14C9C9", OTHER: "#86909C", DEFAULT: "#C9CDD4",
  },
  dark: {
    REGULAR: "#2F77FF", LEADER: "#8542E7", MANAGER: "#FF9529",
    INTERN: "#67E037", CONTRACT: "#20E0E0", OTHER: "#9BA3AD", DEFAULT: "#DCE0E6",
  },
};

export function positionTypeToColor(positionType: Position_Type, theme: "dark" | "light" = "light"): string {
  const colorMap = POSITION_TYPE_COLOR_THEME[theme];
  return colorMap[positionType as string] || colorMap.DEFAULT;
}

// ==============================
// 租户 (tenant)
// ==============================
import type { identityservicev1_Tenant_AuditStatus as Tenant_AuditStatus, identityservicev1_Tenant_Status as Tenant_Status, identityservicev1_Tenant_Type as Tenant_Type } from "@/api/generated/admin/service/v1";

export const tenantTypeList = computed(() => [
  { value: "TRIAL", label: t("enum.tenant.type.TRIAL") },
  { value: "PAID", label: t("enum.tenant.type.PAID") },
  { value: "INTERNAL", label: t("enum.tenant.type.INTERNAL") },
  { value: "PARTNER", label: t("enum.tenant.type.PARTNER") },
  { value: "CUSTOM", label: t("enum.tenant.type.CUSTOM") },
]);

export function tenantTypeToName(tenantType: Tenant_Type) {
  const values = tenantTypeList.value;
  const matchedItem = values.find((item) => item.value === tenantType);
  return matchedItem ? matchedItem.label : "";
}

export function tenantTypeToColor(tenantType: Tenant_Type) {
  switch (tenantType) {
    case "CUSTOM": return "#0050B3";
    case "INTERNAL": return "#1890FF";
    case "PAID": return "#52C41A";
    case "PARTNER": return "#722ED1";
    case "TRIAL": return "#FF7D00";
    default: return "#8C8C8C";
  }
}

export const tenantStatusList = computed(() => [
  { value: "ON", label: t("enum.tenant.status.ON") },
  { value: "OFF", label: t("enum.tenant.status.OFF") },
  { value: "EXPIRED", label: t("enum.tenant.status.EXPIRED") },
  { value: "FREEZE", label: t("enum.tenant.status.FREEZE") },
]);

export function tenantStatusToName(tenantStatus: Tenant_Status) {
  const values = tenantStatusList.value;
  const matchedItem = values.find((item) => item.value === tenantStatus);
  return matchedItem ? matchedItem.label : "";
}

export function tenantStatusToColor(tenantStatus: Tenant_Status) {
  switch (tenantStatus) {
    case "EXPIRED": return "#F5222D";
    case "FREEZE": return "#FAAD14";
    case "OFF": return "#8C8C8C";
    case "ON": return "#52C41A";
    default: return "#8C8C8C";
  }
}

export const tenantAuditStatusList = computed(() => [
  { value: "PENDING", label: t("enum.tenant.auditStatus.PENDING") },
  { value: "APPROVED", label: t("enum.tenant.auditStatus.APPROVED") },
  { value: "REJECTED", label: t("enum.tenant.auditStatus.REJECTED") },
]);

export function tenantAuditStatusToName(tenantAuditStatus: Tenant_AuditStatus) {
  const values = tenantAuditStatusList.value;
  const matchedItem = values.find((item) => item.value === tenantAuditStatus);
  return matchedItem ? matchedItem.label : "";
}

export function tenantAuditStatusToColor(tenantAuditStatus: Tenant_AuditStatus) {
  switch (tenantAuditStatus) {
    case "APPROVED": return "#52C41A";
    case "PENDING": return "#1890FF";
    case "REJECTED": return "#F5222D";
    default: return "#8C8C8C";
  }
}

// ==============================
// 内部消息 (internal-message)
// ==============================
import type {
  internal_messageservicev1_InternalMessage_Status as InternalMessage_Status,
  internal_messageservicev1_InternalMessage_Type as InternalMessage_Type,
  internal_messageservicev1_InternalMessageRecipient_Status as InternalMessageRecipient_Status,
} from "@/api/generated/admin/service/v1";

export const internalMessageStatusList = computed(() => [
  { value: "DRAFT", label: t("enum.internalMessage.status.DRAFT") },
  { value: "PUBLISHED", label: t("enum.internalMessage.status.PUBLISHED") },
  { value: "SCHEDULED", label: t("enum.internalMessage.status.SCHEDULED") },
  { value: "REVOKED", label: t("enum.internalMessage.status.REVOKED") },
  { value: "ARCHIVED", label: t("enum.internalMessage.status.ARCHIVED") },
  { value: "DELETED", label: t("enum.internalMessage.status.DELETED") },
]);

export const internalMessageTypeList = computed(() => [
  { value: "NOTIFICATION", label: t("enum.internalMessage.type.NOTIFICATION") },
  { value: "PRIVATE", label: t("enum.internalMessage.type.PRIVATE") },
  { value: "GROUP", label: t("enum.internalMessage.type.GROUP") },
]);

export const internalMessageRecipientStatusList = computed(() => [
  { value: "SENT", label: t("enum.internalMessageRecipient.status.SENT") },
  { value: "RECEIVED", label: t("enum.internalMessageRecipient.status.RECEIVED") },
  { value: "READ", label: t("enum.internalMessageRecipient.status.READ") },
  { value: "REVOKED", label: t("enum.internalMessageRecipient.status.REVOKED") },
  { value: "DELETED", label: t("enum.internalMessageRecipient.status.DELETED") },
]);

export function internalMessageStatusLabel(value: InternalMessage_Status): string {
  const values = internalMessageStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

const INTERNAL_MESSAGE_STATUS_COLOR_MAP: Record<string, string> = {
  ARCHIVED: "#86909C", DELETED: "#C9CDD4", DRAFT: "#9CA3AF",
  PUBLISHED: "#00B42A", REVOKED: "#F53F3F", SCHEDULED: "#165DFF", DEFAULT: "#E5E7EB",
};

export function internalMessageStatusColor(status: InternalMessage_Status): string {
  return INTERNAL_MESSAGE_STATUS_COLOR_MAP[status as string] || INTERNAL_MESSAGE_STATUS_COLOR_MAP.DEFAULT;
}

export function internalMessageTypeLabel(value: InternalMessage_Type): string {
  const values = internalMessageTypeList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

const INTERNAL_MESSAGE_TYPE_COLOR_MAP: Record<string, string> = {
  GROUP: "#00B42A", NOTIFICATION: "#165DFF", PRIVATE: "#722ED1", DEFAULT: "#C9CDD4",
};

export function internalMessageTypeColor(type: InternalMessage_Type): string {
  return INTERNAL_MESSAGE_TYPE_COLOR_MAP[type as string] || INTERNAL_MESSAGE_TYPE_COLOR_MAP.DEFAULT;
}

export function internalMessageRecipientStatusLabel(value: InternalMessageRecipient_Status): string {
  const values = internalMessageRecipientStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

const INTERNAL_MESSAGE_RECIPIENT_COLOR_THEME: Record<string, Record<string, string>> = {
  light: {
    DELETED: "#C9CDD4", READ: "#86909C", RECEIVED: "#165DFF",
    REVOKED: "#F53F3F", SENT: "#4096FF", DEFAULT: "#E5E7EB",
  },
  dark: {
    DELETED: "#6E7681", READ: "#4E5969", RECEIVED: "#2F77FF",
    REVOKED: "#F87171", SENT: "#69B1FF", DEFAULT: "#4B5563",
  },
};

export function internalMessageRecipientStatusColor(
  status: InternalMessageRecipient_Status,
  theme: "dark" | "light" = "light"
): string {
  const colorMap = INTERNAL_MESSAGE_RECIPIENT_COLOR_THEME[theme];
  return colorMap[status as string] || colorMap.DEFAULT;
}

// ==============================
// 任务 (task)
// ==============================
import type { taskservicev1_Task_Type as Task_Type } from "@/api/generated/admin/service/v1";

export const taskTypeList = computed(() => [
  { value: "PERIODIC", label: t("enum.task.type.Periodic") },
  { value: "DELAY", label: t("enum.task.type.Delay") },
  { value: "WAIT_RESULT", label: t("enum.task.type.WaitResult") },
]);

export function taskTypeToName(taskType: Task_Type) {
  const values = taskTypeList.value;
  const matchedItem = values.find((item) => item.value === taskType);
  return matchedItem ? matchedItem.label : "";
}

export function taskTypeToColor(taskType: Task_Type) {
  switch (taskType) {
    case "DELAY": return "blue";
    case "PERIODIC": return "orange";
    case "WAIT_RESULT": return "purple";
    default: return "gray";
  }
}

// ==============================
// 登录策略 (login-policy)
// ==============================
import type { authenticationservicev1_LoginPolicy_Method as LoginPolicy_Method, authenticationservicev1_LoginPolicy_Type as LoginPolicy_Type } from "@/api/generated/admin/service/v1";

export const loginPolicyTypeList = computed(() => [
  { value: "BLACKLIST", label: t("enum.loginPolicy.type.BLACKLIST") },
  { value: "WHITELIST", label: t("enum.loginPolicy.type.WHITELIST") },
]);

export const loginPolicyMethodList = computed(() => [
  { value: "IP", label: t("enum.loginPolicy.method.IP") },
  { value: "MAC", label: t("enum.loginPolicy.method.MAC") },
  { value: "REGION", label: t("enum.loginPolicy.method.REGION") },
  { value: "TIME", label: t("enum.loginPolicy.method.TIME") },
  { value: "DEVICE", label: t("enum.loginPolicy.method.DEVICE") },
]);

const LOGIN_POLICY_METHOD_COLOR_MAP: Record<string, string> = {
  IP: "#4096FF", MAC: "#909399", REGION: "#FF9A2E",
  TIME: "#F56C6C", DEVICE: "#86909C", DEFAULT: "#86909C",
};

export function loginPolicyMethodToColor(methodName: LoginPolicy_Method) {
  return LOGIN_POLICY_METHOD_COLOR_MAP[methodName as string] || LOGIN_POLICY_METHOD_COLOR_MAP.DEFAULT;
}

export function loginPolicyTypeToName(typeName: LoginPolicy_Type) {
  const values = loginPolicyTypeList.value;
  const matchedItem = values.find((item) => item.value === typeName);
  return matchedItem ? matchedItem.label : "";
}

export function loginPolicyTypeToColor(typeName: LoginPolicy_Type) {
  switch (typeName) {
    case "BLACKLIST": return "red";
    case "WHITELIST": return "green";
    default: return "gray";
  }
}

export function loginPolicyMethodToName(methodName: LoginPolicy_Method) {
  const values = loginPolicyMethodList.value;
  const matchedItem = values.find((item) => item.value === methodName);
  return matchedItem ? matchedItem.label : "";
}

// ==============================
// 登录审计日志 (login-audit-log)
// ==============================
import type {
  auditservicev1_LoginAuditLog_ActionType as LoginAuditLog_ActionType,
  auditservicev1_LoginAuditLog_RiskLevel as LoginAuditLog_RiskLevel,
  auditservicev1_LoginAuditLog_Status as LoginAuditLog_Status,
} from "@/api/generated/admin/service/v1";

const COLORS = {
  neutral: "#86909C", success: "#1F7A34", warning: "#FA8C16",
  danger: "#D32F2F", info: "#1890FF",
};

const LOGIN_AUDIT_LOG_STATUS_COLOR_MAP: Record<string, string> = {
  STATUS_UNSPECIFIED: COLORS.neutral, SUCCESS: COLORS.success,
  FAILED: COLORS.danger, PARTIAL: COLORS.warning, LOCKED: COLORS.warning,
};

const LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP: Record<string, string> = {
  ACTION_TYPE_UNSPECIFIED: COLORS.neutral, LOGIN: COLORS.success,
  LOGOUT: COLORS.info, SESSION_EXPIRED: COLORS.warning,
  KICKED_OUT: COLORS.warning, PASSWORD_RESET: COLORS.warning,
};

const LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP: Record<string, string> = {
  RISK_LEVEL_UNSPECIFIED: COLORS.neutral, LOW: COLORS.success,
  MEDIUM: COLORS.warning, HIGH: COLORS.danger,
};

export function getLoginAuditLogStatusColor(status: LoginAuditLog_Status): string {
  return LOGIN_AUDIT_LOG_STATUS_COLOR_MAP[status as string] || COLORS.neutral;
}

export function getLoginAuditLogActionTypeColor(actionType: LoginAuditLog_ActionType): string {
  return LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP[actionType as string] || COLORS.neutral;
}

export function getLoginAuditLogRiskLevelColor(riskLevel: LoginAuditLog_RiskLevel): string {
  return LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP[riskLevel as string] || COLORS.neutral;
}

export function loginAuditLogStatusToName(status: LoginAuditLog_Status) {
  switch (status) {
    case "FAILED": return t("enum.loginAuditLog.status.FAILED");
    case "PARTIAL": return t("enum.loginAuditLog.status.PARTIAL");
    case "SUCCESS": return t("enum.loginAuditLog.status.SUCCESS");
    default: return "";
  }
}

export const loginAuditLogStatusList = computed(() => [
  { value: "FAILED", label: t("enum.loginAuditLog.status.FAILED") },
  { value: "PARTIAL", label: t("enum.loginAuditLog.status.PARTIAL") },
  { value: "SUCCESS", label: t("enum.loginAuditLog.status.SUCCESS") },
]);

export function loginAuditLogActionTypeToName(status: LoginAuditLog_ActionType) {
  switch (status) {
    case "LOGIN": return t("enum.loginAuditLog.actionType.LOGIN");
    case "LOGOUT": return t("enum.loginAuditLog.actionType.LOGOUT");
    case "SESSION_EXPIRED": return t("enum.loginAuditLog.actionType.SESSION_EXPIRED");
    default: return "";
  }
}

export const loginAuditLogActionTypeList = computed(() => [
  { value: "LOGIN", label: t("enum.loginAuditLog.actionType.LOGIN") },
  { value: "LOGOUT", label: t("enum.loginAuditLog.actionType.LOGOUT") },
  { value: "SESSION_EXPIRED", label: t("enum.loginAuditLog.actionType.SESSION_EXPIRED") },
]);

export function loginAuditLogRiskLevelToName(status: LoginAuditLog_RiskLevel) {
  switch (status) {
    case "HIGH": return t("enum.loginAuditLog.riskLevel.HIGH");
    case "LOW": return t("enum.loginAuditLog.riskLevel.LOW");
    case "MEDIUM": return t("enum.loginAuditLog.riskLevel.MEDIUM");
    default: return "";
  }
}

export const loginAuditLogRiskLevelList = computed(() => [
  { value: "HIGH", label: t("enum.loginAuditLog.riskLevel.HIGH") },
  { value: "LOW", label: t("enum.loginAuditLog.riskLevel.LOW") },
  { value: "MEDIUM", label: t("enum.loginAuditLog.riskLevel.MEDIUM") },
]);

// ==============================
// 操作审计日志 (operation-audit-log)
// ==============================
import type { auditservicev1_OperationAuditLog_ActionType as OperationActionType } from "@/api/generated/admin/service/v1";

export const operationAuditLogActionList = computed(() => [
  { value: "CREATE", label: t("enum.operationAuditLog.action.CREATE") },
  { value: "UPDATE", label: t("enum.operationAuditLog.action.UPDATE") },
  { value: "DELETE", label: t("enum.operationAuditLog.action.DELETE") },
  { value: "READ", label: t("enum.operationAuditLog.action.READ") },
  { value: "ASSIGN", label: t("enum.operationAuditLog.action.ASSIGN") },
  { value: "UNASSIGN", label: t("enum.operationAuditLog.action.UNASSIGN") },
  { value: "EXPORT", label: t("enum.operationAuditLog.action.EXPORT") },
  { value: "IMPORT", label: t("enum.operationAuditLog.action.IMPORT") },
  { value: "OTHER", label: t("enum.operationAuditLog.action.OTHER") },
]);

const OPERATION_AUDIT_LOG_ACTION_COLOR_MAP: Record<string, string> = {
  CREATE: "#1677FF", UPDATE: "#597EF7", DELETE: "#FF4D4F", READ: "#6B7280",
  ASSIGN: "#722ED1", UNASSIGN: "#A855F7", EXPORT: "#00B42A", IMPORT: "#36CFC9",
  OTHER: "#86909C", DEFAULT: "#86909C",
};

export function operationAuditLogActionToColor(action: OperationActionType) {
  return OPERATION_AUDIT_LOG_ACTION_COLOR_MAP[action as string] || OPERATION_AUDIT_LOG_ACTION_COLOR_MAP.DEFAULT;
}

export function operationAuditLogActionToName(action: OperationActionType) {
  const values = operationAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : "";
}

// ==============================
// 权限审计日志 (permission-audit-log)
// ==============================
import type { auditservicev1_PermissionAuditLog_ActionType as PermissionAuditActionType } from "@/api/generated/admin/service/v1";

export const permissionAuditLogActionList = computed(() => [
  { value: "GRANT", label: t("enum.permissionAuditLog.action.GRANT") },
  { value: "REVOKE", label: t("enum.permissionAuditLog.action.REVOKE") },
  { value: "UPDATE", label: t("enum.permissionAuditLog.action.UPDATE") },
  { value: "RESET", label: t("enum.permissionAuditLog.action.RESET") },
  { value: "CREATE", label: t("enum.permissionAuditLog.action.CREATE") },
  { value: "DELETE", label: t("enum.permissionAuditLog.action.DELETE") },
  { value: "ASSIGN", label: t("enum.permissionAuditLog.action.ASSIGN") },
  { value: "UNASSIGN", label: t("enum.permissionAuditLog.action.UNASSIGN") },
  { value: "BULK_GRANT", label: t("enum.permissionAuditLog.action.BULK_GRANT") },
  { value: "BULK_REVOKE", label: t("enum.permissionAuditLog.action.BULK_REVOKE") },
  { value: "EXPIRE", label: t("enum.permissionAuditLog.action.EXPIRE") },
  { value: "RESUME", label: t("enum.permissionAuditLog.action.RESUME") },
  { value: "ROLLBACK", label: t("enum.permissionAuditLog.action.ROLLBACK") },
  { value: "OTHER", label: t("enum.permissionAuditLog.action.OTHER") },
]);

const PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP: Record<string, string> = {
  GRANT: "#1677FF", REVOKE: "#FF4D4F", UPDATE: "#597EF7", RESET: "#6B7280",
  CREATE: "#722ED1", DELETE: "#FF4D4F", ASSIGN: "#00B42A", UNASSIGN: "#FF7875",
  BULK_GRANT: "#36CFC9", BULK_REVOKE: "#FFC0C2", EXPIRE: "#FF4D4F",
  RESUME: "#00B42A", ROLLBACK: "#597EF7", OTHER: "#86909C", DEFAULT: "#86909C",
};

export function permissionAuditLogActionToColor(action: PermissionAuditActionType) {
  return PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP[action as string] || PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP.DEFAULT;
}

export function permissionAuditLogActionToName(action: PermissionAuditActionType) {
  const values = permissionAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : "";
}

// ==============================
// 数据访问审计日志 (data-access-audit-log)
// ==============================
import type { auditservicev1_DataAccessAuditLog_AccessType as AccessType } from "@/api/generated/admin/service/v1";

export const dataAccessAuditLogAccessTypeList = computed(() => [
  { value: "SELECT", label: t("enum.dataAccessAuditLog.accessType.SELECT") },
  { value: "INSERT", label: t("enum.dataAccessAuditLog.accessType.INSERT") },
  { value: "UPDATE", label: t("enum.dataAccessAuditLog.accessType.UPDATE") },
  { value: "DELETE", label: t("enum.dataAccessAuditLog.accessType.DELETE") },
  { value: "VIEW", label: t("enum.dataAccessAuditLog.accessType.VIEW") },
  { value: "BULK_READ", label: t("enum.dataAccessAuditLog.accessType.BULK_READ") },
  { value: "EXPORT", label: t("enum.dataAccessAuditLog.accessType.EXPORT") },
  { value: "IMPORT", label: t("enum.dataAccessAuditLog.accessType.IMPORT") },
  { value: "DDL_CREATE", label: t("enum.dataAccessAuditLog.accessType.DDL_CREATE") },
  { value: "DDL_ALTER", label: t("enum.dataAccessAuditLog.accessType.DDL_ALTER") },
  { value: "DDL_DROP", label: t("enum.dataAccessAuditLog.accessType.DDL_DROP") },
  { value: "METADATA_READ", label: t("enum.dataAccessAuditLog.accessType.METADATA_READ") },
  { value: "SCAN", label: t("enum.dataAccessAuditLog.accessType.SCAN") },
  { value: "ADMIN_OPERATION", label: t("enum.dataAccessAuditLog.accessType.ADMIN_OPERATION") },
  { value: "OTHER", label: t("enum.dataAccessAuditLog.accessType.OTHER") },
]);

const DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP: Record<string, string> = {
  SELECT: "#1677FF", INSERT: "#597EF7", UPDATE: "#597EF7", DELETE: "#FF4D4F",
  VIEW: "#6B7280", BULK_READ: "#6B7280", EXPORT: "#00B42A", IMPORT: "#36CFC9",
  DDL_CREATE: "#722ED1", DDL_ALTER: "#A855F7", DDL_DROP: "#FF4D4F",
  METADATA_READ: "#86909C", SCAN: "#86909C", ADMIN_OPERATION: "#722ED1",
  OTHER: "#86909C", DEFAULT: "#86909C",
};

export function dataAccessAuditLogAccessTypeToColor(accessType: AccessType) {
  return DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP[accessType as string] || DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP.DEFAULT;
}

export function dataAccessAuditLogAccessTypeToName(accessType: AccessType) {
  const values = dataAccessAuditLogAccessTypeList.value;
  const matchedItem = values.find((item) => item.value === accessType);
  return matchedItem ? matchedItem.label : "";
}

// ==============================
// 文件 (file)
// ==============================
import type { storageservicev1_OSSProvider as OSSProvider } from "@/api/generated/admin/service/v1";

export const ossProviderList = computed(() => [
  { value: "LOCAL", label: t("enum.ossProvider.LOCAL") },
  { value: "MINIO", label: t("enum.ossProvider.MINIO") },
  { value: "ALIYUN", label: t("enum.ossProvider.ALIYUN") },
  { value: "QINIU", label: t("enum.ossProvider.QINIU") },
  { value: "TENCENT", label: t("enum.ossProvider.TENCENT") },
  { value: "BAIDU", label: t("enum.ossProvider.BAIDU") },
  { value: "HUAWEI", label: t("enum.ossProvider.HUAWEI") },
  { value: "AWS", label: t("enum.ossProvider.AWS") },
  { value: "AZURE", label: t("enum.ossProvider.AZURE") },
  { value: "GOOGLE", label: t("enum.ossProvider.GOOGLE") },
]);

export function ossProviderLabel(value: OSSProvider): string {
  const values = ossProviderList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

const OSS_PROVIDER_COLOR_MAP: Record<string, string> = {
  LOCAL: "#36D399", MINIO: "#2563EB", QINIU: "#722ED1", ALIYUN: "#FF6A00",
  TENCENT: "#12B7F5", BAIDU: "#4080FF", HUAWEI: "#E64340", AWS: "#FF9900",
  AZURE: "#0078D4", GOOGLE: "#4285F4", DEFAULT: "#C9CDD4",
};

export function ossProviderColor(type: OSSProvider): string {
  return OSS_PROVIDER_COLOR_MAP[type as string] || OSS_PROVIDER_COLOR_MAP.DEFAULT;
}

// ==============================
// 权限 (permission)
// ==============================
import type { permissionservicev1_Permission as Permission, permissionservicev1_PermissionGroup as PermissionGroup } from "@/api/generated/admin/service/v1";

export const roleDataScopeList = computed(() => [
  { label: t("enum.role.dataScope.ALL"), value: "ALL" },
  { label: t("enum.role.dataScope.UNIT_AND_CHILD"), value: "UNIT_AND_CHILD" },
  { label: t("enum.role.dataScope.UNIT_ONLY"), value: "UNIT_ONLY" },
  { label: t("enum.role.dataScope.SELECTED_UNITS"), value: "SELECTED_UNITS" },
  { label: t("enum.role.dataScope.SELF"), value: "SELF" },
]);

const DATA_SCOPE_COLOR_MAP: Record<string, string> = {
  ALL: "#F53F3F", UNIT_AND_CHILD: "#165DFF", UNIT_ONLY: "#FF7D00",
  SELECTED_UNITS: "#722ED1", SELF: "#86909C", DEFAULT: "#C9CDD4",
};

export function dataScopeToColor(dataScope: any): string {
  return DATA_SCOPE_COLOR_MAP[dataScope as string] || DATA_SCOPE_COLOR_MAP.DEFAULT;
}

export function roleDataScopeToName(dataScope: any) {
  const values = roleDataScopeList.value;
  const matchedItem = values.find((item) => item.value === dataScope);
  return matchedItem ? matchedItem.label : "";
}

interface PermissionTreeDataNode {
  key: number | string;
  title: string;
  children?: PermissionTreeDataNode[];
  permission?: Permission;
}

export function buildPermissionTree(
  permissionGroups: PermissionGroup[],
  permissions: Permission[]
): PermissionTreeDataNode[] {
  const groupNodes: Map<string, PermissionTreeDataNode> = new Map();
  const idToKey: Map<number | string, string> = new Map();
  const nameToKey: Map<string, string> = new Map();
  const defaultKey = "group-default";
  groupNodes.set(defaultKey, { key: defaultKey, title: "", children: [] });

  function processGroup(g: PermissionGroup, idxPath: string[]): PermissionTreeDataNode {
    const idPart = g.id ?? `idx-${idxPath.join("-")}`;
    const key = `group-${idPart}`;
    const title = typeof (g as any).name === "string" ? (g as any).name : String(g.id ?? "");
    const node: PermissionTreeDataNode = { key, title, children: [] };
    groupNodes.set(key, node);
    if (g.id !== undefined && g.id !== null) idToKey.set(g.id as any, key);
    if (typeof (g as any).name === "string" && (g as any).name) nameToKey.set((g as any).name, key);
    const children = (g as any).children;
    if (Array.isArray(children) && children.length > 0) {
      children.forEach((child: PermissionGroup, idx: number) => {
        const childNode = processGroup(child, [...idxPath, String(idx)]);
        node.children = node.children ?? [];
        node.children.push(childNode);
      });
    }
    return node;
  }

  permissionGroups.forEach((g, idx) => processGroup(g, [String(idx)]));

  permissions.forEach((perm, idx) => {
    let targetKey: string | undefined;
    if (perm && (perm as any).groupId !== undefined && (perm as any).groupId !== null)
      targetKey = idToKey.get((perm as any).groupId) ?? `group-${(perm as any).groupId}`;
    if (!targetKey && typeof (perm as any).groupName === "string")
      targetKey = nameToKey.get((perm as any).groupName) ?? `group-${(perm as any).groupName}`;
    if (!targetKey || !groupNodes.has(targetKey)) targetKey = defaultKey;
    const childNode: PermissionTreeDataNode = {
      key: perm.id ?? `perm-${idx}`,
      title: typeof (perm as any).name === "string" ? `${(perm as any).name} (${(perm as any).code})` : "",
      permission: perm,
    };
    const parent = groupNodes.get(targetKey);
    if (parent) { parent.children = parent.children ?? []; parent.children.push(childNode); }
  });

  const result: PermissionTreeDataNode[] = permissionGroups.map((g, idx) => {
    const idPart = g.id ?? `idx-${idx}`;
    const key = `group-${idPart}`;
    return groupNodes.get(key) ?? { key, title: typeof (g as any).name === "string" ? (g as any).name : "", children: [] };
  });
  const defaultNode = groupNodes.get(defaultKey);
  if (defaultNode && (defaultNode.children?.length ?? 0) > 0) result.push(defaultNode);
  return result;
}

// ==============================
// 权限分组 (permission-group)
// ==============================

export function travelPermissionGroupChild(
  nodes: PermissionGroup[] | undefined,
  parent: PermissionGroup
): boolean {
  if (nodes === undefined) return false;
  if (parent.parentId === 0 || parent.parentId === undefined) {
    if (parent?.name) parent.name = t(parent?.name ?? "");
    nodes.push(parent);
    return true;
  }
  for (const node of nodes) {
    if (node === undefined) continue;
    if (node.id === parent.parentId) {
      if (parent?.name) parent.name = t(parent?.name ?? "");
      if (node.children !== undefined) node.children.push(parent);
      return true;
    }
    if (travelPermissionGroupChild(node.children, parent)) return true;
  }
  return false;
}

export function buildPermissionGroupTree(groups: PermissionGroup[]): PermissionGroup[] {
  const tree: PermissionGroup[] = [];
  for (const group of groups) {
    if (!group) continue;
    if (group.parentId !== 0 && group.parentId !== undefined) continue;
    if (group?.name) group.name = t(group?.name ?? "");
    tree.push(group);
  }
  for (const group of groups) {
    if (!group) continue;
    if (group.parentId === 0 || group.parentId === undefined) continue;
    if (travelPermissionGroupChild(tree, group)) continue;
    if (group?.name) group.name = t(group?.name ?? "");
    tree.push(group);
  }
  return tree;
}

// ==============================
// API (api) - convertApiToTree
// ==============================

export function convertApiToTree(apis: any[]): any[] {
  const tree: any[] = [];
  for (const api of apis) {
    if (!api) continue;
    if (api.parentId !== 0 && api.parentId !== undefined) continue;
    tree.push(api);
  }
  for (const api of apis) {
    if (!api) continue;
    if (api.parentId === 0 || api.parentId === undefined) continue;
    function findParent(nodes: any[]): boolean {
      for (const node of nodes) {
        if (node.id === api.parentId) {
          if (node.children !== undefined) node.children.push(api);
          return true;
        }
        if (node.children && findParent(node.children)) return true;
      }
      return false;
    }
    if (findParent(tree)) continue;
    tree.push(api);
  }
  return tree;
}
