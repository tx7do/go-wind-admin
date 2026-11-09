import { useRef, useState } from 'react';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import ListTable from '@/components/common/ListTable';
import TableActions from '@/components/common/TableActions';
import {
  ModalForm,
  DrawerForm,
  ProFormText,
  ProFormTextArea,
  ProFormSwitch,
} from '@ant-design/pro-components';
import { App, Button, Descriptions, Tag, Alert } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type {
  notificationservicev1_CreateNotificationTemplateRequest,
  notificationservicev1_NotificationTemplate as NotificationTemplate,
  notificationservicev1_RenderNotificationTemplateRequest,
  notificationservicev1_UpdateNotificationTemplateRequest,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import {
  fetchListNotificationTemplates,
  useCreateNotificationTemplate,
  useDeleteNotificationTemplate,
  useRenderNotificationTemplate,
  useUpdateNotificationTemplate,
} from '@/api/hooks/notification-template';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 通知模板管理（可复用的标题/正文占位模板，{{var}} 占位符）。
 *
 * 发送方以 template_code 引用（SendDirectNotificationRequest.templateCode），
 * 渲染发生在落台账之前——异步载荷装的是渲染结果。建/改模板时服务端会用空变量集
 * 试渲染：模板里只要有占位符、没给值就立刻被拒，坏模板进不了库。
 * 「预览」按钮用给的变量集走服务端 Render，看到的字节即发送字节。
 */

type RenderResult = { title: string; content: string };

const NotificationTemplatePage = () => {
  const { t } = useTranslation('notification-template');
  const actionRef = useRef<ActionType>(null);
  const { message } = App.useApp();
  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const [formOpen, setFormOpen] = useState(false);
  // 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
  const { guardClose, handleValuesChange } = useDrawerCloseGuard(formOpen, () => setFormOpen(false));
  const [formMode, setFormMode] = useState<'create' | 'edit'>('create');
  const [selected, setSelected] = useState<NotificationTemplate | undefined>();
  const [renderTarget, setRenderTarget] = useState<NotificationTemplate | undefined>();
  const [renderResult, setRenderResult] = useState<RenderResult | undefined>();

  const refresh = () => actionRef.current?.reload();

  const createMutation = useCreateNotificationTemplate();
  const updateMutation = useUpdateNotificationTemplate();
  const deleteMutation = useDeleteNotificationTemplate();
  const renderMutation = useRenderNotificationTemplate();

  const handleDelete = async (record: NotificationTemplate) => {
    if (!record.id) return;
    try {
      await deleteMutation.mutateAsync({ id: record.id });
      message.success(t('deleteSuccess'));
      refresh();
    } catch (error: any) {
      // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
      console.error('delete notification template failed', error);
      message.error(error.message || t('deleteFailed'));
    }
  };

  const handleSubmit = async (values: Record<string, any>) => {
    const data = {
      name: values.name,
      code: values.code,
      titleTemplate: values.titleTemplate,
      contentTemplate: values.contentTemplate,
      isEnabled: !!values.isEnabled,
      remark: values.remark,
    };
    try {
      if (formMode === 'create') {
        const req: notificationservicev1_CreateNotificationTemplateRequest = { data };
        await createMutation.mutateAsync(req);
        message.success(t('createSuccess'));
      } else if (selected?.id) {
        const req: notificationservicev1_UpdateNotificationTemplateRequest = {
          id: selected.id,
          data,
          // code 刻意不进掩码：模板编码是发送方的引用锚，改码等于让所有引用悬空，删旧建新才说得清
          updateMask: 'name,titleTemplate,contentTemplate,isEnabled,remark',
        };
        await updateMutation.mutateAsync(req);
        message.success(t('updateSuccess'));
      }
      setFormOpen(false);
      refresh();
      return true;
    } catch (error: any) {
      console.error('save notification template failed', error);
      message.error(error.message || t('saveFailed'));
      return false;
    }
  };

  const handleRender = async (values: { varsJson?: string }) => {
    if (!renderTarget?.id) return false;
    let variables: Record<string, string> = {};
    const raw = (values.varsJson || '').trim();
    if (raw) {
      try {
        variables = JSON.parse(raw);
      } catch (error: any) {
        console.error('parse template vars json failed', error);
        message.error(t('varsJsonInvalid'));
        return false;
      }
    }
    try {
      const req: notificationservicev1_RenderNotificationTemplateRequest = {
        id: renderTarget.id,
        variables,
      };
      const resp = await renderMutation.mutateAsync(req);
      setRenderResult({ title: resp.title ?? '', content: resp.content ?? '' });
      // 返回 false 让弹窗保持打开：结果就地展示，用户看完手动关闭
      return false;
    } catch (error: any) {
      console.error('render notification template failed', error);
      message.error(error.message || t('renderFailed'));
      return false;
    }
  };

  const columns: ProColumns<NotificationTemplate>[] = [
    {
      title: t('name'),
      dataIndex: 'name',
      width: 160,
    },
    {
      title: t('code'),
      dataIndex: 'code',
      width: 180,
      copyable: true,
      render: (_, record) => <Tag color="geekblue">{record.code || '-'}</Tag>,
    },
    {
      title: t('titleTemplate'),
      dataIndex: 'titleTemplate',
      width: 240,
      ellipsis: true,
    },
    {
      title: t('contentTemplate'),
      dataIndex: 'contentTemplate',
      ellipsis: true,
    },
    {
      title: t('isEnabled'),
      dataIndex: 'isEnabled',
      width: 90,
      render: (_, record) =>
        record.isEnabled ? <Tag color="success">{t('enabledOn')}</Tag> : <Tag>{t('enabledOff')}</Tag>,
    },
    {
      title: t('updatedAt'),
      dataIndex: 'updatedAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: t('actions'),
      valueType: 'option',
      // 编辑模板+预览+删除 三个图标+文字按钮实测需 256px，240 会被截断
      width: 260,
      fixed: 'right',
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'edit',
              label: t('edit'),
              icon: <EditOutlined />,
              onClick: () => {
                setFormMode('edit');
                setSelected(record);
                setFormOpen(true);
              },
            },
            {
              key: 'render',
              label: t('render'),
              icon: <EyeOutlined />,
              onClick: () => {
                setRenderTarget(record);
                setRenderResult(undefined);
              },
            },
            {
              key: 'delete',
              label: t('delete'),
              icon: <DeleteOutlined />,
              danger: true,
              confirm: { title: t('deleteConfirm') },
              onClick: () => handleDelete(record),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ListTable<NotificationTemplate>
          actionRef={actionRef}
          columns={columns}
          request={async (params) => {
            const query = new PaginationQuery({
              paging: { page: params.current || 1, pageSize: params.pageSize || 20 },
              formValues: Object.fromEntries(
                Object.entries(params).filter(([key]) => !['current', 'pageSize'].includes(key)),
              ),
            });
            const response = await fetchListNotificationTemplates(query);
            return { data: response.items || [], total: Number(response.total ?? 0), success: true };
          }}
          rowKey="id"
          search={{ optionRender: searchOptionRender, labelWidth: 'auto', defaultCollapsed: false }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
          }}
          options={{ density: true, fullScreen: true, setting: { settingIcon: <ColumnSlidersIcon /> }, reload: true }}
          toolBarRender={() => [
            <Button
              key="create"
              color="primary"
              variant="outlined"
              icon={<PlusOutlined />}
              onClick={() => {
                setFormMode('create');
                setSelected(undefined);
                setFormOpen(true);
              }}
            >
              {t('common:button.add')}
            </Button>,
          ]}
          size="middle"
          bordered
          scroll={{ y: tableScrollY, x: 1200 }}
        />
      </div>

      <DrawerForm
        title={formMode === 'create' ? t('common:modal.create', { moduleName: t('moduleName') }) : t('common:modal.update', { moduleName: t('moduleName') })}
        open={formOpen}
        onOpenChange={(v) => {
          // 关闭一律走 drawerProps.onClose 守卫；这里只放行打开
          if (v) setFormOpen(true);
        }}
        onValuesChange={handleValuesChange}
        drawerProps={{ destroyOnHidden: true, size: 600, onClose: guardClose }}
        submitTimeout={3000}
        onFinish={handleSubmit}
        initialValues={
          formMode === 'create'
            ? { isEnabled: true }
            : { ...selected }
        }
        layout="horizontal"
        labelCol={{ flex: '0 0 120px' }}
        wrapperCol={{ flex: 'auto' }}
      >
        <Alert
          type="info"
          showIcon
          message={t('txCodeHintTitle')}
          description={t('txCodeHint')}
          style={{ marginBottom: 16 }}
        />
        <ProFormText
          name="name"
          label={t('name')}
          rules={[{ required: true, message: t('requiredName') }, { max: 100, message: t('maxChars', { max: 100 }) }]}
        />
        <ProFormText
          name="code"
          label={t('code')}
          disabled={formMode === 'edit'}
          rules={[
            { required: true, message: t('requiredCode') },
            { pattern: /^[A-Za-z0-9_-]{1,64}$/, message: t('codePattern') },
          ]}
          extra={t('codeHint')}
        />
        <ProFormText
          name="titleTemplate"
          label={t('titleTemplate')}
          placeholder={t('titleTemplatePlaceholder')}
          rules={[{ required: true, message: t('requiredTitleTemplate') }, { max: 500, message: t('maxChars', { max: 500 }) }]}
        />
        <ProFormTextArea
          name="contentTemplate"
          label={t('contentTemplate')}
          placeholder={t('contentTemplatePlaceholder')}
          rules={[{ required: true, message: t('requiredContentTemplate') }, { max: 20000, message: t('maxChars', { max: 20000 }) }]}
          fieldProps={{ rows: 6, showCount: true }}
        />
        <ProFormSwitch name="isEnabled" label={t('isEnabled')} />
        <ProFormTextArea
          name="remark"
          label={t('remark')}
          fieldProps={{ rows: 2 }}
        />
      </DrawerForm>

      <ModalForm<{ varsJson?: string }>
        title={t('renderTitle', { name: renderTarget?.name || '' })}
        width={560}
        open={!!renderTarget}
        onOpenChange={(open) => {
          if (!open) setRenderTarget(undefined);
        }}
        modalProps={{ destroyOnHidden: true }}
        submitTimeout={10000}
        onFinish={handleRender}
      >
        <ProFormTextArea
          name="varsJson"
          label={t('varsJson')}
          placeholder={t('varsJsonPlaceholder')}
          fieldProps={{ rows: 4 }}
          extra={t('varsJsonHint')}
        />
        {renderResult && (
          <Descriptions column={1} size="small" bordered style={{ marginTop: 8 }}>
            <Descriptions.Item label={t('renderedTitle')}>{renderResult.title}</Descriptions.Item>
            <Descriptions.Item label={t('renderedContent')}>
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', maxHeight: 240, overflow: 'auto' }}>
                {renderResult.content ?? ''}
              </pre>
            </Descriptions.Item>
          </Descriptions>
        )}
      </ModalForm>
    </ContentContainer>
  );
};

export default NotificationTemplatePage;
