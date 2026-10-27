<template>
  <ProModal
    v-model:visible="visible"
    :close-guard="closeGuard"
    :title="title"
    :config="{
      component: 'drawer',
      drawer: { size: '100%' },
    }"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="110px"
      class="designer-form"
    >
      <ElRow :gutter="16">
        <ElCol :span="6">
          <ElFormItem :label="$t('pages.printTemplate.name')" prop="name">
            <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
          </ElFormItem>
        </ElCol>
        <ElCol :span="6">
          <ElFormItem :label="$t('pages.printTemplate.code')" prop="code">
            <ElInput v-model="formData.code" placeholder="biz.order.a4" :disabled="mode === 'edit'" clearable />
          </ElFormItem>
        </ElCol>
        <ElCol :span="4">
          <ElFormItem :label="$t('pages.printTemplate.paperSize')" prop="paperSize">
            <ElSelect v-model="formData.paperSize">
              <ElOption v-for="p in PAPER_SIZES" :key="p" :label="p" :value="p" />
            </ElSelect>
          </ElFormItem>
        </ElCol>
        <ElCol :span="4">
          <ElFormItem :label="$t('pages.printTemplate.orientation')" prop="orientation">
            <ElSelect v-model="formData.orientation">
              <ElOption :label="$t('pages.printTemplate.portrait')" value="portrait" />
              <ElOption :label="$t('pages.printTemplate.landscape')" value="landscape" />
            </ElSelect>
          </ElFormItem>
        </ElCol>
        <ElCol :span="2">
          <ElFormItem :label="$t('pages.printTemplate.isEnabled')" prop="isEnabled">
            <ElSwitch v-model="formData.isEnabled" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="2">
          <ElFormItem :label="$t('pages.printTemplate.sortOrder')" prop="sortOrder">
            <ElInputNumber v-model="formData.sortOrder" :min="0" style="width: 100%" controls-position="right" />
          </ElFormItem>
        </ElCol>
      </ElRow>
      <ElRow :gutter="16">
        <ElCol :span="12">
          <ElFormItem :label="$t('pages.printTemplate.resourceKey')" prop="resourceKey">
            <ElInput v-model="formData.resourceKey" placeholder="system/user" clearable />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem :label="$t('pages.printTemplate.sampleData')" prop="sampleData">
            <ElInput
              v-model="formData.sampleData"
              type="textarea"
              :rows="2"
              :placeholder='`{"order":{"no":"SO-001","items":[{"sku":"A1","qty":2}]}}`'
            />
          </ElFormItem>
        </ElCol>
      </ElRow>
    </ElForm>

    <!-- 画布容器：任何指针交互视为脏 -->
    <div
      class="designer-host"
      @pointerdown.capture="markDirty"
      @keydown.capture="markDirty"
    >
      <PrintDesigner
        ref="canvasRef"
        is-edit
        :show-help="false"
        :initial-template="initialTemplate"
        :fields="fields"
        style="display: block; height: 100%"
      />
    </div>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="requestClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton :loading="previewLoading" @click="handlePreview">
          {{ $t("pages.printTemplate.preview") }}
        </ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSave">
          {{ $t("common.button.save") }}
        </ElButton>
      </div>
    </template>
  </ProModal>

  <!-- 打印预览 -->
  <ElDialog v-model="previewVisible" :title="$t('pages.printTemplate.previewTitle')" width="82%">
    <div class="preview-body">
      <div v-if="previewLoading" class="preview-loading">{{ $t("pages.printTemplate.rendering") }}</div>
      <iframe ref="previewFrameRef" title="print-preview" class="preview-frame" />
    </div>
    <template #footer>
      <ElButton @click="previewVisible = false">{{ $t("common.button.close") }}</ElButton>
      <ElButton type="primary" @click="handlePreviewPrint">
        {{ $t("pages.printTemplate.print") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import {
  ElButton,
  ElCol,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElRow,
  ElSelect,
  ElSwitch,
} from "element-plus";
// go-wind-print 设计器画布（Vue3 组件，无 UI 库依赖）；样式全局引一次
import { PrintDesigner, createDefaultTemplate } from "@worm-vue3-print/canvas";
import "@worm-vue3-print/canvas/style.css";
import type {
  MultiPageTemplateData,
  PrintBusinessField,
  TemplateData,
} from "@worm-vue3-print/core/designer";
import { renderHtmlPages, browserCodeRenderer } from "@worm-vue3-print/core/browser";

import ProModal from "@/components/Pro/ProModal/index.vue";
import {
  useCreatePrintTemplate,
  useUpdatePrintTemplate,
} from "@/api/composables/print_template";
import { $t } from "@/core/i18n";

const PAPER_SIZES = ["A4", "A3", "A5", "Letter", "Legal"];

const emit = defineEmits<{ success: [] }>();

const { mutateAsync: createTemplate } = useCreatePrintTemplate();
const { mutateAsync: updateTemplate } = useUpdatePrintTemplate();

const visible = ref(false);
const mode = ref<"create" | "edit" | "copy">("create");
const currentRow = ref<any>(null);
const formRef = ref();
const canvasRef = ref<any>(null);
const submitLoading = ref(false);
const previewLoading = ref(false);
const previewVisible = ref(false);
const previewFrameRef = ref<HTMLIFrameElement | null>(null);
const dirty = ref(false);

const formData = ref({
  name: "",
  code: "",
  resourceKey: "",
  paperSize: "A4",
  orientation: "portrait",
  isEnabled: true,
  sortOrder: 100,
  sampleData: "",
});

const initialTemplate = ref<TemplateData | MultiPageTemplateData | undefined>(
  createDefaultTemplate() as any
);
const fields = ref<PrintBusinessField[]>([]);
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

const title = computed(() =>
  mode.value === "create"
    ? $t("pages.printTemplate.createTitle")
    : mode.value === "copy"
      ? $t("pages.printTemplate.copyTitle")
      : $t("pages.printTemplate.editTitle")
);

function markDirty() {
  dirty.value = true;
}

async function closeGuard(): Promise<boolean> {
  if (!dirty.value) return true;
  try {
    await ElMessageBox.confirm(
      $t("common.drawer.unsavedMessage"),
      $t("common.button.confirm"),
      { type: "warning", confirmButtonText: $t("common.drawer.discardButton") }
    );
    dirty.value = false;
    return true;
  } catch {
    return false;
  }
}

function requestClose() {
  closeGuard().then((allow) => allow && (visible.value = false));
}

/** 示例数据 JSON → 字段面板（对象拍平点路径，数组记 list 并派生 items[].xxx） */
function fieldsFromSample(data: Record<string, unknown>): PrintBusinessField[] {
  const out: PrintBusinessField[] = [];
  let order = 0;
  const push = (key: string, label: string, fieldType: string) => {
    out.push({ fieldKey: key, fieldLabel: label, fieldType, sortOrder: ++order });
  };
  const walk = (obj: Record<string, unknown>, prefix: string) => {
    Object.entries(obj || {}).forEach(([k, v]) => {
      const key = prefix ? `${prefix}.${k}` : k;
      if (Array.isArray(v)) {
        push(key, k, "list");
        v.forEach((item) => {
          if (item && typeof item === "object" && !Array.isArray(item)) {
            walk(item as Record<string, unknown>, `${key}[]`);
          }
        });
      } else if (v && typeof v === "object") {
        walk(v as Record<string, unknown>, key);
      } else {
        push(key, k, typeof v === "number" ? "number" : "text");
      }
    });
  };
  walk(data, "");
  return out;
}

function parseSample(text?: string): Record<string, unknown> {
  if (!text || !text.trim()) return {};
  try {
    const v = JSON.parse(text);
    return v && typeof v === "object" && !Array.isArray(v) ? v : {};
  } catch {
    return {};
  }
}

// 示例数据变化 → 重派生字段面板
watch(
  () => formData.value.sampleData,
  (text) => {
    fields.value = fieldsFromSample(parseSample(text));
  }
);

// —— 暗色主题：画布亮色值定义在 .designer-container 上（scoped），宿主注入 !important 覆盖块。
// 与钉死的 1.3.11 类名耦合；fork 内核应加正式 data-theme 块替代（docs/print_template.md）。
const DARK_CSS =
  ".designer-container{--pd-bg:#0b0f19!important;--pd-surface:#111827!important;--pd-sidebar:#111827!important;--pd-sidebar-hover:#1f2937!important;--pd-field-bg:#0d1420!important;--pd-input-bg:#0d1420!important;--pd-hover-tint:rgba(255,255,255,.06)!important;--pd-canvas-bg:#0a0e17!important;--pd-text:#e6e8ef!important;--pd-text-regular:#d1d5db!important;--pd-text-muted:#9aa3b2!important;--pd-text-faint:#6b7280!important;--pd-border:#2a3444!important;--pd-border-soft:#232c3a!important;--pd-accent:#4a8dff!important;--pd-accent-soft:rgba(74,141,255,.16)!important;--pd-accent-2:#7c9dff!important;--pd-accent-secondary:#7c9dff!important;--pd-ruler-bg:#0d1420!important;--pd-ruler:#2a3444!important;--pd-ruler-cursor:#f87171!important;--pd-paper:#ffffff!important;--pd-shadow-paper:rgba(0,0,0,.55)!important;--pd-scrollbar-thumb:#2a3444!important;--pd-scrollbar-thumb-hover:#3a4557!important;--pd-menu-text:#e6e8ef!important;--pd-menu-muted:#9aa3b2!important;--pd-menu-hover:#1f2937!important;--pd-menu-divider:#232c3a!important;--pd-menu-disabled:#4b5563!important;--pd-menu-border:#232c3a!important;--pd-shadow-sm:0 1px 2px rgba(0,0,0,.5)!important;--pd-shadow-md:0 4px 12px rgba(0,0,0,.5)!important}";

function syncDarkStyle() {
  const id = "pd-host-dark-theme";
  let style = document.getElementById(id);
  const isDark = document.documentElement.classList.contains("dark");
  if (isDark && !style) {
    style = document.createElement("style");
    style.id = id;
    style.textContent = DARK_CSS;
    document.head.appendChild(style);
  } else if (!isDark && style) {
    style.remove();
  }
}

let classObserver: MutationObserver | null = null;
watch(visible, (v) => {
  dirty.value = false;
  if (v) {
    syncDarkStyle();
    classObserver = new MutationObserver(syncDarkStyle);
    classObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    // 等画布挂载后注入初始模板
    window.setTimeout(() => {
      initialTemplate.value =
        mode.value === "create"
          ? (createDefaultTemplate() as any)
          : safeParse((currentRow.value?.payload || "") as string);
      formData.value = {
        name: mode.value === "copy" ? `${currentRow.value?.name || ""} ` + $t("pages.printTemplate.copySuffix") : currentRow.value?.name || "",
        code: mode.value === "copy" ? "" : currentRow.value?.code || "",
        resourceKey: currentRow.value?.resourceKey || "",
        paperSize: currentRow.value?.paperSize || "A4",
        orientation: currentRow.value?.orientation || "portrait",
        isEnabled: currentRow.value?.isEnabled ?? true,
        sortOrder: currentRow.value?.sortOrder ?? 100,
        sampleData: "",
      };
      fields.value = [];
    }, 120);
  } else {
    classObserver?.disconnect();
    classObserver = null;
  }
});

function safeParse(text: string): TemplateData | MultiPageTemplateData {
  try {
    return JSON.parse(text);
  } catch {
    return createDefaultTemplate() as any;
  }
}

function currentTemplateJson(): string | null {
  try {
    return canvasRef.value?.getTemplateJson() ?? null;
  } catch {
    return null;
  }
}

async function handlePreview() {
  const json = currentTemplateJson();
  if (!json) {
    ElMessage.warning($t("pages.printTemplate.designerNotReady"));
    return;
  }
  previewVisible.value = true;
  previewLoading.value = true;
  try {
    const result = await renderHtmlPages(
      JSON.parse(json),
      parseSample(formData.value.sampleData),
      window.location.origin,
      browserCodeRenderer
    );
    window.setTimeout(() => {
      const doc = previewFrameRef.value?.contentWindow?.document;
      if (doc) {
        doc.open();
        doc.write(result.html);
        doc.close();
      }
      previewLoading.value = false;
    }, 80);
  } catch (e) {
    previewLoading.value = false;
    ElMessage.error((e as Error).message || $t("pages.printTemplate.renderFailed"));
  }
}

function handlePreviewPrint() {
  previewFrameRef.value?.contentWindow?.print();
}

async function handleSave() {
  await formRef.value?.validate();
  const json = currentTemplateJson();
  if (!json) {
    ElMessage.warning($t("pages.printTemplate.designerNotReady"));
    return;
  }
  let redundant: any = {};
  try {
    const tpl = JSON.parse(json);
    redundant = { paperSize: tpl.paperSize, orientation: tpl.orientation };
  } catch {
    /* 保底走表单值 */
  }
  const values: Record<string, any> = {
    name: formData.value.name,
    code: formData.value.code,
    resourceKey: formData.value.resourceKey || "",
    payload: json,
    paperSize: redundant.paperSize || formData.value.paperSize,
    orientation: redundant.orientation || formData.value.orientation,
    isEnabled: formData.value.isEnabled,
    sortOrder: formData.value.sortOrder,
  };

  submitLoading.value = true;
  try {
    if (mode.value === "edit" && currentRow.value?.id) {
      await updateTemplate({ id: currentRow.value.id, values });
      ElMessage.success($t("pages.printTemplate.updateSuccess"));
    } else {
      await createTemplate({ data: { ...values, isBuiltIn: false } as any });
      ElMessage.success($t("pages.printTemplate.createSuccess"));
    }
    dirty.value = false;
    visible.value = false;
    emit("success");
  } finally {
    submitLoading.value = false;
  }
}

/** 打开设计器（create / edit / copy 三态；copy 预填并在保存时走新建） */
function open(data?: { create?: boolean; row?: any; copy?: boolean }) {
  if (data?.copy && data.row) {
    mode.value = "copy";
    currentRow.value = data.row;
  } else if (!data?.create && data?.row) {
    mode.value = "edit";
    currentRow.value = data.row;
  } else {
    mode.value = "create";
    currentRow.value = null;
  }
  visible.value = true;
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.designer-form {
  padding-right: 10px;
}

.designer-host {
  height: calc(100vh - 320px);
  min-height: 420px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.preview-body {
  height: 68vh;
  background: var(--el-fill-color-light);
  overflow: auto;
}

.preview-loading {
  padding: 24px;
}

.preview-frame {
  width: 100%;
  height: 68vh;
  border: none;
}
</style>
