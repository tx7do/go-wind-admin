<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { StorageManager } from '@vben-core/shared/cache';

import { notification } from 'ant-design-vue';

import { EditorType } from '#/adapter/component/Editor';
import { useVbenForm } from '#/adapter/form';
import {
  type internal_messageservicev1_InternalMessage as InternalMessage,
  type internal_messageservicev1_SendMessageRequest as SendMessageRequest,
} from '@/api/generated/admin/service/v1';
import {
  internalMessageStatusList,
  internalMessageTypeList,
  useInternalMessageCategoryStore,
  useInternalMessageStore,
} from '#/stores';

const internalMessageStore = useInternalMessageStore();
const internalMessageCategoryStore = useInternalMessageCategoryStore();

const storageManager = new StorageManager({
  prefix: 'internal_message',
});

const storageKeyMessage = 'message';

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('page.internalMessage.drawer.create')
    : $t('page.internalMessage.drawer.update'),
);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: {
    formItemClass: 'col-span-2 md:col-span-1',
  },
  wrapperClass: 'grid-cols-2 gap-x-4',

  schema: [
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.internalMessage.status'),
      defaultValue: 'DRAFT',
      componentProps: {
        class: 'w-full',
        placeholder: $t('ui.placeholder.select'),
        options: internalMessageStatusList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        showSearch: true,
      },
      rules: 'selectRequired',
    },
    {
      component: 'Select',
      fieldName: 'type',
      label: $t('page.internalMessage.type'),
      defaultValue: 'NOTIFICATION',
      componentProps: {
        class: 'w-full',
        placeholder: $t('ui.placeholder.select'),
        options: internalMessageTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        showSearch: true,
      },
      rules: 'selectRequired',
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'categoryId',
      label: $t('page.internalMessage.categoryId'),
      rules: 'selectRequired',
      formItemClass: 'col-span-2 md:col-span-2',
      componentProps: {
        class: 'w-full',
        placeholder: $t('ui.placeholder.select'),
        numberToString: true,
        showSearch: true,
        treeDefaultExpandAll: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        api: async () => {
          const result =
            await internalMessageCategoryStore.listInternalMessageCategory(
              undefined,
              {
                is_enabled: 'true',
              },
            );
          return result.items;
        },
      },
    },
    {
      component: 'Input',
      fieldName: 'title',
      label: $t('page.internalMessage.title'),
      rules: 'required',
      formItemClass: 'col-span-2 md:col-span-2',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Editor',
      fieldName: 'content',
      defaultValue: '',
      label: $t('page.internalMessage.content'),
      formItemClass: 'col-span-2 md:col-span-2',
      componentProps: {
        height: '100%',
        placeholder: $t('ui.editor.please_input_content'),
        editorType: EditorType.RICH_TEXT,
        uploadImage: handleUploadImage,
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
        ? internalMessageStore.sendMessage({
            ...values,
            targetAll: true,
          } as SendMessageRequest)
        : internalMessageStore.updateMessage(data.value.row.id, values));

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
      onOpenDrawer();
    } else {
      onCloseDrawer();
    }
  },
});

function onOpenDrawer() {
  // 获取传入的数据
  data.value = drawerApi.getData<Record<string, any>>();

  if (data.value?.create) {
    data.value.row = storageManager.getItem<InternalMessage>(storageKeyMessage);
  }

  // 为表单赋值
  baseFormApi.setValues(data.value?.row);

  setLoading(false);

  console.log('onOpenDrawer', data.value);
}

async function onCloseDrawer() {
  if (data.value?.create) {
    // 获取表单数据
    const values = await baseFormApi.getValues();
    storageManager.setItem(storageKeyMessage, values);
  }
}

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}

async function handleUploadImage(file: File): Promise<string> {
  console.log('Upload image:', file);

  try {
    return '';
  } catch (error) {
    console.error('Image upload failed:', error);
    return '';
  }
}
</script>

<template>
  <Drawer :title="getTitle" class="w-full max-w-[800px]">
    <BaseForm class="mx-4" />
  </Drawer>
</template>
