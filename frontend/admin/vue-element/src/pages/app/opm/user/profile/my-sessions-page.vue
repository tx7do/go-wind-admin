<template>
  <div class="my-sessions-page">
    <div class="page-hint">{{ $t("pages.online_session.mySessionsHint") }}</div>
    <el-table
      v-loading="isLoading"
      :data="items"
      border
      stripe
      row-key="jti"
      :empty-text="$t('pages.online_session.noSessions')"
    >
      <el-table-column :label="$t('pages.online_session.clientType')" width="110">
        <template #default="{ row }">
          <el-tag :type="row.clientType === 'app' ? 'danger' : 'primary'" size="small">
            {{
              row.clientType === "app"
                ? $t("pages.online_session.clientApp")
                : $t("pages.online_session.clientAdmin")
            }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('pages.online_session.ipAddress')" width="150">
        <template #default="{ row }">{{ row.ipAddress || "-" }}</template>
      </el-table-column>
      <el-table-column :label="$t('pages.online_session.userAgent')" min-width="220">
        <template #default="{ row }">
          <span class="ua-cell">{{ row.userAgent || "-" }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('pages.online_session.loginAt')" width="180">
        <template #default="{ row }">{{ formatDateTime(row.loginAt ?? "") }}</template>
      </el-table-column>
      <el-table-column :label="$t('pages.online_session.status')" width="140">
        <template #default="{ row }">
          <el-tag v-if="row.current" type="success" size="small">
            {{ $t("pages.online_session.currentSession") }}
          </el-tag>
          <el-popconfirm
            :title="$t('pages.online_session.revokeConfirm')"
            :confirm-button-text="$t('pages.online_session.revoke')"
            :cancel-button-text="$t('common.button.cancel')"
            @confirm="handleRevoke(row)"
          >
            <template #reference>
              <el-button type="danger" link size="small">
                {{ $t("pages.online_session.revoke") }}
              </el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { ElMessage } from "element-plus";
import { formatDateTime } from "@/utils";
import { $t } from "@go-wind/admin-element";
import {
  useListMyOnlineSessions,
  useRevokeMyOnlineSession,
} from "@/api/composables";
import type { online_sessionservicev1_OnlineSession } from "@/api/generated/admin/service/v1";

const { data, isLoading } = useListMyOnlineSessions();
const { mutateAsync: revoke } = useRevokeMyOnlineSession();

// 通过 computed 派生，保证 useQuery 数据返回后视图响应式更新
const items = computed(
  () => (data.value?.items ?? []) as online_sessionservicev1_OnlineSession[]
);

async function handleRevoke(row: online_sessionservicev1_OnlineSession) {
  if (!row.jti) return;
  try {
    await revoke({ clientType: row.clientType, jti: row.jti });
    ElMessage.success($t("pages.online_session.revokeSuccess"));
  } catch (error: any) {
    console.error("Failed to revoke session:", error);
    ElMessage.error(error?.message || $t("pages.online_session.revokeFailed"));
  }
}
</script>

<style lang="scss" scoped>
.my-sessions-page {
  width: 100%;
}
.page-hint {
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.ua-cell {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}
</style>
