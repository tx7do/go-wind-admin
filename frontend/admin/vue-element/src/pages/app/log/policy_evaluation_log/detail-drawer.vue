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
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.createdAt')">
          {{ formatDateTime(data?.createdAt ?? "") }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.result')">
          <ElTag
            v-if="data?.result !== undefined && data?.result !== null"
            size="small"
            round
            :type="successToType(data.result)"
          >
            {{ successToName(data.result) }}
          </ElTag>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.requestMethod')">
          {{ data?.requestMethod || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.userId')">
          {{ data?.userId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.requestPath')" :span="2">
          {{ data?.requestPath || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.permissionId')">
          {{ data?.permissionId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.policyId')">
          {{ data?.policyId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.membershipId')">
          {{ data?.membershipId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.tenantId')">
          {{ data?.tenantId ?? "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.ipAddress')">
          {{ data?.ipAddress || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.traceId')">
          {{ data?.traceId || "-" }}
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 评估详情 -->
      <ElDivider content-position="left">
        {{ $t("pages.policy_evaluation_log.evaluationContext") }}
      </ElDivider>
      <ElDescriptions :column="1" border>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.evaluationContext')">
          <pre v-if="data?.evaluationContext" class="pre-text">{{ data.evaluationContext }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.effectDetails')">
          <pre v-if="data?.effectDetails" class="pre-text">{{ data.effectDetails }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.scopeSql')">
          <pre v-if="data?.scopeSql" class="pre-text">{{ data.scopeSql }}</pre>
          <span v-else>-</span>
        </ElDescriptionsItem>
      </ElDescriptions>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>
      <ElDescriptions :column="2" border>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.logHash')" :span="2">
          {{ data?.logHash || "-" }}
        </ElDescriptionsItem>
        <ElDescriptionsItem :label="$t('pages.policy_evaluation_log.signature')" :span="2">
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
import { successToType, successToName } from "@/api/composables";
import type { permissionservicev1_PolicyEvaluationLog } from "@/api/generated/admin/service/v1";

const visible = ref(false);
const title = ref("");
const data = ref<permissionservicev1_PolicyEvaluationLog>();

function open(payload: { row: permissionservicev1_PolicyEvaluationLog }) {
  data.value = payload.row;
  title.value = $t("common.modal.view", { moduleName: $t("pages.policy_evaluation_log.moduleName") });
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
