<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  orgUnitStatusList,
  orgUnitTypeList,
  useOrgUnitStore,
  useUserListStore,
} from '#/stores';

const orgUnitStore = useOrgUnitStore();
const userListStore = useUserListStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.orgUnit.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.orgUnit.moduleName') }),
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
      label: $t('page.orgUnit.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.orgUnit.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'parentId',
      label: $t('page.orgUnit.parentId'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        numberToString: true,
        showSearch: true,
        treeDefaultExpandAll: true,
        allowClear: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        api: async () => {
          const result = await orgUnitStore.listOrgUnit(undefined, {
            // parent_id: 0,
            status: 'ON',
          });
          return result.items;
        },
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'leaderId',
      label: $t('page.orgUnit.leaderId'),
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item.nickname,
            value: item.id,
          }));
        },
        api: async () => {
          const result = await userListStore.listUser(undefined, {
            // parent_id: 0,
          });
          return result.items;
        },
      },
    },
    {
      component: 'Select',
      fieldName: 'type',
      label: $t('page.orgUnit.type'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: orgUnitTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
      rules: 'selectRequired',
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
      rules: 'required',
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
        options: orgUnitStatusList,
      },
    },
    {
      component: 'Switch',
      fieldName: 'isLegalEntity',
      defaultValue: false,
      label: $t('page.orgUnit.isLegalEntity'),
      componentProps: {
        class: 'w-auto',
      },
    },
    {
      component: 'Input',
      fieldName: 'registrationNumber',
      label: $t('page.orgUnit.registrationNumber'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'taxId',
      label: $t('page.orgUnit.taxId'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'address',
      label: $t('page.orgUnit.address'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Textarea',
      fieldName: 'description',
      label: $t('page.orgUnit.description'),
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

    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();

    console.log(getTitle.value, values);

    try {
      await (data.value?.create
        ? orgUnitStore.createOrgUnit(values)
        : orgUnitStore.updateOrgUnit(data.value.row.id, values));

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
