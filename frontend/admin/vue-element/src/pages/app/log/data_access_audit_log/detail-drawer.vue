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
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.createdAt')">
          {{ formatDateTime(data?.createdAt ?? "") }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.success')">
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
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.accessType')">
          <ElTag
            v-if="data?.accessType"
            size="small"
            round
            :type="dataAccessAuditLogAccessTypeToType(data.accessType)"
          >
            {{ dataAccessAuditLogAccessTypeToName(data.accessType) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.sensitiveLevel')">
          {{ data?.sensitiveLevel ? $t(`enum.sensitiveLevel.${data.sensitiveLevel}`) : "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.tableName')">
          {{ data?.tableName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.dataCategory')">
          {{ data?.dataCategory || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.username')">
          {{ data?.username || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.userId')">
          {{ data?.userId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.tenantName')">
          {{ data?.tenantName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.latencyMs')">
          {{ data?.latencyMs ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.ipAddress')">
          {{ data?.ipAddress || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- SQL与数据 -->
      <ElDivider content-position="left">{{ $t("pages.data_access_audit_log.sqlText") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.dbUser')">
          {{ data?.dbUser || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.dataSource')">
          {{ data?.dataSource || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.sqlDigest')" :span="2">
          {{ data?.sqlDigest || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.sqlText')" :span="2">
          <pre v-if="data?.sqlText" class="pre-text">{{ data.sqlText }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.dataId')">
          {{ data?.dataId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.affectedRows')">
          {{ data?.affectedRows ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.dataMasked')">
          {{
            data?.dataMasked === true
              ? $t("enum.enable.true")
              : data?.dataMasked === false
                ? $t("enum.enable.false")
                : "-"
          }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.businessPurpose')">
          {{ data?.businessPurpose || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.maskingRules')" :span="2">
          {{ data?.maskingRules || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.requestId')">
          {{ data?.requestId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.logHash')" :span="2">
          {{ data?.logHash || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.data_access_audit_log.signature')" :span="2">
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
  dataAccessAuditLogAccessTypeToType,
  dataAccessAuditLogAccessTypeToName,
} from "@/api/composables";
import type { auditservicev1_DataAccessAuditLog } from "@/api/generated/admin/service/v1";

const visible = ref(false);
const title = ref("");
const data = ref<auditservicev1_DataAccessAuditLog>();

function open(payload: { row: auditservicev1_DataAccessAuditLog }) {
  data.value = payload.row;
  title.value = $t("common.modal.view", { moduleName: $t("pages.data_access_audit_log.moduleName") });
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
