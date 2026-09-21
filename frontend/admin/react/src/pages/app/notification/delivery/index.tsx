import { useRef } from 'react';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { App, Tag, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';
import type {
  notificationservicev1_Channel,
  notificationservicev1_DeliveryStatus,
  notificationservicev1_NotificationDelivery as NotificationDelivery,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListNotificationDeliveries } from '@/api/hooks/notification-delivery';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';

/**
 * 通知投递台账（只读）
 *
 * 一行 = 一次投递事实：写侧在 NotificationService.SendDirect（找回密码验证码、联系人绑定码、
 * 渠道测试邮件），本页只查不改。status 的语义差别值得看清：
 * SKIPPED 是"压根没发出去过"（渠道没配/没启用），FAILED 才是"发了但被退回"。
 *
 * SENDING 有两种读法，靠 attempts 分辨：0 = 已入队还没开拨（异步派发的正常中间态）；
 * >0 = 拨过号但还没定案（还留着 asynq 的重试额度，本次的报错在 last_error；或进程死在中途）。
 */

const STATUS_COLORS: Record<notificationservicev1_DeliveryStatus, string> = {
  SENDING: 'processing',
  SENT: 'success',
  FAILED: 'error',
  SKIPPED: 'warning',
  DELIVERY_STATUS_UNSPECIFIED: 'default',
};

const CHANNEL_COLORS: Record<notificationservicev1_Channel, string> = {
  EMAIL: 'blue',
  SMS: 'green',
  WEBHOOK: 'purple',
  INTERNAL: 'cyan',
  CHANNEL_UNSPECIFIED: 'default',
};

const statusOptions = (t: (key: string) => string) =>
  (['SENDING', 'SENT', 'FAILED', 'SKIPPED'] as const).map((value) => ({
    value,
    label: t(`statusMap.${value}`),
  }));

const eventTypeOptions = (t: (key: string) => string) =>
  (['PASSWORD_RESET_CODE', 'CONTACT_BIND_CODE', 'CHANNEL_TEST_EMAIL', 'INTERNAL_MESSAGE'] as const).map((value) => ({
    value,
    label: t(`eventTypeMap.${value}`),
  }));

// 可筛渠道 = 有 Sender 实现、因此真能写出台账行的渠道。路由现在是数据库表
// （sys_notification_rules，在「通知路由规则」页维护），默认规则没指向 WEBHOOK 也筛得到：
// 管理员建一行 WEBHOOK 规则，下一趟投递就写进台账。只有 SMS 还没有实现，永远筛不出东西。
const channelOptions = (t: (key: string) => string) =>
  (['EMAIL', 'WEBHOOK', 'INTERNAL'] as const).map((value) => ({ value, label: t(`channelMap.${value}`) }));

// ProTable 的 sorter key 来自 dataIndex（camelCase），而 orderBy 要的是数据库列名。
// 直接透传 "createdAt" 在 sqlite 上实测为"静默不参与排序"（go-crud 的列白名单对本仓表
// fail-open，非法列不报错也不生效），换成蛇形才真按投递时间排。
const toSnakeCase = (key: string) => key.replace(/[A-Z]/g, (c) => `_${c.toLowerCase()}`);

const NotificationDeliveryPage = () => {
  const { t } = useTranslation('notification-delivery');
  const actionRef = useRef<ActionType>(null);
  const { message } = App.useApp();
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const columns: ProColumns<NotificationDelivery>[] = [
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
    },
    {
      title: t('eventType'),
      dataIndex: 'eventType',
      width: 130,
      valueType: 'select',
      fieldProps: { options: eventTypeOptions(t) },
      render: (_, record) =>
        record.eventType ? t(`eventTypeMap.${record.eventType}`) : '-',
    },
    {
      title: t('channel'),
      dataIndex: 'channel',
      width: 100,
      valueType: 'select',
      fieldProps: { options: channelOptions(t) },
      render: (_, record) =>
        record.channel ? (
          <Tag color={CHANNEL_COLORS[record.channel]}>{t(`channelMap.${record.channel}`)}</Tag>
        ) : (
          '-'
        ),
    },
    {
      title: t('target'),
      dataIndex: 'target',
      width: 180,
      ellipsis: true,
      // 台账里存的就是脱敏串，搜索框输入完整地址同样只会命中这一形态
      tooltip: t('targetMaskedHint'),
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 110,
      valueType: 'select',
      fieldProps: { options: statusOptions(t) },
      render: (_, record) =>
        record.status ? (
          <Tag color={STATUS_COLORS[record.status]}>{t(`statusMap.${record.status}`)}</Tag>
        ) : (
          '-'
        ),
    },
    {
      title: t('attempts'),
      dataIndex: 'attempts',
      width: 90,
      hideInSearch: true,
      tooltip: t('attemptsHint'),
      render: (_, record) => record.attempts ?? 0,
    },
    {
      title: t('recipientUserId'),
      dataIndex: 'recipientUserId',
      width: 110,
      hideInSearch: true,
      render: (_, record) => record.recipientUserId ?? '-',
    },
    {
      title: t('channelId'),
      dataIndex: 'channelId',
      width: 100,
      hideInSearch: true,
      render: (_, record) => record.channelId ?? '-',
    },
    {
      // 台账不存正文快照，related_id 是"这条投递发的是什么"的唯一回跳入口（站内信 = 消息 ID）。
      title: t('relatedId'),
      dataIndex: 'relatedId',
      width: 100,
      hideInSearch: true,
      tooltip: t('relatedIdHint'),
      render: (_, record) => record.relatedId ?? '-',
    },
    {
      title: t('requestId'),
      dataIndex: 'requestId',
      width: 150,
      hideInSearch: true,
      ellipsis: { showTitle: false },
      tooltip: t('requestIdHint'),
      render: (_, record) =>
        record.requestId ? (
          <Tooltip title={record.requestId} placement="topLeft">
            <span>{record.requestId}</span>
          </Tooltip>
        ) : (
          '-'
        ),
    },
    {
      title: t('sentAt'),
      dataIndex: 'sentAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('lastError'),
      dataIndex: 'lastError',
      width: 260,
      hideInSearch: true,
      ellipsis: { showTitle: false },
      render: (_, record) =>
        record.lastError ? (
          <Tooltip title={record.lastError} placement="topLeft">
            <span style={{ color: 'inherit' }}>{record.lastError}</span>
          </Tooltip>
        ) : (
          '-'
        ),
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ProTable<NotificationDelivery>
          actionRef={actionRef}
          columns={columns}
          request={async (params, sorter) => {
            try {
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || 20,
                },
                formValues: Object.fromEntries(
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize'].includes(key),
                  ),
                ),
                orderBy:
                  sorter && Object.keys(sorter).length > 0
                    ? Object.entries(sorter).map(([key, value]) =>
                        value === 'ascend' ? toSnakeCase(key) : `-${toSnakeCase(key)}`,
                      )
                    : ['-created_at'],
              });

              const response = await fetchListNotificationDeliveries(query);

              return {
                data: response.items || [],
                total: response.total || 0,
                success: true,
              };
            } catch (error: any) {
              message.error(error.message || t('fetchFailed'));
              return {
                data: [],
                total: 0,
                success: false,
              };
            }
          }}
          rowKey="id"
          search={{
            labelWidth: 'auto',
            defaultCollapsed: false,
          }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
          }}
          options={{
            density: true,
            fullScreen: true,
            setting: true,
            reload: true,
          }}
          size="middle"
          bordered
          cardBordered={false}
          scroll={{
            y: tableScrollY,
            x: 1670,
          }}
        />
      </div>
    </ContentContainer>
  );
};

export default NotificationDeliveryPage;
