<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @operate="handleOperate">
      <!-- 租户 -->
      <template #tenant="scope: any">
        <ElTag v-if="(scope.row.tenantId ?? 0) === 0" size="small">
          {{ $t("pages.online_session.platform") }}
        </ElTag>
        <span v-else>#{{ scope.row.tenantId }}</span>
      </template>

      <!-- 客户端类型 -->
      <template #clientType="scope: any">
        <ElTag size="small" :type="scope.row.clientType === 'app' ? 'danger' : 'primary'">
          {{
            scope.row.clientType === "app"
              ? $t("pages.online_session.clientApp")
              : $t("pages.online_session.clientAdmin")
          }}
        </ElTag>
      </template>

      <!-- 浏览器/设备 -->
      <template #userAgent="scope: any">
        <span class="ua-cell" :title="scope.row.userAgent || ''">
          {{ scope.row.userAgent || "-" }}
        </span>
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElMessage, ElMessageBox, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import type { online_sessionservicev1_OnlineSession as OnlineSession } from "@/api/generated/admin/service/v1";
import {
  fetchListOnlineSessions,
  useForceLogoutSession,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { $t } from "@go-wind/admin-element";

const { mutateAsync: forceLogout } = useForceLogoutSession();

const pageRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  rowKey: "jti",
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.online_session.keyword"),
        field: "keyword",
        attrs: {
          placeholder: $t("pages.online_session.keywordPlaceholder"),
          clearable: true,
        },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListOnlineSessions({
        page: page || 1,
        pageSize: pageSize || 10,
        keyword: (queryParams.keyword as string) || undefined,
      });
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "username", label: $t("pages.online_session.username"), minWidth: 120 },
      {
        prop: "tenantId",
        label: $t("pages.online_session.tenant"),
        width: 90,
        slotName: "tenant",
      },
      {
        prop: "clientType",
        label: $t("pages.online_session.clientType"),
        width: 100,
        slotName: "clientType",
      },
      { prop: "ipAddress", label: $t("pages.online_session.ipAddress"), minWidth: 130 },
      {
        prop: "userAgent",
        label: $t("pages.online_session.userAgent"),
        minWidth: 200,
        slotName: "userAgent",
      },
      {
        prop: "deviceId",
        label: $t("pages.online_session.deviceId"),
        width: 120,
        formatter: (row: any) => row.deviceId || "-",
      },
      {
        prop: "loginAt",
        label: $t("pages.online_session.loginAt"),
        width: 170,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 120,
        cellType: "tool",
        buttons: [
          {
            name: "forceLogout",
            label: $t("pages.online_session.forceLogout"),
            icon: "lucide:log-out",
            attrs: { type: "danger", link: true },
          },
        ],
      },
    ],
  },
}));

/* 强制下线：确认后吊销该会话的访问/刷新令牌 */
async function handleOperate(data: { name: string; row: OnlineSession }) {
  if (data.name !== "forceLogout") return;
  const row = data.row;
  if (!row.jti || row.userId === undefined) return;

  try {
    await ElMessageBox.confirm(
      $t("pages.online_session.forceLogoutConfirmContent", {
        user: row.username || String(row.userId),
      }),
      $t("pages.online_session.forceLogoutConfirmTitle"),
      {
        confirmButtonText: $t("pages.online_session.forceLogout"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    );
  } catch {
    return; // 用户取消
  }

  try {
    await forceLogout({
      clientType: row.clientType,
      userId: row.userId,
      jti: row.jti,
    });
    ElMessage.success($t("pages.online_session.forceLogoutSuccess"));
    pageRef.value?.refresh();
  } catch (error: any) {
    ElMessage.error(error?.message || $t("pages.online_session.forceLogoutFailed"));
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
.ua-cell {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}
</style>
