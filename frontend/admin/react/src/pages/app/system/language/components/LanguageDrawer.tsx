import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormDigit,
  ProFormSwitch,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { dictservicev1_Language as Language } from '@/api/generated/admin/service/v1';
import { useCreateLanguage, useUpdateLanguage } from '@/api/hooks/language';

interface LanguageDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: Language;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 语言编辑/创建抽屉组件
 */
const LanguageDrawer: React.FC<LanguageDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('language');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 创建语言
  const createMutation = useCreateLanguage({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listLanguages'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新语言
  const updateMutation = useUpdateLanguage({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listLanguages'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  // 提交表单
  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);

    try {
      const payload = {
        ...values,
        isEnabled: values.isEnabled ?? true,
        isDefault: values.isDefault ?? false,
      };

      if (mode === 'create') {
        await createMutation.mutateAsync({ data: payload });
      } else if (data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values: payload });
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
      initialValues={
        mode === 'edit'
          ? { ...data }
          : {
              sortOrder: 1,
              isEnabled: true,
              isDefault: false,
            }
      }
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
        size: 480,
      }}
    >
      <ProFormText
        name="languageName"
        label={t('languageName')}
        placeholder={t('languageNamePlaceholder')}
        rules={[{ required: true, message: t('requiredLanguageName') }]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormText
        name="languageCode"
        label={t('languageCode')}
        placeholder={t('languageCodePlaceholder')}
        rules={[{ required: true, message: t('requiredLanguageCode') }]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormText
        name="nativeName"
        label={t('nativeName')}
        placeholder={t('nativeNamePlaceholder')}
        rules={[{ required: true, message: t('requiredNativeName') }]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormDigit
        name="sortOrder"
        label={t('sortOrder')}
        placeholder={t('sortOrderPlaceholder')}
        fieldProps={{
          precision: 0,
          min: 0,
        }}
      />

      <ProFormSwitch
        name="isEnabled"
        label={t('isEnabled')}
      />

      <ProFormSwitch
        name="isDefault"
        label={t('isDefault')}
      />
    </DrawerForm>
  );
};

export default LanguageDrawer;
