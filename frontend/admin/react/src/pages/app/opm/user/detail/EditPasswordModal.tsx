import { useState } from 'react';
import { Modal, Form, Input, App } from 'antd';
import { useTranslation } from 'react-i18next';
import { useEditUserPassword } from '@/api/hooks/user';
import { queryClient } from '@/core';

interface EditPasswordModalProps {
  open: boolean;
  userId: string | undefined; // uint64 → proto3 JSON 下为字符串
  onClose: () => void;
}

/**
 * 管理员修改用户密码弹窗
 */
const EditPasswordModal: React.FC<EditPasswordModalProps> = ({ open, userId, onClose }) => {
  const { t } = useTranslation('user-detail');
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [confirmLoading, setConfirmLoading] = useState(false);

  const editPasswordMutation = useEditUserPassword({
    onSuccess: () => {
      setConfirmLoading(false);
      message.success(t('editPasswordSuccess'));
      form.resetFields();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['getUser'] });
    },
    onError: (error: Error) => {
      setConfirmLoading(false);
      message.error(error.message || t('editPasswordFailed'));
    },
  });

  const handleOk = async () => {
    try {
      const values = await form.validateFields();

      if (values.newPassword !== values.confirmPassword) {
        message.error(t('passwordMismatch'));
        return;
      }

      setConfirmLoading(true);
      editPasswordMutation.mutate({
        userId: userId!,
        newPassword: values.newPassword,
      });
    } catch {
      // 表单验证失败
    }
  };

  const handleCancel = () => {
    form.resetFields();
    onClose();
  };

  return (
    <Modal
      title={t('editPasswordTitle')}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      confirmLoading={confirmLoading}
    >
      <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
        <Form.Item
          name="newPassword"
          label={t('newPassword')}
          rules={[{ required: true, message: t('newPasswordPlaceholder') }]}
        >
          <Input.Password placeholder={t('newPasswordPlaceholder')} />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          label={t('confirmPassword')}
          rules={[{ required: true, message: t('confirmPasswordPlaceholder') }]}
        >
          <Input.Password placeholder={t('confirmPasswordPlaceholder')} />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default EditPasswordModal;
