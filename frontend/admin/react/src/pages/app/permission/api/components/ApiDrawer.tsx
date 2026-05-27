import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import { DrawerForm, ProFormText, ProFormSelect } from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { permissionservicev1_Api as Api } from '@/api/generated/admin/service/v1';
import { useCreateApi, useUpdateApi } from '@/api/hooks/api';

import { METHOD_LIST } from '@/config/constants';

interface ApiDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: Api;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * API 编辑/创建抽屉组件
 */
const ApiDrawer: React.FC<ApiDrawerProps> = ({ open, mode, data, onClose, onSuccess }) => {
  const { t } = useTranslation('api');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 创建 API
  const createMutation = useCreateApi({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新 API
  const updateMutation = useUpdateApi({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  // 提交表单
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
    <DrawerForm<Api>
      formRef={formRef}
      open={open}
      title={mode === 'create' ? t('create') : t('edit')}
      initialValues={mode === 'edit' ? data : undefined}
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.reset'),
        },
        submitButtonProps: {
          loading: confirmLoading || createMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: {
          onClick: onClose,
        },
      }}
      drawerProps={{
        destroyOnClose: true,
        onClose,
        size: 600,
      }}
    >
      <ProFormText
        name="description"
        label={t('description')}
        placeholder={t('descriptionPlaceholder')}
        rules={[{ required: false }, { max: 200, message: t('maxChars', { max: 200 }) }]}
      />

      <ProFormText
        name="module"
        label={t('module')}
        placeholder={t('modulePlaceholder')}
        rules={[{ required: false }, { max: 100, message: t('maxChars', { max: 100 }) }]}
      />

      <ProFormText
        name="moduleDescription"
        label={t('moduleDescription')}
        placeholder={t('moduleDescriptionPlaceholder')}
        rules={[{ required: false }, { max: 200, message: t('maxChars', { max: 200 }) }]}
      />

      <ProFormText
        name="path"
        label={t('path')}
        placeholder={t('pathPlaceholder')}
        rules={[
          { required: true, message: t('requiredPath') },
          { max: 500, message: t('maxChars', { max: 500 }) },
        ]}
      />

      <ProFormSelect
        name="method"
        label={t('method')}
        placeholder={t('methodPlaceholder')}
        options={METHOD_LIST}
        rules={[{ required: true, message: t('requiredMethod') }]}
        fieldProps={{
          showSearch: true,
          filterOption: (input, option) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />
    </DrawerForm>
  );
};

export default ApiDrawer;
