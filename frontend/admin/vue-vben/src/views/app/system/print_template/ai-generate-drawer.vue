<script lang="ts" setup>
import { computed, reactive, ref } from 'vue';

import { $t } from '@vben/locales';

import { Button, Form, FormItem, Input, message, Modal, Select, Textarea } from 'ant-design-vue';

import type { printservicev1_PrintTemplate } from '#/api/generated/admin/service/v1';
import { useGeneratePrintTemplate } from '#/api/composables/print_template';

const PAPER_SIZES = ['A4', 'A5', 'A3', 'B5'];

const emit = defineEmits<{ generated: [template: printservicev1_PrintTemplate] }>();

const { mutateAsync: generateTemplate } = useGeneratePrintTemplate();

const visible = ref(false);
const submitting = ref(false);

const formData = reactive({
  description: '',
  paperSize: 'A4',
  orientation: 'portrait',
  resourceKey: '',
});

const descriptionFilled = computed(() => !!formData.description.trim());

/** 描述文本截取为模板预填名（首行取 30 字符足够辨识，完整名用户在设计器里改） */
function nameFromDescription(description: string): string {
  const firstLine = description.split('\n')[0]?.trim() ?? '';
  return firstLine.length > 30 ? `${firstLine.slice(0, 30)}…` : firstLine;
}

async function handleGenerate() {
  const description = formData.description.trim();
  if (!description) {
    message.warning($t('page.printTemplate.aiDescriptionRequired'));
    return;
  }
  submitting.value = true;
  try {
    // 产物不经本弹窗入库——交给设计器（ai 模式）预览，用户点保存才走 Create 链
    const resp = await generateTemplate({
      description,
      paperSize: formData.paperSize || undefined,
      orientation: formData.orientation || undefined,
      resourceKey: formData.resourceKey.trim() || undefined,
    });
    message.success($t('page.printTemplate.aiGenerateSuccess'));
    visible.value = false;
    emit('generated', {
      name: nameFromDescription(description),
      payload: resp.payload,
      paperSize: formData.paperSize,
      orientation: formData.orientation,
      resourceKey: formData.resourceKey.trim() || undefined,
    } as printservicev1_PrintTemplate);
  } catch (e) {
    // message 错误提示由请求层统一弹出；此处带出原始错误对象防静默吞错
    console.error('ai generate print template failed:', e);
  } finally {
    submitting.value = false;
  }
}

function open() {
  formData.description = '';
  formData.paperSize = 'A4';
  formData.orientation = 'portrait';
  formData.resourceKey = '';
  visible.value = true;
}

defineExpose({ open });
</script>

<template>
  <Modal
    v-model:open="visible"
    :title="$t('page.printTemplate.aiGenerateTitle')"
    :mask-closable="false"
    :width="560"
  >
    <Form layout="vertical" class="mt-4">
      <FormItem :label="$t('page.printTemplate.aiDescription')" required>
        <Textarea
          v-model:value="formData.description"
          :rows="4"
          :maxlength="2000"
          show-count
          :placeholder="$t('page.printTemplate.aiDescriptionPlaceholder')"
        />
      </FormItem>
      <div class="flex gap-3">
        <FormItem :label="$t('page.printTemplate.paperSize')" class="flex-1">
          <Select v-model:value="formData.paperSize" :options="PAPER_SIZES.map((p) => ({ value: p, label: p }))" />
        </FormItem>
        <FormItem :label="$t('page.printTemplate.orientation')" class="flex-1">
          <Select
            v-model:value="formData.orientation"
            :options="[
              { value: 'portrait', label: $t('page.printTemplate.portrait') },
              { value: 'landscape', label: $t('page.printTemplate.landscape') },
            ]"
          />
        </FormItem>
      </div>
      <FormItem :label="$t('page.printTemplate.aiResourceKey')">
        <Input v-model:value="formData.resourceKey" :maxlength="100" />
      </FormItem>
    </Form>
    <template #footer>
      <Button @click="visible = false">{{ $t('common.cancel') }}</Button>
      <Button type="primary" :loading="submitting" :disabled="!descriptionFilled" @click="handleGenerate">
        {{ $t('page.printTemplate.aiGenerateButton') }}
      </Button>
    </template>
  </Modal>
</template>
