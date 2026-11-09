import { apiClient } from "@/api/client";
import { PaginationQuery } from "@/core/transport/rest";

// ==============================
// 通知渠道（平台级配置）
// ==============================

/** 分页查询通知渠道 */
export async function fetchListNotificationChannels(query: PaginationQuery) {
  return apiClient.notificationChannelService.ListNotificationChannel(query.toRawParams());
}

/**
 * 创建通知渠道。
 *
 * 三处密钥都是**请求级**字段、不进 data：读视图只有 hasPassword / hasWebhookSecret /
 * hasSmsAccessSecret 三个布尔，所以它们也绝不出现在 updateMask 里。服务端加密存储、不落日志。
 */
export async function createNotificationChannel(
  data: Record<string, any>,
  password?: string,
  webhookSecret?: string,
  smsAccessSecret?: string,
) {
  return apiClient.notificationChannelService.CreateNotificationChannel({
    data: data as any,
    password: password || undefined,
    webhookSecret: webhookSecret || undefined,
    smsAccessSecret: smsAccessSecret || undefined,
  });
}

/**
 * 更新通知渠道：password / webhookSecret / smsAccessSecret 留空表示不修改已存值；
 * updateMask 显式列举业务字段（与 react 基准一致），避免密码等敏感字段混入。
 */
export async function updateNotificationChannel(
  id: string,
  values: Record<string, any>,
  password?: string,
  webhookSecret?: string,
  smsAccessSecret?: string,
) {
  return apiClient.notificationChannelService.UpdateNotificationChannel({
    id,
    data: values as any,
    password: password || undefined,
    webhookSecret: webhookSecret || undefined,
    smsAccessSecret: smsAccessSecret || undefined,
    updateMask:
      "name,type,smtpHost,smtpPort,smtpUsername,smtpFrom,smtpFromName,smtpTls,webhookUrl,webhookSignStyle,webhookPayloadTemplate,smsProvider,smsAccessKeyId,smsSignName,smsTemplateCode,smsAppId,smsRegion,enabled,remark",
  });
}

/** 删除通知渠道 */
export async function deleteNotificationChannel(id: string) {
  return apiClient.notificationChannelService.DeleteNotificationChannel({ id });
}

/** 向指定渠道发送测试邮件 */
export async function sendTestEmail(id: string, recipient: string) {
  return apiClient.notificationChannelService.SendTestEmail({ id, recipient });
}

/** 按渠道类型发送测试消息（EMAIL 走测试邮件，SMS 发测试验证码） */
export async function sendTest(id: string, recipient: string) {
  return apiClient.notificationChannelService.SendTest({ id, recipient });
}
