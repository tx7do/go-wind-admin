interface NotificationItem {
  /** 后端 uint64 → proto3 JSON 为字符串 */
  id: string;
  /** 后端 uint64 → proto3 JSON 为字符串 */
  messageId: string;
  date: string;
  isRead?: boolean;
  message: string;
  title: string;
}

export type { NotificationItem };
