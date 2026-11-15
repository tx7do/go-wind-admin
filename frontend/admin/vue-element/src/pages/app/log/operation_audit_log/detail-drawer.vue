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
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.createdAt')">
          {{ formatDateTime(data?.createdAt ?? "") }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.success')">
          <ElTag
            v-if="data?.success !== undefined && data?.success !== null"
            size="small"
            round
            :type="successToType(data.success)"
          >
            {{ successToName(data.success) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.action')">
          <ElTag
            v-if="data?.action"
            size="small"
            round
            :type="operationAuditLogActionToType(data.action)"
          >
            {{ operationAuditLogActionToName(data.action) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.sensitiveLevel')">
          {{ data?.sensitiveLevel ? $t(`enum.sensitiveLevel.${data.sensitiveLevel}`) : "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.resourceType')">
          {{ data?.resourceType || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.resourceId')">
          {{ data?.resourceId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.username')">
          {{ data?.username || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.userId')">
          {{ data?.userId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.tenantName')">
          {{ data?.tenantName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.ipAddress')">
          {{ data?.ipAddress || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.geoLocation')" :span="2">
          {{ data?.geoLocation?.province }} {{ data?.geoLocation?.city }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 数据变更 -->
      <ElDivider content-position="left">
        {{ $t("pages.operation_audit_log.beforeData") }}
      </ElDivider>
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.beforeData')">
          <pre v-if="data?.beforeData" class="pre-text">{{ data.beforeData }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.afterData')">
          <pre v-if="data?.afterData" class="pre-text">{{ data.afterData }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.failureReason')">
          {{ data?.failureReason || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.requestId')">
          {{ data?.requestId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.traceId')">
          {{ data?.traceId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.logHash')" :span="2">
          {{ data?.logHash || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.operation_audit_log.signature')" :span="2">
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
import {
  successToType,
  successToName,
  operationAuditLogActionToType,
  operationAuditLogActionToName,
} from "@/api/composables";
import type { auditservicev1_OperationAuditLog } from "@/api/generated/admin/service/v1";

const visible = ref(false);
const title = ref("");
const data = ref<auditservicev1_OperationAuditLog>();

function open(payload: { row: auditservicev1_OperationAuditLog }) {
  data.value = payload.row;
  title.value = $t("common.modal.view", { moduleName: $t("pages.operation_audit_log.moduleName") });
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
