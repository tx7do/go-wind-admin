<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{
      component: 'drawer',
      drawer: { size: '780px', closeOnClickModal: false },
    }"
  >
    <div class="detail-drawer">
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.createdAt')">
          {{ formatDateTime(data?.createdAt ?? "") }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.action')">
          <ElTag
            v-if="data?.action"
            size="small"
            round
            :type="permissionAuditLogActionToType(data.action)"
          >
            {{ permissionAuditLogActionToName(data.action) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.targetType')">
          {{ data?.targetType || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.targetName')">
          {{ data?.targetName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.targetId')">
          {{ data?.targetId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.operatorName')">
          {{ data?.operatorName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.operatorId')">
          {{ data?.operatorId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.reason')" :span="2">
          {{ data?.reason || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 变更信息 -->
      <ElDivider content-position="left">{{ $t("pages.permission_audit_log.oldValue") }}</ElDivider>
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.oldValue')">
          <pre v-if="data?.oldValue" class="pre-text">{{ data.oldValue }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.newValue')">
          <pre v-if="data?.newValue" class="pre-text">{{ data.newValue }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.ipAddress')">
          {{ data?.ipAddress || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.tenantId')">
          {{ data?.tenantId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.requestId')">
          {{ data?.requestId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.logHash')" :span="2">
          {{ data?.logHash || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.permission_audit_log.signature')" :span="2">
          {{ data?.signature || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>
    </div>
  </ProModal>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElDivider, ElDescriptions, ElDescriptionsItem, ElTag } from "element-plus";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { formatDateTime } from "@/utils";
import { $t } from "@go-wind/admin-element";
import { permissionAuditLogActionToType, permissionAuditLogActionToName } from "@/api/composables";
import type { auditservicev1_PermissionAuditLog } from "@/api/generated/admin/service/v1";

const visible = ref(false);
const title = ref("");
const data = ref<auditservicev1_PermissionAuditLog>();

function open(payload: { row: auditservicev1_PermissionAuditLog }) {
  data.value = payload.row;
  title.value = $t("common.modal.view", { moduleName: $t("pages.permission_audit_log.moduleName") });
  visible.value = true;
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.detail-drawer {
  padding-right: 10px;
}

.pre-text {
  margin: 0;
  max-height: 240px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: inherit;
}
</style>
