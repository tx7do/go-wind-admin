import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Popover, Switch, Tag, App } from 'antd';
import { ApiOutlined, DeleteOutlined, EditOutlined, PlusOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { scriptservicev1_Script as Script } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@go-wind/admin-react';
import { fetchListScripts, useDeleteScript, useUpdateScript, useListHookPoints } from '@/api/hooks/script';
import { useProTableScrollY } from '@go-wind/admin-react';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { TABLE } from '@/config/constants.ts';
import { getLanguageOptions, languageColorMap } from './constants';
import ScriptDrawer from './components/ScriptDrawer';
import ScriptLogDrawer from './components/ScriptLogDrawer';
import TestRunModal from './components/TestRunModal';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 脚本管理页面
 */
const ScriptManagement = () => {
  const { t } = useTranslation('script');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selectedScript, setSelectedScript] = useState<Script | undefined>();

  const [testRunOpen, setTestRunOpen] = useState(false);
  const [testRunTarget, setTestRunTarget] = useState<Script | undefined>();
  const [logOpen, setLogOpen] = useState(false);

  // 钩子点概览（工具栏弹出卡片）
  const hookPointsQuery = useListHookPoints({
    enabled: false,
  });

  const languageOptions = getLanguageOptions(t);

  const deleteMutation = useDeleteScript({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listScripts'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  const toggleMutation = useUpdateScript({
    onSuccess: () => {
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['listScripts'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
      actionRef.current?.reload();
    },
  });

  const columns: ProColumns<Script>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, _record, index) => {
        const pagination = actionRef.current?.pageInfo;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || TABLE.DEFAULT_PAGE_SIZE;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('name'),
      dataIndex: 'name',
      width: 200,
      ellipsis: true,
      copyable: true,
    },
    {
      title: t('language'),
      dataIndex: 'language',
      width: 110,
      valueType: 'select',
      fieldProps: { options: languageOptions },
      render: (_, record) => {
        const lang = record.language;
        if (!lang) return '-';
        return <Tag color={languageColorMap[lang]}>{lang === 'LUA' ? t('languageLua') : t('languageJavascript')}</Tag>;
      },
    },
    {
      title: t('hookPoint'),
      dataIndex: 'hookPoint',
      width: 180,
      ellipsis: true,
      render: (_, record) =>
        record.hookPoint ? (
          <Tag icon={<ApiOutlined />} color="geekblue">
            {record.hookPoint}
          </Tag>
        ) : (
          <Tag>{t('unmounted')}</Tag>
        ),
    },
    {
      title: t('priority'),
      dataIndex: 'priority',
      width: 80,
      hideInSearch: true,
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 90,
      hideInSearch: true,
      render: (_, record) => (
        <Switch
          checked={!!record.isEnabled}
          checkedChildren={t('enabledTag')}
          unCheckedChildren={t('disabledTag')}
          loading={toggleMutation.isPending && toggleMutation.variables?.id === record.id}
          onChange={(checked) => {
            if (!record.id) return;
            toggleMutation.mutate({ id: record.id, values: { isEnabled: checked } });
          }}
        />
      ),
    },
    {
      title: t('critical'),
      dataIndex: 'critical',
      width: 90,
      hideInSearch: true,
      render: (_, record) =>
        record.critical ? <Tag color="red">{t('critical')}</Tag> : <span>-</span>,
    },
    {
      title: t('version'),
      dataIndex: 'version',
      width: 80,
      hideInSearch: true,
    },
    {
      title: t('description'),
      dataIndex: 'description',
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('updatedAt'),
      dataIndex: 'updatedAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 250,
      fixed: 'right',
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'testRun',
              label: t('testRun'),
              icon: <PlayCircleOutlined />,
              onClick: () => {
                setTestRunTarget(record);
                setTestRunOpen(true);
              },
            },
            {
              key: 'edit',
              label: t('common:button.edit'),
              icon: <EditOutlined />,
              onClick: () => {
                setDrawerMode('edit');
                setSelectedScript(record);
                setDrawerOpen(true);
              },
            },
            {
              key: 'delete',
              label: t('common:button.delete'),
              icon: <DeleteOutlined />,
              danger: true,
              confirm: {
                title: t('deleteConfirmTitle'),
                description: t('deleteConfirmDesc', { moduleName: t('moduleName') }),
              },
              onClick: () => record.id && deleteMutation.mutate({ ids: [record.id] }),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <>
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ListTable<Script>
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

              const response = await fetchListScripts(query);

              return {
                data: response.items || [],
                total: Number(response.total || 0),
                success: true,
              };
            }}
            rowKey="id"
            search={{ optionRender: searchOptionRender,
              labelWidth: 'auto',
              defaultCollapsed: false,
            }}
            pagination={{
              defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
              showSizeChanger: true,
              showQuickJumper: true,
            }}
            toolBarRender={() => [
              <TableExportButton key="export" fetcher={fetchListScripts} columns={columns} filename="scripts" />,
              <Popover
                key="hookPoints"
                title={t('hookPointsTitle')}
                trigger="click"
                onOpenChange={(visible) => visible && hookPointsQuery.refetch()}
                content={
                  <div style={{ maxWidth: 360 }}>
                    <p style={{ marginBottom: 8 }}>
                      <span style={{ opacity: 0.65 }}>{t('languages')}: </span>
                      {(hookPointsQuery.data?.languages || []).join(', ') || '-'}
                    </p>
                    {(hookPointsQuery.data?.items?.length || 0) === 0 ? (
                      <span style={{ opacity: 0.65 }}>{t('hookPointsEmpty')}</span>
                    ) : (
                      <ul style={{ margin: 0, paddingLeft: 18 }}>
                        {(hookPointsQuery.data?.items || []).map((hp) => (
                          <li key={hp.name} style={{ marginBottom: 4 }}>
                            <Tag icon={<ApiOutlined />} color="geekblue">
                              {hp.name}
                            </Tag>
                            {hp.description && (
                              <span style={{ opacity: 0.65, marginRight: 8 }}>{hp.description}</span>
                            )}
                            <span>{t('hookPointScriptCount', { count: hp.scriptCount ?? 0 })}</span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                }
              >
                <Button>{t('hookPoints')}</Button>
              </Popover>,
              <Button key="logs" onClick={() => setLogOpen(true)}>
                {t('logTitle')}
              </Button>,
              <Button
                key="create"
                color="primary"
                variant="outlined"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelectedScript(undefined);
                  setDrawerOpen(true);
                }}
              >
                {t('common:button.add')}
              </Button>,
            ]}
            options={{
              density: true,
              fullScreen: true,
              setting: { settingIcon: <ColumnSlidersIcon /> },
              reload: true,
            }}
            size="middle"
            bordered
            cardBordered={false}
            scroll={{ y: tableScrollY }}
          />
        </div>
      </ContentContainer>

      {/* 脚本编辑/创建抽屉 */}
      <ScriptDrawer
        open={drawerOpen}
        mode={drawerMode}
        data={selectedScript}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedScript(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />

      {/* 试运行对话框 */}
      <TestRunModal
        open={testRunOpen}
        script={testRunTarget}
        onClose={() => {
          setTestRunOpen(false);
          setTestRunTarget(undefined);
        }}
        onSuccess={() => actionRef.current?.reload()}
      />

      {/* 执行日志抽屉 */}
      <ScriptLogDrawer open={logOpen} onClose={() => setLogOpen(false)} />
    </>
  );
};

export default ScriptManagement;
