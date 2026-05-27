<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @row-click="handleRowClick"
    >
      <!-- 启用状态 -->
      <template #isEnabled="scope: any">
        <ElTag size="small" effect="dark" round :color="enableBoolToColor(scope.row.isEnabled)">
          {{ enableBoolToName(scope.row.isEnabled) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <DictTypeDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import DictTypeDrawer from "./dict-type-drawer.vue";

import { enableBoolToColor, enableBoolToName, useDeleteDictType } from "@/api/composables";
import { $t } from "@/i18n";
import { useDictViewStore } from "@/views/app/system/dict/dict-view.state";

const { mutateAsync: deleteDictType } = useDeleteDictType();
const dictViewStore = useDictViewStore();

const pageRef = ref();
const drawerRef = ref();

const pageConfig: ProPageConfig = {

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.dict.typeCode"),
        field: "type_code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await dictViewStore.fetchTypeList(page || 1, pageSize || 10, queryParams);
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteDictType({ ids: [ids] as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      {
        prop: "typeName",
        label: $t("pages.dict.typeName"),
        minWidth: 150,
        align: "left",
      },
      {
        prop: "typeCode",
        label: $t("pages.dict.typeCode"),
        minWidth: 150,
        align: "left",
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
          { name: "edit", text: $t("common.button.edit") },
          { name: "delete", text: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
    ],
  },
};

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}

// 行点击联动 - 切换字典类型时刷新字典项列表
function handleRowClick(row: any) {
  if (row?.id) {
    dictViewStore.setCurrentTypeId(row.id);
  }
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
