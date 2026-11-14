import { useState } from 'react';
import { Form, Input, Button, App } from 'antd';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@go-wind/admin-react';
import { useChangePassword } from '@/api/hooks/user-profile';

const EditPassword = () => {
  const { t } = useTranslation('profile');
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [changingPwd, setChangingPwd] = useState(false);

  // 后端在改密成功后已吊销本人全部会话令牌（含当前会话），
  // 这里前端同步 forceLogout（不调已失效的 logout API），由路由守卫重定向到登录页。
  const changePwdMutation = useChangePassword({
    onSuccess: () => {
      message.success(t('changePasswordSuccess'));
      form.resetFields();
      useAuthStore.getState().forceLogout();
    },
    onError: (err: Error) => {
      console.error('Failed to change password:', err);
      message.error(err.message || t('changePasswordFailed'));
    },
  });

  const handleChangePassword = async () => {
    try {
      const values = await form.validateFields();
      if (values.newPassword !== values.confirmPassword) {
        message.error(t('passwordMismatch'));
        return;
      }
      setChangingPwd(true);
      await changePwdMutation.mutateAsync({
        oldPassword: values.oldPassword,
        newPassword: values.newPassword,
      });
    } catch {
      // 表单校验失败，Antd 已就地标红字段
    } finally {
      setChangingPwd(false);
    }
  };

  return (
    <div style={{ maxWidth: 460 }}>
      <Form form={form} layout="horizontal" labelCol={{ flex: '0 0 120px' }} wrapperCol={{ flex: 'auto' }}>
        <Form.Item
          name="oldPassword"
          label={t('oldPassword')}
          rules={[{ required: true, message: t('requiredOldPassword') }]}
        >
          <Input.Password placeholder={t('oldPasswordPlaceholder')} />
        </Form.Item>
        <Form.Item
          name="newPassword"
          label={t('newPassword')}
          rules={[{ required: true, message: t('requiredNewPassword') }]}
        >
          <Input.Password placeholder={t('newPasswordPlaceholder')} />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          label={t('confirmPassword')}
          rules={[{ required: true, message: t('requiredConfirmPassword') }]}
        >
          <Input.Password placeholder={t('confirmPasswordPlaceholder')} />
        </Form.Item>
        <Form.Item>
          <Button type="primary" loading={changingPwd} onClick={handleChangePassword}>
            {t('changePassword')}
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
};

export default EditPassword;
