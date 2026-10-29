<template>
  <ProModal
    v-model:visible="visible"
    :close-guard="closeGuard"
    :title="title"
    :config="{
      component: 'drawer',
      drawer: {
        size: '100%',
        // class 走 EP $attrs 合并到 .el-drawer 面板（供全局样式把 body 变 flex 撑满）；
        // EP 的 DrawerProps 类型漏声明了该 attr，运行时有效
        // @ts-expect-error class 未在 EP DrawerProps 类型中声明
        class: 'bi-designer-drawer',
      },
    }"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="100px"
      class="designer-form"
    >
      <ElRow :gutter="16">
        <ElCol :span="7">
          <ElFormItem :label="$t('pages.biDashboard.name')" prop="name">
            <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
          </ElFormItem>
        </ElCol>
        <ElCol :span="7">
          <ElFormItem :label="$t('pages.biDashboard.code')" prop="code">
            <ElInput v-model="formData.code" placeholder="biz.sales.board" clearable />
          </ElFormItem>
        </ElCol>
        <ElCol :span="7">
          <ElFormItem :label="$t('pages.biDashboard.remark')" prop="remark">
            <ElInput v-model="formData.remark" clearable />
          </ElFormItem>
        </ElCol>
        <ElCol :span="3">
          <ElFormItem :label="$t('pages.biDashboard.isEnabled')" prop="isEnabled">
            <ElSwitch v-model="formData.isEnabled" />
          </ElFormItem>
        </ElCol>
      </ElRow>
    </ElForm>

    <!-- 画布容器：任何指针/键盘交互视为脏 -->
    <div
      class="designer-host"
      @pointerdown.capture="markDirty"
      @keydown.capture="markDirty"
    >
        <!-- 不能写 display:block：会压过组件 scoped 的 display:flex，
             内部 __body flex:1 失效 → 画布塌成内容高、无法滚动、抽屉里不撑满 -->
        <BiDesigner
          :key="canvasKey"
          :demo="mode === 'create'"
          :initial-schema="initialSchema"
          style="height: 100%"
          @change="handleSchemaChange"
          />

    </div>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="requestClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSave">
          {{ $t("common.button.save") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
/**
 * BI 设计器抽屉（ele 端）：上部基础信息 + 下部 @go-wind/bi-vue 画布。
 * edit 态载入行 payload；保存整份覆盖 schema，edit 走 version 乐观锁（409 提示刷新）。
 */
import { computed, ref } from "vue";
import {
  ElButton,
  ElCol,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElRow,
  ElSwitch,
} from "element-plus";
import { BiDesigner } from "@go-wind/bi-vue";
import "@go-wind/bi-vue/style.css";
import type { BiDashboardSchema } from "@go-wind/bi-core";

import ProModal from "@/components/Pro/ProModal/index.vue";
import {
  useCreateBiDashboard,
  useUpdateBiDashboard,
} from "@/api/composables/bi_dashboard";
import { $t } from "@/core/i18n";

const emit = defineEmits<{ success: [] }>();

const { mutateAsync: createDashboard } = useCreateBiDashboard();
const { mutateAsync: updateDashboard } = useUpdateBiDashboard();

const visible = ref(false);
const mode = ref<"create" | "edit">("create");
const currentRow = ref<any>(null);
const formRef = ref();
const submitLoading = ref(false);
const dirty = ref(false);

const formData = ref({
  name: "",
  code: "",
  remark: "",
  isEnabled: true,
  isPublic: false,
});

const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

const title = computed(() =>
  mode.value === "create" ? $t("pages.biDashboard.createTitle") : $t("pages.biDashboard.editTitle")
);

/** 画布 key：换行编辑时强制重挂载（initialSchema 是挂载快照，不回灌） */
const canvasKey = computed(() => `${mode.value}-${currentRow.value?.id ?? "new"}`);

const initialSchema = ref<Partial<BiDashboardSchema> | undefined>(undefined);

/** 画布抛出的最新 schema（保存时整份覆盖） */
const schemaRef = ref<BiDashboardSchema | null>(null);

function markDirty() {
  dirty.value = true;
}

function handleSchemaChange(schema: BiDashboardSchema) {
  schemaRef.value = schema;
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

function open(options: { row?: any }) {
  dirty.value = false;
  schemaRef.value = null;
  if (options.row) {
    mode.value = "edit";
    currentRow.value = options.row;
    formData.value = {
      name: options.row.name || "",
      code: options.row.code || "",
      remark: options.row.remark || "",
      isEnabled: options.row.isEnabled ?? true,
      isPublic: options.row.isPublic ?? false,
    };
    if (options.row.payload) {
      try {
        initialSchema.value = JSON.parse(options.row.payload);
      } catch (error) {
        // 脏 payload 不阻塞打开：记原始错误，按空白画布编辑
        console.error("[bi-designer] payload 解析失败，按空白画布打开", error);
        initialSchema.value = undefined;
      }
    } else {
      initialSchema.value = undefined;
    }
  } else {
    mode.value = "create";
    currentRow.value = null;
    initialSchema.value = undefined;
    formData.value = {
      name: "",
      code: "",
      remark: "",
      isEnabled: true,
      isPublic: false,
    };
  }
  visible.value = true;
}

async function handleSave() {
  const valid = await formRef.value?.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  const schema = schemaRef.value;
  if (!schema) {
    ElMessage.warning($t("pages.biDashboard.designerNotReady"));
    return;
  }
  const payload = JSON.stringify(schema);
  const chartCount = schema.charts?.length ?? 0;
  const themeKey = schema.theme?.key ?? "";

  submitLoading.value = true;
  try {
    if (mode.value === "edit" && currentRow.value?.id) {
      await updateDashboard({
        id: currentRow.value.id,
        values: {
          name: formData.value.name,
          code: formData.value.code,
          remark: formData.value.remark,
          isEnabled: formData.value.isEnabled,
          isPublic: formData.value.isPublic,
          // 乐观锁：回传打开时读到的版本，冲突时后端 409
          version: currentRow.value.version ?? 1,
          payload,
          chartCount,
          themeKey,
        },
      });
      ElMessage.success($t("pages.biDashboard.updateSuccess"));
    } else {
      await createDashboard({
        data: {
          name: formData.value.name,
          code: formData.value.code,
          kind: "DASHBOARD",
          remark: formData.value.remark,
          payload,
          chartCount,
          themeKey,
          isEnabled: formData.value.isEnabled,
          isPublic: formData.value.isPublic,
          isBuiltIn: false,
          sortOrder: 100,
        } as any,
      });
      ElMessage.success($t("pages.biDashboard.createSuccess"));
    }
    dirty.value = false;
    visible.value = false;
    emit("success");
  } catch (error: any) {
    const msg = String(error?.message || "");
    if (msg.includes("modified by another session") || msg.includes("CONFLICT")) {
      ElMessage.error($t("pages.biDashboard.saveConflict"));
    } else {
      // 不吞错：原始错误进控制台；提示语义化兜底
      console.error("[bi-designer] save failed", error);
      ElMessage.error(msg || $t("pages.biDashboard.updateFailed"));
    }
  } finally {
    submitLoading.value = false;
  }
}

defineExpose({ open });
</script>

<style scoped lang="scss">
.designer-form {
  margin-bottom: 8px;
}

.designer-host {
  flex: 1;
  min-height: 0;
}
</style>

<style lang="scss">
// 抽屉 body 高度链（全局块：抽屉 appendToBody 后脱离本组件子树，scoped 够不到）。
// ProModal 的 v-loading 包裹 div 无类名，只能按 body 直子代选中。
// canvas 由 .designer-host flex:1 撑满剩余高度（不再用 calc 魔法数）。
.bi-designer-drawer {
  .el-drawer__body {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .el-drawer__body > div {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
}
</style>
