<template>
  <ElDialog
    v-model="visible"
    :title="$t('pages.biDashboard.revisionTitle')"
    width="720px"
    :close-on-click-modal="false"
  >
    <ElTable v-loading="loading" :data="rows" size="small" max-height="420">
      <ElTableColumn :label="$t('pages.biDashboard.revisionVersion')" width="80">
        <template #default="scope">
          <ElTag size="small" type="primary">v{{ scope.row.version }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('pages.biDashboard.name')" prop="name" show-overflow-tooltip />
      <ElTableColumn :label="$t('pages.biDashboard.revisionSource')" width="100">
        <template #default="scope">
          <ElTag v-if="scope.row.source === 'RESTORE'" size="small" type="warning">
            {{ $t("pages.biDashboard.revisionSourceRestore") }}
          </ElTag>
          <ElTag v-else size="small" type="info">
            {{ $t("pages.biDashboard.revisionSourceSave") }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('common.table.createdAt')" prop="createdAt" width="160" />
      <ElTableColumn :label="$t('pages.biDashboard.action')" width="90" fixed="right">
        <template #default="scope">
          <ElButton link type="primary" @click="handleRestore(scope.row as biservicev1_BiDashboardRevision)">
            {{ $t("pages.biDashboard.revisionRestore") }}
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>
  </ElDialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import {
  ElButton,
  ElDialog,
  ElMessage,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";
import type { biservicev1_BiDashboardRevision } from "@/api/generated/admin/service/v1";
import {
  useBiDashboardRevisions,
  useRestoreBiDashboardRevision,
} from "@/api/composables/bi_dashboard";
import { queryClient } from "@/plugins/vue-query";
import { $t } from "@/core/i18n";

/**
 * 版本历史弹窗：每次保存留旧版快照（新快照在前），恢复 = 快照写回主表。
 * 恢复前服务端会先以 RESTORE 来源留底当前版本——误恢复可再恢复回来（可逆）。
 */

const visible = ref(false);
const dashboardId = ref<number | undefined>(undefined);

const revisions = useBiDashboardRevisions(
  () => dashboardId.value,
  () => visible.value && !!dashboardId.value
);
const loading = computed(() => revisions.isLoading.value);
const rows = computed<biservicev1_BiDashboardRevision[]>(
  () => (revisions.data.value?.items as any) || []
);

const { mutateAsync: restoreRevision } = useRestoreBiDashboardRevision();

async function handleRestore(row: biservicev1_BiDashboardRevision) {
  try {
    await restoreRevision({ id: row.id });
    ElMessage.success($t("pages.biDashboard.revisionRestoreSuccess"));
    // 恢复会改主表（version+1）：刷新快照列表与列表页
    await queryClient.invalidateQueries({ queryKey: ["biDashboardRevisions", dashboardId.value] });
    await queryClient.invalidateQueries({ queryKey: ["listBiDashboards"] });
  } catch (e) {
    // 错误提示由请求层统一弹出；此处带出原始错误对象防静默吞错
    console.error("restore bi dashboard revision failed:", e);
  }
}

watch(visible, (v) => {
  if (v && dashboardId.value) {
    queryClient.invalidateQueries({ queryKey: ["biDashboardRevisions", dashboardId.value] });
  }
});

defineExpose({
  open(id: number) {
    dashboardId.value = id;
    visible.value = true;
  },
});
</script>
