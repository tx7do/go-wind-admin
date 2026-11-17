import { useRef } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ListTable } from '@go-wind/admin-react';
import { Button, Drawer, Popconfirm, Tag, Tooltip, App } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { scriptservicev1_ScriptLog as ScriptLog } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@go-wind/admin-react';
import { fetchListScriptLogs, usePurgeScriptLogs } from '@/api/hooks/script-log';
import { useProTableScrollY } from '@go-wind/admin-react';
import { TABLE } from '@go-wind/admin-react';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

interface ScriptLogDrawerProps {
  open: boolean;
  onClose: () => void;
}

const triggerColorMap: Record<string, string> = {
  hook: 'geekblue',
  task: 'cyan',
  test_run: 'orange',
};

/**
 * 脚本执行日志抽屉（挂在脚本管理页工具栏）。
 */
const ScriptLogDrawer: React.FC<ScriptLogDrawerProps> = ({ open, onClose }) => {
  const { t } = useTranslation('script');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const actionRef = useRef<ActionType>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const purgeMutation = usePurgeScriptLogs({
    onSuccess: (resp) => {
      message.success(t('logPurgeSuccess', { count: Number(resp.deleted ?? 0) }));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listScriptLogs'] });
    },
    onError: (error: Error) => message.error(error.message || t('fetchFailed')),
  });

  const columns: ProColumns<ScriptLog>[] = [
    {
      title: t('logScriptName'),
      dataIndex: 'scriptName',
      width: 160,
      ellipsis: true,
    },
    {
      title: t('language'),
      dataIndex: 'language',
      width: 80,
      hideInSearch: true,
      render: (_, record) => (record.language ? <Tag>{record.language}</Tag> : '-'),
    },
    {
      title: t('logTrigger'),
      dataIndex: 'triggerType',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: [
          { label: 'hook', value: 'hook' },
          { label: 'task', value: 'task' },
          { label: 'test_run', value: 'test_run' },
        ],
      },
      render: (_, record) => {
        const trigger = record.triggerType || '';
        return <Tag color={triggerColorMap[trigger] || 'default'}>{trigger}</Tag>;
      },
    },
    {
      title: t('hookPoint'),
      dataIndex: 'hookPoint',
      width: 140,
      ellipsis: true,
      hideInSearch: true,
      render: (_, record) => record.hookPoint || '-',
    },
    {
      title: t('logSuccess'),
      dataIndex: 'success',
      width: 90,
      valueType: 'select',
      fieldProps: {
        options: [
          { label: t('testRunSuccess'), value: 'true' },
          { label: t('testRunFailed'), value: 'false' },
        ],
      },
      render: (_, record) =>
        record.success ? (
          <Tag color="success">{t('testRunSuccess')}</Tag>
        ) : (
          <Tag color="error">{t('testRunFailed')}</Tag>
        ),
    },
    {
      title: t('testRunDuration'),
      dataIndex: 'durationMs',
      width: 90,
      hideInSearch: true,
      render: (_, record) => `${record.durationMs ?? 0}ms`,
    },
    {
      title: t('logError'),
      dataIndex: 'error',
      hideInSearch: true,
      ellipsis: true,
      render: (_, record) =>
        record.error ? (
          <Tooltip title={<pre style={{ margin: 0, whiteSpace: 'pre-wrap', maxHeight: 300, overflow: 'auto' }}>{record.error}</pre>}>
            <span style={{ color: 'var(--gowind-danger-text)' }}>{record.error}</span>
          </Tooltip>
        ) : (
          '-'
        ),
    },
    {
      title: t('logCreatedAt'),
      dataIndex: 'createdAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
  ];

  return (
    <Drawer
      title={t('viewTitle')}
      open={open}
      onClose={onClose}
      size={960}
      forceRender
    >
      <div ref={containerRef} className="page-container-content h-full">
      <ListTable<ScriptLog>
        actionRef={actionRef}
        columns={columns}
        request={async (params, _sorter, _filter) => {
          const query = new PaginationQuery({
            paging: {
              page: params.current || 1,
              pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
            },
            formValues: Object.fromEntries(
              Object.entries(params).filter(
                ([key]) => !['current', 'pageSize'].includes(key),
              ),
            ),
          });

          const response = await fetchListScriptLogs(query);

          return {
            data: response.items || [],
            total: Number(response.total || 0),
            success: true,
          };
        }}
        rowKey="id"
        search={{ optionRender: searchOptionRender,
          labelWidth: 'auto',
          defaultCollapsed: true,
        }}
        pagination={{
          defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
          showSizeChanger: true,
        }}
        toolBarRender={() => [
          <Popconfirm
            key="purge"
            title={t('logPurgeConfirmTitle')}
            description={t('logPurgeConfirmDesc')}
            onConfirm={() =>
              purgeMutation.mutate({
                before: new Date(Date.now() - 90 * 24 * 3600 * 1000).toISOString() as any,
              })
            }
            okText={t('common:button.ok')}
            cancelText={t('common:button.cancel')}
          >
            <Button danger icon={<DeleteOutlined />}>
              {t('logPurge')}
            </Button>
          </Popconfirm>,
        ]}
        options={{ setting: { settingIcon: <ColumnSlidersIcon /> }, density: true, reload: true }}
        size="middle"
        scroll={{ y: tableScrollY }}
      />
      </div>
    </Drawer>
  );
};

export default ScriptLogDrawer;
