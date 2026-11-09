<template>
  <ProModal
    v-model:visible="visible"
    :title="runTitle"
    :loading="running"
    :config="{
      component: 'drawer',
      drawer: { size: 520, closeOnClickModal: false },
    }"
  >
    <ElForm ref="formRef" :model="runForm" label-width="120px" class="drawer-form">
      <ElFormItem
        v-for="v in inputVars"
        :key="v.name"
        :label="v.label || v.name"
        :prop="v.name"
        :rules="v.required ? [{ required: true, message: t('pages.ai_agent.runs.requiredInput') }] : []"
      >
        <ElInput v-model="runForm[v.name!]" type="textarea" :rows="2" />
      </ElFormItem>
      <div v-if="inputVars.length === 0" class="field-tip">
        {{ t("pages.ai_agent.runs.noInputs") }}
      </div>
      <div v-if="liveSteps.length > 0" class="field-tip">{{ liveSteps.join(" · ") }}</div>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="visible = false">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="running" @click="handleRun">
          {{ t("pages.ai_agent.runs.run") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch, onUnmounted } from "vue";
import { ElButton, ElForm, ElFormItem, ElInput, ElMessage } from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useI18n } from "@/core/i18n";
import { globalSSEClient, SSE_EVENT } from "@/core/transport/sse";
import { fetchAiAgentApp, runAiAgentApp } from "@/api/composables";

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const formRef = ref();
const running = ref(false);
const app = ref<any>();
const runForm = reactive<Record<string, string>>({});

const visible = ref(false);

const runTitle = computed(() => t("pages.ai_agent.runs.testTitle", { name: app.value?.name || "" }));

// start 节点 inputVars 声明解析（graph JSON 里取）
const inputVars = computed(() => {
  if (!app.value?.graph) return [];
  try {
    const g = JSON.parse(app.value.graph);
    const start = (g.nodes ?? []).find((n: any) => n.type === "start");
    return (start?.data?.inputVars ?? []) as { name: string; label?: string; required?: boolean }[];
  } catch {
    return [];
  }
});

const liveSteps = ref<string[]>([]);

function onFrame(data: any) {
  if (!data || typeof data !== "object" || data.appId !== app.value.id) return;
  if (data.phase === "node_end") {
    liveSteps.value.push(`${data.nodeId} ok`);
  }
}

function open(row: any) {
  app.value = row;
  Object.keys(runForm).forEach((k) => delete runForm[k]);
  liveSteps.value = [];
  visible.value = true;
  // 本用户流上的 ai_agent_run 帧（appId 过滤在 handler 内）；尽力而为，同步响应为准。
  // 注销用同一回调引用（组件重挂载反复 on 会累加）；unmount 兜底
  globalSSEClient.on(SSE_EVENT.AgentRun, onFrame);
  // 行字段可能被裁剪/滞后：graph 的事实源是 Get，拉全量刷新 inputVars
  if (row?.id) {
    fetchAiAgentApp(row.id)
      .then((resp: any) => {
        app.value = resp;
      })
      .catch((error: any) => {
        console.error("[ai-agent-run-drawer] load app detail failed:", error);
      });
  }
}

onUnmounted(() => {
  globalSSEClient.off(SSE_EVENT.AgentRun, onFrame);
});

watch(visible, (v) => {
  if (!v) {
    globalSSEClient.off(SSE_EVENT.AgentRun, onFrame);
    liveSteps.value = [];
  }
});

async function handleRun() {
  const valid = await formRef.value
    ?.validate()
    .then(() => true, () => false);
  if (!valid) return;
  running.value = true;
  try {
    await runAiAgentApp(app.value.id, { ...runForm });
    ElMessage.success(t("pages.ai_agent.runs.testSent"));
    visible.value = false;
    emit("success");
  } catch (error: any) {
    console.error("agent run failed:", error);
    ElMessage.error(error?.message || t("pages.ai_agent.runs.testFailed"));
  } finally {
    running.value = false;
  }
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
