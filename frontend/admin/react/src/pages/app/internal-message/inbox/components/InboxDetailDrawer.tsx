import { Avatar, Divider, Drawer, Tag } from 'antd';
import { UserOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import DOMPurify from 'dompurify';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useTranslation } from 'react-i18next';
import type { internal_messageservicev1_InternalMessageRecipient as InboxItem } from '@/api/generated/admin/service/v1';
import { useGetInternalMessage } from '@/api/hooks/internal-message';
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
 * 站内信全文阅读抽屉：消息标题入头部 + 发送人元信息头 + 正文（DOMPurify 净化）。
 * 发送人名称不在列表行里，打开时按 messageId 补拉一次 GetMessage（ele 同款路径）；
 * 未读消息的"打开即已读"由父组件在打开时触发，这里只负责展示。
 */
export const InboxDetailDrawer = ({ open, record, onClose }: InboxDetailDrawerProps) => {
  const { t } = useTranslation('inbox');
  const statusMap = getRecipientStatusMap(t);

  const status = record?.status as keyof typeof statusMap | undefined;
  const statusConfig = status ? statusMap[status] : undefined;

  // 列表行只有标题/正文/状态/时间，发送人要按 messageId 补拉；失败不影响抽屉其余部分。
  // sender_id=0 的系统消息（定时任务产出）senderName 为空，显示"系统"而非占位符
  const { data: messageDetail, isPending: detailPending } = useGetInternalMessage(
    { id: record?.messageId },
    { enabled: open && !!record?.messageId }, // messageId 为 uint64（字符串），enabled 已兜住空值
  );
  const senderName = messageDetail?.senderName || (detailPending ? '-' : t('senderSystem'));

  // 正文格式自适应（仿 cms ContentViewer）：
  // - 富文本编辑器产出 → HTML 标签特征 → DOMPurify 净化渲染
  // - 纯文本 / Markdown（AI 日报等）→ react-markdown（gfm 表格/列表，
  //   v10 默认不渲染裸 HTML，路径本身安全；换行/段落由 markdown 语义保留）
  const content = record?.content ? String(record.content) : '';
  const isHtml = /<\/?[a-z][\s\S]*>/i.test(content);

  return (
    <Drawer title={record?.title || t('detailTitle')} open={open} onClose={onClose} size={640}>
      {record && (
        <div className="flex flex-col gap-4">
          {/* 发送人元信息头：头像 + 发送人/接收时间，状态 Tag 右置 */}
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <Avatar size={40} icon={<UserOutlined />} />
              <div className="flex min-w-0 flex-col gap-1">
                <div className="truncate text-sm font-semibold text-[color:var(--ant-color-text)]">
                  {senderName}
                </div>
                <div className="text-xs text-[color:var(--ant-color-text-secondary)]">
                  {t('createdAt')} {formatTime(record.createdAt)}
                </div>
                {status === 'READ' && (
                  <div className="text-xs text-[color:var(--ant-color-text-secondary)]">
                    {t('readAt')} {formatTime(record.readAt)}
                  </div>
                )}
              </div>
            </div>
            {statusConfig ? (
              <Tag color={statusConfig.color}>{statusConfig.text}</Tag>
            ) : (
              record.status || '-'
            )}
          </div>

          <Divider style={{ margin: '4px 0' }} />

          {/* 消息正文：富文本内容 DOMPurify 净化渲染；纯文本/Markdown 走 react-markdown
              （gfm 表格/列表；v10 默认忽略裸 HTML，路径安全）。样式约束在内容块内
              避免外部样式溢出 */}
          <div className="inbox-detail-content rounded-lg border border-[color:var(--ant-color-border-secondary)] bg-[color:var(--ant-color-bg-container)] p-4 text-sm leading-6 break-words">
            {content ? (
              isHtml ? (
                <div dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(content) }} />
              ) : (
                // 纯文本的孤立换行在 markdown 语义里是软换行（会折叠）——
                // 升格为段落换行，AI 多段报告不再挤成一坨
                <ReactMarkdown remarkPlugins={[remarkGfm]}>
                  {content.replace(/(?<!\n)\n(?!\n)/g, '\n\n')}
                </ReactMarkdown>
              )
            ) : (
              '-'
            )}
          </div>
        </div>
      )}
    </Drawer>
  );
};
