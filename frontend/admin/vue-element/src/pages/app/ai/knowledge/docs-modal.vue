<template>
  <ElDrawer
    :model-value="visible"
    :title="t('pages.ai_knowledge.docsTitle', { name: base?.name || '' })"
    width="680px"
    :close-on-click-modal="false"
    @update:model-value="(v: boolean) => emit('update:visible', v)"
    @closed="resetForm"
  >
    <!-- 上传区：纯文本直接入库（切片 → 向量化） -->
    <div class="mb-4 space-y-2">
      <ElInput v-model="docName" :placeholder="t('pages.ai_knowledge.docNamePlaceholder')" clearable maxlength="100" />
      <ElInput
        v-model="docContent"
        type="textarea"
        :rows="5"
        :placeholder="t('pages.ai_knowledge.docContentPlaceholder')"
        maxlength="50000"
        show-word-limit
      />
      <ElButton
        type="primary"
        :loading="uploading"
        :disabled="!docName.trim() || !docContent.trim()"
        @click="handleUpload"
      >
        {{ t("pages.ai_knowledge.upload") }}
      </ElButton>
    </div>

    <ElTable :data="docs" size="small" v-loading="loading">
      <ElTableColumn prop="name" :label="t('pages.ai_knowledge.docName')" show-overflow-tooltip />
      <ElTableColumn prop="status" :label="t('pages.ai_knowledge.docStatus')" width="90">
        <template #default="{ row }">
          <ElTag v-if="row.status === 'READY'" type="success" size="small">READY</ElTag>
          <ElTag v-else type="danger" size="small">{{ row.status || "FAILED" }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn prop="chunkCount" :label="t('pages.ai_knowledge.docChunks')" width="80" />
      <ElTableColumn :label="t('pages.ai_knowledge.action')" width="80">
        <template #default="{ row }">
          <ElPopconfirm
            :title="t('pages.ai_knowledge.deleteDocConfirm')"
            @confirm="handleDelete(row)"
          >
            <!-- EP 2.14 起 Popconfirm 触发器是具名 #reference 插槽，默认插槽被无视（按钮静默消失） -->
            <template #reference>
              <ElButton type="danger" text size="small">
                <Icon icon="lucide:trash-2" />
              </ElButton>
            </template>
          </ElPopconfirm>
        </template>
      </ElTableColumn>
    </ElTable>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";
import {
  ElButton,
  ElDialog,
  ElInput,
  ElMessage,
  ElPopconfirm,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";
import { Icon } from "@iconify/vue";
import { useI18n } from "@go-wind/admin-element";
import type { aiservicev1_AiDoc as AiDoc, aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from "@/api/generated/admin/service/v1";
import { deleteAiDoc, fetchListAiDocs, uploadAiDoc } from "@/api/composables";

const props = defineProps<{ visible: boolean; base?: AiKnowledgeBase }>();
const emit = defineEmits<{ "update:visible": [value: boolean] }>();
const { t } = useI18n();

const docs = ref<AiDoc[]>([]);
const loading = ref(false);
const uploading = ref(false);
const docName = ref("");
const docContent = ref("");

function loadDocs() {
  if (!props.base?.id) return;
  loading.value = true;
  fetchListAiDocs(props.base.id)
    .then((res) => {
      docs.value = (res.items || []) as AiDoc[];
    })
    .catch((error: any) => {
      console.error("fetch ai docs failed:", error);
      ElMessage.error(t("pages.ai_knowledge.fetchFailed"));
    })
    .finally(() => {
      loading.value = false;
    });
}

function resetForm() {
  docName.value = "";
  docContent.value = "";
}

watch(
  () => props.visible,
  (v) => {
    if (v) loadDocs();
  },
);

async function handleUpload() {
  if (!props.base?.id || !docName.value.trim() || !docContent.value.trim()) return;
  uploading.value = true;
  try {
    const res = await uploadAiDoc(props.base.id, docName.value.trim(), docContent.value);
    ElMessage.success(t("pages.ai_knowledge.uploadSuccess", { chunks: res.chunkCount ?? 0 }));
    resetForm();
    loadDocs();
  } catch (error: any) {
    console.error("upload ai doc failed:", error);
    ElMessage.error(error?.message || t("pages.ai_knowledge.uploadFailed"));
  } finally {
    uploading.value = false;
  }
}

async function handleDelete(row: AiDoc) {
  if (!props.base?.id) return;
  try {
    await deleteAiDoc(props.base.id, row.id!);
    ElMessage.success(t("pages.ai_knowledge.deleteSuccess"));
    loadDocs();
  } catch (error: any) {
    console.error("delete ai doc failed:", error);
    ElMessage.error(error?.message || t("pages.ai_knowledge.fetchFailed"));
  }
}
</script>
