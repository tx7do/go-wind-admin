<template>
  <Popover
    v-model:open="visible"
    trigger="click"
    placement="bottomRight"
  >
    <template #content>
      <div class="flex w-72 flex-col gap-2">
        <Input
          v-model:value="topic"
          size="small"
          allow-clear
          :disabled="loading"
          :placeholder="$t('page.aiContent.topicPlaceholder')"
          @press-enter="handleGenerate"
        />
        <Button
          type="primary"
          size="small"
          :loading="loading"
          :disabled="!topic.trim()"
          @click="handleGenerate"
        >
          <template #icon>
            <Iconify icon="lucide:zap" />
          </template>
          {{ $t('page.aiContent.generateNow') }}
        </Button>
      </div>
    </template>
    <Button size="small" class="ml-2" :disabled="disabled">
      <template #icon>
        <Iconify icon="lucide:zap" />
      </template>
      {{ label || $t('page.aiContent.button') }}
    </Button>
  </Popover>
</template>

<script lang="ts" setup>
import { ref } from 'vue';

import { $t } from '@vben/locales';
import { preferences } from '@vben/preferences';

import { Icon as Iconify } from '@iconify/vue';
import { Button, Input, Popover, message } from 'ant-design-vue';

import { generateAiContent } from '#/api';

defineOptions({ name: 'AiGenerateButton' });

const props = defineProps<{
  /** 补充上下文（可选） */
  context?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 按钮文字，默认"AI 生成" */
  label?: string;
  /** 场景：DESCRIPTION / ANNOUNCEMENT / REPLY / GENERAL */
  scene: 'DESCRIPTION' | 'ANNOUNCEMENT' | 'REPLY' | 'GENERAL';
}>();

const emit = defineEmits<{
  /** 生成结果回调（父组件将文本填入目标字段） */
  generate: [content: string];
}>();

const topic = ref('');
const visible = ref(false);
const loading = ref(false);

async function handleGenerate() {
  const trimmed = topic.value.trim();
  if (!trimmed || loading.value) return;
  loading.value = true;
  try {
    const resp = await generateAiContent({
      scene: props.scene,
      topic: trimmed,
      context: props.context,
      lang: preferences.app.locale || 'zh-CN',
    });
    emit('generate', resp.content ?? '');
    visible.value = false;
    topic.value = '';
  } catch (error) {
    // 不吞错：带出原始错误对象，供链路排查
    console.error('ai content generate failed:', error);
    message.error($t('page.aiContent.failed'));
  } finally {
    loading.value = false;
  }
}
</script>
