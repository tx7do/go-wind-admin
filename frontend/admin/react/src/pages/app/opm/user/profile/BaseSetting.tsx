import { useState } from 'react';
import { Form, Input, Select, Button, Avatar, Row, Col, Descriptions, Tag, App, Upload } from 'antd';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import type { identityservicev1_User as User } from '@/api/generated/admin/service/v1';
import {
  useGetUserProfile,
  useUpdateUserProfile,
  useUploadAvatar,
  useDeleteAvatar,
} from '@/api/hooks/user-profile';
import { getCharColor, getRandomColor } from '@go-wind/admin-react';
import { formatDateTime } from '@go-wind/admin-react';

const BaseSetting = () => {
  const { t } = useTranslation(['profile', 'common']);
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  const { data: userData, isLoading } = useGetUserProfile();
  const user = userData as User | undefined | null;
  const { mutateAsync: uploadAvatar } = useUploadAvatar();
  const { mutateAsync: deleteAvatar } = useDeleteAvatar();

  const firstChar = user?.username?.slice(0, 1).toUpperCase() || '?';
  const avatarColor = getCharColor(firstChar);

  const updateMutation = useUpdateUserProfile({
    onSuccess: () => {
      message.success(t('saveSuccess'));
      queryClient.invalidateQueries({ queryKey: ['getMe'] });
    },
    onError: (err: Error) => {
      console.error('Failed to update profile:', err);
      message.error(err.message || t('saveFailed'));
    },
  });

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      if (!user?.id) return;
      setSaving(true);
      await updateMutation.mutateAsync({ id: user.id, values });
    } catch {
      // 表单校验失败，Antd 已就地标红字段
    } finally {
      setSaving(false);
    }
  };

  /**
   * 头像上传：选中文件转 base64（剥离 dataURL 前缀）后提交。
   * 后端用 base64.StdEncoding.DecodeString 解码，不识别 dataURL 前缀，故必须传纯 base64。
   */
  const handleUploadAvatar = async (file: File) => {
    const reader = new FileReader();
    reader.addEventListener('load', async () => {
      try {
        const dataUrl = reader.result as string;
        const base64 = dataUrl.replace(/^data:[^;]+;base64,/, '');
        await uploadAvatar({ imageBase64: base64 });
        await queryClient.refetchQueries({ queryKey: ['getMe'] });
        message.success(t('avatarUploadSuccess'));
      } catch (error: any) {
        console.error('Failed to upload avatar:', error);
        message.error(error?.message || t('avatarUploadFailed'));
      }
    });
    reader.addEventListener('error', () => {
      console.error('Failed to read avatar file:', reader.error);
      message.error(t('avatarUploadFailed'));
    });
    reader.readAsDataURL(file);
    return false; // 阻止 antd 默认上传行为
  };

  const beforeAvatarUpload = (file: File) => {
    const isImage = file.type.startsWith('image/');
    const withinSize = file.size / 1024 / 1024 < 5;
    if (!isImage) {
      message.error(t('avatarNotImage'));
      return false;
    }
    if (!withinSize) {
      message.error(t('avatarTooLarge'));
      return false;
    }
    return true;
  };

  const handleDeleteAvatar = async () => {
    try {
      await deleteAvatar();
      await queryClient.refetchQueries({ queryKey: ['getMe'] });
      message.success(t('avatarDeleted'));
    } catch (error: any) {
      console.error('Failed to delete avatar:', error);
      message.error(error?.message || t('avatarDeleteFailed'));
    }
  };

  if (isLoading) return null;

  return (
    <div style={{ overflowX: 'hidden' }}>
      <Row gutter={24}>
      <Col span={14}>
        <Form
          form={form}
          layout="horizontal"
          labelCol={{ flex: '0 0 120px' }}
          wrapperCol={{ flex: 'auto' }}
          initialValues={{
            nickname: user?.nickname || '',
            realname: user?.realname || '',
            email: user?.email || '',
            mobile: user?.mobile || '',
            telephone: user?.telephone || '',
            gender: user?.gender,
            region: user?.region || '',
            address: user?.address || '',
            description: user?.description || '',
            locale: user?.locale || '',
          }}
        >
          <Form.Item name="nickname" label={t('field.nickname')}>
            <Input placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="realname" label={t('field.realname')}>
            <Input placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="email" label={t('field.email')}>
            <Input placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="mobile" label={t('field.mobile')}>
            <Input placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="telephone" label={t('field.telephone')}>
            <Input placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="gender" label={t('field.gender')}>
            <Select placeholder={t('common:placeholder.select')} allowClear options={[
              { value: 'MALE', label: t('genderMap.MALE') },
              { value: 'FEMALE', label: t('genderMap.FEMALE') },
              { value: 'SECRET', label: t('genderMap.SECRET') },
            ]} />
          </Form.Item>
          <Form.Item name="region" label={t('field.region')}>
            <Input placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="address" label={t('field.address')}>
            <Input placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="description" label={t('field.description')}>
            <Input.TextArea rows={3} placeholder={t('common:placeholder.input')} />
          </Form.Item>
          <Form.Item name="locale" label={t('field.locale')}>
            <Select placeholder={t('common:placeholder.select')} allowClear options={[
              { value: 'zh-CN', label: '中文' },
              { value: 'en-US', label: 'English' },
            ]} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" loading={saving} onClick={handleSave}>
              {t('save')}
            </Button>
          </Form.Item>
        </Form>
      </Col>

      <Col span={10}>
        <div style={{ paddingTop: 20 }}>
          <div style={{ fontSize: 14, fontWeight: 500, marginBottom: 16 }}>{t('field.avatar')}</div>
          <Avatar
            src={user?.avatar || undefined}
            size={96}
            style={!user?.avatar ? { backgroundColor: avatarColor, fontSize: 40, fontWeight: 700 } : {}}
          >
            {firstChar}
          </Avatar>
          <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
            <Upload
              showUploadList={false}
              accept="image/*"
              beforeUpload={beforeAvatarUpload}
              customRequest={({ file }) => handleUploadAvatar(file as File)}
            >
              <Button type="primary">{t('button.uploadAvatar')}</Button>
            </Upload>
            {user?.avatar && (
              <Button danger onClick={handleDeleteAvatar}>{t('button.deleteAvatar')}</Button>
            )}
          </div>
        </div>

        {/* 账号归属信息（只读） */}
        <Descriptions column={1} size="small" style={{ marginTop: 24 }}>
          <Descriptions.Item label={t('info.username')}>
            {user?.username || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('info.roleNames')}>
            {user?.roleNames?.length ? (
              user.roleNames.map((role) => (
                <Tag key={role} color={getRandomColor(role)} style={{ marginRight: 4 }}>
                  {role}
                </Tag>
              ))
            ) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('info.orgUnitNames')}>
            {user?.orgUnitNames?.length ? (
              user.orgUnitNames.map((org) => (
                <Tag key={org} color={getRandomColor(org)} style={{ marginRight: 4 }}>
                  {org}
                </Tag>
              ))
            ) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('info.positionNames')}>
            {user?.positionNames?.length ? (
              user.positionNames.map((pos) => (
                <Tag key={pos} color={getRandomColor(pos)} style={{ marginRight: 4 }}>
                  {pos}
                </Tag>
              ))
            ) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('info.tenantName')}>
            {user?.tenantName || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('info.lastLoginAt')}>
            {formatDateTime(user?.lastLoginAt ?? '') || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('info.lastLoginIp')}>
            {user?.lastLoginIp || '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('info.createdAt')}>
            {formatDateTime(user?.createdAt ?? '') || '-'}
          </Descriptions.Item>
        </Descriptions>
      </Col>
    </Row>
    </div>
  );
};

export default BaseSetting;
