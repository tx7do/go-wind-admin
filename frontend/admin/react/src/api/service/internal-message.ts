import {
  createInternalMessageServiceClient,
  createInternalMessageCategoryServiceClient,
  createInternalMessageRecipientServiceClient,
  type internal_messageservicev1_GetInternalMessageRequest,
  type internal_messageservicev1_UpdateInternalMessageRequest,
  type internal_messageservicev1_DeleteInternalMessageRequest,
  type internal_messageservicev1_SendMessageRequest,
  type internal_messageservicev1_RevokeMessageRequest,
  type internal_messageservicev1_GetInternalMessageCategoryRequest,
  type internal_messageservicev1_CreateInternalMessageCategoryRequest,
  type internal_messageservicev1_UpdateInternalMessageCategoryRequest,
  type internal_messageservicev1_DeleteInternalMessageCategoryRequest,
  type internal_messageservicev1_DeleteNotificationFromInboxRequest,
  type internal_messageservicev1_MarkNotificationAsReadRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _messageInstance: ReturnType<typeof createInternalMessageServiceClient> | null = null;
let _categoryInstance: ReturnType<typeof createInternalMessageCategoryServiceClient> | null = null;
let _recipientInstance: ReturnType<typeof createInternalMessageRecipientServiceClient> | null = null;

export function getInternalMessageService() {
  if (!_messageInstance) {
    _messageInstance = createInternalMessageServiceClient(requestApi);
  }
  return _messageInstance;
}

export function getMessageCategoryService() {
  if (!_categoryInstance) {
    _categoryInstance = createInternalMessageCategoryServiceClient(requestApi);
  }
  return _categoryInstance;
}

export function getMessageRecipientService() {
  if (!_recipientInstance) {
    _recipientInstance = createInternalMessageRecipientServiceClient(requestApi);
  }
  return _recipientInstance;
}

// ==============================
// 内部消息管理
// ==============================

export async function listInternalMessages(query: PaginationQuery) {
  const params = query.toRawParams();
  return getInternalMessageService().ListMessage(params);
}

export async function getInternalMessage(request: internal_messageservicev1_GetInternalMessageRequest) {
  return getInternalMessageService().GetMessage(request);
}

export async function updateInternalMessage(request: internal_messageservicev1_UpdateInternalMessageRequest) {
  return getInternalMessageService().UpdateMessage(request);
}

export async function deleteInternalMessage(request: internal_messageservicev1_DeleteInternalMessageRequest) {
  return getInternalMessageService().DeleteMessage(request);
}

export async function sendMessage(request: internal_messageservicev1_SendMessageRequest) {
  return getInternalMessageService().SendMessage(request);
}

export async function revokeMessage(request: internal_messageservicev1_RevokeMessageRequest) {
  return getInternalMessageService().RevokeMessage(request);
}

// ==============================
// 消息分类管理
// ==============================

export async function listMessageCategories(query: PaginationQuery) {
  const params = query.toRawParams();
  return getMessageCategoryService().List(params);
}

export async function getMessageCategory(request: internal_messageservicev1_GetInternalMessageCategoryRequest) {
  return getMessageCategoryService().Get(request);
}

export async function createMessageCategory(request: internal_messageservicev1_CreateInternalMessageCategoryRequest) {
  return getMessageCategoryService().Create(request);
}

export async function updateMessageCategory(request: internal_messageservicev1_UpdateInternalMessageCategoryRequest) {
  return getMessageCategoryService().Update(request);
}

export async function deleteMessageCategory(request: internal_messageservicev1_DeleteInternalMessageCategoryRequest) {
  return getMessageCategoryService().Delete(request);
}

// ==============================
// 消息接收者管理
// ==============================

export async function listUserInbox(query: PaginationQuery) {
  const params = query.toRawParams();
  return getMessageRecipientService().ListUserInbox(params);
}

export async function deleteNotificationFromInbox(request: internal_messageservicev1_DeleteNotificationFromInboxRequest) {
  return getMessageRecipientService().DeleteNotificationFromInbox(request);
}

export async function markNotificationAsRead(request: internal_messageservicev1_MarkNotificationAsReadRequest) {
  return getMessageRecipientService().MarkNotificationAsRead(request);
}
