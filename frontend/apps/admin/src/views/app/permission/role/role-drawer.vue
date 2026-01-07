<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  buildMenuTree,
  convertApiToTree, convertPermissionToTree,
  statusList,
  useApiResourceStore,
  useMenuStore, usePermissionStore,
  useRoleStore,
} from '#/stores';
import { deepClone } from '#/utils';

const roleStore = useRoleStore();
const apiStore = useApiResourceStore();
const menuStore = useMenuStore();
const permissionStore = usePermissionStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.role.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.role.moduleName') }),
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
      label: $t('page.role.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.role.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'InputNumber',
      fieldName: 'sortOrder',
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
      label: $t('ui.table.status'),
      defaultValue: 'ON',
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: statusList,
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
      component: 'ApiTree',
      fieldName: 'permissions',
      componentProps: {
        title: $t('page.role.permissions'),
        showSearch: true,
        treeDefaultExpandAll: false,
        loadingSlot: 'suffixIcon',
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        resultField: 'items',
        api: async () => {
          return await permissionStore.listPermission(undefined, {
            status: 'ON',
          });
        },
        afterFetch: (data: any) => {
          return convertPermissionToTree(data.items);
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
    // @ts-ignore JSON.stringify
    const finalValues = deepClone(values);

    if (
      finalValues.apis !== null &&
      Array.isArray(finalValues.apis) &&
      finalValues.apis.length > 0
    ) {
      finalValues.apis = filterNumbers(values.apis);
    }

    console.log(getTitle.value, finalValues, data.value.row);

    try {
      await (data.value?.create
        ? roleStore.createRole(finalValues)
        : roleStore.updateRole(data.value.row.id, finalValues));

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

      // setLoading(true);
      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ loading });
}

/**
 * 从数组或对象中提取所有有效数字（原始 number 类型，排除 NaN/Infinity）
 * @returns 仅包含有效数字的数组
 */
function filterNumbers(arr: unknown[]): number[] {
  if (!Array.isArray(arr)) {
    throw new TypeError('输入必须是 Array 类型');
  }

  const is_valid_number = (value: unknown): value is number => {
    return (
      typeof value === 'number' && // 必须是原始 number 类型
      Object.prototype.toString.call(value) === '[object Number]' && // 排除数字包装对象（new Number()）
      !Number.isNaN(value) && // 排除 NaN
      Number.isFinite(value) // 排除 Infinity
    );
  };

  return arr.filter((element) => is_valid_number(element));
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
