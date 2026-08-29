/**
 * 通知中心逻辑
 */
import { ref, onMounted, onBeforeUnmount } from "vue";
import {
  fetchListUserInbox,
  fetchGetInternalMessage,
  useDeleteNotificationFromInbox,
  useMarkNotificationAsRead,
} from "@/api/composables";
import { useAppUserStore } from "@/stores";
import { PaginationQuery } from "@/core/transport/rest";
import { router } from "@/router";
import type { internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from "@/api/generated/admin/service/v1";
import { dateUtil } from "@/utils";
import { globalSSEClient } from "@/core/transport/sse";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

const PAGE_SIZE = 5;

// SSE 事件名称：通知消息
const NOTICE_EVENT = "notification";

export function useNotice() {
  const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();
  const { mutateAsync: deleteNotificationFromInbox } = useDeleteNotificationFromInbox();
  const userStore = useAppUserStore();

  // 状态
  const list = ref<InternalMessageRecipient[]>([]);
  const unreadTotal = ref(0);
  const detail = ref<any | null>(null);
  const dialogVisible = ref(false);

  // ============================================
  // 数据获取
  // ============================================

  async function fetchList(params?: any) {
    const userId = userStore.userInfo?.id;
    if (!userId) return;

    const result = await fetchListUserInbox(
      new PaginationQuery({
        paging: { page: 1, pageSize: PAGE_SIZE },
        formValues: {
          recipient_user_id: userId.toString(),
          status: "RECEIVED",
          ...params,
        },
        orderBy: ["-created_at"],
      })
    );
    // 转换数据格式
    list.value = (result.items || []).map((item) => convertInternalMessageRecipient(item));
    unreadTotal.value = result.total ?? 0;
  }

  async function read(item: { id?: number; messageId?: number }) {
    // 注意区分两种 id：
    // - item.id 是收件记录 id (InternalMessageRecipient.id)，用于标记已读 recipientIds
    // - item.messageId 是消息本身 id，用于 fetchGetInternalMessage 取详情
    // 之前错误地把收件记录 id 当作消息 id 传给 fetchGetInternalMessage，
    // 导致详情取错或 404。
    const recipientId = item.id;
    const messageId = item.messageId;
    if (recipientId == null && messageId == null) return;

    detail.value = await fetchGetInternalMessage({ id: messageId });
    dialogVisible.value = true;

    // 标记为已读（用收件记录 id）
    const userId = userStore.userInfo?.id;
    if (userId && recipientId != null) {
      try {
        await markNotificationAsRead({ userId, recipientIds: [recipientId] });
        ElMessage.success(t("common.notice.markedRead"));
      } catch {
        ElMessage.error(t("common.notice.markFailed"));
      }
    }

    // 从列表中移除已读项
    if (recipientId != null) {
      const idx = list.value.findIndex((row) => row.id === recipientId);
      if (idx >= 0) {
        list.value.splice(idx, 1);
        if (unreadTotal.value > 0) unreadTotal.value -= 1;
      }
    }

    await fetchList();
  }

  async function readAll() {
    const userId = userStore.userInfo?.id;
    if (!userId) return;

    if (unreadTotal.value === 0) {
      ElMessage.info(t("common.notice.noUnread"));
      return;
    }

    try {
      // 空 recipientIds 表示标记该用户全部未读（服务端按用户维度兜底）：
      // 下拉只加载了当前页，无法枚举全部未读ID。
      await markNotificationAsRead({ userId, recipientIds: [] });
      ElMessage.success(t("common.notice.allMarkedRead"));
    } catch {
      ElMessage.error(t("common.message.error"));
      return;
    }

    // 清空列表并重置计数
    list.value = [];
    unreadTotal.value = 0;
  }

  async function clearAll() {
    const userId = userStore.userInfo?.id;
    if (!userId) return;

    try {
      await ElMessageBox.confirm(t("common.notice.confirmClear"), t("common.message.info"), {
        confirmButtonText: t("common.button.confirm"),
        cancelButtonText: t("common.button.cancel"),
        type: "warning",
      });
    } catch {
      return;
    }

    if (list.value.length === 0 && unreadTotal.value === 0) {
      ElMessage.info(t("common.notice.noMessages"));
      return;
    }

    try {
      // 空 recipientIds 表示清空该用户收件箱（服务端按用户维度兜底，
      // 下拉只加载了当前页，无法枚举全部ID），危险操作已有上方二次确认。
      await deleteNotificationFromInbox({ userId, recipientIds: [] });
      ElMessage.success(t("common.notice.cleared"));
    } catch {
      ElMessage.error(t("common.message.error"));
      return;
    }

    // 清空列表并重置计数
    list.value = [];
    unreadTotal.value = 0;
  }

  function goMore() {
    router.push("/internal-message/inbox");
  }

  /**
   * 检查消息是否已存在
   */
  function hasMessage(data: InternalMessageRecipient): boolean {
    return list.value.some((item) => item.messageId === data.messageId);
  }

  /**
   * 转换内部消息为UI数据
   */
  function convertInternalMessageRecipient(item: InternalMessageRecipient) {
    const date = dateUtil(item.createdAt as string).fromNow();
    return {
      id: item.id ?? 0,
      messageId: item.messageId ?? 0,
      title: item.title || "",
      content: item.content || "",
      status: item.status,
      createdAt: item.createdAt,
      date,
      isRead: item.status === "READ",
    };
  }

  /**
   * 处理SSE通知事件
   */
  function handleSseNotification(data: InternalMessageRecipient) {
    try {
      if (!data.id || !data.messageId) return;

      // 避免重复
      if (hasMessage(data)) return;

      // 转换数据格式并添加到列表头部
      const convertedData = convertInternalMessageRecipient(data);
      list.value.unshift(convertedData);

      // 如果超过页面大小，移除最后一个
      if (list.value.length > PAGE_SIZE) {
        list.value.pop();
      }

      // 更新未读总数
      unreadTotal.value += 1;

      // 显示桌面通知
      ElNotification({
        title: t("common.notice.newNotificationTitle"),
        message: data.title || t("common.notice.newMessage"),
        type: "success",
        position: "bottom-right",
      });
    } catch (e) {
      console.error("解析通知消息失败", e);
    }
  }

  /**
   * 处理撤回通知事件
   */
  function handleSseRevoke(data: any) {
    try {
      if (!data.id && !data.messageId) return;

      // 从列表中移除已撤回的通知
      const idx = list.value.findIndex(
        (item) => item.id === data.id || item.messageId === data.messageId
      );
      if (idx >= 0) {
        list.value.splice(idx, 1);
        if (unreadTotal.value > 0) unreadTotal.value -= 1;
      }
    } catch (e) {
      console.error("处理撤回通知失败", e);
    }
  }

  // ============================================
  // SSE 订阅
  // ============================================

  function setupSubscription() {
    // 订阅新通知事件
    globalSSEClient.on<InternalMessageRecipient>(NOTICE_EVENT, handleSseNotification);

    // 订阅撤回通知事件
    globalSSEClient.on<any>("notification-revoke", handleSseRevoke);
  }

  // ============================================
  // 生命周期
  // ============================================

  onMounted(() => {
    fetchList();
    setupSubscription();
  });

  onBeforeUnmount(() => {
    // 必须按回调引用显式注销，否则 globalSSEClient.handlers 会持续累加：
    // NoticeDropdown 每次重新挂载（布局切换/路由往返/热重载）都会重新 on()，
    // 若不 off()，一条新消息会触发 N 次回调（N=累计挂载次数），
    // 导致 unreadTotal 重复自增、桌面通知重复弹出、list 反复 unshift。
    globalSSEClient.off(NOTICE_EVENT, handleSseNotification);
    globalSSEClient.off("notification-revoke", handleSseRevoke);
  });

  return {
    list,
    unreadTotal,
    detail,
    dialogVisible,
    fetchList,
    read,
    readAll,
    clearAll,
    goMore,
  };
}
