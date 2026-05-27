import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormSelect,
  ProFormTextArea,
  ProFormDigit,
  ProFormDateTimePicker,
  ProFormRadio,
  ProFormDependency,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { taskservicev1_Task as Task } from '@/api/generated/admin/service/v1';
import { useCreateTask, useUpdateTask, fetchListTaskTypeNames } from '@/api/hooks/task';
import { getTaskTypeOptions } from '../constants';

interface TaskDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: Task;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 任务编辑/创建抽屉组件
 */
const TaskDrawer: React.FC<TaskDrawerProps> = ({ open, mode, data, onClose, onSuccess }) => {
  const { t } = useTranslation('task');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);
  const [typeNameOptions, setTypeNameOptions] = useState<{ label: string; value: string }[]>([]);

  // 加载类型名称列表
  useEffect(() => {
    if (open) {
      fetchListTaskTypeNames()
        .then((res) => {
          const names = res?.typeNames || [];
          setTypeNameOptions(names.map((n: string) => ({ label: n, value: n })));
        })
        .catch(() => setTypeNameOptions([]));
    }
  }, [open]);

  // 编辑模式填充表单
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      const taskOptions = (data as any).taskOptions || {};
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          type: data.type || 'PERIODIC',
          typeName: data.typeName || '',
          taskPayload: data.taskPayload || '',
          cronSpec: data.cronSpec || '',
          enable: data.enable ?? true,
          remark: data.remark || '',
          'taskOptions.maxRetry': taskOptions.maxRetry ?? 3,
          'taskOptions.timeout': taskOptions.timeout,
          'taskOptions.deadline': taskOptions.deadline,
          'taskOptions.processIn': taskOptions.processIn,
          'taskOptions.processAt': taskOptions.processAt,
        });
      }, 0);
    }
  }, [open, mode, data]);

  const createMutation = useCreateTask({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listTasks'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  const updateMutation = useUpdateTask({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listTasks'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {
      // 聚合 taskOptions 嵌套字段
      const payload = {
        type: values.type,
        typeName: values.typeName,
        taskPayload: values.taskPayload || '',
        cronSpec: values.cronSpec || '',
        enable: values.enable ?? true,
        remark: values.remark || '',
        taskOptions: {
          maxRetry: values['taskOptions.maxRetry'] ?? 3,
          timeout: values['taskOptions.timeout'],
          deadline: values['taskOptions.deadline'],
          processIn: values['taskOptions.processIn'],
          processAt: values['taskOptions.processAt'],
        },
      };

      if (mode === 'create') {
        await createMutation.mutateAsync({ data: payload });
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
      initialValues={{
        type: 'PERIODIC',
        enable: true,
        'taskOptions.maxRetry': 3,
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
      <ProFormSelect
        name="type"
        label={t('type')}
        placeholder={t('typePlaceholder')}
        options={getTaskTypeOptions(t)}
        rules={[{ required: true, message: t('requiredType') }]}
        fieldProps={{
          showSearch: true,
          filterOption: (input: string, option: any) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormSelect
        name="typeName"
        label={t('typeName')}
        placeholder={t('typeNamePlaceholder')}
        options={typeNameOptions}
        rules={[{ required: true, message: t('requiredTypeName') }]}
        fieldProps={{
          showSearch: true,
          filterOption: (input: string, option: any) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />

      <ProFormTextArea
        name="taskPayload"
        label={t('taskPayload')}
        placeholder={t('taskPayloadPlaceholder')}
        fieldProps={{ allowClear: true, rows: 3 }}
      />

      {/* Cron 表达式 - 仅周期任务 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (type !== 'PERIODIC') return null;
          return (
            <ProFormText
              name="cronSpec"
              label={t('cronSpec')}
              placeholder={t('cronSpecPlaceholder')}
              fieldProps={{ allowClear: true }}
            />
          );
        }}
      </ProFormDependency>

      <ProFormDigit
        name="taskOptions.maxRetry"
        label={t('taskOptionsMaxRetry')}
        fieldProps={{ precision: 0, min: 0 }}
      />

      {/* 超时时间 - 延迟/等待结果任务 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (!['DELAY', 'WAIT_RESULT'].includes(type)) return null;
          return (
            <ProFormDigit
              name="taskOptions.timeout"
              label={t('taskOptionsTimeout')}
              fieldProps={{ precision: 0, min: 0 }}
            />
          );
        }}
      </ProFormDependency>

      {/* 以下字段仅延迟任务 */}
      <ProFormDependency name={['type']}>
        {({ type }) => {
          if (type !== 'DELAY') return null;
          return (
            <>
              <ProFormDateTimePicker
                name="taskOptions.deadline"
                label={t('taskOptionsDeadline')}
                fieldProps={{ allowClear: true, showTime: true, style: { width: '100%' } }}
              />
              <ProFormDateTimePicker
                name="taskOptions.processIn"
                label={t('taskOptionsProcessIn')}
                fieldProps={{ allowClear: true, showTime: true, style: { width: '100%' } }}
              />
              <ProFormDateTimePicker
                name="taskOptions.processAt"
                label={t('taskOptionsProcessAt')}
                fieldProps={{ allowClear: true, showTime: true, style: { width: '100%' } }}
              />
            </>
          );
        }}
      </ProFormDependency>

      <ProFormRadio.Group
        name="enable"
        label={t('enable')}
        rules={[{ required: true }]}
        options={[
          { label: t('enableMap.true'), value: true },
          { label: t('enableMap.false'), value: false },
        ]}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />

      <ProFormTextArea
        name="remark"
        label={t('remark')}
        placeholder={t('remarkPlaceholder')}
        fieldProps={{ allowClear: true, rows: 2 }}
      />
    </DrawerForm>
  );
};

export default TaskDrawer;
