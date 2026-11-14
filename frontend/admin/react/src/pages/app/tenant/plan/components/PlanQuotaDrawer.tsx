import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormSelect,
  ProFormDigit,
} from '@ant-design/pro-components';
import { App, Select, Form } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@go-wind/admin-react';
import { useCreatePlanQuota, useUpdatePlanQuota, fetchListPlans } from '@/api/hooks/plan';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';
import { getQuotaTypeOptions } from '../constants';
import { SELECT_FILTER_PROPS } from '../constants';

interface PlanQuotaDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: any;
  planId: number;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 套餐配额编辑/创建抽屉组件
 */
const PlanQuotaDrawer: React.FC<PlanQuotaDrawerProps> = ({
  open,
  mode,
  data,
  planId,
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

  // 套餐下拉数据
  const [planOptions, setPlanOptions] = useState<{ label: string; value: number }[]>([]);

  // 加载套餐下拉数据
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

  // 编辑模式下设置表单值
  useEffect(() => {
    if (!(open && mode === 'edit' && data)) return;
    let cancelled = false;
    let attempts = 0;
    const applyValues = () => {
      if (cancelled) return;
      const form = formRef.current;
      if (form) {
        form.setFieldsValue({
          planId: data.planId || planId,
          quotaType: data.quotaType,
          quotaValue: data.quotaValue,
        });
        return;
      }
      if (attempts++ < 30) {
        requestAnimationFrame(applyValues);
      }
    };
    requestAnimationFrame(applyValues);
    return () => {
      cancelled = true;
    };
  }, [open, mode, data, planId]);

  // 创建 mutation
  const createMutation = useCreatePlanQuota({
    onSuccess: () => {
      message.success(t('createSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listPlanQuotas'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新 mutation
  const updateMutation = useUpdatePlanQuota({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listPlanQuotas'] });
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
        await updateMutation.mutateAsync({ id: data.id, values });
      } else {
        await createMutation.mutateAsync({ data: values });
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
      title={mode === 'create' ? t('quotaCreate') : t('quotaEdit')}
      open={open}
      onValuesChange={handleValuesChange}
      initialValues={{
        planId,
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
        resetButtonProps: { onClick: guardClose },
      }}
      drawerProps={{ destroyOnHidden: true, onClose: guardClose, size: 600 }}
      layout="horizontal"
      labelCol={{ flex: '0 0 120px' }}
      wrapperCol={{ flex: 'auto' }}
    >
      {/* 所属套餐 - 只读 Select */}
      <Form.Item
        name="planId"
        label={t('planId')}
        rules={[{ required: true, message: t('requiredPlanId') }]}
      >
        <Select
          options={planOptions}
          placeholder={t('planIdPlaceholder')}
          disabled
        />
      </Form.Item>

      <ProFormSelect
        name="quotaType"
        label={t('quotaTypeLabel')}
        placeholder={t('quotaTypePlaceholder')}
        options={getQuotaTypeOptions(t)}
        rules={[{ required: true, message: t('requiredQuotaType') }]}
        fieldProps={SELECT_FILTER_PROPS}
      />

      <ProFormDigit
        name="quotaValue"
        label={t('quotaValue')}
        placeholder={t('quotaValuePlaceholder')}
        fieldProps={{ precision: 0, min: 0 }}
      />
    </DrawerForm>
  );
};

export default PlanQuotaDrawer;
