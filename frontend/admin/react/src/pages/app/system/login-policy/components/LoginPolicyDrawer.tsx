import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormSelect,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { authenticationservicev1_LoginPolicy as LoginPolicy } from '@/api/generated/admin/service/v1';
import { useCreateLoginPolicy, useUpdateLoginPolicy } from '@/api/hooks/login-policy';
import { getPolicyTypeOptions, getPolicyMethodOptions } from '../constants';

interface LoginPolicyDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: LoginPolicy;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 登录策略编辑/创建抽屉组件
 */
const LoginPolicyDrawer: React.FC<LoginPolicyDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('login-policy');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 创建策略
  const createMutation = useCreateLoginPolicy({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listLoginPolicies'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新策略
  const updateMutation = useUpdateLoginPolicy({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listLoginPolicies'] });
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
        await createMutation.mutateAsync(values);
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
      initialValues={mode === 'edit' ? { ...data } : undefined}
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
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
        size: 520,
      }}
    >
      <ProFormText
        name="targetId"
        label={t('targetId')}
        placeholder={t('targetIdPlaceholder')}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormSelect
        name="type"
        label={t('type')}
        placeholder={t('typePlaceholder')}
        options={getPolicyTypeOptions(t)}
        rules={[{ required: true, message: t('requiredType') }]}
        fieldProps={{
          showSearch: true,
          filterOption: (input: string, option: any) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormSelect
        name="method"
        label={t('method')}
        placeholder={t('methodPlaceholder')}
        options={getPolicyMethodOptions(t)}
        rules={[{ required: true, message: t('requiredMethod') }]}
        fieldProps={{
          showSearch: true,
          filterOption: (input: string, option: any) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormText
        name="value"
        label={t('value')}
        placeholder={t('valuePlaceholder')}
        rules={[{ required: true, message: t('requiredValue') }]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormTextArea
        name="reason"
        label={t('reason')}
        placeholder={t('reasonPlaceholder')}
        fieldProps={{
          allowClear: true,
          rows: 3,
        }}
      />
    </DrawerForm>
  );
};

export default LoginPolicyDrawer;
