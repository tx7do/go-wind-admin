import { useEffect, useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormDatePicker,
  ProFormRadio,
  ProFormText,
} from '@ant-design/pro-components';
import { App, Modal, Typography } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { access_keyservicev1_AccessKey as AccessKey } from '@/api/generated/admin/service/v1';
import { useCreateAccessKey, useUpdateAccessKey } from '@/api/hooks/access-key';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';

interface AccessKeyDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: AccessKey;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 凭证创建/编辑抽屉。
 * 创建成功后弹出一次性 Secret 展示（服务端只存 SHA-256 摘要，关闭后不可再查看），
 * 因此创建成功时阻断抽屉自动关闭，用户确认后一并收起并刷新列表。
 */
export default function AccessKeyDrawer({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}: AccessKeyDrawerProps) {
  const { t } = useTranslation('access-key');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const formRef = useRef<ProFormInstance>(null);

  const [confirmLoading, setConfirmLoading] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);

  const createMutation = useCreateAccessKey();
  const updateMutation = useUpdateAccessKey();

  useEffect(() => {
    if (!open) return;
    setTimeout(() => {
      if (mode === 'edit' && data) {
        formRef.current?.setFieldsValue({
          name: data.name,
          status: data.status ?? 'ON',
          expiresAt: data.expiresAt,
        });
      }
    }, 0);
  }, [open, mode, data]);

  // 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
  const { guardClose, handleValuesChange, markClean } = useDrawerCloseGuard(open, onClose);

  const resetAndClose = () => {
    formRef.current?.resetFields();
    onClose();
  };

  const handleSecretClose = () => {
    setCreatedSecret(null);
    resetAndClose();
    onSuccess();
  };

  const handleSubmit = async (values: Record<string, any>) => {
    setConfirmLoading(true);
    try {
      if (mode === 'create') {
        const resp = await createMutation.mutateAsync({
          data: {
            name: values.name,
            expiresAt: values.expiresAt ?? undefined,
          },
        } as any);
        queryClient.invalidateQueries({ queryKey: ['listAccessKeys'] });
        setCreatedSecret(resp.secret ?? null);
        markClean(); // Secret 已生成即已保存成功，后续关闭不再确认
        return true; // 阻止抽屉自动关闭：先展示 Secret
      }
      await updateMutation.mutateAsync({
        id: data!.id!,
        values: {
          name: values.name,
          status: values.status,
          expiresAt: values.expiresAt ?? undefined,
        },
      });
      message.success(t('updateSuccess'));
      resetAndClose();
      onSuccess();
      return true;
    } catch (error: any) {
      message.error(error?.message || t('fetchFailed'));
      return false;
    } finally {
      setConfirmLoading(false);
    }
  };

  return (
    <>
      <DrawerForm
        title={mode === 'create' ? t('create') : t('edit')}
        open={open}
        onValuesChange={handleValuesChange}
        formRef={formRef}
        drawerProps={{
          onClose: guardClose,
          destroyOnHidden: true,
          // antd6 Drawer width 已废弃：size 直接收数字
          size: 480,
        }}
        onFinish={handleSubmit}
        submitter={{ submitButtonProps: { loading: confirmLoading } }}
        layout="horizontal"
        labelCol={{ flex: '0 0 120px' }}
        wrapperCol={{ flex: 'auto' }}
      >
        <ProFormText
          name="name"
          label={t('name')}
          placeholder={t('namePlaceholder')}
          rules={[{ required: true, message: t('requiredName') }]}
        />
        {mode === 'edit' && (
          <ProFormRadio.Group
            name="status"
            label={t('status')}
            options={[
              { label: t('statusMap.ON'), value: 'ON' },
              { label: t('statusMap.OFF'), value: 'OFF' },
            ]}
          />
        )}
        <ProFormDatePicker
          name="expiresAt"
          label={t('expiresAt')}
          fieldProps={{ showTime: true, style: { width: '100%' } }}
          extra={t('expiresAtHint')}
        />
      </DrawerForm>

      <Modal
        title={t('secretDialogTitle')}
        open={createdSecret !== null}
        onOk={handleSecretClose}
        onCancel={handleSecretClose}
        cancelButtonProps={{ style: { display: 'none' } }}
        width={560}
      >
        <Typography.Paragraph type="warning">
          {t('secretDialogHint')}
        </Typography.Paragraph>
        <Typography.Paragraph copyable={{ text: createdSecret ?? '' }}>
          <Typography.Text code style={{ wordBreak: 'break-all' }}>
            {createdSecret}
          </Typography.Text>
        </Typography.Paragraph>
      </Modal>
    </>
  );
}
