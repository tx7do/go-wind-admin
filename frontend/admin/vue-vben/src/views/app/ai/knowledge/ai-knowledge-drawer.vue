<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { PaginationQuery, createAiKnowledgeBase, fetchListAiProviders, updateAiKnowledgeBase } from '#/api';

const emit = defineEmits<{ success: [] }>();

const data = ref<{ create?: boolean; row?: any }>({});
const providerOptions = ref<{ label: string; value: number }[]>([]);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: { componentProps: { class: 'w-full' } },
  schema: [
    {
      fieldName: 'name',
      label: $t('page.aiKnowledge.name'),
      component: 'Input',
      rules: 'required',
      componentProps: { placeholder: $t('ui.placeholder.input') },
    },
    {
      fieldName: 'description',
      label: $t('page.aiKnowledge.description'),
      component: 'Textarea',
      componentProps: { rows: 2 },
    },
    {
      fieldName: 'providerId',
      label: $t('page.aiKnowledge.provider'),
      component: 'Select',
      rules: 'selectRequired',
      componentProps: {
        options: providerOptions,
        placeholder: $t('page.aiKnowledge.provider'),
      },
    },
    {
      fieldName: 'embeddingModel',
      label: $t('page.aiKnowledge.embeddingModel'),
      component: 'Input',
      rules: 'required',
      componentProps: { placeholder: 'text-embedding-3-small / bge-m3' },
    },
  ],
});

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.aiKnowledge.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.aiKnowledge.moduleName') }),
);

const [Drawer, drawerApi] = useVbenDrawer({
  class: 'w-[520px]',
  onConfirm: async () => {
    const { valid } = await baseFormApi.validate();
    if (!valid) return;
    const values = (await baseFormApi.getValues()) as Record<string, any>;
    try {
      drawerApi.setState({ loading: true });
      if (data.value?.create) {
        await createAiKnowledgeBase(values);
        message.success($t('ui.notification.create_success'));
      } else {
        await updateAiKnowledgeBase(data.value?.row?.id, values);
        message.success($t('ui.notification.update_success'));
      }
      emit('success');
      drawerApi.close();
    } catch (error) {
      console.error('save ai knowledge base failed:', error);
      message.error(error instanceof Error ? error.message : 'save failed');
    } finally {
      drawerApi.setState({ loading: false });
    }
  },
  onCancel: () => drawerApi.close(),
  onOpenChange(isOpen) {
    if (isOpen) {
      data.value = drawerApi.getData<{ create?: boolean; row?: any }>() ?? {};
      baseFormApi.setValues(data.value?.row ?? {});
      // provider 下拉（启用项）；失败不阻断抽屉打开
      fetchListAiProviders(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }))
        .then((res) => {
          providerOptions.value = ((res.items || []) as any[]).map((p) => ({
            label: `${p.name || ''} / ${p.modelName || ''}`,
            value: p.id,
          }));
        })
        .catch((error) => console.error('fetch providers for knowledge drawer failed:', error));
    }
  },
});
</script>

<template>
  <Drawer class="w-[520px]" :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
