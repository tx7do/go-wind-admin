interface NotificationItem {
  id: number;
  messageId: number;
  avatar: string;
  date: string;
  isRead?: boolean;
  message: string;
  title: string;
}

export type { NotificationItem };
