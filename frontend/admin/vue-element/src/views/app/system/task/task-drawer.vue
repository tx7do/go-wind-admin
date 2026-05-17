<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { enableBoolList, taskTypeList, useTaskStore } from '#/stores';

const taskStore = useTaskStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.task.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.task.moduleName') }),
);
// const isCreate = computed(() => data.value?.create);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  // 所有表单项共用，可单独在表单内覆盖
  commonConfig: {
    // 所有表单项
    componentProps: {
      class: 'w-full',
    },
  },
  schema: [
    {
      component: 'Select',
      fieldName: 'type',
      label: $t('page.task.type'),
      defaultValue: 'PERIODIC',
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: taskTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
      rules: 'selectRequired',
    },

    {
      component: 'ApiSelect',
      fieldName: 'typeName',
      label: $t('page.task.typeName'),
      rules: 'required',
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        api: async () => {
          const result = await taskStore.listTaskTypeName();
          return result.typeNames;
        },
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item,
            value: item,
          }));
        },
      },
    },

    {
      component: 'Textarea',
      fieldName: 'taskPayload',
      label: $t('page.task.taskPayload'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'Input',
      fieldName: 'cronSpec',
      label: $t('page.task.cronSpec'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      dependencies: {
        show: (values) => {
          return ['PERIODIC'].includes(values.type);
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'InputNumber',
      fieldName: 'taskOptions.maxRetry',
      label: $t('page.task.taskOptionsMaxRetry'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
        defaultValue: 3,
      },
    },

    {
      component: 'InputNumber',
      fieldName: 'taskOptions.timeout',
      label: $t('page.task.taskOptionsTimeout'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      dependencies: {
        show: (values) => {
          return ['DELAY', 'WAIT_RESULT'].includes(values.type);
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'DatePicker',
      fieldName: 'taskOptions.deadline',
      label: $t('page.task.taskOptionsDeadline'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
        showTime: true,
      },
      dependencies: {
        show: (values) => {
          return ['DELAY'].includes(values.type);
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'DatePicker',
      fieldName: 'taskOptions.processIn',
      label: $t('page.task.taskOptionsProcessIn'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
        showTime: true,
      },
      dependencies: {
        show: (values) => {
          return ['DELAY'].includes(values.type);
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'DatePicker',
      fieldName: 'taskOptions.processAt',
      label: $t('page.task.taskOptionsProcessAt'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
        showTime: true,
      },
      dependencies: {
        show: (values) => {
          return ['DELAY'].includes(values.type);
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'RadioGroup',
      fieldName: 'enable',
      defaultValue: true,
      label: $t('page.task.enable'),
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: enableBoolList,
      },
    },

    {
      component: 'Textarea',
      fieldName: 'remark',
      label: $t('ui.table.remark'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {
    console.log('onConfirm');

    // 校验输入的数据
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();

    console.log(getTitle.value, values);

    try {
      await (data.value?.create
        ? taskStore.createTask(values)
        : taskStore.updateTask(data.value.row.id, values));

      notification.success({
        message: data.value?.create
          ? $t('ui.notification.create_success')
          : $t('ui.notification.update_success'),
      });
    } catch {
      notification.error({
        message: data.value?.create
          ? $t('ui.notification.create_failed')
          : $t('ui.notification.update_failed'),
      });
    } finally {
      // 关闭窗口
      drawerApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      // 获取传入的数据
      data.value = drawerApi.getData<Record<string, any>>();

      // 为表单赋值
      baseFormApi.setValues(data.value?.row);

      setLoading(false);

      console.log('onOpenChange', data.value, data.value?.create);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
