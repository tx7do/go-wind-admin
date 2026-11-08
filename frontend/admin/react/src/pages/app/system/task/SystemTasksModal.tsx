import { Modal, Tag, Typography } from 'antd';
import type { ProColumns } from '@ant-design/pro-components';
import { useTranslation } from 'react-i18next';
import ListTable from '@/components/common/ListTable';
import { useInspectSystemTasks } from '@/api/hooks/task-monitor';
import { formatDateTime } from '@/utils/date';

const { Text } = Typography;

/**
 * 系统级常驻任务弹窗（方案 C'：只读 asynq Inspector，零新表）。
 *
 * 数据来自 asynq 本身：
 *  - 调度计划表：cron + 下次/上次入队时间（上次为空 = 注册后从未运行）；
 *  - 运行状态表：active/pending/retry/archived 计数 + 最近失败明细
 *    （错误原文/失败时间/已重试次数，失败时间倒序、每类型最多 10 条）。
 * asynq 队列无命名空间：后端只汇总调度条目覆盖到的任务类型，
 * 共库错开 URI 后看到的即本实例自己的任务。
 */
const SystemTasksModal = ({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) => {
  const { t } = useTranslation('task');
  const inspectQuery = useInspectSystemTasks({ enabled: open });

  const scheduleColumns: ProColumns<Record<string, any>>[] = [
    {
      title: t('sysTaskType'),
      dataIndex: 'taskType',
      render: (_, record) => <Tag color="geekblue">{record.taskType}</Tag>,
    },
    { title: t('sysTaskCron'), dataIndex: 'cronSpec', width: 130 },
    {
      title: t('sysTaskNext'),
      dataIndex: 'nextEnqueueAt',
      width: 170,
      render: (_, record) =>
        record.nextEnqueueAt ? formatDateTime(record.nextEnqueueAt) : '-',
    },
    {
      title: t('sysTaskPrev'),
      dataIndex: 'prevEnqueueAt',
      width: 170,
      render: (_, record) =>
        record.prevEnqueueAt ? (
          formatDateTime(record.prevEnqueueAt)
        ) : (
          <Text type="secondary">{t('sysTaskNeverRun')}</Text>
        ),
    },
  ];

  const summaryColumns: ProColumns<Record<string, any>>[] = [
    {
      title: t('sysTaskType'),
      dataIndex: 'taskType',
      render: (_, record) => <Tag color="geekblue">{record.taskType}</Tag>,
    },
    { title: t('sysTaskActive'), dataIndex: 'active', width: 70, align: 'center' },
    { title: t('sysTaskPending'), dataIndex: 'pending', width: 80, align: 'center' },
    {
      title: t('sysTaskRetry'),
      dataIndex: 'retry',
      width: 70,
      align: 'center',
      render: (_, record) =>
        record.retry > 0 ? <Tag color="warning">{record.retry}</Tag> : record.retry,
    },
    {
      title: t('sysTaskArchived'),
      dataIndex: 'archived',
      width: 80,
      align: 'center',
      render: (_, record) =>
        record.archived > 0 ? <Tag color="error">{record.archived}</Tag> : record.archived,
    },
  ];

  const failureColumns: ProColumns<Record<string, any>>[] = [
    { title: t('sysTaskType'), dataIndex: 'taskType', width: 160 },
    {
      title: t('sysTaskState'),
      dataIndex: 'state',
      width: 90,
      render: (_, record) => (
        <Tag color={record.state === 'archived' ? 'error' : 'warning'}>{record.state}</Tag>
      ),
    },
    { title: t('sysTaskLastError'), dataIndex: 'lastError', ellipsis: true },
    {
      title: t('sysTaskFailedAt'),
      dataIndex: 'lastFailedAt',
      width: 170,
      render: (_, record) =>
        record.lastFailedAt ? formatDateTime(record.lastFailedAt) : '-',
    },
    {
      title: t('sysTaskRetried'),
      key: 'retried',
      width: 90,
      align: 'center',
      render: (_, record) => `${record.retried ?? 0}/${record.maxRetry ?? 0}`,
    },
  ];

  const summaries = inspectQuery.data?.summaries ?? [];
  const failures = summaries.flatMap((s: any) => s.recentFailures ?? []);

  return (
    <Modal
      title={t('sysTasksTitle')}
      open={open}
      onCancel={onClose}
      footer={null}
      width={860}
      destroyOnHidden
    >
      {inspectQuery.isFetching && (
        <div style={{ padding: 48, textAlign: 'center' }}>{t('sysTaskLoading')}</div>
      )}
      {inspectQuery.isError && (
        <Text type="danger">
          {inspectQuery.error?.message || t('sysTaskLoadFailed')}
        </Text>
      )}
      {inspectQuery.data && (
        <>
          <Typography.Title level={5} style={{ marginBottom: 8 }}>
            {t('sysTaskSchedules')}
            <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
              {t('sysTaskQueueHint', { queue: inspectQuery.data.queue ?? '' })}
            </Text>
          </Typography.Title>
          <ListTable
            columns={scheduleColumns}
            dataSource={(inspectQuery.data.schedules ?? []) as Record<string, any>[]}
            search={false}
            options={false}
            toolBarRender={false}
            pagination={false}
            rowKey="taskType"
            size="small"
            bordered
          />

          <Typography.Title level={5} style={{ margin: '20px 0 8px' }}>
            {t('sysTaskStates')}
          </Typography.Title>
          <ListTable
            columns={summaryColumns}
            dataSource={summaries as Record<string, any>[]}
            search={false}
            options={false}
            toolBarRender={false}
            pagination={false}
            rowKey="taskType"
            size="small"
            bordered
          />

          <Typography.Title level={5} style={{ margin: '20px 0 8px' }}>
            {t('sysTaskFailures')}
          </Typography.Title>
          {failures.length === 0 ? (
            <Text type="secondary">{t('sysTaskNoFailures')}</Text>
          ) : (
            <ListTable
              columns={failureColumns}
              dataSource={failures as Record<string, any>[]}
              search={false}
              options={false}
              toolBarRender={false}
              pagination={false}
              rowKey={(r: any) =>
                `${r.taskType}-${r.state}-${r.lastFailedAt ?? ''}-${r.retried ?? 0}`
              }
              size="small"
              bordered
            />
          )}
        </>
      )}
    </Modal>
  );
};

export default SystemTasksModal;
