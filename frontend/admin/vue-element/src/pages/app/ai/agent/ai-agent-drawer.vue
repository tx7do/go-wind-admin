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
    <ElForm
      ref="formRef"
      :model="drawer.formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <ElFormItem :label="t('pages.ai_agent.name')" prop="name">
        <ElInput v-model="drawer.formData.name" clearable />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_agent.description')" prop="description">
        <ElInput v-model="drawer.formData.description" type="textarea" :rows="2" />
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_agent.graph')" prop="graph">
        <ElInput
          v-model="drawer.formData.graph"
          type="textarea"
          :rows="10"
          :placeholder="t('pages.ai_agent.graphPlaceholder')"
          class="font-mono"
        />
        <div class="field-tip">{{ t("pages.ai_agent.graphHint") }}</div>
      </ElFormItem>

      <ElFormItem :label="t('pages.ai_agent.enabled')" prop="enabled">
        <ElSwitch v-model="drawer.formData.enabled" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="drawer.guard.requestClose(drawer.close)">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton
          type="primary"
          :loading="drawer.submitLoading.value"
          @click="drawer.handleSubmit(formRef, () => emit('success'))"
        >
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElSwitch,
  ElMessage,
} from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import { useI18n } from "@/core/i18n";
import {
  createAiAgentApp,
  updateAiAgentApp,
} from "@/api/composables";

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const formRef = ref();
const isCreate = ref(true);

const drawer = useDrawerForm({
  moduleKey: "pages.ai_agent.moduleName",
  defaults: {
    name: "",
    description: "",
    graph: "",
    enabled: true,
  },
  // graph 保存期由后端强校验（唯一 start/可达/无环/condition 分支），前端不复制第二份
  createFn: async (values: Record<string, any>) => {
    return createAiAgentApp(values);
  },
  updateFn: (id: string, values: Record<string, any>) => {
    return updateAiAgentApp(id, values, "name,description,graph,enabled");
  },
});

const formRules = computed(() => ({
  name: [
    { required: true, message: t("pages.ai_agent.requiredName"), trigger: "blur" },
  ],
  // graph 允许空串（先建壳后画图）；非空时必须是合法 JSON（结构校验在后端）
  graph: [
    {
      validator: (_rule: any, value: string, callback: (err?: Error) => void) => {
        if (!value) return callback();
        try {
          JSON.parse(value);
          callback();
        } catch {
          callback(new Error(t("pages.ai_agent.graphInvalid")));
        }
      },
      trigger: "blur",
    },
  ],
}));

// 包装 open：追踪创建/编辑模式，编辑时显式回填行数据
function open(options: { create: boolean; row?: any }) {
  isCreate.value = options.create;
  drawer.open(options, (row: any) => {
    Object.assign(drawer.formData, {
      name: row.name || "",
      description: row.description || "",
      graph: row.graph || "",
      enabled: !!row.enabled,
    });
  });
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.field-tip {
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
