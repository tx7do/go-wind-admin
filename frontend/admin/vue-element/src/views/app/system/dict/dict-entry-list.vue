<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <!-- 搜索 -->
    <PageSearch
      ref="searchRef"
      :search-config="searchConfig"
      @query-click="handleQueryClick"
      @reset-click="handleResetClick"
    />

    <!-- 列表 -->
    <PageContent
      ref="contentRef"
      :content-config="contentConfig"
      @add-click="handleAddClick"
      @operate-click="handleOperateClick"
      @toolbar-click="handleToolbarClick"
    >
      <!-- 启用状态 -->
      <template #isEnabled="{ row }">
        <ElTag size="small" effect="dark" round :color="enableBoolToColor(row.isEnabled)">
          {{ enableBoolToName(row.isEnabled) }}
        </ElTag>
      </template>
      <!-- 标签（多语言） -->
      <template #entryLabel="{ row }">
        {{ getEntryLabel(row) }}
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <DictEntryDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElTag, ElMessage, ElMessageBox } from "element-plus";
import { watch, onMounted } from "vue";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import DictEntryDrawer from "./dict-entry-drawer.vue";

import { enableBoolToColor, enableBoolToName, useDeleteDictEntry } from "@/api/composables";
import { $t } from "@/i18n";
import { getEntryLabel, useDictViewStore } from "@/views/app/system/dict/dict-view.state";

const { mutateAsync: deleteDictEntry } = useDeleteDictEntry();
const dictViewStore = useDictViewStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true,
  formItems: [
    {
      type: "input",
      label: $t("pages.dict.entryValue"),
      prop: "entry_value",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:dict_entry",
  toolbarRight: ["add"],
  defaultToolbar: ["refresh", "filter"],
  table: {
    border: true,
    stripe: true,
    height: "auto",
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await dictViewStore.fetchEntryList(
      dictViewStore.currentTypeId,
      page || 1,
      pageSize || 10,
      queryParams
    );
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
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
    },
    {
      prop: "sortOrder",
      label: $t("common.table.sortOrder"),
      width: 95,
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
      template: "tool",
      action: [
        {
          name: "edit",
          text: $t("common.button.edit"),
        },
        {
          name: "delete",
          text: $t("common.button.delete"),
          attrs: {
            type: "danger",
          },
        },
      ],
    },
  ],
};

// 新增按钮点击
function handleAddClick() {
  drawerRef.value?.open({ create: true });
}

// 操作按钮点击
async function handleOperateClick(data: IOperateData) {
  const { name, row } = data;

  if (name === "edit") {
    drawerRef.value?.open({ create: false, row });
  } else if (name === "delete") {
    try {
      await ElMessageBox.confirm(
        $t("common.message.confirmDelete", { moduleName: $t("pages.dict.moduleName") }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );

      await deleteDictEntry({ ids: [row.id] });
      ElMessage.success($t("common.notification.deleteSuccess"));
      handleSuccess();
    } catch (error) {
      if (error !== "cancel") {
        ElMessage.error($t("common.notification.deleteFailed"));
      }
    }
  }
}

// 工具栏按钮点击
function handleToolbarClick(name: string) {
  console.log("toolbar click:", name);
}

// 成功回调
function handleSuccess() {
  contentRef.value?.fetchPageData();
}

// 监听字典类型切换,自动刷新字典项列表
watch(
  () => dictViewStore.needReloadEntryList,
  (needReload) => {
    if (needReload && contentRef.value) {
      contentRef.value.fetchPageData({}, true);
      dictViewStore.needReloadEntryList = false;
    }
  }
);

// 初始化时加载一次
onMounted(() => {
  // 如果有初始字典类型ID,加载字典项列表
  if (dictViewStore.currentTypeId) {
    dictViewStore.needReloadEntryList = true;
  }
});
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
