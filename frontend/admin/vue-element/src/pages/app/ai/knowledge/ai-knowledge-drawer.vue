<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :close-guard="drawer.closeGuard"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{
      component: 'drawer',
      drawer: { size: drawer.drawerWidth },
    }"
  >
    <ElForm ref="formRef" :model="drawer.formData" :rules="formRules" label-width="140px" class="drawer-form">
      <ElFormItem :label="t('pages.ai_knowledge.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="t('pages.ai_knowledge.namePlaceholder')" clearable />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_knowledge.description')" prop="description">
        <ElInput v-model="drawer.formData.description" type="textarea" :rows="2" />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_knowledge.provider')" prop="providerId">
        <ElSelect v-model="drawer.formData.providerId" :placeholder="t('pages.ai_knowledge.requiredProvider')">
          <ElOption
            v-for="p in providerOptions"
            :key="p.value"
            :label="p.label"
            :value="p.value"
          />
        </ElSelect>
        <div class="field-tip">{{ t("pages.ai_knowledge.providerTooltip") }}</div>
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_knowledge.embeddingModel')" prop="embeddingModel">
        <ElInput v-model="drawer.formData.embeddingModel" placeholder="text-embedding-3-small / bge-m3" clearable />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <ElButton @click="drawer.guard.requestClose(drawer.close)">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton
        type="primary"
        :loading="drawer.submitLoading.value"
        @click="drawer.handleSubmit(formRef, () => emit('success'))"
      >
        {{ $t("common.button.confirm") }}
      </ElButton>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
  type FormInstance,
  type FormRules,
} from "element-plus";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import { useI18n } from "@/core/i18n";
import { PaginationQuery } from "@/core/transport/rest";
import { createAiKnowledgeBase, fetchListAiKnowledgeBases, fetchListAiProviders, updateAiKnowledgeBase } from "@/api/composables";

const emit = defineEmits<{ success: [] }>();
const { t } = useI18n();
const formRef = ref<FormInstance>();

const providerOptions = ref<{ label: string; value: string }[]>([]);

const drawer = useDrawerForm({
  moduleKey: "pages.ai_knowledge.moduleName",
  defaults: {
    name: "",
    description: "",
    providerId: undefined,
    embeddingModel: "",
  },
  createFn: async (values: Record<string, any>) => {
    return createAiKnowledgeBase(values);
  },
  updateFn: (id: string, values: Record<string, any>) => {
    return updateAiKnowledgeBase(id, values);
  },
  asyncSetup: async () => {
    // provider 下拉（启用项）
    try {
      const res = await fetchListAiProviders(
        new PaginationQuery({ paging: { page: 1, pageSize: 100 }, formValues: { isEnabled: true } }),
      );
      providerOptions.value = (res.items || []).map((p: any) => ({
        label: `${p.name || ""} / ${p.modelName || ""}`,
        value: p.id,
      }));
    } catch (error) {
      console.error("fetch providers for knowledge drawer failed:", error);
    }
  },
});

const formRules: FormRules = {
  name: [{ required: true, message: t("pages.ai_knowledge.requiredName"), trigger: "blur" }],
  providerId: [{ required: true, message: t("pages.ai_knowledge.requiredProvider"), trigger: "change" }],
  embeddingModel: [{ required: true, message: t("pages.ai_knowledge.requiredEmbeddingModel"), trigger: "blur" }],
};

// 包装 open：编辑模式下显式回填表单声明字段（避免 useDrawerForm 默认不回填导致编辑态空白）
function open(data?: { create?: boolean; row?: any }) {
  drawer.open(data, (row: any) => {
    if (!row) return;
    // 仅回填 defaults 声明的字段，不拷贝 id/createdAt 等不可变字段
    drawer.formData.name = row.name ?? "";
    drawer.formData.description = row.description ?? "";
    drawer.formData.providerId = row.providerId ?? undefined;
    drawer.formData.embeddingModel = row.embeddingModel ?? "";
  });
}

// 暴露方法给父组件
defineExpose({ open });
</script>

<style lang="scss" scoped>
.field-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  margin-top: 4px;
  width: 100%;
}
</style>
