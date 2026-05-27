import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormDigit,
  ProFormRadio,
} from '@ant-design/pro-components';
import { App, TreeSelect, Form } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import {
  useCreatePermissionGroup,
  useUpdatePermissionGroup,
  fetchListPermissionGroups,
} from '@/api/hooks/permission-group';
import { getStatusOptions, buildTreeSelectData } from './constants';

interface PermissionGroupDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: any;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 权限分组编辑/创建抽屉组件
 */
const PermissionGroupDrawer: React.FC<PermissionGroupDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('permission-group');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);
  const [parentTreeData, setParentTreeData] = useState<any[]>([]);

  // 加载父级分组树
  useEffect(() => {
    if (open) {
      fetchListPermissionGroups(new PaginationQuery({ formValues: { status: 'ON' } }))
        .then((res) => {
          setParentTreeData(buildTreeSelectData(res?.items || [], data?.id));
        })
        .catch(() => setParentTreeData([]));
    }
  }, [open, data?.id]);

  // 编辑模式下设置表单值
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          name: data.name || '',
          module: data.module || '',
          parentId: data.parentId || undefined,
          sortOrder: data.sortOrder ?? 1,
          status: data.status || 'ON',
        });
      }, 0);
    }
  }, [open, mode, data]);

  // 创建 mutation
  const createMutation = useCreatePermissionGroup({
    onSuccess: () => {
      message.success(t('createSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listPermissionGroups'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新 mutation
  const updateMutation = useUpdatePermissionGroup({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listPermissionGroups'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  // 提交表单
  const handleSubmit = async (values: Record<string, any>) => {
    try {
      setConfirmLoading(true);
      if (mode === 'edit' && data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values });
      } else {
        await createMutation.mutateAsync(values as any);
      }
      return true;
    } catch {
      return false;
    } finally {
      setConfirmLoading(false);
    }
  };

  return (
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? t('create') : t('edit')}
      open={open}
      onOpenChange={(visible) => {
        if (!visible) {
          formRef.current?.resetFields();
          onClose();
        }
      }}
      initialValues={{
        sortOrder: 1,
        status: 'ON',
      }}
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading || createMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: { onClick: onClose },
      }}
      drawerProps={{ destroyOnClose: true, onClose, placement: 'left', size: 600 }}
    >
      <ProFormText
        name="name"
        label={t('name')}
        placeholder={t('namePlaceholder')}
        rules={[{ required: true, message: t('requiredName') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="module"
        label={t('module')}
        placeholder={t('modulePlaceholder')}
        rules={[{ required: true, message: t('requiredModule') }]}
        fieldProps={{ allowClear: true }}
      />

      {/* 父级分组 - TreeSelect */}
      <Form.Item
        name="parentId"
        label={t('parentId')}
      >
        <TreeSelect
          treeData={parentTreeData}
          placeholder={t('parentIdPlaceholder')}
          allowClear
          showSearch
          treeDefaultExpandAll
        />
      </Form.Item>

      <ProFormDigit
        name="sortOrder"
        label={t('sortOrder')}
        placeholder={t('sortOrderPlaceholder')}
        fieldProps={{ precision: 0, min: 0 }}
      />

      <ProFormRadio.Group
        name="status"
        label={t('status')}
        rules={[{ required: true, message: t('requiredStatus') }]}
        options={getStatusOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />
    </DrawerForm>
  );
};

export default PermissionGroupDrawer;
