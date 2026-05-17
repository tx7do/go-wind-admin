<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  buildPermissionGroupTree,
  statusList,
  usePermissionGroupStore,
} from '#/stores';

const permissionGroupStore = usePermissionGroupStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', {
        moduleName: $t('page.permissionGroup.moduleName'),
      })
    : $t('ui.modal.update', {
        moduleName: $t('page.permissionGroup.moduleName'),
      }),
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
      label: $t('page.permissionGroup.name'),
      rules: 'required',
      componentProps() {
        return {
          placeholder: $t('ui.placeholder.input'),
          allowClear: true,
        };
      },
    },
    {
      component: 'Input',
      fieldName: 'module',
      label: $t('page.permissionGroup.module'),
      rules: 'required',
      componentProps() {
        return {
          placeholder: $t('ui.placeholder.input'),
          allowClear: true,
        };
      },
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'parentId',
      label: $t('page.permissionGroup.parentId'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        class: 'w-full',
        showSearch: true,
        treeDefaultExpandAll: true,
        numberToString: true,
        allowClear: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: any) => {
          return buildPermissionGroupTree(data);
        },
        api: async () => {
          const fieldValue = baseFormApi.form.values;
          const result = await permissionGroupStore.listPermissionGroup(
            undefined,
            {
              parentId: fieldValue.parentId,
              status: 'ON',
            },
          );
          return result.items;
        },
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'sortOrder',
      defaultValue: 1,
      label: $t('ui.table.sortOrder'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'RadioGroup',
      fieldName: 'status',
      defaultValue: 'ON',
      label: $t('ui.table.status'),
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: statusList,
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
        ? permissionGroupStore.createPermissionGroup(values)
        : permissionGroupStore.updatePermissionGroup(
            data.value.row.id,
            values,
          ));

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
      drawerApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen) {
    if (isOpen) {
      // 获取传入的数据
      data.value = drawerApi.getData<Record<string, any>>();

      // 为表单赋值
      baseFormApi.setValues(data.value?.row);

      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
