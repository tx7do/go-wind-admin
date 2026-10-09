<script lang="ts" setup>
import { ref, watch } from 'vue';

import { $t } from '@vben/locales';
import { LucideTrash2 } from '@vben/icons';

import { Input, message, Modal, Popconfirm, Table, Tag } from 'ant-design-vue';

import { deleteAiDoc, fetchListAiDocs, uploadAiDoc } from '#/api';
import type {
  aiservicev1_AiDoc as AiDoc,
  aiservicev1_AiKnowledgeBase as AiKnowledgeBase,
} from '#/api/generated/admin/service/v1';

const props = defineProps<{ visible: boolean; base?: AiKnowledgeBase }>();
const emit = defineEmits<{ 'update:visible': [value: boolean] }>();

const docs = ref<AiDoc[]>([]);
const loading = ref(false);
const uploading = ref(false);
const docName = ref('');
const docContent = ref('');

function loadDocs() {
  if (!props.base?.id) return;
  loading.value = true;
  fetchListAiDocs(props.base.id)
    .then((res) => {
      docs.value = (res.items || []) as AiDoc[];
    })
    .catch((error) => {
      console.error('fetch ai docs failed:', error);
      message.error(error instanceof Error ? error.message : 'fetch failed');
    })
    .finally(() => {
      loading.value = false;
    });
}

function resetForm() {
  docName.value = '';
  docContent.value = '';
}

watch(
  () => props.visible,
  (v) => {
    if (v) loadDocs();
  },
);

const columns = [
  { title: $t('page.aiKnowledge.docName'), dataIndex: 'name', ellipsis: true },
  { title: $t('page.aiKnowledge.docStatus'), dataIndex: 'status', width: 90 },
  { title: $t('page.aiKnowledge.docChunks'), dataIndex: 'chunkCount', width: 80 },
  { title: $t('page.aiKnowledge.action'), dataIndex: 'action', width: 80 },
];

async function handleUpload() {
  if (!props.base?.id || !docName.value.trim() || !docContent.value.trim()) return;
  uploading.value = true;
  try {
    const res = await uploadAiDoc(props.base.id, docName.value.trim(), docContent.value);
    message.success($t('page.aiKnowledge.uploadSuccess', { chunks: res.chunkCount ?? 0 }));
    resetForm();
    loadDocs();
  } catch (error) {
    console.error('upload ai doc failed:', error);
    message.error(error instanceof Error ? error.message : 'upload failed');
  } finally {
    uploading.value = false;
  }
}

async function handleDelete(row: AiDoc) {
  if (!props.base?.id) return;
  try {
    await deleteAiDoc(props.base.id, row.id!);
    message.success($t('ui.notification.delete_success'));
    loadDocs();
  } catch (error) {
    console.error('delete ai doc failed:', error);
    message.error(error instanceof Error ? error.message : 'delete failed');
  }
}
</script>

<template>
  <Modal
    :footer="null"
    :open="visible"
    :title="`${$t('page.aiKnowledge.docs')}：${base?.name || ''}`"
    width="680px"
    @cancel="emit('update:visible', false)"
    @close="resetForm"
  >
    <!-- 上传区：纯文本直接入库（切片 → 向量化） -->
    <div class="mb-4 space-y-2">
      <Input v-model:value="docName" :maxlength="100" :placeholder="$t('page.aiKnowledge.docName')" />
      <Input.TextArea
        v-model:value="docContent"
        :maxlength="50000"
        :placeholder="$t('page.aiKnowledge.docContentPlaceholder')"
        :rows="5"
        show-count
      />
      <a-button
        :disabled="!docName.trim() || !docContent.trim()"
        :loading="uploading"
        type="primary"
        @click="handleUpload"
      >
        {{ $t('page.aiKnowledge.upload') }}
      </a-button>
    </div>

    <Table
      :columns="columns"
      :data-source="docs"
      :loading="loading"
      :pagination="false"
      row-key="id"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'status'">
          <Tag v-if="record.status === 'READY'" color="success">READY</Tag>
          <Tag v-else color="error">{{ record.status || 'FAILED' }}</Tag>
        </template>
        <template v-else-if="column.dataIndex === 'action'">
          <Popconfirm
            :title="$t('page.aiKnowledge.deleteDocConfirm')"
            @confirm="handleDelete(record)"
          >
            <a-button danger size="small" type="text">
              <LucideTrash2 />
            </a-button>
          </Popconfirm>
        </template>
      </template>
    </Table>
  </Modal>
</template>
