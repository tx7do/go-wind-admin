<template>
  <ElDialog
    v-model="visible"
    :title="$t('pages.printTemplate.aiGenerateTitle')"
    width="560px"
    :close-on-click-modal="false"
    @closed="formRef?.resetFields()"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <ElFormItem :label="$t('pages.printTemplate.aiDescription')" prop="description">
        <ElInput
          v-model="formData.description"
          type="textarea"
          :rows="4"
          maxlength="2000"
          show-word-limit
          :placeholder="$t('pages.printTemplate.aiDescriptionPlaceholder')"
        />
      </ElFormItem>
      <ElRow :gutter="12">
        <ElCol :span="12">
          <ElFormItem :label="$t('pages.printTemplate.paperSize')" prop="paperSize">
            <ElSelect v-model="formData.paperSize">
              <ElOption v-for="p in PAPER_SIZES" :key="p" :label="p" :value="p" />
            </ElSelect>
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem :label="$t('pages.printTemplate.orientation')" prop="orientation">
            <ElSelect v-model="formData.orientation">
              <ElOption :label="$t('pages.printTemplate.portrait')" value="portrait" />
              <ElOption :label="$t('pages.printTemplate.landscape')" value="landscape" />
            </ElSelect>
          </ElFormItem>
        </ElCol>
      </ElRow>
      <ElFormItem :label="$t('pages.printTemplate.aiResourceKey')" prop="resourceKey">
        <ElInput v-model="formData.resourceKey" maxlength="100" clearable />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton type="primary" :loading="submitting" @click="handleGenerate">
        {{ $t("pages.printTemplate.aiGenerateButton") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import {
  ElButton,
  ElCol,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElOption,
  ElRow,
  ElSelect,
} from "element-plus";
import type { printservicev1_PrintTemplate } from "@/api/generated/admin/service/v1";
import { useGeneratePrintTemplate } from "@/api/composables/print_template";
import { $t } from "@/core/i18n";

const PAPER_SIZES = ["A4", "A5", "A3", "B5"];

const emit = defineEmits<{ generated: [template: printservicev1_PrintTemplate] }>();

const { mutateAsync: generateTemplate } = useGeneratePrintTemplate();

const visible = ref(false);
const submitting = ref(false);
const formRef = ref();
const formData = reactive({
  description: "",
  paperSize: "A4",
  orientation: "portrait",
  resourceKey: "",
});
const formRules = {
  description: [{ required: true, message: $t("pages.printTemplate.aiDescriptionRequired") || $t("common.validation.required"), trigger: "blur" }],
};

/** 描述文本截取为模板预填名（首行取 30 字符足够辨识，完整名用户在设计器里改） */
function nameFromDescription(description: string): string {
  const firstLine = description.split("\n")[0]?.trim() ?? "";
  return firstLine.length > 30 ? `${firstLine.slice(0, 30)}…` : firstLine;
}

async function handleGenerate() {
  await formRef.value?.validate();
  const description = formData.description.trim();
  submitting.value = true;
  try {
    // 产物不经本弹窗入库——交给设计器（ai 模式）预览，用户点保存才走 Create 链
    const resp = await generateTemplate({
      description,
      paperSize: formData.paperSize || undefined,
      orientation: formData.orientation || undefined,
      resourceKey: formData.resourceKey.trim() || undefined,
    });
    ElMessage.success($t("pages.printTemplate.aiGenerateSuccess"));
    visible.value = false;
    emit("generated", {
      name: nameFromDescription(description),
      payload: resp.payload,
      paperSize: formData.paperSize,
      orientation: formData.orientation,
      resourceKey: formData.resourceKey.trim() || undefined,
    } as printservicev1_PrintTemplate);
  } catch (e) {
    // ElMessage 错误提示由调用方（ProPage 请求层）统一弹出；此处带出原始错误对象防静默吞错
    console.error("ai generate print template failed:", e);
  } finally {
    submitting.value = false;
  }
}

defineExpose({ open: () => (visible.value = true) });
</script>
