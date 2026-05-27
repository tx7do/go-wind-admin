import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { storageservicev1_File as FileItem } from '@/api/generated/admin/service/v1';
import { useCreateFile, useUpdateFile } from '@/api/hooks/file';

interface FileDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: FileItem;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 文件编辑/创建抽屉组件
 */
const FileDrawer: React.FC<FileDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('file');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 创建文件
  const createMutation = useCreateFile({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listFiles'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新文件
  const updateMutation = useUpdateFile({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listFiles'] });
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
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? t('upload') : t('edit')}
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
        size: 480,
      }}
    >
      <ProFormText
        name="fileName"
        label={t('fileName')}
        placeholder={t('fileNamePlaceholder')}
        rules={[{ required: true, message: t('requiredFileName') }]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormTextArea
        name="remark"
        label={t('remark')}
        placeholder={t('remarkPlaceholder')}
        fieldProps={{
          allowClear: true,
          rows: 3,
        }}
      />
    </DrawerForm>
  );
};

export default FileDrawer;
