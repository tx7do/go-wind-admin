import React, { useEffect, useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormSelect,
  ProFormRadio,
  ProFormDigit,
  ProFormTreeSelect,
  ProFormCheckbox,
  ProFormDependency,
} from '@ant-design/pro-components';
import { Divider, App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { permissionservicev1_Menu as Menu } from '@/api/generated/admin/service/v1';
import { useCreateMenu, useUpdateMenu, fetchListMenus } from '@/api/hooks/menu';
import { PaginationQuery } from '@/core';
import {
  getMenuTypeOptions,
  getStatusOptions,
  isButton,
  isCatalog,
  isEmbedded,
  isMenu,
  normalizeAuthority,
} from '../constants';

interface MenuDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: Menu;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 菜单编辑/创建抽屉组件
 */
const MenuDrawer: React.FC<MenuDrawerProps> = ({ open, mode, data, onClose, onSuccess }) => {
  const { t } = useTranslation('menu');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 创建菜单
  const createMutation = useCreateMenu({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listMenus'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新菜单
  const updateMutation = useUpdateMenu({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listMenus'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  // 编辑模式时填充表单
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      // 将 meta 内部字段展开到表单顶级
      const meta = (data as any).meta || {};
      const formValues: Record<string, any> = {
        type: data.type || 'MENU',
        status: data.status || 'ON',
        path: data.path || '',
        component: data.component || '',
        parentId: data.parentId || undefined,
        'meta.title': meta.title || '',
        'meta.order': meta.order || 0,
        'meta.icon': meta.icon || '',
        'meta.authority': normalizeAuthority(meta.authority),
        'meta.keepAlive': meta.keepAlive || false,
        'meta.affixTab': meta.affixTab || false,
        'meta.hideInMenu': meta.hideInMenu || false,
        'meta.hideChildrenInMenu': meta.hideChildrenInMenu || false,
        'meta.hideInBreadcrumb': meta.hideInBreadcrumb || false,
        'meta.hideInTab': meta.hideInTab || false,
      };
      setTimeout(() => {
        formRef.current?.setFieldsValue(formValues);
      }, 0);
    }
  }, [open, mode, data]);

  // 构建菜单树（用于父级菜单选择）
  const buildMenuTree = (items: any[], parentId: number | string | undefined): any[] => {
    return items
      .filter((item) => item.parentId === parentId)
      .map((item) => ({
        title: item.meta?.title || item.path || '',
        value: item.id,
        key: item.id,
        children: buildMenuTree(items, item.id),
      }));
  };

  // 加载菜单树数据
  const fetchMenuTree = async () => {
    try {
      const query = new PaginationQuery({
        paging: { page: 1, pageSize: 1000 },
        formValues: { status: 'ON' },
      });
      const response = await fetchListMenus(query);
      const items = response.items || [];
      return buildMenuTree(items, undefined);
    } catch {
      return [];
    }
  };

  // 提交表单
  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);

    try {
      // 将 meta 字段重新聚合
      const payload: Record<string, any> = {
        type: values.type,
        status: values.status,
        path: values.path,
        component: values.component,
        parentId: values.parentId || 0,
        children: [],
        meta: {
          title: values['meta.title'],
          order: values['meta.order'] || 0,
          icon: values['meta.icon'] || '',
          authority: values['meta.authority'] || [],
          keepAlive: values['meta.keepAlive'] || false,
          affixTab: values['meta.affixTab'] || false,
          hideInMenu: values['meta.hideInMenu'] || false,
          hideChildrenInMenu: values['meta.hideChildrenInMenu'] || false,
          hideInBreadcrumb: values['meta.hideInBreadcrumb'] || false,
          hideInTab: values['meta.hideInTab'] || false,
        },
      };

      if (mode === 'create') {
        await createMutation.mutateAsync({ data: payload as any });
      } else if (data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values: payload });
      }
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
      drawerProps={{
        destroyOnClose: true,
        onClose,
        size: 720,
      }}
      initialValues={{
        type: 'MENU',
        status: 'ON',
        component: 'BasicLayout',
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
        resetButtonProps: {
          onClick: onClose,
        },
      }}
    >
      {/* 菜单类型 */}
      <ProFormRadio.Group
        name="type"
        label={t('type')}
        options={getMenuTypeOptions(t)}
        fieldProps={{
          optionType: 'button',
          buttonStyle: 'solid',
        }}
      />

      {/* 菜单标题（i18n key） */}
      <ProFormText
        name="meta.title"
        label={t('menuTitle')}
        placeholder={t('menuTitlePlaceholder')}
        rules={[{ required: true, message: t('requiredTitle') }]}
        fieldProps={{
          allowClear: true,
        }}
      />

      {/* 上级菜单 */}
      <ProFormTreeSelect
        name="parentId"
        label={t('parentId')}
        placeholder={t('parentIdPlaceholder')}
        request={fetchMenuTree}
        fieldProps={{
          showSearch: true,
          treeDefaultExpandAll: true,
          allowClear: true,
          treeNodeFilterProp: 'title',
          placeholder: t('parentIdPlaceholder'),
        }}
      />

      {/* 排序 */}
      <ProFormDigit
        name="meta.order"
        label={t('order')}
        placeholder={t('orderPlaceholder')}
        fieldProps={{
          precision: 0,
          min: 0,
        }}
      />

      {/* 图标 - 仅非按钮类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (isButton(type)) return null;
          return (
            <ProFormText
              name="meta.icon"
              label={t('icon')}
              placeholder={t('iconPlaceholder')}
              fieldProps={{
                allowClear: true,
              }}
            />
          );
        }}
      </ProFormDependency>

      {/* 路由路径 - 仅非按钮类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (isButton(type)) return null;
          return (
            <ProFormText
              name="path"
              label={t('path')}
              placeholder={t('pathPlaceholder')}
              rules={[{ required: true, message: t('requiredPath') }]}
              fieldProps={{
                allowClear: true,
              }}
            />
          );
        }}
      </ProFormDependency>

      {/* 组件路径 - 仅菜单类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (!isMenu(type)) return null;
          return (
            <ProFormText
              name="component"
              label={t('component')}
              placeholder={t('componentPlaceholder')}
              rules={[{ required: true, message: t('requiredComponent') }]}
              fieldProps={{
                allowClear: true,
              }}
            />
          );
        }}
      </ProFormDependency>

      {/* 权限标识 - 非目录类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (isCatalog(type)) return null;
          return (
            <ProFormSelect
              name="meta.authority"
              label={t('authority')}
              placeholder={t('authorityPlaceholder')}
              mode="tags"
              fieldProps={{
                tokenSeparators: [','],
              }}
            />
          );
        }}
      </ProFormDependency>

      {/* 状态 */}
      <ProFormRadio.Group
        name="status"
        label={t('status')}
        options={getStatusOptions(t)}
        rules={[{ required: true }]}
        fieldProps={{
          optionType: 'button',
          buttonStyle: 'solid',
        }}
      />

      {/* 高级设置分割线 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (isButton(type) || type === 'LINK') return null;
          return <Divider titlePlacement="left">{t('advancedSettings')}</Divider>;
        }}
      </ProFormDependency>

      {/* KeepAlive - 仅菜单类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (!isMenu(type)) return null;
          return (
            <ProFormCheckbox name="meta.keepAlive">
              {t('keepAlive')}
            </ProFormCheckbox>
          );
        }}
      </ProFormDependency>

      {/* 固定标签页 - 菜单和内嵌类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (!isMenu(type) && !isEmbedded(type)) return null;
          return (
            <ProFormCheckbox name="meta.affixTab">
              {t('affixTab')}
            </ProFormCheckbox>
          );
        }}
      </ProFormDependency>

      {/* 隐藏菜单 - 非按钮类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (isButton(type)) return null;
          return (
            <ProFormCheckbox name="meta.hideInMenu">
              {t('hideInMenu')}
            </ProFormCheckbox>
          );
        }}
      </ProFormDependency>

      {/* 隐藏子菜单 - 目录和菜单类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (!isCatalog(type) && !isMenu(type)) return null;
          return (
            <ProFormCheckbox name="meta.hideChildrenInMenu">
              {t('hideChildrenInMenu')}
            </ProFormCheckbox>
          );
        }}
      </ProFormDependency>

      {/* 隐藏面包屑 - 非按钮/链接类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (isButton(type) || type === 'LINK') return null;
          return (
            <ProFormCheckbox name="meta.hideInBreadcrumb">
              {t('hideInBreadcrumb')}
            </ProFormCheckbox>
          );
        }}
      </ProFormDependency>

      {/* 隐藏标签页 - 非按钮/链接类型 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (isButton(type) || type === 'LINK') return null;
          return (
            <ProFormCheckbox name="meta.hideInTab">
              {t('hideInTab')}
            </ProFormCheckbox>
          );
        }}
      </ProFormDependency>
    </DrawerForm>
  );
};

export default MenuDrawer;
