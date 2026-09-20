<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 渠道：色由 composable 的枚举映射给，槽里不硬编码 -->
      <template #channel="scope: any">
        <ElTag
          v-if="scope.row.channel"
          size="small"
          round
          :type="notificationRuleChannelToType(scope.row.channel)"
        >
          {{ notificationRuleChannelToName(scope.row.channel) }}
        </ElTag>
        <span v-else>-</span>
      </template>

      <!-- 派发方式：异步/同步的判据就写在提示里，读表的人不必去翻文档 -->
      <template #isAsync="scope: any">
        <ElTooltip :content="t('pages.notification_rule.isAsyncHint')" placement="top">
          <ElTag v-if="scope.row.isAsync" size="small" round type="primary">
            {{ t("pages.notification_rule.isAsyncOn") }}
          </ElTag>
          <ElTag v-else size="small" round>
            {{ t("pages.notification_rule.isAsyncOff") }}
          </ElTag>
        </ElTooltip>
      </template>

      <!-- 启用：停用不是「删了这条路由」，而是「这条路由暂时不生效」，颜色区分开 -->
      <template #isEnabled="scope: any">
        <ElTooltip :content="t('pages.notification_rule.isEnabledHint')" placement="top">
          <ElTag v-if="scope.row.isEnabled" size="small" round type="success">
            {{ t("pages.notification_rule.enabledOn") }}
          </ElTag>
          <ElTag v-else size="small" round>
            {{ t("pages.notification_rule.enabledOff") }}
          </ElTag>
        </ElTooltip>
      </template>
    </ProPage>

    <!-- 创建/编辑抽屉 -->
    <NotificationRuleDrawer ref="drawerRef" @success="handleSuccess" />

    <!-- 测试投递：target 可空（Webhook 留空即取渠道登记的回调地址） -->
    <ElDialog
      v-model="testVisible"
      :title="t('pages.notification_rule.testDispatchTitle', { event: testEventLabel })"
      width="480px"
      align-center
      :close-on-click-modal="false"
      @closed="testTarget = undefined"
    >
      <div class="test-dispatch-tip">{{ t("pages.notification_rule.testDispatchHint") }}</div>
      <ElForm :model="testForm" label-width="120px">
        <ElFormItem :label="t('pages.notification_rule.testTarget')" prop="target">
          <ElInput
            v-model="testForm.target"
            :placeholder="t('pages.notification_rule.testTargetPlaceholder')"
            clearable
          />
          <div class="test-dispatch-field-tip">
            {{ t("pages.notification_rule.testTargetHint") }}
          </div>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="testVisible = false">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="testing" @click="handleTestDispatch">
          {{ t("pages.notification_rule.testDispatch") }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script lang="ts" setup>
/**
 * 通知路由规则管理（事件类型 → 渠道 + 派发方式）
 *
 * 这张表是投递时的唯一真相：改完这一行，下一次投递立刻按新路由走（服务端不缓存）。
 * 两个决定来自同一行，所以「渠道」与「异步」必须一起看：异步 = 请求入队即回，
 * 结论由队列回写台账；同步 = 当场拨号，接口的回话就是投递结论。
 *
 * 删掉一行 = 「这个事件不再通知」：服务端不会在下次启动补回来，但那条事件的调用
 * 会拿到「没有启用路由规则」的报错（不静默丢通知）。
 */
import { computed, ref } from "vue";
import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElTag,
  ElTooltip,
} from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import NotificationRuleDrawer from "./notification-rule-drawer.vue";
import type { notificationservicev1_NotificationRule as NotificationRule } from "@/api/generated/admin/service/v1";
import {
  fetchListNotificationRules,
  notificationRuleChannelOptionList,
  notificationRuleChannelToName,
  notificationRuleChannelToType,
  notificationRuleEventTypeList,
  notificationRuleEventTypeToName,
  useDeleteNotificationRule,
  useTestDispatchNotification,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { useI18n } from "@/core/i18n";

const { t } = useI18n();

const pageRef = ref();
const drawerRef = ref();

const { mutateAsync: deleteNotificationRule } = useDeleteNotificationRule();
const { mutateAsync: testDispatch } = useTestDispatchNotification();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    // 只有两个枚举筛选项：ID/枚举字段一律下拉精确匹配，不进模糊搜索（全仓铁律）
    fields: [
      {
        type: "select",
        label: t("pages.notification_rule.eventType"),
        field: "eventType",
        attrs: {
          placeholder: t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: notificationRuleEventTypeList.value,
      },
      {
        type: "select",
        label: t("pages.notification_rule.channel"),
        field: "channel",
        attrs: {
          placeholder: t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: notificationRuleChannelOptionList.value,
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      try {
        const result = await fetchListNotificationRules(
          new PaginationQuery({
            paging: { page: page || 1, pageSize: pageSize || 20 },
            formValues: {
              eventType: queryParams.eventType,
              channel: queryParams.channel,
            },
          })
        );
        return { items: result.items || [], total: result.total || 0 };
      } catch (error: any) {
        // 不吞错：ElMessage 只是给用户看的文案，排查要靠控制台里的原始错误对象
        console.error("list notification rules failed", error);
        ElMessage.error(error?.message || t("pages.notification_rule.fetchFailed"));
        return { items: [], total: 0 };
      }
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      {
        prop: "eventType",
        label: t("pages.notification_rule.eventType"),
        minWidth: 150,
        formatter: (row: NotificationRule) => notificationRuleEventTypeToName(row.eventType) || "-",
      },
      {
        prop: "channel",
        label: t("pages.notification_rule.channel"),
        width: 110,
        slotName: "channel",
      },
      {
        prop: "isAsync",
        label: t("pages.notification_rule.isAsync"),
        width: 110,
        slotName: "isAsync",
      },
      {
        prop: "isEnabled",
        label: t("pages.notification_rule.isEnabled"),
        width: 100,
        slotName: "isEnabled",
      },
      {
        prop: "remark",
        label: t("pages.notification_rule.remark"),
        minWidth: 240,
        formatter: (row: NotificationRule) => row.remark || "-",
      },
      {
        prop: "updatedAt",
        label: t("pages.notification_rule.updatedAt"),
        width: 170,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: t("common.table.action"),
        fixed: "right",
        width: 240,
        cellType: "tool",
        buttons: [
          { name: "edit", label: t("common.button.edit"), icon: "lucide:pen-line" },
          // 站内信规则不给测试按钮：它需要一条真实的消息本体，填个收件人 ID 测不出东西
          {
            name: "testDispatch",
            label: t("pages.notification_rule.testDispatch"),
            icon: "lucide:zap",
            visible: (row: NotificationRule) => row.channel !== "INTERNAL",
          },
          // 删除按钮刻意不叫 "delete"：ProPage 内置的删除确认只有通用文案，
          // 而这一行的后果（该事件从此不再通知、重启也不补回）必须说清楚
          {
            name: "remove",
            label: t("pages.notification_rule.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
          },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: NotificationRule) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}

async function handleOperate(data: { name: string; row: NotificationRule }) {
  if (data.name === "testDispatch") {
    testTarget.value = data.row;
    testForm.value.target = "";
    testVisible.value = true;
    return;
  }
  if (data.name !== "remove") return;

  const row = data.row;
  if (!row.id) return;

  try {
    await ElMessageBox.confirm(
      t("pages.notification_rule.deleteConfirm"),
      t("common.title.confirm"),
      {
        confirmButtonText: t("common.button.confirm"),
        cancelButtonText: t("common.button.cancel"),
        type: "warning",
        lockScroll: false,
      }
    );
  } catch (error) {
    // 取消是 ElMessageBox 的 reject("cancel")，属正常路径；其余形态（API 误用等）留痕
    if (error !== "cancel" && error !== "close") {
      console.error("delete confirm dialog rejected unexpectedly", error);
    }
    return;
  }

  try {
    await deleteNotificationRule({ id: row.id });
    ElMessage.success(t("pages.notification_rule.deleteSuccess"));
    pageRef.value?.refresh();
  } catch (error: any) {
    // 用户看到的那句翻译不含服务端原因，原始错误必须留在控制台
    console.error("delete notification rule failed", error);
    ElMessage.error(error?.message || t("pages.notification_rule.deleteFailed"));
  }
}

// === 测试投递 ===
const testVisible = ref(false);
const testing = ref(false);
const testTarget = ref<NotificationRule>();
const testForm = ref({ target: "" });

const testEventLabel = computed(() =>
  testTarget.value?.eventType ? notificationRuleEventTypeToName(testTarget.value.eventType) : ""
);

async function handleTestDispatch() {
  if (!testTarget.value?.id) return;
  testing.value = true;
  try {
    // 自定义 RPC（body:"*"）收**扁平**请求体：包一层 { data } 会被 protojson
    // 当未知字段丢掉，接口照样 200、字段全为空
    const resp = await testDispatch({
      id: testTarget.value.id,
      target: testForm.value.target || undefined,
    });
    // 异步规则回的是 SENDING：这次只是「已交给队列」，说「已投递」而不说「已送达」
    ElMessage.success(
      resp.status === "SENDING"
        ? t("pages.notification_rule.testDispatchQueued", { id: resp.deliveryId })
        : t("pages.notification_rule.testDispatchSent", { id: resp.deliveryId })
    );
    testVisible.value = false;
  } catch (error: any) {
    console.error("test dispatch failed", error);
    ElMessage.error(error?.message || t("pages.notification_rule.testDispatchFailed"));
  } finally {
    testing.value = false;
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

.test-dispatch-tip {
  margin-bottom: 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.test-dispatch-field-tip {
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
</style>
