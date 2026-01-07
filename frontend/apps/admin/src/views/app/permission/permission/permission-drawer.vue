<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  buildMenuTree,
  convertApiToTree,
  statusList,
  useApiResourceStore,
  useMenuStore,
  usePermissionGroupStore,
  usePermissionStore,
} from '#/stores';

const permissionStore = usePermissionStore();
const permissionGroupStore = usePermissionGroupStore();
const apiStore = useApiResourceStore();
const menuStore = useMenuStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.permission.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.permission.moduleName') }),
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
      label: $t('page.permission.name'),
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
      fieldName: 'code',
      label: $t('page.permission.code'),
      rules: 'required',
      componentProps() {
        return {
          placeholder: $t('ui.placeholder.input'),
          allowClear: true,
        };
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'groupId',
      label: $t('page.permission.groupId'),
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item.name,
            value: item.id,
          }));
        },
        api: async () => {
          const result = await permissionGroupStore.listPermissionGroup(
            undefined,
            {},
          );
          return result.items;
        },
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
    {
      component: 'ApiTree',
      fieldName: 'menuIds',
      componentProps: {
        title: $t('page.permission.menuIds'),
        showSearch: true,
        treeDefaultExpandAll: false,
        loadingSlot: 'suffixIcon',
        childrenField: 'children',
        labelField: 'meta.title',
        valueField: 'id',
        resultField: 'items',
        api: async () => {
          return await menuStore.listMenu(undefined, {
            status: 'ON',
          });
        },
        afterFetch: (data: any) => {
          return buildMenuTree(data.items);
        },
      },
    },
    {
      component: 'ApiTree',
      fieldName: 'apiResourceIds',
      componentProps: {
        title: $t('page.permission.apiResourceIds'),
        toolbar: true,
        search: true,
        checkable: true,
        numberToString: false,
        loadingSlot: 'suffixIcon',
        childrenField: 'children',
        labelField: 'title',
        valueField: 'key',
        api: async () => {
          const data = await apiStore.listApiResource(undefined, {});
          return convertApiToTree(data.items ?? []);
        },
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
        ? permissionStore.createPermission(values)
        : permissionStore.updatePermission(data.value.row.id, values));

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
