import { Descriptions, Drawer, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import DOMPurify from 'dompurify';
import { useTranslation } from 'react-i18next';
import type { internal_messageservicev1_InternalMessageRecipient as InboxItem } from '@/api/generated/admin/service/v1';
import { getRecipientStatusMap } from '../constants';

interface InboxDetailDrawerProps {
  open: boolean;
  record: InboxItem | null;
  onClose: () => void;
}

/** ISO 时间串 → 本地可读格式；空值返回占位符 */
const formatTime = (value?: string) =>
  value && dayjs(value).isValid() ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-';

/**
 * 站内信全文阅读抽屉：标题 + 状态/时间元信息 + 正文（保留换行）。
 * 未读消息的"打开即已读"由父组件在打开时触发，这里只负责展示。
 */
export const InboxDetailDrawer = ({ open, record, onClose }: InboxDetailDrawerProps) => {
  const { t } = useTranslation('inbox');
  const statusMap = getRecipientStatusMap(t);

  const status = record?.status as keyof typeof statusMap | undefined;
  const statusConfig = status ? statusMap[status] : undefined;

  return (
    <Drawer title={t('detailTitle')} open={open} onClose={onClose} width={480}>
      {record && (
        <div className="flex flex-col gap-4">
          <Typography.Title level={5} style={{ marginBottom: 0 }}>
            {record.title || '-'}
          </Typography.Title>

          <Descriptions column={1} size="small">
            <Descriptions.Item label={t('status')}>
              {statusConfig ? (
                <Tag color={statusConfig.color}>{statusConfig.text}</Tag>
              ) : (
                record.status || '-'
              )}
            </Descriptions.Item>
            <Descriptions.Item label={t('createdAt')}>
              {formatTime(record.createdAt)}
            </Descriptions.Item>
            <Descriptions.Item label={t('readAt')}>{formatTime(record.readAt)}</Descriptions.Item>
          </Descriptions>

          <div>
            <div className="mb-2 text-sm text-[color:var(--ant-color-text-secondary)]">
              {t('content')}
            </div>
            {/* 消息正文是富文本编辑器产出的 HTML：DOMPurify 净化防存储型 XSS 后渲染，
                样式上约束在内容块内避免外部样式溢出 */}
            <div className="inbox-detail-content rounded-lg border border-white/8 bg-[color:var(--ant-color-bg-container)] p-3 text-sm leading-6 break-words">
              {record.content ? (
                <div
                  dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(String(record.content)) }}
                />
              ) : (
                '-'
              )}
            </div>
          </div>
        </div>
      )}
    </Drawer>
  );
};
