<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  planExpiryPolicyList,
  planVersionList,
  planModuleList,
  useCreatePlan,
  useUpdatePlan,
  useCreatePlanModule,
  useDeletePlanModule,
  fetchListPlanModules,
} from '#/api';
import { PaginationQuery } from '#/transport/rest';

const { mutateAsync: createPlan } = useCreatePlan();
const { mutateAsync: updatePlan } = useUpdatePlan();
const { mutateAsync: createModuleMut } = useCreatePlanModule();
const { mutateAsync: deleteModuleMut } = useDeletePlanModule();

const existingModules = ref<string[]>([]);

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.plan.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.plan.moduleName') }),
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
      component: 'Input',
      fieldName: 'name',
      label: $t('page.plan.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Select',
      fieldName: 'version',
      label: $t('page.plan.version'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: planVersionList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
      rules: 'selectRequired',
    },
    {
      component: 'Select',
      fieldName: 'expiryPolicy',
      label: $t('page.plan.expiryPolicy'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: planExpiryPolicyList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
      rules: 'selectRequired',
    },
    {
      component: 'InputNumber',
      fieldName: 'dataRetentionDays',
      label: $t('page.plan.dataRetentionDays'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'moduleWhitelist',
      label: $t('page.plan.moduleWhitelist'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: planModuleList,
        mode: 'multiple',
        allowClear: true,
        showSearch: true,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
      },
    },
    {
      component: 'Textarea',
      fieldName: 'description',
      label: $t('ui.table.description'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
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

    // 加载条设置为加载状态
    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();

    // 剥离模块白名单字段，不传入套餐主体 CRUD。
    const { moduleWhitelist, ...planValues } = values;
    const selectedModules: string[] = Array.isArray(moduleWhitelist)
      ? moduleWhitelist
      : [];

    console.log(getTitle.value, Object.keys(planValues));

    try {
      // useCreatePlan 内部已包 { data: {...} }，这里再包一层会让后端解析出空 Plan
      await (data.value?.create
        ? createPlan({ ...planValues })
        : updatePlan({ id: data.value.row.id, values: planValues }));

      notification.success({
        message: data.value?.create
          ? $t('ui.notification.create_success')
          : $t('ui.notification.update_success'),
      });

      // 编辑模式下同步模块白名单 diff
      if (!data.value?.create && data.value.row?.id) {
        const planId = data.value.row.id;
        const toAdd = selectedModules.filter(
          (m) => !existingModules.value.includes(m),
        );
        const toRemove = existingModules.value.filter(
          (m) => !selectedModules.includes(m),
        );

        // 查找已有模块记录以获取删除所需的 ID
        let existingItems: any[] = [];
        try {
          const resp = await fetchListPlanModules(
            new PaginationQuery({
              paging: { page: 1, pageSize: 999 },
              formValues: { plan_id: planId },
            }),
          );
          existingItems = resp?.items || [];
        } catch {
          existingItems = [];
        }

        for (const mod of toAdd) {
          await createModuleMut({ planId: planId, module: mod as any } as any);
        }
        for (const mod of toRemove) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const item = existingItems.find((it: any) => it?.module === mod);
          if (item?.id) {
            await deleteModuleMut({ id: item.id } as any);
          }
        }
      }
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
      if (data.value.row !== undefined) {
        baseFormApi.setValues(data.value?.row);

        // 加载该套餐已有的模块白名单
        try {
          fetchListPlanModules(
            new PaginationQuery({
              paging: { page: 1, pageSize: 999 },
              formValues: { plan_id: data.value.row.id },
            }),
          ).then((resp: any) => {
            const names: string[] = [];
            if (resp?.items) {
              for (const item of resp.items) {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const mod = (item as any)?.module;
                if (typeof mod === 'string' && mod.length > 0) {
                  names.push(mod);
                }
              }
            }
            existingModules.value = names;
            baseFormApi.setValues({ moduleWhitelist: [...names] });
          });
        } catch {
          existingModules.value = [];
        }
      }

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
  <Drawer :title="getTitle" class="w-full max-w-[800px]">
    <BaseForm class="mx-0" />
  </Drawer>
</template>
