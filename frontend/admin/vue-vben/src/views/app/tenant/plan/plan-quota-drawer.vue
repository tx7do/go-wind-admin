<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  fetchListPlans,
  PaginationQuery,
  planQuotaTypeList,
  useCreatePlanQuota,
  useUpdatePlanQuota,
} from '#/api';
import { usePlanViewStore } from '#/views/app/tenant/plan/plan-view.state';

const { mutateAsync: createPlanQuota } = useCreatePlanQuota();
const { mutateAsync: updatePlanQuota } = useUpdatePlanQuota();
const planViewStore = usePlanViewStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.plan.planQuota') })
    : $t('ui.modal.update', { moduleName: $t('page.plan.planQuota') }),
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
      component: 'ApiSelect',
      fieldName: 'planId',
      label: $t('page.plan.plan'),
      defaultValue: planViewStore.currentPlanId ?? undefined,
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        disabled: true,
        showSearch: true,
        allowClear: false,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        labelField: 'name',
        valueField: 'id',
        api: async () => {
          const result = await fetchListPlans(
            new PaginationQuery({
              formValues: {},
            }),
          );
          return result.items;
        },
      },
    },
    {
      component: 'Select',
      fieldName: 'quotaType',
      label: $t('page.plan.quotaType'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: planQuotaTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
      rules: 'selectRequired',
    },
    {
      component: 'InputNumber',
      fieldName: 'quotaValue',
      label: $t('page.plan.quotaValue'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {

    // 校验输入的数据
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    // 加载条设置为加载状态
    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();


    try {
      await (data.value?.create
        ? createPlanQuota(values)
        : updatePlanQuota({ id: data.value.row.id, values }));

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
      if (data.value.row === undefined) {
        baseFormApi.setValues({ planId: planViewStore.currentPlanId });
      } else {
        data.value.row.planId = planViewStore.currentPlanId;
        baseFormApi.setValues(data.value?.row);
      }

      setLoading(false);

    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle" class="w-full max-w-[800px]">
    <BaseForm class="mx-0" />
  </Drawer>
</template>
