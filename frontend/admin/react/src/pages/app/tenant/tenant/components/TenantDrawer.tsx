import React, { useEffect, useRef, useState } from 'react';
import { DrawerForm, ProFormText, ProFormSelect, ProFormTextArea, ProFormDateTimePicker } from '@ant-design/pro-components';
import type { ProFormInstance } from '@ant-design/pro-components';
import { Button, message, Divider, Popconfirm, Descriptions, Tag, Progress } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { useUpdateTenant, useCreateTenantWithAdminUser, useGetTenantUsage, useCleanupTenantData } from '@/api/hooks/tenant';
import { toProtoTimestamp } from '@/utils/datetime';
import { fetchListPlans } from '@/api/hooks/plan';
import { PaginationQuery } from '@/core';
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

  // 编辑模式下加载租户用量与配额对比数据。
  const usageQuery = useGetTenantUsage(
    { id: data?.id ?? 0 },
    { enabled: open && mode === 'edit' && !!data?.id },
  );
  const cleanupMutation = useCleanupTenantData();

  // 套餐下拉数据
  const [planOptions, setPlanOptions] = useState<{ label: string; value: number }[]>([]);

  // 打开抽屉时加载套餐目录下拉
  useEffect(() => {
    if (open) {
      fetchListPlans(new PaginationQuery({ formValues: {} }))
        .then((res) => {
          const options = (res.items || []).map((item: any) => ({
            label: item.name,
            value: item.id,
          }));
          setPlanOptions(options);
        })
        .catch(() => setPlanOptions([]));
    }
  }, [open]);

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
            planId: data.planId,
            expiredAt: data.expiredAt,
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
            planId: values.planId,
            expiredAt: toProtoTimestamp(values.expiredAt),
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
            planId: values.planId,
            expiredAt: toProtoTimestamp(values.expiredAt),
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

      <ProFormSelect
        name="planId"
        label={t('subscriptionPlan')}
        placeholder={t('subscriptionPlanPlaceholder')}
        options={planOptions}
        fieldProps={SELECT_FILTER_PROPS}
      />

      <ProFormDateTimePicker
        name="expiredAt"
        label={t('expiredAt')}
        placeholder={t('expiredAtPlaceholder')}
        fieldProps={{ allowClear: true, showTime: true, style: { width: '100%' } }}
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

      {/* 编辑模式下显示用量与配额对比 + 清理数据按钮 */}
      {mode === 'edit' && usageQuery.data && (
        <>
          <Divider style={{ margin: '24px 0 16px' }}>{t('usageSection')}</Divider>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t('usageUserCount')}>
              {usageQuery.data.userCount ?? 0}
            </Descriptions.Item>
            <Descriptions.Item label={t('usageStorage')}>
              {usageQuery.data.storageUsedBytes ?? 0} bytes
            </Descriptions.Item>
            <Descriptions.Item label={t('usageApiCalls')}>
              {usageQuery.data.apiCallCount ?? 0}
            </Descriptions.Item>
            <Descriptions.Item label={t('usagePlan')}>
              {usageQuery.data.planName ?? t('usageNoPlan')}
            </Descriptions.Item>
          </Descriptions>
          {usageQuery.data.quotas && usageQuery.data.quotas.length > 0 && (
            <div style={{ marginTop: 16 }}>
              {usageQuery.data.quotas.map((q, idx) => {
                let label = q.quotaType?.toString() ?? '';
                let current = 0;
                let limit = q.quotaValue ?? 0;
                if (q.quotaType === 1) {
                  label = t('usageUserCount');
                  current = usageQuery.data.userCount ?? 0;
                } else if (q.quotaType === 2) {
                  label = t('usageStorage');
                  current = usageQuery.data.storageUsedBytes ?? 0;
                } else if (q.quotaType === 3) {
                  label = t('usageApiCalls');
                  current = usageQuery.data.apiCallCount ?? 0;
                }
                const pct = limit > 0 ? Math.min(100, (current / limit) * 100) : 0;
                return (
                  <div key={idx} style={{ marginBottom: 12 }}>
                    <div style={{ marginBottom: 4 }}>
                      {label}: {current} / {limit}
                    </div>
                    <Progress percent={pct} size="small" />
                  </div>
                );
              })}
            </div>
          )}
          <div style={{ marginTop: 24 }}>
            <Popconfirm
              title={t('cleanupConfirmTitle')}
              description={t('cleanupConfirmDesc')}
              okText={t('common:button.confirm')}
              okButtonProps={{ danger: true }}
              cancelText={t('common:button.cancel')}
              onConfirm={async () => {
                try {
                  await cleanupMutation.mutateAsync({ id: data?.id ?? 0 });
                  message.success(t('cleanupSuccess'));
                  queryClient.invalidateQueries({ queryKey: ['listTenants'] });
                  onClose();
                } catch (err: any) {
                  message.error(err?.message || t('cleanupFailed'));
                }
              }}
            >
              <Button danger loading={cleanupMutation.isPending}>
                {t('cleanupData')}
              </Button>
            </Popconfirm>
          </div>
        </>
      )}

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
