import { useRef, useState } from 'react';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import ListTable from '@/components/common/ListTable';
import {
  ModalForm,
  ProFormDependency,
  ProFormText,
  ProFormTextArea,
  ProFormSelect,
  ProFormDigit,
  ProFormSwitch,
} from '@ant-design/pro-components';
import { App, Button, Popconfirm, Tag, Tooltip } from 'antd';
import {
  CaretRightOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type {
  monitor_alertservicev1_CreateMonitorAlertRuleRequest,
  monitor_alertservicev1_MonitorAlertRule as MonitorAlertRule,
  monitor_alertservicev1_UpdateMonitorAlertRuleRequest,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import {
  fetchListMonitorAlertRules,
  useCreateMonitorAlertRule,
  useDeleteMonitorAlertRule,
  useEvaluateMonitorAlerts,
  useUpdateMonitorAlertRule,
} from '@/api/hooks/monitor-alert';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';

/**
 * 监控告警规则：指标阈值 → 触发通知（联动通知域）。
 *
 * 渠道与目标显式写在规则上（不经路由表）：告警是"点对点"的运营配置。
 * 评估每 5 分钟自动跑一轮（系统级常驻任务），「立即评估」手动触发同一内核；
 * 告警→恢复成对通知，持续越限按冷却间隔重发。
 */

const metricOptions = (t: (key: string) => string) =>
  (
    [
      'GO_GOROUTINES',
      'GO_MEM_ALLOC_MB',
      'DB_OPEN_CONNECTIONS',
      'DB_PING_FAIL',
      'REDIS_DB_SIZE',
    ] as const
  ).map((value) => ({ value, label: t(`metricMap.${value}`) }));

const opOptions = (t: (key: string) => string) =>
  (['GE', 'LE'] as const).map((value) => ({ value, label: t(`opMap.${value}`) }));

const channelOptions = (t: (key: string) => string) =>
  (['EMAIL', 'WEBHOOK'] as const).map((value) => ({ value, label: t(`channelMap.${value}`) }));

const MonitorAlertPage = () => {
  const { t } = useTranslation('monitor-alert');
  const actionRef = useRef<ActionType>(null);
  const { message, modal } = App.useApp();
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const [formOpen, setFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<MonitorAlertRule | undefined>();

  const refresh = () => actionRef.current?.reload();

  const createMutation = useCreateMonitorAlertRule();
  const updateMutation = useUpdateMonitorAlertRule();
  const deleteMutation = useDeleteMonitorAlertRule();
  const evaluateMutation = useEvaluateMonitorAlerts();

  const handleDelete = async (record: MonitorAlertRule) => {
    if (!record.id) return;
    try {
      await deleteMutation.mutateAsync({ id: record.id });
      message.success(t('deleteSuccess'));
      refresh();
    } catch (error: any) {
      console.error('delete monitor alert rule failed', error);
      message.error(error.message || t('deleteFailed'));
    }
  };

  const handleEvaluate = async () => {
    try {
      const resp = await evaluateMutation.mutateAsync({});
      const lines = (resp.outcomes ?? []).map((o) => {
        const value = o.currentValue != null ? o.currentValue.toFixed(2) : '-';
        const state = o.firing ? t('outcomeFiring') : t('outcomeNormal');
        const extra = o.notified
          ? ` · ${t('outcomeNotified')}`
          : o.reason
            ? ` · ${o.reason}`
            : '';
        return `${o.name}: ${state} (${value}${extra})`;
      });
      modal.info({
        title: t('evaluateResultTitle'),
        content: <div style={{ whiteSpace: 'pre-wrap' }}>{lines.join('\n') || t('evaluateEmpty')}</div>,
        width: 560,
      });
    } catch (error: any) {
      console.error('evaluate monitor alerts failed', error);
      message.error(error.message || t('evaluateFailed'));
    }
  };

  const handleSubmit = async (values: Record<string, any>) => {
    const isPingFail = values.metric === 'DB_PING_FAIL';
    const data = {
      name: values.name,
      metric: values.metric,
      // DB_PING_FAIL 是布尔指标，op/threshold 无意义不给
      op: isPingFail ? undefined : values.op,
      threshold: isPingFail ? undefined : values.threshold,
      cooldownMinutes: values.cooldownMinutes ?? 30,
      channel: values.channel,
      target: values.target,
      isEnabled: !!values.isEnabled,
      remark: values.remark,
    };
    try {
      if (formMode === 'create') {
        const req: monitor_alertservicev1_CreateMonitorAlertRuleRequest = { data };
        await createMutation.mutateAsync(req);
        message.success(t('createSuccess'));
      } else if (selected?.id) {
        const req: monitor_alertservicev1_UpdateMonitorAlertRuleRequest = {
          id: selected.id,
          data,
          updateMask: 'name,metric,op,threshold,cooldownMinutes,channel,target,isEnabled,remark',
        };
        await updateMutation.mutateAsync(req);
        message.success(t('updateSuccess'));
      }
      setFormOpen(false);
      refresh();
      return true;
    } catch (error: any) {
      console.error('save monitor alert rule failed', error);
      message.error(error.message || t('saveFailed'));
      return false;
    }
  };

  const columns: ProColumns<MonitorAlertRule>[] = [
    { title: t('name'), dataIndex: 'name', width: 160 },
    {
      title: t('metric'),
      dataIndex: 'metric',
      width: 180,
      valueType: 'select',
      fieldProps: { options: metricOptions(t) },
      render: (_, record) =>
        record.metric ? <Tag color="geekblue">{t(`metricMap.${record.metric}`)}</Tag> : '-',
    },
    {
      title: t('condition'),
      width: 150,
      hideInSearch: true,
      render: (_, record) =>
        record.metric === 'DB_PING_FAIL'
          ? t('pingFailCondition')
          : `${t(`opMap.${record.op}`)} ${record.threshold ?? '-'}`,
    },
    {
      title: t('cooldownMinutes'),
      dataIndex: 'cooldownMinutes',
      width: 110,
      hideInSearch: true,
      render: (_, record) => `${record.cooldownMinutes ?? 30} min`,
    },
    {
      title: t('channel'),
      dataIndex: 'channel',
      width: 100,
      valueType: 'select',
      fieldProps: { options: channelOptions(t) },
      render: (_, record) => (record.channel ? <Tag>{t(`channelMap.${record.channel}`)}</Tag> : '-'),
    },
    {
      title: t('target'),
      dataIndex: 'target',
      width: 220,
      ellipsis: true,
      copyable: true,
    },
    {
      title: t('lastFiring'),
      width: 110,
      hideInSearch: true,
      tooltip: t('lastFiringHint'),
      render: (_, record) =>
        record.lastFiring ? (
          <Tag color="error">{t('firingNow')}</Tag>
        ) : (
          <Tag color="success">{t('firingNo')}</Tag>
        ),
    },
    {
      title: t('lastAlertedAt'),
      dataIndex: 'lastAlertedAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 90,
      render: (_, record) =>
        record.isEnabled ? (
          <Tag color="success">{t('enabledOn')}</Tag>
        ) : (
          <Tag>{t('enabledOff')}</Tag>
        ),
    },
    {
      title: t('actions'),
      valueType: 'option',
      width: 180,
      fixed: 'right',
      render: (_, record) => [
        <Button
          key="edit"
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => {
            setFormMode('edit');
            setSelected(record);
            setFormOpen(true);
          }}
        >
          {t('edit')}
        </Button>,
        <Popconfirm key="delete" title={t('deleteConfirm')} onConfirm={() => handleDelete(record)}>
          <Button danger type="link" size="small" icon={<DeleteOutlined />}>
            {t('delete')}
          </Button>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ListTable<MonitorAlertRule>
          actionRef={actionRef}
          columns={columns}
          request={async (params) => {
            const query = new PaginationQuery({
              paging: { page: params.current || 1, pageSize: params.pageSize || 20 },
              formValues: Object.fromEntries(
                Object.entries(params).filter(([key]) => !['current', 'pageSize'].includes(key)),
              ),
            });
            const response = await fetchListMonitorAlertRules(query);
            return { data: response.items || [], total: response.total || 0, success: true };
          }}
          rowKey="id"
          search={{ labelWidth: 'auto', defaultCollapsed: false }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
          }}
          options={{ density: true, fullScreen: true, setting: true, reload: true }}
          toolBarRender={() => [
            <Tooltip key="evaluate-tip" title={t('evaluateHint')}>
              <Button
                key="evaluate"
                icon={<CaretRightOutlined />}
                loading={evaluateMutation.isPending}
                onClick={handleEvaluate}
              >
                {t('evaluateNow')}
              </Button>
            </Tooltip>,
            <Button
              key="create"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                setFormMode('create');
                setSelected(undefined);
                setFormOpen(true);
              }}
            >
              {t('create')}
            </Button>,
          ]}
          size="middle"
          bordered
          scroll={{ y: tableScrollY, x: 1300 }}
        />
      </div>

      <ModalForm
        title={formMode === 'create' ? t('create') : t('edit')}
        width={560}
        open={formOpen}
        onOpenChange={setFormOpen}
        modalProps={{ destroyOnHidden: true, mask: { closable: false } }}
        submitTimeout={3000}
        onFinish={handleSubmit}
        initialValues={
          formMode === 'create'
            ? { cooldownMinutes: 30, channel: 'EMAIL', isEnabled: true }
            : { ...selected }
        }
      >
        <ProFormText
          name="name"
          label={t('name')}
          rules={[{ required: true, message: t('requiredName') }, { max: 100, message: t('maxChars', { max: 100 }) }]}
        />
        <ProFormSelect
          name="metric"
          label={t('metric')}
          options={metricOptions(t)}
          rules={[{ required: true, message: t('requiredMetric') }]}
        />
        <ProFormDependency name={['metric']}>
          {({ metric }) =>
            metric === 'DB_PING_FAIL' ? null : (
              <>
                <ProFormSelect
                  name="op"
                  label={t('op')}
                  options={opOptions(t)}
                  rules={[{ required: true, message: t('requiredOp') }]}
                />
                <ProFormDigit
                  name="threshold"
                  label={t('threshold')}
                  fieldProps={{ precision: 2, step: 1 }}
                  rules={[{ required: true, message: t('requiredThreshold') }]}
                />
              </>
            )
          }
        </ProFormDependency>
        <ProFormDigit
          name="cooldownMinutes"
          label={t('cooldownMinutes')}
          tooltip={t('cooldownHint')}
          min={1}
          max={1440}
          fieldProps={{ precision: 0 }}
        />
        <ProFormSelect
          name="channel"
          label={t('channel')}
          options={channelOptions(t)}
          rules={[{ required: true, message: t('requiredChannel') }]}
        />
        <ProFormText
          name="target"
          label={t('target')}
          tooltip={t('targetHint')}
          rules={[{ required: true, message: t('requiredTarget') }, { max: 500, message: t('maxChars', { max: 500 }) }]}
        />
        <ProFormSwitch name="isEnabled" label={t('isEnabled')} />
        <ProFormTextArea name="remark" label={t('remark')} fieldProps={{ rows: 2 }} />
      </ModalForm>
    </ContentContainer>
  );
};

export default MonitorAlertPage;
