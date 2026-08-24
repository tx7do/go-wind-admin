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
        <ElDescriptionsItem :label="$t('pages.api_audit_log.createdAt')">
          {{ formatDateTime(data?.createdAt ?? "") }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.success')">
          <ElTag
            v-if="data?.success !== undefined && data?.success !== null"
            size="small"
            effect="dark"
            round
            :color="successToColor(data.success)"
          >
            {{ successToNameWithStatusCode(data.success, data?.statusCode ?? 0) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.username')">
          {{ data?.username || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.userId')">
          {{ data?.userId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.httpMethod')">
          <ElTag v-if="data?.httpMethod" size="small" effect="dark" round>
            {{ data.httpMethod }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.statusCode')">
          {{ data?.statusCode ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.path')" :span="2">
          {{ data?.path || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.requestUri')" :span="2">
          {{ data?.requestUri || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.latencyMs')">
          {{ data?.latencyMs ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.ipAddress')">
          {{ data?.ipAddress || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.tenantName')">
          {{ data?.tenantName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.geoLocation')" :span="2">
          {{ data?.geoLocation?.province }} {{ data?.geoLocation?.city }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 设备信息 -->
      <ElDivider content-position="left">{{ $t("pages.api_audit_log.device") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.platform')">
          {{ data?.deviceInfo?.platform || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.osName')">
          {{ data?.deviceInfo?.osName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.browserName')">
          {{ data?.deviceInfo?.browserName || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.userAgent')" :span="2">
          {{ data?.deviceInfo?.userAgent || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 接口信息 -->
      <ElDivider content-position="left">{{ $t("pages.api_audit_log.apiModule") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.apiModule')">
          {{ data?.apiModule || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.apiOperation')">
          {{ data?.apiOperation || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.apiDescription')" :span="2">
          {{ data?.apiDescription || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.referer')" :span="2">
          {{ data?.referer || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.reason')" :span="2">
          {{ data?.reason || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.spanId')">
          {{ data?.spanId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.requestId')">
          {{ data?.requestId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.traceId')">
          {{ data?.traceId || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.logHash')" :span="2">
          {{ data?.logHash || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.signature')" :span="2">
          {{ data?.signature || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.requestBody')" :span="2">
          <pre v-if="data?.requestBody" class="pre-text">{{ data.requestBody }}</pre>
          <template v-else>-</template>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.requestHeader')" :span="2">
          <pre v-if="data?.requestHeader" class="pre-text">{{ data.requestHeader }}</pre>
          <template v-else>-</template>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.api_audit_log.response')" :span="2">
          <pre v-if="data?.response" class="pre-text">{{ data.response }}</pre>
          <template v-else>-</template>
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
import { $t } from "@/core/i18n";
import { successToColor, successToNameWithStatusCode } from "@/api/composables";
import type { auditservicev1_ApiAuditLog } from "@/api/generated/admin/service/v1";

const visible = ref(false);
const title = ref("");
const data = ref<auditservicev1_ApiAuditLog>();

function open(payload: { row: auditservicev1_ApiAuditLog }) {
  data.value = payload.row;
  title.value = $t("pages.api_audit_log.moduleName");
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
