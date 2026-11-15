/**
 * 站内信浮窗（顶栏铃铛下拉）逻辑。
 * 行为以 react 端 HeaderContent 的 inboxContent 为基准：
 * - 未读数走独立 COUNT（pageSize=1 取服务端 total）
 * - 列表为最近 5 条「已读+未读混合」（不按状态过滤）
 * - 点击条目/底部链接进入收件箱页；「全部已读」一键标记
 * - SSE notification 事件触发重拉，另有 5 分钟兜底轮询
 */
import { ref, onMounted, onBeforeUnmount } from "vue";
import {
  fetchListUserInbox,
  useMarkNotificationAsRead,
} from "@/api/composables";
import { useAppUserStore } from '@go-wind/admin-element';
import { PaginationQuery } from "@go-wind/admin-core";
import { router } from "@/router";
import type { internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from "@/api/generated/admin/service/v1";
import { globalSSEClient, SSE_EVENT } from "@go-wind/admin-core";
import { i18n } from "@go-wind/admin-element";

const t = i18n.global.t;

const PAGE_SIZE = 5;
// 兜底轮询：实时更新靠 SSE notification 事件触发重拉，这里只在 SSE 失联时补拉
const POLL_INTERVAL = 300_000;

export function useNotice() {
  const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();
  const userStore = useAppUserStore();

  const list = ref<InternalMessageRecipient[]>([]);
  const unreadTotal = ref(0);
  const listLoading = ref(false);
  const markAllLoading = ref(false);

  async function fetchUnreadCount() {
    const userId = userStore.userInfo?.id;
    if (!userId) return;
    const result = await fetchListUserInbox(
      new PaginationQuery({
        paging: { page: 1, pageSize: 1 },
        // recipient_user_id 必传：不传只靠租户过滤，会查到同租户其他用户的收件记录
        formValues: {
          recipient_user_id: userId.toString(),
          status: "RECEIVED",
        },
      })
    );
    // proto int64 经 protojson 运行时是字符串（TS 类型却标 number），须显式转数字。
    // 未读数用列表响应的服务端 total（独立 COUNT），不能用当前页条数——后者封顶在 pageSize。
    unreadTotal.value = Number(result.total ?? 0);
  }

  async function fetchList() {
    const userId = userStore.userInfo?.id;
    if (!userId) return;
    listLoading.value = true;
    try {
      const result = await fetchListUserInbox(
        new PaginationQuery({
          paging: { page: 1, pageSize: PAGE_SIZE },
          formValues: { recipient_user_id: userId.toString() },
          orderBy: ["-created_at"],
        })
      );
      list.value = result.items || [];
    } finally {
      listLoading.value = false;
    }
  }

  function reloadAll() {
    fetchUnreadCount();
    fetchList();
  }

  async function readAll() {
    const userId = userStore.userInfo?.id;
    if (!userId || unreadTotal.value <= 0 || markAllLoading.value) return;

    markAllLoading.value = true;
    try {
      // 后端约定：recipientIds 为空 = 标记该用户全部未读
      // （下拉只加载了当前页，无法枚举全部未读ID）。
      await markNotificationAsRead({ userId, recipientIds: [] });
      ElMessage.success(t("core.notice.markAllReadSuccess"));
      reloadAll();
    } catch (error) {
      console.error("markAllRead failed:", error);
      ElMessage.error(t("core.notice.markAllReadFailed"));
    } finally {
      markAllLoading.value = false;
    }
  }

  function goMore() {
    router.push("/internal-message/inbox");
  }

  // SSE 实时通知：收到新站内信即整体重拉（不做增量插入，避免去重/截断逻辑分叉）
  function handleSseNotification() {
    reloadAll();
  }

  let pollTimer: ReturnType<typeof setInterval> | undefined;

  onMounted(() => {
    reloadAll();
    globalSSEClient.on(SSE_EVENT.Notification, handleSseNotification);
    pollTimer = setInterval(reloadAll, POLL_INTERVAL);
  });

  onBeforeUnmount(() => {
    // 必须按回调引用显式注销，否则 globalSSEClient.handlers 会持续累加：
    // NoticeDropdown 每次重新挂载（布局切换/路由往返/热重载）都会重新 on()，
    // 若不 off()，一条新消息会触发 N 次回调（N=累计挂载次数）。
    globalSSEClient.off(SSE_EVENT.Notification, handleSseNotification);
    if (pollTimer) clearInterval(pollTimer);
  });

  return {
    list,
    unreadTotal,
    listLoading,
    markAllLoading,
    readAll,
    goMore,
  };
}
