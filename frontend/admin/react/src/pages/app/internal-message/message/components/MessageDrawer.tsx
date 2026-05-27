import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormSelect,
} from '@ant-design/pro-components';
import { App, TreeSelect, Form } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { internal_messageservicev1_InternalMessage as InternalMessage } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { useUpdateInternalMessage, useSendMessage, fetchListMessageCategories } from '@/api/hooks/internal-message';
import { Editor, EditorType } from '@/components/common/Editor';
import { getStatusOptions, getTypeOptions } from '../constants';

interface MessageDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: InternalMessage;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 内部消息编辑/创建抽屉组件
 */
const MessageDrawer: React.FC<MessageDrawerProps> = ({ open, mode, data, onClose, onSuccess }) => {
  const { t } = useTranslation('internal-message');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);
  const [categoryTreeData, setCategoryTreeData] = useState<any[]>([]);

  // 加载分类列表
  useEffect(() => {
    if (open) {
      fetchListMessageCategories(new PaginationQuery({ formValues: { isEnabled: 'true' } }))
        .then((res) => {
          const items = res?.items || [];
          // 分类是平级列表，转为 select 选项
          setCategoryTreeData(
            items.map((c: any) => ({
              id: c.id,
              key: c.id,
              value: c.id,
              title: c.name || String(c.id),
              label: c.name || String(c.id),
            })),
          );
        })
        .catch(() => setCategoryTreeData([]));
    }
  }, [open]);

  // 编辑模式填充表单
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          status: data.status || 'DRAFT',
          type: (data as any).type || 'NOTIFICATION',
          categoryId: (data as any).categoryId || undefined,
          title: data.title || '',
          content: (data as any).content || '',
        });
      }, 0);
    }
  }, [open, mode, data]);

  const sendMessageMutation = useSendMessage({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listInternalMessages'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  const updateMutation = useUpdateInternalMessage({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listInternalMessages'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {
      if (mode === 'create') {
        await sendMessageMutation.mutateAsync({
          ...values,
          targetAll: true,
        });
      } else if (data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values });
      }
    } finally {
      setConfirmLoading(false);
    }
  };

  return (
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? t('drawerCreate') : t('drawerUpdate')}
      open={open}
      onOpenChange={(visible) => {
        if (!visible) {
          formRef.current?.resetFields();
          onClose();
        }
      }}
      initialValues={{
        status: 'DRAFT',
        type: 'NOTIFICATION',
      }}
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading || sendMessageMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: { onClick: onClose },
      }}
      drawerProps={{ destroyOnClose: true, onClose, size: 800 }}
    >
      <ProFormSelect
        name="status"
        label={t('status')}
        placeholder={t('statusPlaceholder')}
        options={getStatusOptions(t)}
        rules={[{ required: true, message: t('requiredStatus') }]}
        fieldProps={{
          showSearch: true,
          filterOption: (input: string, option: any) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormSelect
        name="type"
        label={t('type')}
        placeholder={t('typePlaceholder')}
        options={getTypeOptions(t)}
        rules={[{ required: true, message: t('requiredType') }]}
        fieldProps={{
          showSearch: true,
          filterOption: (input: string, option: any) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      {/* 消息分类 - TreeSelect */}
      <Form.Item
        name="categoryId"
        label={t('categoryId')}
        rules={[{ required: true, message: t('requiredCategoryId') }]}
      >
        <TreeSelect
          showSearch
          allowClear
          treeDefaultExpandAll
          placeholder={t('categoryIdPlaceholder')}
          treeData={categoryTreeData}
        />
      </Form.Item>

      <ProFormText
        name="title"
        label={t('title')}
        placeholder={t('titlePlaceholder')}
        rules={[{ required: true, message: t('requiredTitle') }]}
        fieldProps={{ allowClear: true }}
      />

      {/* 消息内容 - Tiptap 富文本编辑器 */}
      <Form.Item
        name="content"
        label={t('content')}
        rules={[{ required: true, message: t('requiredContent', '请输入消息内容') }]}
      >
        <Editor
          editorType={EditorType.RICH_TEXT}
          height={400}
          placeholder={t('contentPlaceholder')}
        />
      </Form.Item>
    </DrawerForm>
  );
};

export default MessageDrawer;
