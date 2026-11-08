<template>
  <div class="app-container h-full flex flex-1 flex-col">
  <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit" @operate="handleOperate">
    <!-- 状态 -->
    <template #status="scope: any">
      <ElTag size="small" :type="scope.row.status === 'ON' ? 'success' : 'info'">
        {{ scope.row.status === "ON" ? t("pages.access_key.statusMap.ON") : t("pages.access_key.statusMap.OFF") }}
      </ElTag>
    </template>
  </ProPage>

  <!-- 创建/编辑抽屉 -->
  <AccessKeyDrawer ref="drawerRef" @success="handleSuccess" />

  <!-- 重置后的新密钥（一次性展示） -->
  <ElDialog
    v-model="secretVisible"
    :title="t('pages.access_key.secretDialogTitle')"
    width="560px"
    align-center
    :close-on-click-modal="false"
    @closed="handleSecretClosed"
  >
    <ElAlert type="warning" :closable="false" :title="t('pages.access_key.secretDialogHint')" />
    <ElInput :model-value="createdSecret" readonly class="secret-input">
      <template #append>
        <ElButton @click="copySecret">{{ t('pages.access_key.secretCopied') }}</ElButton>
      </template>
    </ElInput>
    <template #footer>
      <ElButton type="primary" @click="secretVisible = false">
        {{ t("common.button.confirm") }}
      </ElButton>
    </template>
  </ElDialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag, ElDialog, ElAlert, ElInput, ElButton, ElMessage, ElMessageBox } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import AccessKeyDrawer from "./access-key-drawer.vue";
import { useI18n } from "@/core/i18n";
import {
  deleteAccessKey,
  fetchListAccessKeys,
  resetAccessKeySecret,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();
const createdSecret = ref("");
const secretVisible = ref(false);

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: t("pages.access_key.name"),
        field: "name",
        attrs: { placeholder: t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListAccessKeys(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        }),
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteAccessKey(ids);
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      { prop: "name", label: t("pages.access_key.name"), minWidth: 160 },
      { prop: "accessKey", label: t("pages.access_key.accessKey"), minWidth: 220 },
      {
        prop: "status",
        label: t("pages.access_key.status"),
        minWidth: 90,
        slotName: "status",
      },
      {
        prop: "expiresAt",
        label: t("pages.access_key.expiresAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "lastUsedAt",
        label: t("pages.access_key.lastUsedAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "createdAt",
        label: t("common.table.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: t("common.table.action"),
        fixed: "right",
        // 编辑+重置密钥+删除 三钮实测需 183px，150 会截断
        width: 190,
        cellType: "tool",
        buttons: [
          { name: "edit", label: t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "reset", label: t("pages.access_key.resetSecret"), icon: "lucide:key-round" },
          {
            name: "delete",
            label: t("common.button.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
          },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}

function copySecret() {
  navigator.clipboard?.writeText(createdSecret.value).catch((err: unknown) => {
    console.error("copy secret failed:", err);
  });
  ElMessage.success(t("pages.access_key.secretCopied"));
}

function handleSecretClosed() {
  createdSecret.value = "";
  handleSuccess();
}

async function handleOperate(data: { name: string; row: any }) {
  if (data.name !== "reset") return;
  try {
    await ElMessageBox.confirm(t("pages.access_key.resetConfirm"), t("pages.access_key.resetSecret"), {
      type: "warning",
    });
  } catch {
    return; // 用户取消：不是错误
  }
  try {
    const resp = await resetAccessKeySecret(data.row.id);
    createdSecret.value = resp.secret ?? "";
    secretVisible.value = true;
    ElMessage.success(t("pages.access_key.resetSuccess"));
  } catch (err: any) {
    console.error("reset secret failed:", err);
    ElMessage.error(err?.message || t("pages.access_key.resetFailed"));
  }
}
</script>

<style lang="scss" scoped>
:deep(.el-tag) {
  text-transform: none;
}
</style>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}

.secret-input {
  margin-top: 12px;
}
</style>
