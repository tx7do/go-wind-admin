<template>
  <ElDialog
    v-model="visible"
    :title="$t('pages.script.testRunTitle', { name: script?.name ?? '-' })"
    width="640px"
    :close-on-click-modal="false"
    append-to-body
  >
    <ElDescriptions :column="2" size="small" class="mb-4">
      <ElDescriptionsItem :label="$t('pages.script.language')">
        {{ scriptLanguageToName(script?.language) }}
      </ElDescriptionsItem>
      <ElDescriptionsItem :label="$t('pages.script.hookPoint')">
        {{ script?.hookPoint || $t("pages.script.unmounted") }}
      </ElDescriptionsItem>
    </ElDescriptions>

    <div class="mb-2 text-sm text-[var(--el-text-color-secondary)]">
      {{ $t("pages.script.testRunInput") }}
    </div>

    <div
      v-for="(row, index) in inputRows"
      :key="index"
      class="mb-2 flex items-center gap-2"
    >
      <ElInput
        v-model="row.key"
        :placeholder="$t('pages.script.testRunInputKey')"
        style="width: 180px"
      />
      <ElInput v-model="row.value" :placeholder="$t('pages.script.testRunInputValue')" />
      <ElButton :icon="Delete" @click="inputRows.splice(index, 1)" />
    </div>

    <ElButton class="mb-4 w-full" dashed :icon="Plus" @click="addInputRow">
      {{ $t("pages.script.testRunAddInput") }}
    </ElButton>

    <ElAlert
      v-if="hadInvalidJson"
      type="warning"
      :title="$t('pages.script.testRunInvalidJson')"
      show-icon
      class="mb-3"
      :closable="false"
    />

    <template v-if="result">
      <ElAlert
        :type="result.success ? 'success' : 'error'"
        :title="result.success ? $t('pages.script.testRunSuccess') : $t('pages.script.testRunFailed')"
        show-icon
        :closable="false"
        class="mb-3"
      >
        <template v-if="!result.success && result.error" #default>
          <pre class="m-0 whitespace-pre-wrap">{{ result.error }}</pre>
        </template>
      </ElAlert>

      <div class="mb-1 text-sm">
        {{ $t("pages.script.testRunOutput") }} ·
        {{ $t("pages.script.testRunDuration") }}: {{ result.durationMs ?? 0 }}ms
      </div>
      <pre class="run-output">{{ contextText }}</pre>
    </template>

    <template #footer>
      <ElButton @click="visible = false">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton
        type="primary"
        :icon="VideoPlay"
        :loading="loading"
        :disabled="!script?.id"
        @click="handleRun"
      >
        {{ $t("pages.script.testRunRun") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElAlert, ElButton, ElDescriptions, ElDescriptionsItem, ElDialog, ElInput, ElMessage } from "element-plus";
import { Delete, Plus, VideoPlay } from "@element-plus/icons-vue";

import {
  scriptLanguageToName,
  useTestRunScript,
} from "@/api/composables";
import type { scriptservicev1_Script, scriptservicev1_TestRunScriptResponse } from "@/api/generated/admin/service/v1";
import { $t } from "@go-wind/admin-element";

const emit = defineEmits<{
  success: [];
}>();

const visible = ref(false);
const script = ref<scriptservicev1_Script | undefined>();
const inputRows = ref<Array<{ key: string; value: string }>>([]);
const result = ref<scriptservicev1_TestRunScriptResponse>();
const hadInvalidJson = ref(false);

const { mutateAsync: testRun, isPending: loading } = useTestRunScript();

const contextText = computed(() => {
  const context = result.value?.context || {};
  const entries = Object.entries(context);
  if (entries.length === 0) return $t("pages.script.testRunNoOutput");
  return JSON.stringify(
    Object.fromEntries(entries.map(([k, v]) => [k, safeParse(v)])),
    null,
    2
  );
});

function addInputRow() {
  inputRows.value.push({ key: "", value: "" });
}

function open(row: scriptservicev1_Script) {
  script.value = row;
  inputRows.value = [];
  result.value = undefined;
  hadInvalidJson.value = false;
  visible.value = true;
}

async function handleRun() {
  if (!script.value?.id) return;

  const input: Record<string, string> = {};
  let invalid = false;
  for (const row of inputRows.value) {
    const key = row.key.trim();
    if (!key) continue;
    input[key] = row.value;
    if (row.value.trim() !== "") {
      try {
        JSON.parse(row.value);
      } catch {
        invalid = true;
      }
    }
  }
  hadInvalidJson.value = invalid;

  try {
    result.value = await testRun({ id: script.value.id, input });
    if (result.value.success) {
      emit("success");
    }
  } catch (error) {
    console.error("脚本试运行失败", error);
    ElMessage.error($t("pages.script.fetchFailed"));
  }
}

/** 后端上下文值是 JSON 字符串，展示时还原为对象字面量 */
function safeParse(value: string): unknown {
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.run-output {
  max-height: 220px;
  padding: 12px;
  margin: 0;
  overflow: auto;
  font-size: 12px;
  line-height: 1.5;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}
</style>
