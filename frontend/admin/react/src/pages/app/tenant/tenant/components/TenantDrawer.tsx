import React, { useEffect, useRef } from 'react';
import { DrawerForm, ProFormText, ProFormSelect, ProFormTextArea } from '@ant-design/pro-components';
import type { ProFormInstance } from '@ant-design/pro-components';
import { Button, message, Divider } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { createTenantWithAdminUser, updateTenant } from '@/api/service/tenant';
import type {
  identityservicev1_Tenant,
  identityservicev1_Tenant_Type,
  identityservicev1_Tenant_Status,
  identityservicev1_Tenant_AuditStatus,
} from '@/api/generated/admin/service/v1';

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
  const queryClient = useQueryClient();
  const formRef = useRef<ProFormInstance>(null);

  // 当 Drawer 打开时，设置表单初始值
  useEffect(() => {
    if (open && formRef.current) {
      if (mode === 'edit' && data) {
        formRef.current.setFieldsValue({
          name: data.name,
          code: data.code,
          tenantType: data.type,
          auditStatus: data.auditStatus,
          status: data.status,
          remark: data.remark,
        });
      } else {
        // 创建模式下的默认值
        formRef.current.setFieldsValue({
          tenantType: 'PAID',
          auditStatus: 'APPROVED',
          status: 'ON',
        });
      }
    }
  }, [open, mode, data]);

  // 提交处理
  const handleSubmit = async (values: any) => {
    try {
      if (mode === 'create') {
        // 检查密码和确认密码是否一致
        if (values.password !== values.passwordConfirm) {
          message.error('两次输入的密码不一致');
          return false;
        }

        // 创建租户及管理员用户
        await createTenantWithAdminUser({
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
        message.success('创建成功');
      } else {
        // 更新租户
        if (!data?.id) {
          message.error('租户 ID 不存在');
          return false;
        }
        await updateTenant({
          id: data.id,
          data: {
            name: values.name,
            code: values.code,
            type: values.tenantType as identityservicev1_Tenant_Type,
            auditStatus: values.auditStatus as identityservicev1_Tenant_AuditStatus,
            status: values.status as identityservicev1_Tenant_Status,
            remark: values.remark,
          },
          updateMask: {
            paths: ['name', 'code', 'type', 'auditStatus', 'status', 'remark'],
          },
        });
        message.success('更新成功');
      }

      // 刷新列表
      queryClient.invalidateQueries({ queryKey: ['listTenants'] });

      // 重置表单并关闭
      formRef.current?.resetFields();
      onSuccess();

      return true;
    } catch (error: any) {
      message.error(error.message || (mode === 'create' ? '创建失败' : '更新失败'));
      return false;
    }
  };

  return (
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? '新建租户' : '编辑租户'}
      open={open}
      onOpenChange={(visible) => {
        if (!visible) {
          formRef.current?.resetFields();
          onClose();
        }
      }}
      width={600}
      initialValues={{
        tenantType: 'PAID',
        auditStatus: 'APPROVED',
        status: 'ON',
      }}
      onFinish={handleSubmit}
      submitter={{
        render: (_, dom) => [
          <Button key="reset" onClick={() => formRef.current?.resetFields()}>
            重置
          </Button>,
          ...dom,
        ],
      }}
    >
      <ProFormText
        name="name"
        label="租户名称"
        placeholder="请输入租户名称"
        rules={[
          { required: true, message: '请输入租户名称' },
          { max: 50, message: '最多 50 个字符' },
        ]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormText
        name="code"
        label="租户编码"
        placeholder="请输入租户编码"
        rules={[
          { required: true, message: '请输入租户编码' },
          { max: 50, message: '最多 50 个字符' },
        ]}
        fieldProps={{
          allowClear: true,
        }}
      />

      <ProFormSelect
        name="tenantType"
        label="租户类型"
        placeholder="请选择租户类型"
        rules={[{ required: true, message: '请选择租户类型' }]}
        options={[
          { label: '试用', value: 'TRIAL' },
          { label: '付费', value: 'PAID' },
          { label: '内部', value: 'INTERNAL' },
          { label: '合作伙伴', value: 'PARTNER' },
          { label: '自定义', value: 'CUSTOM' },
        ]}
        fieldProps={{
          showSearch: true,
          allowClear: true,
          filterOption: (input, option) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormSelect
        name="auditStatus"
        label="审核状态"
        placeholder="请选择审核状态"
        rules={[{ required: true, message: '请选择审核状态' }]}
        options={[
          { label: '待审核', value: 'PENDING' },
          { label: '已通过', value: 'APPROVED' },
          { label: '已拒绝', value: 'REJECTED' },
        ]}
        fieldProps={{
          showSearch: true,
          allowClear: true,
          filterOption: (input, option) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormSelect
        name="status"
        label="状态"
        placeholder="请选择状态"
        rules={[{ required: true, message: '请选择状态' }]}
        options={[
          { label: '启用', value: 'ON' },
          { label: '禁用', value: 'OFF' },
          { label: '已过期', value: 'EXPIRED' },
          { label: '已冻结', value: 'FREEZE' },
        ]}
        fieldProps={{
          showSearch: true,
          allowClear: true,
          filterOption: (input, option) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormTextArea
        name="remark"
        label="备注"
        placeholder="请输入备注"
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
          <Divider style={{ margin: '24px 0 16px' }}>管理员账号配置</Divider>

          <ProFormText
            name="username"
            label="管理员用户名"
            placeholder="请输入管理员用户名"
            rules={[
              { required: true, message: '请输入管理员用户名' },
              { max: 50, message: '最多 50 个字符' },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />

          <ProFormText.Password
            name="password"
            label="初始密码"
            placeholder="请输入初始密码"
            rules={[
              { required: true, message: '请输入初始密码' },
              { min: 6, message: '密码至少 6 个字符' },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />

          <ProFormText.Password
            name="passwordConfirm"
            label="确认密码"
            placeholder="请再次输入密码"
            rules={[
              { required: true, message: '请再次输入密码' },
              { min: 6, message: '密码至少 6 个字符' },
              {
                validator: async (_, value) => {
                  const password = formRef.current?.getFieldValue('password');
                  if (value && password !== value) {
                    return Promise.reject(new Error('两次输入的密码不一致'));
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
            label="手机号"
            placeholder="请输入手机号"
            rules={[
              { required: true, message: '请输入手机号' },
              { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号' },
            ]}
            fieldProps={{
              allowClear: true,
            }}
          />

          <ProFormText
            name="email"
            label="电子邮箱"
            placeholder="请输入电子邮箱"
            rules={[
              { required: true, message: '请输入电子邮箱' },
              { type: 'email', message: '请输入正确的邮箱地址' },
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
