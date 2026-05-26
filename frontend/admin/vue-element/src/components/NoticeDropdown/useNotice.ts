/**
 * 通知中心逻辑
 */
import { ref, onMounted, onBeforeUnmount } from "vue";
import { fetchListUserInbox, fetchGetInternalMessage, useMarkNotificationAsRead } from "@/api/composables";
import { useAppUserStore } from "@/stores";
import { PaginationQuery } from "@/core/transport/rest";
import { router } from "@/router";
import type { internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from "@/api/generated/admin/service/v1";
import { dateUtil } from "@/utils";
import { globalSSEClient } from "@/core/transport/sse";

const PAGE_SIZE = 5;

// SSE 事件名称：通知消息
const NOTICE_EVENT = "notification";

export function useNotice() {
  const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();
  const userStore = useAppUserStore();

  // 状态
  const list = ref<InternalMessageRecipient[]>([]);
  const unreadTotal = ref(0);
  const detail = ref<any | null>(null);
  const dialogVisible = ref(false);

  let unsubscribe: (() => void) | null = null;

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
          ...params,
        },
        orderBy: ["-created_at"],
      })
    );
    // 转换数据格式
    list.value = (result.items || []).map((item) => convertInternalMessageRecipient(item));
    unreadTotal.value = result.total ?? 0;
  }

  async function read(id: string | number) {
    const numericId = Number(id);
    detail.value = await fetchGetInternalMessage({ id: numericId });
    dialogVisible.value = true;

    // 标记为已读
    const userId = userStore.userInfo?.id;
    if (userId) {
      try {
        await markNotificationAsRead({ userId, notificationIds: [numericId] });
        ElMessage.success("已标记为已读");
      } catch {
        ElMessage.error("标记失败");
      }
    }

    // 从列表中移除已读项
    const idx = list.value.findIndex((item) => item.id === numericId);
    if (idx >= 0) {
      list.value.splice(idx, 1);
      if (unreadTotal.value > 0) unreadTotal.value -= 1;
    }

    await fetchList();
  }

  async function readAll() {
    const userId = userStore.userInfo?.id;
    if (!userId) return;

    // 获取所有未读消息ID
    const unreadIds = list.value
      .filter((item) => item.status !== "READ")
      .map((item) => Number(item.id));

    if (unreadIds.length === 0) {
      ElMessage.info("没有未读消息");
      return;
    }

    try {
      await markNotificationAsRead({ userId, notificationIds: unreadIds });
      ElMessage.success("已全部标记为已读");
    } catch {
      ElMessage.error("操作失败");
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
      await ElMessageBox.confirm("确定要清空所有消息吗？", "提示", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      });
    } catch {
      return;
    }

    // 获取所有消息ID
    const allIds = list.value.map((item) => Number(item.id));

    if (allIds.length === 0) {
      ElMessage.info("没有消息可清空");
      return;
    }

    try {
      // TODO: 调用后端 API 删除消息
      // await internalMessageStore.deleteMessages(userId, allIds);
      ElMessage.success("已清空所有消息");
    } catch {
      ElMessage.error("操作失败");
      return;
    }

    // 清空列表并重置计数
    list.value = [];
    unreadTotal.value = 0;
  }

  function goMore() {
    router.push("/inbox/messages");
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
        title: "您收到一条新的通知消息！",
        message: data.title || "新消息",
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
    if (unsubscribe) {
      unsubscribe();
      unsubscribe = null;
    }
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
