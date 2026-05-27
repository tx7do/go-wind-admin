import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormDigit,
  ProFormRadio,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { useCreateDictType, useUpdateDictType } from '@/api/hooks/dict';
import { enableBoolRadioOptions } from './constants';

interface DictTypeDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: any;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 字典类型编辑/创建抽屉组件
 */
const DictTypeDrawer: React.FC<DictTypeDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('dict-type');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 编辑模式下设置表单值（destroyOnClose 时需延迟赋值）
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          typeName: data.typeName || '',
          typeCode: data.typeCode || '',
          sortOrder: data.sortOrder ?? 1,
          isEnabled: data.isEnabled ?? true,
        });
      }, 0);
    }
  }, [open, mode, data]);

  // 创建 mutation
  const createMutation = useCreateDictType({
    onSuccess: () => {
      message.success(t('createSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listDictTypes'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新 mutation
  const updateMutation = useUpdateDictType({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listDictTypes'] });
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
        await createMutation.mutateAsync({ data: values });
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
        isEnabled: true,
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
        name="typeName"
        label={t('typeName')}
        placeholder={t('typeNamePlaceholder')}
        rules={[{ required: true, message: t('requiredTypeName') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="typeCode"
        label={t('typeCode')}
        placeholder={t('typeCodePlaceholder')}
        rules={[{ required: true, message: t('requiredTypeCode') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormDigit
        name="sortOrder"
        label={t('sortOrder')}
        placeholder={t('sortOrderPlaceholder')}
        fieldProps={{ precision: 0, min: 0 }}
      />

      <ProFormRadio.Group
        name="isEnabled"
        label={t('status')}
        rules={[{ required: true, message: t('requiredStatus') }]}
        options={enableBoolRadioOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />
    </DrawerForm>
  );
};

export default DictTypeDrawer;
