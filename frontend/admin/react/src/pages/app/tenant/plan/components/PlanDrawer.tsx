import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormSelect,
  ProFormDigit,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import {
  useCreatePlan,
  useUpdatePlan,
  useListPlanModules,
  useCreatePlanModule,
  useDeletePlanModule,
} from '@/api/hooks/plan';
import { PaginationQuery } from '@go-wind/admin-react';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';
import { getPlanVersionOptions, getExpiryPolicyOptions, getModuleOptions } from '../constants';
import { SELECT_FILTER_PROPS } from '../constants';

// 将后端 PlanModule 列表映射为前端多选值（字符串模块名）。
// 后端 entity 的 module 字段为字符串枚举值，与 MODULE_LABEL_KEYS 中的字符串一致。
function extractModuleNames(items: any[] | undefined): string[] {
  const names: string[] = [];
  if (items) {
    for (const item of items) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const mod = (item as any)?.module;
      if (typeof mod === 'string' && mod.length > 0) {
        names.push(mod);
      }
    }
  }
  return names;
}

interface PlanDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: any;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 套餐目录编辑/创建抽屉组件
 */
const PlanDrawer: React.FC<PlanDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('plan');
  const formRef = useRef<ProFormInstance>(null);

  // 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
  const { guardClose, handleValuesChange } = useDrawerCloseGuard(open, onClose);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 编辑模式下加载套餐的模块白名单。
  // 通过 plan_id 过滤查询 sys_plan_modules，提取 module 字段填充多选框。
  const moduleIdQuery = new PaginationQuery({
    paging: { page: 1, pageSize: 999 },
    formValues: mode === 'edit' && data?.id ? { plan_id: data.id } : {},
  });
  const { data: planModuleData } = useListPlanModules(moduleIdQuery, {
    enabled: open && mode === 'edit' && !!data?.id,
  });
  const existingModules = extractModuleNames(planModuleData?.items);

  // 编辑模式下设置表单值（destroyOnHidden 时需延迟赋值）
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          name: data.name || '',
          version: data.version,
          expiryPolicy: data.expiryPolicy,
          dataRetentionDays: data.dataRetentionDays,
          description: data.description || '',
          remark: data.remark || '',
          moduleWhitelist: existingModules,
        });
      }, 0);
    }
  }, [open, mode, data, existingModules]);

  // 创建 mutation
  const createMutation = useCreatePlan({
    onSuccess: () => {
      message.success(t('createSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listPlans'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新 mutation
  const updateMutation = useUpdatePlan({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listPlans'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  // 模块白名单 CRUD mutations
  const createModuleMutation = useCreatePlanModule({
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['listPlanModules'] });
    },
  });
  const deleteModuleMutation = useDeletePlanModule({
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['listPlanModules'] });
    },
  });

  // 提交表单
  const handleSubmit = async (values: Record<string, any>) => {
    try {
      setConfirmLoading(true);

      // 剥离模块白名单字段，不传入套餐主体 CRUD。
      const { moduleWhitelist, ...planValues } = values;
      const selectedModules: string[] = Array.isArray(moduleWhitelist) ? moduleWhitelist : [];

      if (mode === 'edit' && data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values: planValues });
      } else {
        await createMutation.mutateAsync({ data: planValues });
      }

      // 同步模块白名单：计算 diff（新增、删除），逐一调用 create/delete。
      // 新套餐创建后 data.id 不可用（create 不回传 id），此处仅对编辑模式同步。
      if (mode === 'edit' && data?.id) {
        const toAdd = selectedModules.filter((m) => !existingModules.includes(m));
        const toRemove = existingModules.filter((m) => !selectedModules.includes(m));

        for (const mod of toAdd) {
          await createModuleMutation.mutateAsync({
            data: { planId: data.id, module: mod as any },
          } as any);
        }
        for (const mod of toRemove) {
          // 查找该模块对应的 plan_module 记录 ID 以执行删除。
          const item = planModuleData?.items?.find(
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (it: any) => it?.module === mod,
          );
          if (item?.id) {
            await deleteModuleMutation.mutateAsync({ id: item.id } as any);
          }
        }
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
      onValuesChange={handleValuesChange}
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading || createMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: { onClick: guardClose },
      }}
      drawerProps={{ destroyOnHidden: true, onClose: guardClose, size: 600 }}
      layout="horizontal"
      labelCol={{ flex: '0 0 120px' }}
      wrapperCol={{ flex: 'auto' }}
    >
      <ProFormText
        name="name"
        label={t('name')}
        placeholder={t('namePlaceholder')}
        rules={[{ required: true, message: t('requiredName') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormSelect
        name="version"
        label={t('versionLabel')}
        placeholder={t('versionPlaceholder')}
        options={getPlanVersionOptions(t)}
        rules={[{ required: true, message: t('requiredVersion') }]}
        fieldProps={SELECT_FILTER_PROPS}
      />

      <ProFormSelect
        name="expiryPolicy"
        label={t('expiryPolicyLabel')}
        placeholder={t('expiryPolicyPlaceholder')}
        options={getExpiryPolicyOptions(t)}
        rules={[{ required: true, message: t('requiredExpiryPolicy') }]}
        fieldProps={SELECT_FILTER_PROPS}
      />

      <ProFormSelect
        name="moduleWhitelist"
        label={t('moduleWhitelist')}
        placeholder={t('moduleWhitelistPlaceholder')}
        options={getModuleOptions(t)}
        fieldProps={{ ...SELECT_FILTER_PROPS, mode: 'multiple' }}
      />

      <ProFormDigit
        name="dataRetentionDays"
        label={t('dataRetentionDays')}
        placeholder={t('dataRetentionDaysPlaceholder')}
        fieldProps={{ precision: 0, min: 0 }}
      />

      <ProFormTextArea
        name="description"
        label={t('description')}
        placeholder={t('descriptionPlaceholder')}
        fieldProps={{ allowClear: true, autoSize: { minRows: 2, maxRows: 4 } }}
      />

      <ProFormTextArea
        name="remark"
        label={t('remark')}
        placeholder={t('remarkPlaceholder')}
        fieldProps={{ allowClear: true, autoSize: { minRows: 2, maxRows: 4 } }}
      />
    </DrawerForm>
  );
};

export default PlanDrawer;
