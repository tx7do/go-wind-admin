import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormDigit,
  ProFormTextArea,
  ProFormSelect,
  ProFormRadio,
} from '@ant-design/pro-components';
import { App, TreeSelect, Form } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { identityservicev1_Position as Position } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { useCreatePosition, useUpdatePosition } from '@/api/hooks/position';
import { fetchListOrgUnits } from '@/api/hooks/org-unit';
import { getStatusOptions, getPositionTypeOptions, buildOrgTreeData } from '../constants';

interface PositionDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: Position;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 职位编辑/创建抽屉组件
 */
const PositionDrawer: React.FC<PositionDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('position');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);
  const [orgTreeData, setOrgTreeData] = useState<any[]>([]);

  // 加载组织树
  useEffect(() => {
    if (open) {
      fetchListOrgUnits(new PaginationQuery({ formValues: { status: 'ON' } }))
        .then((res) => {
          const items = res?.items || [];
          setOrgTreeData(buildOrgTreeData(items as any[]));
        })
        .catch(() => setOrgTreeData([]));
    }
  }, [open]);

  // 编辑模式填充表单
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      // 使用 setTimeout 确保 DrawerForm 内部初始化完成后再设值
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          name: data.name || '',
          code: data.code || '',
          type: data.type || 'REGULAR',
          orgUnitId: (data as any).orgUnitId || undefined,
          headcount: (data as any).headcount ?? 1,
          sortOrder: (data as any).sortOrder ?? 1,
          status: data.status || 'ON',
          description: (data as any).description || '',
          remark: (data as any).remark || '',
        });
      }, 0);
    }
  }, [open, mode, data]);

  const createMutation = useCreatePosition({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listPositions'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  const updateMutation = useUpdatePosition({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listPositions'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {
      if (mode === 'create') {
        await createMutation.mutateAsync({ data: values });
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
      title={mode === 'create' ? t('create') : t('edit')}
      open={open}
      onOpenChange={(visible) => {
        if (!visible) {
          formRef.current?.resetFields();
          onClose();
        }
      }}
      initialValues={{
        type: 'REGULAR',
        headcount: 1,
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
      drawerProps={{ destroyOnClose: true, onClose, size: 600 }}
    >
      <ProFormText
        name="name"
        label={t('name')}
        placeholder={t('namePlaceholder')}
        rules={[{ required: true, message: t('requiredName') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="code"
        label={t('code')}
        placeholder={t('codePlaceholder')}
        rules={[{ required: true, message: t('requiredCode') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormSelect
        name="type"
        label={t('type')}
        placeholder={t('typePlaceholder')}
        options={getPositionTypeOptions(t)}
        rules={[{ required: true, message: t('requiredType') }]}
        fieldProps={{
          showSearch: true,
          allowClear: true,
          filterOption: (input: string, option: any) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      {/* 所属组织 - TreeSelect */}
      <Form.Item
        name="orgUnitId"
        label={t('orgUnit')}
        rules={[{ required: true, message: t('requiredOrgUnit') }]}
      >
        <TreeSelect
          showSearch
          allowClear
          treeDefaultExpandAll
          placeholder={t('orgUnitPlaceholder')}
          treeData={orgTreeData}
        />
      </Form.Item>

      <ProFormDigit
        name="headcount"
        label={t('headcount')}
        placeholder={t('headcountPlaceholder')}
        rules={[{ required: true }]}
        fieldProps={{ precision: 0, min: 0 }}
      />

      <ProFormDigit
        name="sortOrder"
        label={t('sortOrder')}
        placeholder={t('sortOrderPlaceholder')}
        rules={[{ required: true }]}
        fieldProps={{ precision: 0, min: 0 }}
      />

      <ProFormRadio.Group
        name="status"
        label={t('status')}
        rules={[{ required: true }]}
        options={getStatusOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />

      <ProFormTextArea
        name="description"
        label={t('description')}
        placeholder={t('descriptionPlaceholder')}
        fieldProps={{ allowClear: true, rows: 2 }}
      />

      <ProFormTextArea
        name="remark"
        label={t('remark')}
        placeholder={t('remarkPlaceholder')}
        fieldProps={{ allowClear: true, rows: 2 }}
      />
    </DrawerForm>
  );
};

export default PositionDrawer;
