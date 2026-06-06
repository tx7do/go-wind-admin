<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 启用状态 -->
      <template #isEnabled="scope: any">
        <ElTag size="small" :type="scope.row.isEnabled ? 'success' : 'info'" effect="plain">
          {{ enableBoolToName(scope.row.isEnabled) }}
        </ElTag>
      </template>
      <!-- 标签（多语言） -->
      <template #entryLabel="scope: any">
        {{ getEntryLabel(scope.row) }}
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <DictEntryDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import DictEntryDrawer from "./dict-entry-drawer.vue";

import { enableBoolToName, useDeleteDictEntry } from "@/api/composables";
import { $t } from "@/core/i18n";
import { getEntryLabel, useDictViewStore } from "@/pages/app/system/dict/dict-view.state";

const { mutateAsync: deleteDictEntry } = useDeleteDictEntry();
const dictViewStore = useDictViewStore();

const pageRef = ref();
const drawerRef = ref();

// 监听字典类型切换,自动刷新字典项列表
watch(
  () => dictViewStore.needReloadEntryList,
  (needReload) => {
    if (needReload && pageRef.value) {
      pageRef.value.refresh();
      dictViewStore.needReloadEntryList = false;
    }
  }
);

// 初始化时加载一次
onMounted(() => {
  if (dictViewStore.currentTypeId) {
    dictViewStore.needReloadEntryList = true;
  }
});

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.dict.entryValue"),
        field: "entry_value",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await dictViewStore.fetchEntryList(
        dictViewStore.currentTypeId,
        page || 1,
        pageSize || 10,
        queryParams
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteDictEntry({ ids: [ids] as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      {
        prop: "entryLabel",
        label: $t("pages.dict.entryLabel"),
        minWidth: 95,
        slotName: "entryLabel",
        align: "left",
      },
      {
        prop: "entryValue",
        label: $t("pages.dict.entryValue"),
        minWidth: 95,
        align: "left",
      },
      {
        prop: "numericValue",
        label: $t("pages.dict.numericValue"),
        minWidth: 95,
        align: "right",
      },
      {
        prop: "sortOrder",
        label: $t("common.table.sortOrder"),
        width: 95,
        align: "right",
      },
      {
        prop: "isEnabled",
        label: $t("common.table.status"),
        width: 95,
        slotName: "isEnabled",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          {
            name: "delete",
            label: $t("common.button.delete"),
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
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
