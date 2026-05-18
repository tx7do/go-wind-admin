<script lang="ts" setup>
import type { ChangeEvent } from 'ant-design-vue/es/_util/EventInterface';

import { computed, reactive, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t, $te } from '@vben/locales';

import lucide from '@iconify/json/json/lucide.json';
import { addCollection } from '@iconify/vue';
import { notification } from 'ant-design-vue';

import { useVbenForm } from '@/adapter/form';
import {
  buildMenuTree,
  isButton,
  isCatalog,
  isMenu,
  menuTypeList,
  statusList,
  useMenuStore,
} from '@/stores';

const menuStore = useMenuStore();

addCollection(lucide);

const data = ref();

const titleSuffix = reactive({ title: '' });

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: t('pages.menu.moduleName') })
    : $t('ui.modal.update', { moduleName: t('pages.menu.moduleName') }),
);

// const isCreate = computed(() => data.value?.create);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  // 所有表单项共用，可单独在表单内覆盖
  commonConfig: {
    formItemClass: 'col-span-2 md:col-span-1',
  },
  wrapperClass: 'grid-cols-2 gap-x-4',

  schema: [
    {
      component: 'RadioGroup',
      fieldName: 'type',
      label: t('pages.menu.type'),
      defaultValue: 'MENU',
      formItemClass: 'col-span-2 md:col-span-2',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        options: menuTypeList,
      },
    },

    {
      component: 'Input',
      fieldName: 'meta.title',
      label: t('pages.menu.title'),
      rules: 'required',
      componentProps() {
        // 不需要处理多语言时就无需这么做
        return {
          placeholder: $t('ui.placeholder.input'),
          allowClear: true,
          addonAfter: titleSuffix.title,
          onChange({ target: { value } }: ChangeEvent) {
            titleSuffix.title = value && $te(value) ? $t(value) : '';
          },
        };
      },
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'parentId',
      label: t('pages.menu.parentId'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        class: 'w-full',
        showSearch: true,
        treeDefaultExpandAll: true,
        numberToString: true,
        allowClear: true,
        childrenField: 'children',
        labelField: 'meta.title',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        api: async () => {
          const fieldValue = baseFormApi.form.values;
          const result = await menuStore.listMenu(undefined, {
            parentId: fieldValue.parentId,
            status: 'ON',
          });
          return result.items;
        },

        afterFetch: (data: any) => {
          return buildMenuTree(data);
        },
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'meta.order',
      label: t('pages.menu.order'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'IconPicker',
      fieldName: 'meta.icon',
      label: t('pages.menu.icon'),
      componentProps: {
        prefix: 'lucide',
      },
      dependencies: {
        show: (values) => !isButton(values.type),
        triggerFields: ['type'],
      },
    },
    {
      component: 'Input',
      fieldName: 'path',
      label: t('pages.menu.path'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      dependencies: {
        show: (values) => !isButton(values.type),
        triggerFields: ['type'],
      },
    },
    {
      component: 'Input',
      fieldName: 'component',
      label: t('pages.menu.component'),
      defaultValue: 'BasicLayout',
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      dependencies: {
        show: (values) => isMenu(values.type),
        triggerFields: ['type'],
      },
    },
    {
      component: 'Select',
      fieldName: 'meta.authority',
      label: t('pages.menu.authority'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
        mode: 'tags',
        tokenSeparators: [','],
        class: 'w-full',
      },
      dependencies: {
        show: (values) => !isCatalog(values.type),
        triggerFields: ['type'],
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
      component: 'Divider',
      dependencies: {
        show: (values) => {
          return !['BUTTON', 'LINK'].includes(values.type);
        },
        triggerFields: ['type'],
      },
      fieldName: 'divider1',
      formItemClass: 'col-span-2 md:col-span-2 pb-0',
      hideLabel: true,
      renderComponentContent() {
        return {
          default: () => t('pages.menu.advancedSettings'),
        };
      },
    },

    {
      component: 'Checkbox',
      fieldName: 'meta.keepAlive',
      dependencies: {
        show: (values) => {
          return ['MENU'].includes(values.type);
        },
        triggerFields: ['type'],
      },
      renderComponentContent() {
        return {
          default: () => t('pages.menu.keepAlive'),
        };
      },
    },
    {
      component: 'Checkbox',
      fieldName: 'meta.affixTab',
      dependencies: {
        show: (values) => {
          return ['EMBEDDED', 'MENU'].includes(values.type);
        },
        triggerFields: ['type'],
      },
      renderComponentContent() {
        return {
          default: () => t('pages.menu.affixTab'),
        };
      },
    },
    {
      component: 'Checkbox',
      fieldName: 'meta.hideInMenu',
      dependencies: {
        show: (values) => {
          return !['BUTTON'].includes(values.type);
        },
        triggerFields: ['type'],
      },
      renderComponentContent() {
        return {
          default: () => t('pages.menu.hideInMenu'),
        };
      },
    },
    {
      component: 'Checkbox',
      fieldName: 'meta.hideChildrenInMenu',
      dependencies: {
        show: (values) => {
          return ['CATALOG', 'MENU'].includes(values.type);
        },
        triggerFields: ['type'],
      },
      renderComponentContent() {
        return {
          default: () => t('pages.menu.hideChildrenInMenu'),
        };
      },
    },
    {
      component: 'Checkbox',
      fieldName: 'meta.hideInBreadcrumb',
      dependencies: {
        show: (values) => {
          return !['BUTTON', 'LINK'].includes(values.type);
        },
        triggerFields: ['type'],
      },
      renderComponentContent() {
        return {
          default: () => t('pages.menu.hideInBreadcrumb'),
        };
      },
    },
    {
      component: 'Checkbox',
      fieldName: 'meta.hideInTab',
      dependencies: {
        show: (values) => {
          return !['BUTTON', 'LINK'].includes(values.type);
        },
        triggerFields: ['type'],
      },
      renderComponentContent() {
        return {
          default: () => t('pages.menu.hideInTab'),
        };
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
        ? menuStore.createMenu(values)
        : menuStore.updateMenu(data.value.row.id, values));

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

      titleSuffix.title = data.value?.row?.meta?.title
        ? $t(data.value?.row?.meta?.title)
        : '';

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
  <Drawer :title="getTitle" class="w-full max-w-[800px]">
    <BaseForm class="mx-4" />
  </Drawer>
</template>
