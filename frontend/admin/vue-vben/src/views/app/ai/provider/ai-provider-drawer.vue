<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { createAiProvider, updateAiProvider } from '#/api';

const emit = defineEmits<{ success: [] }>();

const data = ref<{ create?: boolean; row?: any }>({});
const currentHint = ref('');

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: { componentProps: { class: 'w-full' } },
  schema: [
    {
      fieldName: 'name',
      label: $t('page.aiProvider.name'),
      component: 'Input',
      rules: 'required',
      componentProps: { placeholder: $t('ui.placeholder.input') },
    },
    {
      fieldName: 'modelType',
      label: $t('page.aiProvider.modelType'),
      component: 'RadioGroup',
      rules: 'selectRequired',
      componentProps: {
        options: [
          { label: $t('page.aiProvider.modelTypeCloud'), value: 'CLOUD' },
          { label: $t('page.aiProvider.modelTypeLocal'), value: 'LOCAL' },
        ],
      },
      defaultValue: 'CLOUD',
    },
    {
      fieldName: 'modelName',
      label: $t('page.aiProvider.modelName'),
      component: 'Input',
      rules: 'required',
      componentProps: { placeholder: 'deepseek-chat / qwen-max / llama3' },
    },
    {
      // 云端 / 本地各自的连接配置：依赖 modelType 联动显示（if + provider 值）
      fieldName: 'baseUrl',
      label: $t('page.aiProvider.baseUrl'),
      component: 'Input',
      dependencies: {
        triggerFields: ['modelType'],
        if: (values) => values.modelType === 'CLOUD',
      },
      componentProps: { placeholder: 'https://api.deepseek.com/v1' },
    },
    {
      fieldName: 'organization',
      label: $t('page.aiProvider.organization'),
      component: 'Input',
      dependencies: {
        triggerFields: ['modelType'],
        if: (values) => values.modelType === 'CLOUD',
      },
    },
    {
      fieldName: 'apiKey',
      label: $t('page.aiProvider.apiKey'),
      component: 'InputPassword',
      dependencies: {
        triggerFields: ['modelType'],
        if: (values) => values.modelType === 'CLOUD',
      },
      componentProps: { placeholder: $t('page.aiProvider.apiKeyKeepExisting'), autocomplete: 'new-password' },
      help: undefined,
    },
    {
      fieldName: 'localHost',
      label: $t('page.aiProvider.localHost'),
      component: 'Input',
      dependencies: {
        triggerFields: ['modelType'],
        if: (values) => values.modelType === 'LOCAL',
      },
      componentProps: { placeholder: 'localhost' },
    },
    {
      fieldName: 'localPort',
      label: $t('page.aiProvider.localPort'),
      component: 'InputNumber',
      dependencies: {
        triggerFields: ['modelType'],
        if: (values) => values.modelType === 'LOCAL',
      },
      componentProps: { min: 1, max: 65535, precision: 0 },
      defaultValue: 11434,
    },
    {
      fieldName: 'timeoutSeconds',
      label: $t('page.aiProvider.timeoutSeconds'),
      component: 'InputNumber',
      componentProps: { min: 1, max: 600, precision: 0 },
      defaultValue: 60,
    },
    {
      fieldName: 'systemPrompt',
      label: $t('page.aiProvider.systemPrompt'),
      component: 'Textarea',
      componentProps: { rows: 3 },
    },
    {
      fieldName: 'isDefault',
      label: $t('page.aiProvider.isDefault'),
      component: 'Switch',
      defaultValue: false,
    },
    {
      fieldName: 'isEnabled',
      label: $t('page.aiProvider.isEnabled'),
      component: 'Switch',
      defaultValue: true,
    },
    {
      fieldName: 'remark',
      label: $t('page.aiProvider.remark'),
      component: 'Textarea',
      componentProps: { rows: 2 },
    },
  ],
});

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.aiProvider.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.aiProvider.moduleName') }),
);

const [Drawer, drawerApi] = useVbenDrawer({
  class: 'w-[560px]',
  onConfirm: async () => {
    const { valid } = await baseFormApi.validate();
    if (!valid) return;
    const values = (await baseFormApi.getValues()) as Record<string, any>;
    // 密钥请求级字段：留空表示不改（不进 updateMask）；hint 是服务端只读字段，剥掉
    const { apiKeyHint: _hint, ...payload } = values;
    try {
      drawerApi.setState({ loading: true });
      if (data.value?.create) {
        await createAiProvider(payload);
        message.success($t('ui.notification.create_success'));
      } else {
        await updateAiProvider(data.value?.row?.id, payload);
        message.success($t('ui.notification.update_success'));
      }
      emit('success');
      drawerApi.close();
    } catch (error) {
      console.error('save ai provider failed:', error);
      message.error(error instanceof Error ? error.message : 'save failed');
    } finally {
      drawerApi.setState({ loading: false });
    }
  },
  onCancel: () => drawerApi.close(),
  onOpenChange(isOpen) {
    if (isOpen) {
      data.value = drawerApi.getData<{ create?: boolean; row?: any }>() ?? {};
      currentHint.value = data.value?.row?.apiKeyHint || '';
      // 密钥永不回显：编辑回填剥掉 apiKey（undefined），只带其余字段
      const row = { ...(data.value?.row ?? {}) };
      delete row.apiKey;
      delete row.apiKeyHint;
      baseFormApi.setValues(row);
    }
  },
});
</script>

<template>
  <Drawer class="w-[560px]" :title="getTitle">
    <div v-if="!data.create && currentHint" class="mb-2 text-xs text-gray-400">
      {{ $t('page.aiProvider.apiKeyHintLabel') }}: {{ currentHint }}
    </div>
    <BaseForm />
  </Drawer>
</template>
