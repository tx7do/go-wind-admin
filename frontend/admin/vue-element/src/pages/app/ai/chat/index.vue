<template>
  <div class="app-container chat-page h-full flex min-h-0 flex-col">
    <div class="flex min-h-0 flex-1 gap-3">
      <!-- 左栏：会话列表 -->
      <aside class="chat-card flex w-60 shrink-0 flex-col">
        <div class="panel-head">
          <Icon icon="lucide:messages-square" :width="15" />
          <span class="panel-head__title">{{ t("pages.ai_chat.conversations") }}</span>
          <span v-if="conversations.length" class="panel-head__badge">
            {{ conversations.length }}
          </span>
        </div>
        <div class="px-3 pt-2">
          <!-- 不用 type="primary" plain：EP 的 plain 主色按钮暗色下实测是 #006BE6 实底 + 白字，
               与侧边栏选中项同色同形（规则见 docs/design-language.md §4「实底主色标位置，不标次级动作」）。 -->
          <ElButton class="new-conv-btn" @click="handleNewConversation">
            <Icon icon="lucide:plus" :width="14" class="mr-1" />
            {{ t("pages.ai_chat.newConversation") }}
          </ElButton>
        </div>
        <div class="conv-list">
          <div v-if="conversations.length === 0" class="conv-list__empty">
            <Icon icon="lucide:inbox" :width="26" />
            <span>{{ t("pages.ai_chat.noConversations") }}</span>
          </div>
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="conv-item"
            :class="{ 'is-active': activeId === conv.id }"
            @click="activeId = conv.id"
          >
            <div class="conv-item__text">
              <span class="conv-item__title">{{ conv.title || `#${conv.id}` }}</span>
              <span v-if="conv.createdAt" class="conv-item__time">
                {{ formatDate(conv.createdAt, "MM-DD HH:mm") }}
              </span>
            </div>
            <div class="conv-item__ops">
              <button
                type="button"
                class="row-op"
                :title="t('pages.ai_chat.rename')"
                @click.stop="handleRename(conv)"
              >
                <Icon icon="lucide:pen-line" :width="14" />
              </button>
              <button
                type="button"
                class="row-op row-op--danger"
                :title="t('pages.ai_chat.deleteConversation')"
                @click.stop="confirmDelete(conv)"
              >
                <Icon icon="lucide:trash-2" :width="14" />
              </button>
            </div>
          </div>
        </div>
      </aside>

      <!-- 右栏：消息区 + 输入区 -->
      <section class="chat-card flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="panel-head">
          <span class="panel-head__title" :title="activeTitle">{{ activeTitle }}</span>
          <span v-if="activeKbName" class="panel-head__tag">
            <Icon icon="lucide:book-open" :width="12" />
            {{ activeKbName }}
          </span>
          <span v-if="messages.length" class="panel-head__badge">{{ messages.length }}</span>
        </div>

        <div ref="scrollRef" class="msg-scroll">
          <div class="msg-list">
            <!-- 空态 -->
            <div v-if="messages.length === 0 && !sending" class="chat-empty">
              <div class="chat-empty__icon">
                <Icon icon="lucide:sparkles" :width="24" />
              </div>
              <div class="chat-empty__title">{{ t("pages.ai_chat.emptyTitle") }}</div>
              <div class="chat-empty__desc">{{ t("pages.ai_chat.emptyDesc") }}</div>
            </div>

            <div
              v-for="msg in messages"
              :key="msg.id"
              class="msg"
              :class="msg.role === 'USER' ? 'msg--user' : 'msg--ai'"
            >
              <div class="msg__avatar">
                <Icon :icon="msg.role === 'USER' ? 'lucide:user' : 'lucide:bot'" :width="16" />
              </div>
              <div class="msg__main">
                <div class="msg__meta">
                  <span class="msg__who">
                    {{ t(msg.role === "USER" ? "pages.ai_chat.you" : "pages.ai_chat.assistant") }}
                  </span>
                  <span v-if="msg.createdAt">{{ formatDate(msg.createdAt, "HH:mm:ss") }}</span>
                </div>

                <!-- 用户消息：纯文本气泡 -->
                <div v-if="msg.role === 'USER'" class="msg__bubble msg__bubble--user">
                  {{ msg.content }}
                </div>
                <!-- AI 消息：markdown 渲染气泡 -->
                <div v-else class="msg__bubble msg__bubble--ai">
                  <!-- eslint-disable-next-line vue/no-v-html -- 已经过 DOMPurify 消毒，见 renderMarkdown -->
                  <div class="markdown-body" v-html="renderMarkdown(msg.content || '')" />
                  <div
                    v-if="msg.promptTokens || msg.completionTokens"
                    class="msg__foot"
                    :title="`${msg.modelName || ''} · ${msg.durationMs || 0}ms`"
                  >
                    <Icon icon="lucide:hash" :width="11" />
                    {{
                      t("pages.ai_chat.tokensUsage", {
                        prompt: msg.promptTokens ?? 0,
                        completion: msg.completionTokens ?? 0,
                      })
                    }}
                  </div>
                </div>
              </div>
            </div>

            <!-- 流式占位气泡 -->
            <div v-if="sending" class="msg msg--ai">
              <div class="msg__avatar">
                <Icon icon="lucide:bot" :width="16" />
              </div>
              <div class="msg__main">
                <div class="msg__meta">
                  <span class="msg__who">{{ t("pages.ai_chat.assistant") }}</span>
                </div>
                <div class="msg__bubble msg__bubble--ai">
                  <!-- 工具调用可见化：模型调工具期间的中间过程帧 -->
                  <div v-for="(tc, i) in toolCalls" :key="`${i}-${tc.name}`" class="tool-call">
                    <span class="tool-call__head">🛠 {{ tc.name }}({{ tc.arguments }})</span>
                    <span class="tool-call__result">{{ tc.result }}</span>
                  </div>
                  <!-- eslint-disable-next-line vue/no-v-html -- 已经过 DOMPurify 消毒，见 renderMarkdown -->
                  <div class="markdown-body" v-html="renderMarkdown(streamingText)" />
                  <div v-if="!streamingText" class="typing">
                    <i />
                    <i />
                    <i />
                  </div>
                  <div class="msg__foot">
                    <Icon icon="lucide:loader-2" :width="11" class="is-spinning" />
                    {{ t("pages.ai_chat.streaming") }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区 -->
        <div class="composer">
          <div class="composer__box">
            <ElInput
              v-model="input"
              type="textarea"
              :bordered="false"
              :autosize="{ minRows: 2, maxRows: 6 }"
              :placeholder="t('pages.ai_chat.inputPlaceholder')"
              :disabled="sending"
              class="composer__input"
              @keydown.enter.exact.prevent="handleSend"
            />
            <div class="composer__bar">
              <div class="composer__tool">
                <Icon icon="lucide:book-open" :width="13" />
                <span class="composer__tool-label">
                  {{ t("pages.ai_knowledge.knowledgeBase") }}
                </span>
                <ElSelect
                  v-model="knowledgeBaseId"
                  clearable
                  size="small"
                  class="composer__kb"
                  :placeholder="t('pages.ai_knowledge.knowledgeBasePlaceholder')"
                >
                  <ElOption
                    v-for="kb in knowledgeBases"
                    :key="kb.id"
                    :label="kb.name || `#${kb.id}`"
                    :value="kb.id!"
                  />
                </ElSelect>
              </div>
              <div class="composer__actions">
                <span class="composer__hint">{{ t("pages.ai_chat.sendHint") }}</span>
                <ElButton
                  type="primary"
                  round
                  :loading="sending"
                  :disabled="!input.trim()"
                  @click="handleSend"
                >
                  <Icon icon="lucide:send" :width="14" class="mr-1" />
                  {{ sending ? t("pages.ai_chat.sending") : t("pages.ai_chat.send") }}
                </ElButton>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { ElButton, ElInput, ElMessage, ElMessageBox, ElOption, ElSelect } from "element-plus";
import { Icon } from "@iconify/vue";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { useI18n } from "@/core/i18n";
import { formatDate } from "@/utils";
import { PaginationQuery } from "@/core/transport/rest";
import { globalSSEClient, SSE_EVENT } from "@/core/transport/sse";
import type {
  aiservicev1_AiConversation as AiConversation,
  aiservicev1_AiMessage as AiMessage,
} from "@/api/generated/admin/service/v1";
import {
  deleteAiConversation,
  fetchListAiConversations,
  fetchListAiKnowledgeBases,
  fetchListAiMessages,
  sendAiChat,
  updateAiConversation,
} from "@/api/composables";
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from "@/api/generated/admin/service/v1";

const { t } = useI18n();

// marked 单行换行按 GFM 处理（聊天场景常见单换行段落）
marked.setOptions({ gfm: true, breaks: true });

function renderMarkdown(content: string): string {
  // AI 回复经 DOMPurify 消毒后再 v-html；LLM 输出不可信，防注入
  return DOMPurify.sanitize(marked.parse(content, { async: false }) as string);
}

// ── 会话 ──────────────────────────────────────────────────────────
const conversations = ref<AiConversation[]>([]);
const activeId = ref<string | undefined>(undefined);

// 仅在首次拉到列表时自动选中最近会话：写在 loadConversations 里会让「新建对话」
// 立刻被重新选中顶回去，空态与"新会话"标题都渲染不出来
let autoSelected = false;

const activeTitle = computed(() => {
  if (activeId.value === undefined) return t("pages.ai_chat.untitled");
  const conv = conversations.value.find((c) => c.id === activeId.value);
  return conv?.title || `#${conv?.id}`;
});

async function loadConversations() {
  try {
    const r = await fetchListAiConversations(
      new PaginationQuery({ paging: { page: 1, pageSize: 100 }, orderBy: ["-last_message_at"] })
    );
    conversations.value = (r.items || []) as AiConversation[];
    if (!autoSelected && conversations.value.length > 0) {
      autoSelected = true;
      activeId.value = conversations.value[0]!.id;
    }
  } catch (error) {
    console.error("load ai conversations failed:", error);
    ElMessage.error(t("pages.ai_chat.fetchFailed"));
  }
}

// ── 消息 ──────────────────────────────────────────────────────────
const messages = ref<AiMessage[]>([]);

async function loadMessages() {
  if (activeId.value === undefined) {
    messages.value = [];
    return;
  }
  try {
    const r = await fetchListAiMessages(
      new PaginationQuery({
        paging: { page: 1, pageSize: 200 },
        formValues: { conversation_id: activeId.value },
        orderBy: ["id"],
      })
    );
    messages.value = (r.items || []) as AiMessage[];
    scrollToBottom();
  } catch (error) {
    console.error("load ai messages failed:", error);
    ElMessage.error(t("pages.ai_chat.fetchFailed"));
  }
}

watch(activeId, () => {
  streamingText.value = "";
  toolCalls.value = [];
  loadMessages();
});

// ── 流式 ──────────────────────────────────────────────────────────
const streamingText = ref("");
// 工具调用可见化：ai_chat_tool 帧按到达顺序累积，随流式气泡一并渲染
const toolCalls = ref<ChatToolEvent[]>([]);
const scrollRef = ref<HTMLElement>();

function scrollToBottom() {
  nextTick(() => {
    scrollRef.value?.scrollTo({ top: scrollRef.value?.scrollHeight ?? 0, behavior: "smooth" });
  });
}

interface ChatChunk {
  conversationId?: string;
  seq?: number;
  delta?: string;
}

interface ChatToolEvent {
  conversationId?: string;
  name?: string;
  arguments?: string;
  result?: string;
}

function handleChunk(data: ChatChunk) {
  if (!data || typeof data !== "object" || !data.conversationId) return;
  // 只渲染当前打开会话的片段；其余会话的 chunk 静默丢弃（响应到达后列表刷新）。
  // 新会话（activeId 未定）放行：首条消息的会话由响应才创建，过滤会让新会话流式全丢。
  if (activeId.value !== undefined && data.conversationId !== activeId.value) return;
  streamingText.value += data.delta || "";
  scrollToBottom();
}

function handleToolEvent(data: ChatToolEvent) {
  if (!data || typeof data !== "object" || !data.name) return;
  // 同 handleChunk：新会话（activeId 未定）放行，工具帧才对首条消息可见
  if (activeId.value !== undefined && data.conversationId !== activeId.value) return;
  toolCalls.value.push(data);
  scrollToBottom();
}

onMounted(() => {
  loadConversations();
  loadMessages();
  // 必须按回调引用显式注销：组件重挂载反复 on() 会让回调累加（见 useNotice.ts 的注释）
  globalSSEClient.on<ChatChunk>(SSE_EVENT.AIChatChunk, handleChunk);
  globalSSEClient.on<ChatToolEvent>(SSE_EVENT.AIChatTool, handleToolEvent);
});

onBeforeUnmount(() => {
  globalSSEClient.off(SSE_EVENT.AIChatChunk, handleChunk);
  globalSSEClient.off(SSE_EVENT.AIChatTool, handleToolEvent);
});

// ── 知识库选择（RAG：发送时携带 knowledgeBaseId） ─────────────────
const knowledgeBases = ref<AiKnowledgeBase[]>([]);
const knowledgeBaseId = ref<string | undefined>(undefined);

const activeKbName = computed(() => {
  if (knowledgeBaseId.value === undefined) return "";
  const kb = knowledgeBases.value.find((k) => k.id === knowledgeBaseId.value);
  return kb?.name || (kb?.id ? `#${kb.id}` : "");
});

onMounted(() => {
  fetchListAiKnowledgeBases(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }))
    .then((res) => {
      knowledgeBases.value = (res.items || []) as AiKnowledgeBase[];
    })
    .catch((error: any) => console.error("fetch ai knowledge bases failed:", error));
});

// ── 发送 ──────────────────────────────────────────────────────────
const input = ref("");
const sending = ref(false);

async function handleSend() {
  const content = input.value.trim();
  if (!content || sending.value) return;
  streamingText.value = "";
  toolCalls.value = [];
  sending.value = true;
  scrollToBottom();
  try {
    const resp = await sendAiChat({
      conversationId: activeId.value,
      content,
      knowledgeBaseId: knowledgeBaseId.value,
    });
    input.value = "";
    streamingText.value = "";
    toolCalls.value = [];
    if (resp.conversation?.id) {
      const exists = conversations.value.some((c) => c.id === resp.conversation!.id);
      if (!exists) {
        activeId.value = resp.conversation.id;
        await loadConversations();
      }
    }
    await loadMessages();
  } catch (error: any) {
    console.error("send ai chat failed:", error);
    streamingText.value = "";
    toolCalls.value = [];
    ElMessage.error(error?.message || t("pages.ai_chat.chatFailed"));
  } finally {
    sending.value = false;
  }
}

// ── 删除会话 ──────────────────────────────────────────────────────
async function handleRename(conv: AiConversation) {
  try {
    const { value } = await ElMessageBox.prompt(
      t("pages.ai_chat.renamePrompt"),
      t("pages.ai_chat.rename"),
      { inputValue: conv.title || "" }
    );
    const title = (value || "").trim();
    if (!title || title === conv.title) return;
    await updateAiConversation(conv.id!, { title });
    ElMessage.success(t("pages.ai_chat.renameSuccess"));
    await loadConversations();
  } catch {
    // 用户取消
  }
}

async function confirmDelete(conv: AiConversation) {
  const confirmed = await ElMessageBox.confirm(
    t("pages.ai_chat.deleteConversationConfirm"),
    t("pages.ai_chat.deleteConversation"),
    {
      type: "warning",
    }
  ).then(
    () => true,
    () => false
  );
  if (!confirmed) return;
  try {
    await deleteAiConversation(conv.id!);
    ElMessage.success(t("pages.ai_chat.deleteSuccess"));
    if (activeId.value === conv.id) {
      activeId.value = undefined;
      messages.value = [];
    }
    await loadConversations();
  } catch (error: any) {
    console.error("delete ai conversation failed:", error);
    ElMessage.error(error?.message || t("pages.ai_chat.fetchFailed"));
  }
}

function handleNewConversation() {
  activeId.value = undefined;
  messages.value = [];
  streamingText.value = "";
  toolCalls.value = [];
  loadConversations();
}
</script>

<style lang="scss" scoped>
// 全部颜色走 Element Plus 主题变量：主色可配置、暗色自动跟随，
// 不写死 tailwind 调色板（bg-blue-500 / bg-gray-100 在换主题色或切暗色时会脱节）。
.chat-card {
  min-height: 0;
  overflow: hidden;
  background: var(--el-bg-color);
  // 与四张 Pro 卡同取 §2.2 的卡片静止档（变量定义见 styles/vendors/_element-plus.scss 的
  // :root / html.dark 两块）。原先吃 --el-box-shadow-light：亮色 0 0 12px 12% 黑，
  // 暗色实测 0 0 12px 72% 黑 —— 正是 _dark-mode.scss 给 .el-card 定档时点名"会在窄间距卡片间
  // 叠出黑缝"的那一档，而会话列表卡与对话卡之间的缝就是窄间距。
  border: 1px solid var(--gowind-card-border);
  border-radius: 12px;
  box-shadow: var(--gowind-card-shadow);
}

// ── 面板头部 ────────────────────────────────────────────────────────
.panel-head {
  display: flex;
  flex-shrink: 0;
  gap: 6px;
  align-items: center;
  height: 40px;
  padding: 0 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.panel-head__title {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.panel-head__badge {
  flex-shrink: 0;
  padding: 0 6px;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 17px;
  // 芯片底是 --el-fill-color（浅 #F0F2F5 / 暗 #262D3A），次要档在上面只有 4.31 / 4.50:1
  // （11px 需 4.5）；改用「常规」档两端同时合格（9.19 / ~8）
  color: var(--el-text-color-regular);
  background: var(--el-fill-color);
  border-radius: 999px;
}

.panel-head__tag {
  display: inline-flex;
  flex-shrink: 0;
  gap: 4px;
  align-items: center;
  max-width: 40%;
  padding: 1px 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 11px;
  font-weight: 400;
  color: var(--gowind-primary-text);
  white-space: nowrap;
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-8);
  border-radius: 999px;
}

// ── 会话列表 ────────────────────────────────────────────────────────
.new-conv-btn {
  width: 100%;
}

.conv-list {
  flex: 1;
  min-height: 0;
  padding: 6px 8px 10px;
  overflow-y: auto;
}

.conv-list__empty {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: center;
  justify-content: center;
  padding: 32px 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.conv-item {
  position: relative;
  display: flex;
  gap: 4px;
  align-items: center;
  padding: 7px 10px;
  cursor: pointer;
  border-radius: 8px;
  transition: background-color 0.15s ease;

  & + & {
    margin-top: 2px;
  }

  &:hover {
    background: var(--el-fill-color-light);
  }

  &.is-active {
    /* 选中态按 docs/design-language.md §4 定稿：主色实底 + 前景色文字，
       不用左侧竖条 / 淡色底 / 字重加粗。旧写法实测暗色下标题仅 3.29:1。 */
    color: var(--el-color-white);
    background: var(--el-color-primary);

    .conv-item__title,
    .conv-item__time,
    .row-op {
      color: inherit;
    }
  }
}

.conv-item__text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.conv-item__title {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 13px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.conv-item__time {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-secondary);
}

.conv-item__ops {
  display: flex;
  flex-shrink: 0;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.conv-item:hover .conv-item__ops,
.conv-item:focus-within .conv-item__ops {
  opacity: 1;
}

.row-op {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 6px;
  transition:
    color 0.15s ease,
    background-color 0.15s ease;

  &:hover {
    color: var(--gowind-primary-text);
    background: var(--el-fill-color);
  }
}

.row-op--danger:hover {
  color: var(--gowind-danger-text);
  background: var(--el-color-danger-light-9);
}

// ── 消息区 ──────────────────────────────────────────────────────────
.msg-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

// 正文与输入框同宽，形成一条居中的阅读柱
.msg-list {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 56rem;
  min-height: 100%;
  padding: 20px 24px 12px;
  margin: 0 auto;
}

.chat-empty {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  justify-content: center;
  padding: 24px;
  text-align: center;
}

.chat-empty__icon {
  display: grid;
  place-items: center;
  width: 56px;
  height: 56px;
  margin-bottom: 8px;
  color: var(--gowind-primary-text);
  background: var(--el-color-primary-light-9);
  border-radius: 50%;
}

.chat-empty__title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.chat-empty__desc {
  max-width: 26rem;
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.msg {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.msg--user {
  flex-direction: row-reverse;

  .msg__main {
    align-items: flex-end;
  }

  .msg__meta {
    flex-direction: row-reverse;
  }
}

.msg__avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
}

.msg--ai .msg__avatar {
  color: var(--gowind-primary-text);
  background: var(--el-color-primary-light-9);
}

.msg--user .msg__avatar {
  color: var(--el-text-color-regular);
  background: var(--el-fill-color);
}

.msg__main {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  max-width: 78%;
}

.msg__meta {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 0 2px;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-secondary);
}

.msg__who {
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.msg__bubble {
  padding: 9px 13px;
  font-size: 14px;
  line-height: 1.7;
  overflow-wrap: break-word;
  border-radius: 12px;
}

.msg__bubble--user {
  color: var(--el-color-white);
  white-space: pre-wrap;
  background: var(--el-color-primary);
  border-top-right-radius: 4px;
}

.msg__bubble--ai {
  min-width: 0;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-top-left-radius: 4px;
}

.msg__foot {
  display: flex;
  gap: 4px;
  align-items: center;
  margin-top: 6px;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-placeholder);
}

.typing {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  height: 20px;

  i {
    width: 6px;
    height: 6px;
    background: var(--el-text-color-disabled);
    border-radius: 50%;
    animation: typing-bounce 1.2s ease-in-out infinite;

    &:nth-child(2) {
      animation-delay: 0.15s;
    }

    &:nth-child(3) {
      animation-delay: 0.3s;
    }
  }
}

@keyframes typing-bounce {
  0%,
  60%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }

  30% {
    opacity: 1;
    transform: translateY(-3px);
  }
}

:deep(.is-spinning) {
  animation: typing-spin 1s linear infinite;
}

@keyframes typing-spin {
  to {
    transform: rotate(360deg);
  }
}

// ── 输入区 ──────────────────────────────────────────────────────────
.composer {
  flex-shrink: 0;
  padding: 12px 24px 16px;
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color-lighter);
}

.composer__box {
  max-width: 56rem;
  padding: 6px 10px 8px;
  margin: 0 auto;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;

  &:hover {
    border-color: var(--el-border-color-hover);
  }

  &:focus-within {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-9);
  }
}

// _dark-mode.scss 与 vendors/_element-plus.scss 对 .el-textarea__inner 的 box-shadow /
// background 都写成了 !important，普通覆盖打不过（实测暗色下文本域自己一圈内阴影，
// 与 composer 外框叠成双线）；这里用更高优先级 + !important 压回无边框透明态。
.composer .composer__box .composer__input :deep(.el-textarea__inner) {
  padding: 4px 2px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  background: transparent !important;
  box-shadow: none !important;
}

.composer__bar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 12px;
  align-items: center;
  justify-content: space-between;
  margin-top: 2px;
}

.composer__tool {
  display: flex;
  flex: 1 1 auto;
  gap: 6px;
  align-items: center;
  min-width: 0;
  color: var(--el-text-color-secondary);
}

.composer__tool-label {
  flex-shrink: 0;
  font-size: 12px;
}

// 窄屏下可收缩：固定 width 会被 flex 压缩到placeholder 都放不下
.composer__kb {
  flex: 1 1 180px;
  min-width: 96px;
  max-width: 260px;
}

.composer__actions {
  display: flex;
  flex-shrink: 0;
  gap: 10px;
  align-items: center;
  margin-left: auto;
}

.composer__hint {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

// ── Markdown 正文 ───────────────────────────────────────────────────
// tailwind preflight 会清掉列表符号/标题字号/引用缩进，
// LLM 输出以列表与标题为主，必须在气泡内逐项还原，否则回复糊成一片纯文本。
.markdown-body {
  :deep(p) {
    margin: 0 0 0.5em;

    &:last-child {
      margin-bottom: 0;
    }
  }

  :deep(h1),
  :deep(h2),
  :deep(h3),
  :deep(h4),
  :deep(h5),
  :deep(h6) {
    margin: 0.8em 0 0.4em;
    font-weight: 600;
    line-height: 1.5;
    color: var(--el-text-color-primary);

    &:first-child {
      margin-top: 0;
    }
  }

  :deep(h1) {
    font-size: 1.25em;
  }

  :deep(h2) {
    font-size: 1.15em;
  }

  :deep(h3) {
    font-size: 1.05em;
  }

  :deep(h4),
  :deep(h5),
  :deep(h6) {
    font-size: 1em;
  }

  :deep(ul),
  :deep(ol) {
    padding-left: 1.5em;
    margin: 0.4em 0;

    &:last-child {
      margin-bottom: 0;
    }
  }

  :deep(ul) {
    list-style: disc;

    ul {
      list-style: circle;
    }
  }

  :deep(ol) {
    list-style: decimal;
  }

  :deep(li) {
    margin: 0.2em 0;

    &::marker {
      color: var(--el-text-color-secondary);
    }

    > p {
      margin: 0;
    }
  }

  :deep(a) {
    color: var(--gowind-primary-text);
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }

  :deep(blockquote) {
    padding: 2px 0 2px 12px;
    margin: 0.5em 0;
    color: var(--el-text-color-secondary);
    border-left: 3px solid var(--el-border-color);
  }

  :deep(hr) {
    margin: 0.8em 0;
    border: none;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  :deep(strong) {
    font-weight: 600;
  }

  :deep(img) {
    max-width: 100%;
    border-radius: 8px;
  }

  :deep(code) {
    padding: 1px 5px;
    font-size: 0.92em;
    background: var(--el-fill-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
  }

  :deep(pre) {
    max-width: 100%;
    padding: 12px;
    margin: 0.5em 0;
    overflow-x: auto;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    &:last-child {
      margin-bottom: 0;
    }

    code {
      padding: 0;
      font-size: 12.5px;
      line-height: 1.6;
      background: transparent;
      border: none;
    }
  }

  :deep(table) {
    display: block;
    max-width: 100%;
    margin: 0.5em 0;
    overflow-x: auto;
    border-collapse: collapse;
  }

  :deep(th),
  :deep(td) {
    padding: 5px 10px;
    border: 1px solid var(--el-border-color-lighter);
  }

  :deep(th) {
    font-weight: 600;
    background: var(--el-fill-color-light);
  }
}

// 工具调用可见化：ai_chat_tool 帧的展示条（模型调工具期间的中间过程，流式结束即清）
.tool-call {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-width: 100%;
  margin-bottom: 6px;
  padding: 6px 10px;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  line-height: 1.5;
  border: 1px dashed var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.tool-call__head {
  color: var(--el-text-color-secondary);
}

.tool-call__result {
  color: var(--el-text-color-primary);
  word-break: break-all;
}
</style>
