import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import { DrawerForm, ProFormText, ProFormRadio, ProFormTextArea } from '@ant-design/pro-components';
import { App, TreeSelect, Select, Form } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { useCreateUser, useUpdateUser } from '@/api/hooks/user';
import { fetchListRoles } from '@/api/hooks/role';
import { fetchListOrgUnits } from '@/api/hooks/org-unit';
import { fetchListPositions } from '@/api/hooks/position';
import { getUserStatusOptions, getGenderOptions } from '../constants';
import { buildOrgTreeSelectData } from './tree-utils';

interface UserDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: any;
  tenantId: number | undefined;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 用户创建/编辑抽屉组件
 */
const UserDrawer: React.FC<UserDrawerProps> = ({
  open,
  mode,
  data,
  tenantId,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('user');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 下拉数据
  const [roleTreeData, setRoleTreeData] = useState<any[]>([]);
  const [orgUnitTreeData, setOrgUnitTreeData] = useState<any[]>([]);
  const [positionOptions, setPositionOptions] = useState<{ label: string; value: number }[]>([]);

  // 加载下拉数据
  useEffect(() => {
    if (open) {
      // 角色
      fetchListRoles(
        new PaginationQuery({
          formValues: { status: 'ON', type__not: 'TEMPLATE', tenant_id: tenantId ?? 0 },
        }),
      )
        .then((res) => {
          const tree = buildRoleTreeSelect(res?.items || []);
          setRoleTreeData(tree);
        })
        .catch(() => setRoleTreeData([]));

      // 组织
      fetchListOrgUnits(
        new PaginationQuery({ formValues: { status: 'ON', tenant_id: tenantId ?? 0 } }),
      )
        .then((res) => {
          const tree = buildOrgTreeSelectData(res?.items || []);
          setOrgUnitTreeData(tree);
        })
        .catch(() => setOrgUnitTreeData([]));

      // 职位
      fetchListPositions(new PaginationQuery({ formValues: { status: 'ON' } }))
        .then((res) => {
          const opts = (res.items || []).map((item: any) => ({
            label: item.name || '',
            value: item.id,
          }));
          setPositionOptions(opts);
        })
        .catch(() => setPositionOptions([]));
    }
  }, [open, tenantId]);

  // 编辑模式下设置表单值
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          username: data.username || '',
          password: undefined,
          nickname: data.nickname || '',
          realname: data.realname || '',
          email: data.email || '',
          mobile: data.mobile || '',
          gender: data.gender || 'SECRET',
          status: data.status || 'NORMAL',
          roleIds: data.roleIds || [],
          orgUnitIds: data.orgUnitIds || [],
          positionIds: data.positionIds || [],
          remark: data.remark || '',
        });
      }, 0);
    }
  }, [open, mode, data]);

  // 创建 mutation
  const createMutation = useCreateUser({
    onSuccess: () => {
      message.success(t('createSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listUsers'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新 mutation
  const updateMutation = useUpdateUser({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listUsers'] });
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
        // 更新：不发送 password（除非有值）
        const updateValues = { ...values };
        if (!updateValues.password) {
          delete updateValues.password;
        }
        await updateMutation.mutateAsync({ id: data.id, values: updateValues });
      } else {
        // 创建：需要 data + password 包装
        const { password, ...restValues } = values;
        await createMutation.mutateAsync({
          data: restValues as any,
          password: password || undefined,
        });
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
        gender: 'SECRET',
        status: 'NORMAL',
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
      drawerProps={{ destroyOnClose: true, onClose, size: 600 }}
    >
      <ProFormText
        name="username"
        label={t('username')}
        placeholder={t('usernamePlaceholder')}
        rules={[{ required: true, message: t('requiredUsername') }]}
        fieldProps={{
          allowClear: true,
          disabled: mode === 'edit',
        }}
      />

      <ProFormText.Password
        name="password"
        label={t('password')}
        placeholder={t('passwordPlaceholder')}
        fieldProps={{ allowClear: true }}
      />

      {/* 角色 - TreeSelect 多选 */}
      <Form.Item name="roleIds" label={t('roleIds')}>
        <TreeSelect
          treeData={roleTreeData}
          placeholder={t('roleIdsPlaceholder')}
          allowClear
          showSearch
          multiple
          treeDefaultExpandAll={false}
          style={{ width: '100%' }}
        />
      </Form.Item>

      {/* 组织 - TreeSelect 多选 */}
      <Form.Item name="orgUnitIds" label={t('orgUnitIds')}>
        <TreeSelect
          treeData={orgUnitTreeData}
          placeholder={t('orgUnitIdsPlaceholder')}
          allowClear
          showSearch
          multiple
          treeDefaultExpandAll
          style={{ width: '100%' }}
        />
      </Form.Item>

      {/* 职位 - Select 多选 */}
      <Form.Item name="positionIds" label={t('positionIds')}>
        <Select
          mode="multiple"
          options={positionOptions}
          placeholder={t('positionIdsPlaceholder')}
          allowClear
          showSearch
          style={{ width: '100%' }}
        />
      </Form.Item>

      {/* 性别 */}
      <ProFormRadio.Group
        name="gender"
        label={t('gender')}
        options={getGenderOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />

      <ProFormText
        name="nickname"
        label={t('nickname')}
        placeholder={t('nicknamePlaceholder')}
        rules={[{ required: true, message: t('requiredNickname') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="realname"
        label={t('realname')}
        placeholder={t('realnamePlaceholder')}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="email"
        label={t('email')}
        placeholder={t('emailPlaceholder')}
        rules={[{ required: true, message: t('requiredEmail') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="mobile"
        label={t('mobile')}
        placeholder={t('mobilePlaceholder')}
        fieldProps={{ allowClear: true }}
      />

      {/* 状态 */}
      <ProFormRadio.Group
        name="status"
        label={t('status')}
        rules={[{ required: true, message: t('requiredStatus') }]}
        options={getUserStatusOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />

      <ProFormTextArea
        name="remark"
        label={t('remark')}
        placeholder={t('remarkPlaceholder')}
        fieldProps={{ allowClear: true, rows: 3 }}
      />
    </DrawerForm>
  );
};

export default UserDrawer;

// ========== 工具函数 ==========

/**
 * 将角色扁平列表构建为 TreeSelect 数据
 */
function buildRoleTreeSelect(items: any[]): any[] {
  if (!items || items.length === 0) return [];

  const map = new Map<number, any>();
  const roots: any[] = [];

  items.forEach((item) => {
    if (item.id == null) return;
    map.set(item.id, {
      value: item.id,
      label: item.name || '',
      title: item.name || '',
      children: [],
      parentId: item.parentId,
    });
  });

  map.forEach((node) => {
    if (node.parentId && map.has(node.parentId)) {
      map.get(node.parentId)!.children.push(node);
    } else {
      roots.push(node);
    }
  });

  // 清理空 children
  const cleanEmpty = (nodes: any[]) => {
    nodes.forEach((n) => {
      if (n.children && n.children.length === 0) {
        delete n.children;
      } else if (n.children) {
        cleanEmpty(n.children);
      }
    });
  };
  cleanEmpty(roots);

  return roots;
}
