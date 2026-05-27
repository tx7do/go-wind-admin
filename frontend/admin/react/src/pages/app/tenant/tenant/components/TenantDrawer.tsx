import React, { useEffect, useRef } from 'react';
import { DrawerForm, ProFormText, ProFormSelect, ProFormTextArea } from '@ant-design/pro-components';
import type { ProFormInstance } from '@ant-design/pro-components';
import { Button, message, Divider } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { useUpdateTenant, useCreateTenantWithAdminUser } from '@/api/hooks/tenant';
import type {
  identityservicev1_Tenant,
  identityservicev1_Tenant_Type,
  identityservicev1_Tenant_Status,
  identityservicev1_Tenant_AuditStatus,
} from '@/api/generated/admin/service/v1';
import {
  getTenantTypeOptions,
  getAuditStatusOptions,
  getTenantStatusOptions,
  SELECT_FILTER_PROPS,
} from '../constants';

interface TenantDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: identityservicev1_Tenant;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 租户编辑/创建 Drawer 组件（基于 DrawerForm）
 */
const TenantDrawer: React.FC<TenantDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('tenant');
  const queryClient = useQueryClient();
  const formRef = useRef<ProFormInstance>(null);

  const updateMutation = useUpdateTenant();
  const createMutation = useCreateTenantWithAdminUser();

  // 当 Drawer 打开时，设置表单初始值
  useEffect(() => {
    if (open) {
      setTimeout(() => {
        if (mode === 'edit' && data && formRef.current) {
          formRef.current.setFieldsValue({
            name: data.name,
            code: data.code,
            tenantType: data.type,
            auditStatus: data.auditStatus,
            status: data.status,
            remark: data.remark,
          });
        } else if (formRef.current) {
          // 创建模式下的默认值
          formRef.current.setFieldsValue({
            tenantType: 'PAID',
            auditStatus: 'APPROVED',
            status: 'ON',
          });
        }
      }, 0);
    }
  }, [open, mode, data]);

  // 提交处理
  const handleSubmit = async (values: any) => {
    try {
      if (mode === 'create') {
        // 检查密码和确认密码是否一致
        if (values.password !== values.passwordConfirm) {
          message.error(t('passwordMismatch'));
          return false;
        }

        // 创建租户及管理员用户
        await createMutation.mutateAsync({
          tenant: {
            name: values.name,
            code: values.code,
            type: values.tenantType as identityservicev1_Tenant_Type,
            auditStatus: values.auditStatus as identityservicev1_Tenant_AuditStatus,
            status: values.status as identityservicev1_Tenant_Status,
            remark: values.remark,
          },
          user: {
            username: values.username,
            mobile: values.mobile,
            email: values.email,
            orgUnitIds: undefined,
            orgUnitNames: undefined,
            positionIds: undefined,
            positionNames: undefined,
            roleIds: undefined,
            roles: undefined,
            roleNames: undefined,
          },
          password: values.password,
        });
        message.success(t('createSuccess'));
      } else {
        // 更新租户
        if (!data?.id) {
          message.error(t('tenantIdMissing'));
          return false;
        }
        await updateMutation.mutateAsync({
          id: data.id,
          values: {
            name: values.name,
            code: values.code,
            type: values.tenantType as identityservicev1_Tenant_Type,
            auditStatus: values.auditStatus as identityservicev1_Tenant_AuditStatus,
            status: values.status as identityservicev1_Tenant_Status,
            remark: values.remark,
          },
        });
        message.success(t('updateSuccess'));
      }

      // 刷新列表
      queryClient.invalidateQueries({ queryKey: ['listTenants'] });

      // 重置表单并关闭
      formRef.current?.resetFields();
      onSuccess();

      return true;
    } catch (error: any) {
      message.error(error.message || (mode === 'create' ? t('createFailed') : t('updateFailed')));
      return false;
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
        tenantType: 'PAID',
        auditStatus: 'APPROVED',
        status: 'ON',
      }}
      onFinish={handleSubmit}
      submitter={{
        render: (_, dom) => [
          <Button key="reset" onClick={() => formRef.current?.resetFields()}>
            {t('reset')}
          </Button>,
          ...dom,
        ],
      }}
      drawerProps={{
        destroyOnClose: true,
        onClose,
        size: 600,
      }}
    >
      <ProFormText
        name="name"
        label={t('name')}
        placeholder={t('namePlaceholder')}
        rules={[
          { required: true, message: t('requiredName') },
          { max: 50, message: t('maxChars', { max: 50 }) },
        ]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormText
        name="code"
        label={t('code')}
        placeholder={t('codePlaceholder')}
        rules={[
          { required: true, message: t('requiredCode') },
          { max: 50, message: t('maxChars', { max: 50 }) },
        ]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormSelect
        name="tenantType"
        label={t('tenantType')}
        placeholder={t('tenantTypePlaceholder')}
        options={getTenantTypeOptions(t)}
        rules={[{ required: true, message: t('requiredType') }]}
        fieldProps={SELECT_FILTER_PROPS}
      />

      <ProFormSelect
        name="auditStatus"
        label={t('auditStatus')}
        placeholder={t('auditStatusPlaceholder')}
        options={getAuditStatusOptions(t)}
        rules={[{ required: true, message: t('requiredAuditStatus') }]}
        fieldProps={SELECT_FILTER_PROPS}
      />

      <ProFormSelect
        name="status"
        label={t('status')}
        placeholder={t('statusPlaceholder')}
        options={getTenantStatusOptions(t)}
        rules={[{ required: true, message: t('requiredStatus') }]}
        fieldProps={SELECT_FILTER_PROPS}
      />

      <ProFormTextArea
        name="remark"
        label={t('remark')}
        placeholder={t('remarkPlaceholder')}
        fieldProps={{
          rows: 4,
          allowClear: true,
          maxLength: 500,
          showCount: true,
        }}
      />

      {/* 创建模式下显示管理员账号配置 */}
      {mode === 'create' && (
        <>
          <Divider style={{ margin: '24px 0 16px' }}>{t('adminSection')}</Divider>

          <ProFormText
            name="username"
            label={t('adminUsernameLabel')}
            placeholder={t('adminUsernamePlaceholder')}
            rules={[
              { required: true, message: t('requiredAdminUsername') },
              { max: 50, message: t('maxChars', { max: 50 }) },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />

          <ProFormText.Password
            name="password"
            label={t('password')}
            placeholder={t('passwordPlaceholder')}
            rules={[
              { required: true, message: t('requiredPassword') },
              { min: 6, message: t('passwordMin', { min: 6 }) },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />

          <ProFormText.Password
            name="passwordConfirm"
            label={t('passwordConfirm')}
            placeholder={t('passwordConfirmPlaceholder')}
            rules={[
              { required: true, message: t('requiredPasswordConfirm') },
              { min: 6, message: t('passwordMin', { min: 6 }) },
              {
                validator: async (_, value) => {
                  const password = formRef.current?.getFieldValue('password');
                  if (value && password !== value) {
                    return Promise.reject(new Error(t('passwordMismatch')));
                  }
                  return Promise.resolve();
                },
              },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />

          <ProFormText
            name="mobile"
            label={t('mobile')}
            placeholder={t('mobilePlaceholder')}
            rules={[
              { required: true, message: t('requiredMobile') },
              { pattern: /^1[3-9]\d{9}$/, message: t('mobileInvalid') },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />

          <ProFormText
            name="email"
            label={t('email')}
            placeholder={t('emailPlaceholder')}
            rules={[
              { required: true, message: t('requiredEmail') },
              { type: 'email', message: t('emailInvalid') },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />
        </>
      )}
    </DrawerForm>
  );
};

export default TenantDrawer;
