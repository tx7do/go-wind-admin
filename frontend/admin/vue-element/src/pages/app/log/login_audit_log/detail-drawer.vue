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
        <ElDescriptionsItem :label="$t('pages.login_audit_log.createdAt')">
          {{ formatDateTime(data?.createdAt ?? "") }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.status')">
          <ElTag
            v-if="data?.status"
            size="small"
            round
            :type="loginAuditLogStatusToType(data.status)"
          >
            {{ loginAuditLogStatusToName(data.status) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.username')">
          {{ data?.username || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.userId')">
          {{ data?.userId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.actionType')">
          <ElTag
            v-if="data?.actionType"
            size="small"
            round
            :type="loginAuditLogActionTypeToType(data.actionType)"
          >
            {{ loginAuditLogActionTypeToName(data.actionType) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.loginMethod')">
          {{ data?.loginMethod ? $t(`enum.loginMethod.${data.loginMethod}`) : "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.riskLevel')">
          <ElTag
            v-if="data?.riskLevel"
            size="small"
            round
            :type="loginAuditLogRiskLevelToType(data.riskLevel)"
          >
            {{ loginAuditLogRiskLevelToName(data.riskLevel) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.riskScore')">
          {{ data?.riskScore ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.ipAddress')">
          {{ data?.ipAddress || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.tenantName')">
          {{ data?.tenantName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.geoLocation')" :span="2">
          {{ data?.geoLocation?.province }} {{ data?.geoLocation?.city }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 设备信息 -->
      <ElDivider content-position="left">{{ $t("pages.login_audit_log.device") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.platform')">
          {{ data?.deviceInfo?.platform || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.osName')">
          {{ data?.deviceInfo?.osName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.browserName')">
          {{ data?.deviceInfo?.browserName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.userAgent')" :span="2">
          {{ data?.deviceInfo?.userAgent || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.mfaStatus')">
          {{ data?.mfaStatus || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.sessionId')">
          {{ data?.sessionId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.failureReason')" :span="2">
          {{ data?.failureReason || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.riskFactors')" :span="2">
          <ElTag
            v-for="factor in data?.riskFactors"
            :key="factor"
            size="small"
            effect="plain"
            round
            class="tag-item"
          >
            {{ factor }}
          </ElTag>
          <span v-if="!data?.riskFactors?.length">-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.requestId')">
          {{ data?.requestId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.traceId')">
          {{ data?.traceId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.logHash')" :span="2">
          {{ data?.logHash || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.login_audit_log.signature')" :span="2">
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
  loginAuditLogStatusToType,
  loginAuditLogActionTypeToType,
  loginAuditLogRiskLevelToType,
  loginAuditLogStatusToName,
  loginAuditLogActionTypeToName,
  loginAuditLogRiskLevelToName,
} from "@/api/composables";
import type { auditservicev1_LoginAuditLog } from "@/api/generated/admin/service/v1";

const visible = ref(false);
const title = ref("");
const data = ref<auditservicev1_LoginAuditLog>();

function open(payload: { row: auditservicev1_LoginAuditLog }) {
  data.value = payload.row;
  title.value = $t("common.modal.view", { moduleName: $t("pages.login_audit_log.moduleName") });
  visible.value = true;
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.detail-drawer {
  padding-right: 10px;
}

.tag-item {
  margin: 0 4px 4px 0;
}
</style>
