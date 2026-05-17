<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import { enableBoolList, useDictStore } from '#/stores';

const dictStore = useDictStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.dict.dictType') })
    : $t('ui.modal.update', { moduleName: $t('page.dict.dictType') }),
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
      fieldName: 'typeName',
      label: $t('page.dict.typeName'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: z.string().min(1, { message: $t('ui.formRules.required') }),
    },
    {
      component: 'Input',
      fieldName: 'typeCode',
      label: $t('page.dict.typeCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: z.string().min(1, { message: $t('ui.formRules.required') }),
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
      fieldName: 'isEnabled',
      label: $t('ui.table.status'),
      defaultValue: true,
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: enableBoolList,
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

    console.log(getTitle.value, Object.keys(values));

    try {
      await (data.value?.create
        ? dictStore.createDictType(values)
        : dictStore.updateDictType(data.value.row.id, values));

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
      if (data.value.row !== undefined) {
        baseFormApi.setValues(data.value?.row);
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
