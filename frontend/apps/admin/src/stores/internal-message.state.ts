import { computed } from 'vue';

import { $t } from '@vben/locales';

import { defineStore } from 'pinia';

import {
  createInternalMessageRecipientServiceClient,
  createInternalMessageServiceClient,
  type internal_messageservicev1_InternalMessage_Status as InternalMessage_Status,
  type internal_messageservicev1_InternalMessage_Type as InternalMessage_Type,
  type internal_messageservicev1_InternalMessageRecipient_Status as InternalMessageRecipient_Status,
  type internal_messageservicev1_SendMessageRequest as SendMessageRequest,
} from '#/generated/api/admin/service/v1';
import { makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useInternalMessageStore = defineStore('internal_message', () => {
  const internalMessageService = createInternalMessageServiceClient(
    requestClientRequestHandler,
  );

  const internalMessageRecipientService =
    createInternalMessageRecipientServiceClient(requestClientRequestHandler);

  /**
   * 查询消息列表
   */
  async function listMessage(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await internalMessageService.ListMessage({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: orderBy ?? [],
      query: makeQueryString(formValues ?? null),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取消息
   */
  async function getMessage(id: number) {
    return await internalMessageService.GetMessage({ id });
  }

  /**
   * 更新消息
   */
  async function updateMessage(id: number, values: object) {
    return await internalMessageService.UpdateMessage({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除消息
   */
  async function deleteMessage(id: number) {
    return await internalMessageService.DeleteMessage({
      id,
    });
  }

  /**
   * 获取用户的收件箱列表
   */
  async function listUserInbox(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await internalMessageRecipientService.ListUserInbox({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: orderBy ?? [],
      query: makeQueryString(formValues ?? null),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 将通知标记为已读
   */
  async function markNotificationAsRead(
    userId: number,
    recipientIds: number[],
  ) {
    return await internalMessageRecipientService.MarkNotificationAsRead({
      userId,
      recipientIds,
    });
  }

  /**
   * 删除收件箱中的通知
   */
  async function deleteNotificationFromInbox(
    userId: number,
    recipientIds: number[],
  ) {
    return await internalMessageRecipientService.DeleteNotificationFromInbox({
      userId,
      recipientIds,
    });
  }

  /**
   * 撤销某条消息
   */
  async function revokeMessage(userId: number, messageId: number) {
    return await internalMessageService.RevokeMessage({
      messageId,
      userId,
    });
  }

  /**
   * 发送消息
   */
  async function sendMessage(request: SendMessageRequest) {
    return await internalMessageService.SendMessage(request);
  }

  function $reset() {}

  return {
    $reset,
    listMessage,
    getMessage,
    updateMessage,
    deleteMessage,
    listUserInbox,
    sendMessage,
    revokeMessage,
    markNotificationAsRead,
    deleteNotificationFromInbox,
  };
});

export const internalMessageStatusList = computed(() => [
  {
    value: 'DRAFT',
    label: $t('enum.internalMessageStatus.DRAFT'),
  },
  {
    value: 'PUBLISHED',
    label: $t('enum.internalMessageStatus.PUBLISHED'),
  },
  {
    value: 'SCHEDULED',
    label: $t('enum.internalMessageStatus.SCHEDULED'),
  },
  {
    value: 'REVOKED',
    label: $t('enum.internalMessageStatus.REVOKED'),
  },
  {
    value: 'ARCHIVED',
    label: $t('enum.internalMessageStatus.ARCHIVED'),
  },
  {
    value: 'DELETED',
    label: $t('enum.internalMessageStatus.DELETED'),
  },
]);

export const internalMessageTypeList = computed(() => [
  {
    value: 'NOTIFICATION',
    label: $t('enum.internalMessageType.NOTIFICATION'),
  },
  {
    value: 'PRIVATE',
    label: $t('enum.internalMessageType.PRIVATE'),
  },
  {
    value: 'GROUP',
    label: $t('enum.internalMessageType.GROUP'),
  },
]);

export const internalMessageRecipientStatusList = computed(() => [
  {
    value: 'SENT',
    label: $t('enum.internalMessageRecipientStatus.SENT'),
  },
  {
    value: 'RECEIVED',
    label: $t('enum.internalMessageRecipientStatus.RECEIVED'),
  },
  {
    value: 'READ',
    label: $t('enum.internalMessageRecipientStatus.READ'),
  },
  {
    value: 'REVOKED',
    label: $t('enum.internalMessageRecipientStatus.REVOKED'),
  },
  {
    value: 'DELETED',
    label: $t('enum.internalMessageRecipientStatus.DELETED'),
  },
]);

export function internalMessageStatusLabel(
  value: InternalMessage_Status,
): string {
  const values = internalMessageStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const INTERNAL_MESSAGE_STATUS_COLOR_MAP = {
  ARCHIVED: '#86909C', // 归档：中深灰（已存档，弱化但可识别）
  DELETED: '#C9CDD4', // 已删除：浅灰（极弱化，接近背景）
  DRAFT: '#9CA3AF', // 草稿：中灰（未完成，中性状态）
  PUBLISHED: '#00B42A', // 已发布：企业绿（成功、正向状态）
  REVOKED: '#F53F3F', // 已撤回：企业红（异常、高危状态）
  SCHEDULED: '#165DFF', // 计划发送：企业蓝（待执行、流程中）
  DEFAULT: '#E5E7EB', // 默认：浅中性灰（兜底，避免空值）
} as const satisfies Record<'DEFAULT' | InternalMessage_Status, string>;

/**
 * 内部消息状态映射对应颜色
 * @param status 内部消息状态（ARCHIVED/DELETED/DRAFT/PUBLISHED/REVOKED/SCHEDULED）
 * @returns 标准化十六进制颜色值（兜底中性灰，避免样式异常）
 */
export function internalMessageStatusColor(
  status: InternalMessage_Status,
): string {
  return (
    INTERNAL_MESSAGE_STATUS_COLOR_MAP[status] ||
    INTERNAL_MESSAGE_STATUS_COLOR_MAP.DEFAULT
  );
}

export function internalMessageTypeLabel(value: InternalMessage_Type): string {
  const values = internalMessageTypeList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const INTERNAL_MESSAGE_TYPE_COLOR_MAP = {
  GROUP: '#00B42A', // 群聊：企业绿（协作、活跃、多人互动）
  NOTIFICATION: '#165DFF', // 通知：企业蓝（官方、提醒、系统推送）
  PRIVATE: '#722ED1', // 私信：企业紫（私密、一对一、个人沟通）
  DEFAULT: '#C9CDD4', // 默认：中性浅灰（兜底，避免样式异常）
} as const satisfies Record<'DEFAULT' | InternalMessage_Type, string>;

/**
 * 内部消息类型映射对应颜色
 * @param type 内部消息类型（GROUP/NOTIFICATION/PRIVATE）
 * @returns 标准化十六进制颜色值（兜底中性灰，避免样式错乱）
 */
export function internalMessageTypeColor(type: InternalMessage_Type): string {
  return (
    INTERNAL_MESSAGE_TYPE_COLOR_MAP[type] ||
    INTERNAL_MESSAGE_TYPE_COLOR_MAP.DEFAULT
  );
}

export function internalMessageRecipientStatusLabel(
  value: InternalMessageRecipient_Status,
): string {
  const values = internalMessageRecipientStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const INTERNAL_MESSAGE_RECIPIENT_COLOR_THEME = {
  light: {
    DELETED: '#C9CDD4',
    READ: '#86909C',
    RECEIVED: '#165DFF',
    REVOKED: '#F53F3F',
    SENT: '#4096FF',
    DEFAULT: '#E5E7EB',
  },
  dark: {
    DELETED: '#6E7681', // 深色模式下的浅灰
    READ: '#4E5969', // 深色模式下的中深灰
    RECEIVED: '#2F77FF', // 深色模式下更亮的蓝
    REVOKED: '#F87171', // 深色模式下更柔和的红
    SENT: '#69B1FF', // 深色模式下更柔和的浅蓝
    DEFAULT: '#4B5563', // 深色模式下的中性灰
  },
} as const;

/**
 * 支持主题的内部消息接收状态颜色映射
 * @param status 内部消息接收状态
 * @param theme 主题模式（light/dark），默认浅色
 * @returns 对应主题的十六进制颜色值
 */
export function internalMessageRecipientStatusColor(
  status: InternalMessageRecipient_Status,
  theme: 'dark' | 'light' = 'light',
): string {
  const colorMap = INTERNAL_MESSAGE_RECIPIENT_COLOR_THEME[theme];
  return colorMap[status] || colorMap.DEFAULT;
}
