<script lang="ts" setup>
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { dateUtil } from '@vben/utils';

import { message } from 'ant-design-vue';
import { Icon as Iconify } from '@iconify/vue';
import { marked } from 'marked';
import DOMPurify from 'dompurify';

import {
  PaginationQuery,
  deleteAiConversation,
  fetchListAiConversations,
  fetchListAiKnowledgeBases,
  fetchListAiMessages,
  sendAiChat,
  updateAiConversation,
} from '#/api';
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from '#/api/generated/admin/service/v1';
import { globalSSEClient, SSE_EVENT } from '#/transport/sse';
import type {
  aiservicev1_AiConversation as AiConversation,
  aiservicev1_AiMessage as AiMessage,
} from '#/api/generated/admin/service/v1';

// marked 单行换行按 GFM 处理（聊天场景常见单换行段落）
marked.setOptions({ gfm: true, breaks: true });

function renderMarkdown(content: string): string {
  // AI 回复经 DOMPurify 消毒后再 v-html；LLM 输出不可信，防注入
  return DOMPurify.sanitize(marked.parse(content, { async: false }) as string);
}

// ── 会话 ──────────────────────────────────────────────────────────
const conversations = ref<AiConversation[]>([]);
const activeId = ref<number | undefined>(undefined);

// 仅在首次拉到列表时自动选中最近会话：写在 loadConversations 里会让「新建对话」
// 立刻被重新选中顶回去，空态与"新会话"标题都渲染不出来
let autoSelected = false;

const activeTitle = computed(() => {
  if (activeId.value === undefined) return $t('page.aiChat.untitled');
  const conv = conversations.value.find((c) => c.id === activeId.value);
  return conv?.title || `#${conv?.id}`;
});

async function loadConversations() {
  try {
    const r = await fetchListAiConversations(
      new PaginationQuery({
        paging: { page: 1, pageSize: 100 },
        orderBy: ['-last_message_at'],
      }),
    );
    conversations.value = (r.items || []) as AiConversation[];
    if (!autoSelected && conversations.value.length > 0) {
      autoSelected = true;
      activeId.value = conversations.value[0]!.id;
    }
  } catch (error) {
    console.error('load ai conversations failed:', error);
    message.error($t('page.aiChat.fetchFailed'));
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
        orderBy: ['id'],
      }),
    );
    messages.value = (r.items || []) as AiMessage[];
    scrollToBottom();
  } catch (error) {
    console.error('load ai messages failed:', error);
    message.error($t('page.aiChat.fetchFailed'));
  }
}

watch(activeId, () => {
  streamingText.value = '';
  toolCalls.value = [];
  loadMessages();
});

// ── 流式 ──────────────────────────────────────────────────────────
const streamingText = ref('');
// 工具调用可见化：ai_chat_tool 帧按到达顺序累积，随流式气泡一并渲染
const toolCalls = ref<ChatToolEvent[]>([]);
const scrollRef = ref<HTMLElement>();

function scrollToBottom() {
  nextTick(() => {
    scrollRef.value?.scrollTo({
      top: scrollRef.value?.scrollHeight ?? 0,
      behavior: 'smooth',
    });
  });
}

interface ChatChunk {
  conversationId?: number;
  seq?: number;
  delta?: string;
}

interface ChatToolEvent {
  conversationId?: number;
  name?: string;
  arguments?: string;
  result?: string;
}

function handleChunk(data: ChatChunk) {
  if (!data || typeof data !== 'object' || !data.conversationId) return;
  // 只渲染当前打开会话的片段；其余会话的 chunk 静默丢弃（响应到达后列表刷新）。
  // 新会话（activeId 未定）放行：首条消息的会话由响应才创建，过滤会让新会话流式全丢。
  if (activeId.value !== undefined && data.conversationId !== activeId.value) return;
  streamingText.value += data.delta || '';
  scrollToBottom();
}

function handleToolEvent(data: ChatToolEvent) {
  if (!data || typeof data !== 'object' || !data.name) return;
  // 同 handleChunk：新会话（activeId 未定）放行，工具帧才对首条消息可见
  if (activeId.value !== undefined && data.conversationId !== activeId.value) return;
  toolCalls.value.push(data);
  scrollToBottom();
}

onMounted(() => {
  loadConversations();
  loadMessages();
  // 必须按回调引用显式注销：组件重挂载反复 on() 会让回调累加（同 useNotice 的坑）
  globalSSEClient.on<ChatChunk>(SSE_EVENT.AIChatChunk, handleChunk);
  globalSSEClient.on<ChatToolEvent>(SSE_EVENT.AIChatTool, handleToolEvent);
});

onBeforeUnmount(() => {
  globalSSEClient.off(SSE_EVENT.AIChatChunk, handleChunk);
  globalSSEClient.off(SSE_EVENT.AIChatTool, handleToolEvent);
});

// ── 会话重命名（内联编辑） ────────────────────────────────────────
const renamingId = ref<number | undefined>(undefined);
const renameText = ref('');

function startRename(conv: AiConversation) {
  renamingId.value = conv.id;
  renameText.value = conv.title || '';
}

function commitRename(conv: AiConversation) {
  const title = renameText.value.trim();
  renamingId.value = undefined;
  if (!title || title === (conv.title ?? '')) return;
  updateAiConversation(conv.id!, { title })
    .then(() => loadConversations())
    .catch((error) => {
      console.error('rename conversation failed:', error);
      message.error($t('page.aiChat.fetchFailed'));
    });
}

// ── 知识库选择（RAG：发送时携带 knowledgeBaseId） ─────────────────
const knowledgeBases = ref<AiKnowledgeBase[]>([]);
const knowledgeBaseId = ref<number | undefined>(undefined);

const activeKbName = computed(() => {
  if (knowledgeBaseId.value === undefined) return '';
  const kb = knowledgeBases.value.find((k) => k.id === knowledgeBaseId.value);
  return kb?.name || (kb?.id ? `#${kb.id}` : '');
});

onMounted(() => {
  fetchListAiKnowledgeBases(
    new PaginationQuery({ paging: { page: 1, pageSize: 100 } }),
  )
    .then((res) => {
      knowledgeBases.value = (res.items || []) as AiKnowledgeBase[];
    })
    .catch((error) => console.error('fetch ai knowledge bases failed:', error));
});

// ── 发送 ──────────────────────────────────────────────────────────
const input = ref('');
const sending = ref(false);

async function handleSend() {
  const content = input.value.trim();
  if (!content || sending.value) return;
  streamingText.value = '';
  toolCalls.value = [];
  sending.value = true;
  scrollToBottom();
  try {
    const resp = await sendAiChat({
      conversationId: activeId.value ?? 0,
      content,
      knowledgeBaseId: knowledgeBaseId.value ?? 0,
    });
    input.value = '';
    streamingText.value = '';
    toolCalls.value = [];
    if (resp.conversation?.id) {
      const exists = conversations.value.some(
        (c) => c.id === resp.conversation!.id,
      );
      if (!exists) {
        activeId.value = resp.conversation.id;
        await loadConversations();
      }
    }
    await loadMessages();
  } catch (error) {
    console.error('send ai chat failed:', error);
    streamingText.value = '';
    toolCalls.value = [];
    message.error(
      error instanceof Error ? error.message : $t('page.aiChat.chatFailed'),
    );
  } finally {
    sending.value = false;
  }
}

// ── 删除会话 ──────────────────────────────────────────────────────
async function confirmDelete(conv: AiConversation) {
  try {
    await deleteAiConversation(conv.id!);
    message.success($t('page.aiChat.deleteSuccess'));
    if (activeId.value === conv.id) {
      activeId.value = undefined;
      messages.value = [];
    }
    await loadConversations();
  } catch (error) {
    console.error('delete ai conversation failed:', error);
    message.error(
      error instanceof Error ? error.message : $t('page.aiChat.fetchFailed'),
    );
  }
}

function handleNewConversation() {
  activeId.value = undefined;
  messages.value = [];
  streamingText.value = '';
  toolCalls.value = [];
  loadConversations();
}
</script>

<template>
  <Page auto-content-height>
    <div class="flex h-full min-h-0 gap-3">
      <!-- 左栏：会话列表 -->
      <aside class="chat-card flex w-60 shrink-0 flex-col">
        <div class="panel-head">
          <Iconify icon="lucide:messages-square" :width="15" />
          <span class="panel-head__title">{{
            $t('page.aiChat.conversations')
          }}</span>
          <span v-if="conversations.length" class="panel-head__badge">
            {{ conversations.length }}
          </span>
        </div>
        <div class="px-3 pt-2">
          <a-button block @click="handleNewConversation">
            <template #icon>
              <Iconify icon="lucide:plus" />
            </template>
            {{ $t('page.aiChat.newConversation') }}
          </a-button>
        </div>
        <div class="conv-list">
          <div v-if="conversations.length === 0" class="conv-list__empty">
            <Iconify icon="lucide:inbox" :width="26" />
            <span>{{ $t('page.aiChat.noConversations') }}</span>
          </div>
          <div
            v-for="conv in conversations"
            :key="conv.id"
            class="conv-item"
            :class="{ 'is-active': activeId === conv.id }"
            @click="activeId = conv.id"
            @dblclick="startRename(conv)"
          >
            <div class="conv-item__text">
              <input
                v-if="renamingId === conv.id"
                v-model="renameText"
                class="conv-item__input"
                @click.stop
                @keydown.enter.prevent="commitRename(conv)"
                @blur="commitRename(conv)"
              />
              <template v-else>
                <span class="conv-item__title">{{
                  conv.title || `#${conv.id}`
                }}</span>
                <span v-if="conv.createdAt" class="conv-item__time">
                  {{ dateUtil(conv.createdAt).format('MM-DD HH:mm') }}
                </span>
              </template>
            </div>
            <div class="conv-item__ops">
              <button
                class="row-op"
                :title="$t('page.aiChat.rename')"
                type="button"
                @click.stop="startRename(conv)"
              >
                <Iconify icon="lucide:pen-line" :width="14" />
              </button>
              <a-popconfirm
                :cancel-text="$t('ui.button.cancel')"
                :ok-text="$t('ui.button.ok')"
                :title="$t('page.aiChat.deleteConversationConfirm')"
                @confirm="confirmDelete(conv)"
              >
                <button
                  class="row-op row-op--danger"
                  :title="$t('page.aiChat.deleteConversation')"
                  type="button"
                  @click.stop
                >
                  <Iconify icon="lucide:trash-2" :width="14" />
                </button>
              </a-popconfirm>
            </div>
          </div>
        </div>
      </aside>

      <!-- 右栏：消息区 + 输入区 -->
      <section class="chat-card flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="panel-head">
          <span class="panel-head__title" :title="activeTitle">{{
            activeTitle
          }}</span>
          <span v-if="activeKbName" class="panel-head__tag">
            <Iconify icon="lucide:book-open" :width="12" />
            {{ activeKbName }}
          </span>
          <span v-if="messages.length" class="panel-head__badge">{{
            messages.length
          }}</span>
        </div>

        <div ref="scrollRef" class="msg-scroll">
          <div class="msg-list">
            <!-- 空态 -->
            <div v-if="messages.length === 0 && !sending" class="chat-empty">
              <div class="chat-empty__icon">
                <Iconify icon="lucide:sparkles" :width="24" />
              </div>
              <div class="chat-empty__title">
                {{ $t('page.aiChat.emptyTitle') }}
              </div>
              <div class="chat-empty__desc">
                {{ $t('page.aiChat.emptyDesc') }}
              </div>
            </div>

            <div
              v-for="msg in messages"
              :key="msg.id"
              class="msg"
              :class="msg.role === 'USER' ? 'msg--user' : 'msg--ai'"
            >
              <div class="msg__avatar">
                <Iconify
                  :icon="msg.role === 'USER' ? 'lucide:user' : 'lucide:bot'"
                  :width="16"
                />
              </div>
              <div class="msg__main">
                <div class="msg__meta">
                  <span class="msg__who">
                    {{
                      $t(
                        msg.role === 'USER'
                          ? 'page.aiChat.you'
                          : 'page.aiChat.assistant',
                      )
                    }}
                  </span>
                  <span v-if="msg.createdAt">{{
                    dateUtil(msg.createdAt).format('HH:mm:ss')
                  }}</span>
                </div>

                <!-- 用户消息：纯文本气泡 -->
                <div
                  v-if="msg.role === 'USER'"
                  class="msg__bubble msg__bubble--user"
                >
                  {{ msg.content }}
                </div>
                <!-- AI 消息：markdown 渲染气泡 -->
                <div v-else class="msg__bubble msg__bubble--ai">
                  <!-- eslint-disable-next-line vue/no-v-html -- 已经过 DOMPurify 消毒，见 renderMarkdown -->
                  <div
                    class="markdown-body"
                    v-html="renderMarkdown(msg.content || '')"
                  />
                  <div
                    v-if="msg.promptTokens || msg.completionTokens"
                    class="msg__foot"
                    :title="`${msg.modelName || ''} · ${msg.durationMs || 0}ms`"
                  >
                    <Iconify icon="lucide:hash" :width="11" />
                    {{
                      $t('page.aiChat.tokensUsage', {
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
                <Iconify icon="lucide:bot" :width="16" />
              </div>
              <div class="msg__main">
                <div class="msg__meta">
                  <span class="msg__who">{{
                    $t('page.aiChat.assistant')
                  }}</span>
                </div>
                <div class="msg__bubble msg__bubble--ai">
                  <!-- 工具调用可见化：模型调工具期间的中间过程帧 -->
                  <div
                    v-for="(tc, i) in toolCalls"
                    :key="`${i}-${tc.name}`"
                    class="tool-call"
                  >
                    <span class="tool-call__head">
                      🛠 {{ tc.name }}({{ tc.arguments }})
                    </span>
                    <span class="tool-call__result">{{ tc.result }}</span>
                  </div>
                  <!-- eslint-disable-next-line vue/no-v-html -- 已经过 DOMPurify 消毒，见 renderMarkdown -->
                  <div
                    v-if="streamingText"
                    class="markdown-body"
                    v-html="renderMarkdown(streamingText)"
                  />
                  <div v-else class="typing">
                    <i />
                    <i />
                    <i />
                  </div>
                  <div class="msg__foot">
                    <Iconify
                      class="is-spinning"
                      icon="lucide:loader-2"
                      :width="11"
                    />
                    {{ $t('page.aiChat.streaming') }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区 -->
        <div class="composer">
          <div class="composer__box">
            <a-textarea
              v-model:value="input"
              class="composer__input"
              :bordered="false"
              :auto-size="{ minRows: 2, maxRows: 6 }"
              :disabled="sending"
              :placeholder="$t('page.aiChat.inputPlaceholder')"
              @keydown.enter.exact.prevent="handleSend"
            />
            <div class="composer__bar">
              <div class="composer__tool">
                <Iconify icon="lucide:book-open" :width="13" />
                <span class="composer__tool-label">{{
                  $t('page.aiChat.knowledgeBase')
                }}</span>
                <a-select
                  v-model:value="knowledgeBaseId"
                  allow-clear
                  class="composer__kb"
                  size="small"
                  :placeholder="$t('page.aiChat.knowledgeBasePlaceholder')"
                  :options="
                    knowledgeBases.map((kb) => ({
                      label: kb.name || `#${kb.id}`,
                      value: kb.id,
                    }))
                  "
                />
              </div>
              <div class="composer__actions">
                <span class="composer__hint">{{
                  $t('page.aiChat.sendHint')
                }}</span>
                <a-button
                  :disabled="!input.trim()"
                  :loading="sending"
                  shape="round"
                  type="primary"
                  @click="handleSend"
                >
                  <template #icon>
                    <Iconify v-if="!sending" icon="lucide:send" />
                  </template>
                  {{
                    sending ? $t('page.aiChat.sending') : $t('page.aiChat.send')
                  }}
                </a-button>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </Page>
</template>

<style scoped>
/* 颜色一律走 vben 设计令牌（--card/--border/--primary/…，见 design-tokens）：
   主色可配置、暗色自动跟随。tailwind 调色板类（gray-200/blue-500）在切主题时会脱节。 */

.chat-card {
  min-height: 0;
  overflow: hidden;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-radius: 12px;
}

/* ── 面板头部 ─────────────────────────────────────────────────────── */
.panel-head {
  display: flex;
  flex-shrink: 0;
  gap: 6px;
  align-items: center;
  height: 40px;
  padding: 0 12px;
  font-size: 13px;
  font-weight: 600;
  color: hsl(var(--muted-foreground));
  border-bottom: 1px solid hsl(var(--border));
}

.panel-head__title {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  color: hsl(var(--foreground));
  white-space: nowrap;
}

.panel-head__badge {
  flex-shrink: 0;
  padding: 0 6px;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 17px;
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted));
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
  color: hsl(var(--primary));
  white-space: nowrap;
  background: hsl(var(--primary) / 10%);
  border: 1px solid hsl(var(--primary) / 20%);
  border-radius: 999px;
}

/* ── 会话列表 ─────────────────────────────────────────────────────── */
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
  color: hsl(var(--muted-foreground));
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
}

.conv-item + .conv-item {
  margin-top: 2px;
}

.conv-item:hover {
  background: hsl(var(--accent));
}

/* 选中态按 docs/design-language.md §4 定稿：主色实底 + 前景色文字 + 8px 圆角，
   不得用左侧竖条 / 淡色底 / 字重加粗区分。旧写法（primary/10% 淡底 + 3px 竖条 +
   主色加粗文字）实测暗色下标题对比度仅 3.29:1，低于 13px 正文要求的 4.5:1。 */
.conv-item.is-active {
  color: hsl(var(--primary-foreground));
  background: hsl(var(--primary));
}

.conv-item.is-active .conv-item__title {
  color: inherit;
}

.conv-item.is-active .conv-item__time {
  /* 主色实底上再压透明度会掉到 3.3:1（11px 文字要求 4.5:1），故用满不透明的前景色，
     层次靠字号与位置表达 */
  color: hsl(var(--primary-foreground));
}

.conv-item.is-active .row-op {
  color: hsl(var(--primary-foreground));
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
  color: hsl(var(--card-foreground));
  white-space: nowrap;
}

.conv-item__time {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: hsl(var(--muted-foreground));
}

.conv-item__input {
  width: 100%;
  min-width: 0;
  padding: 2px 6px;
  font-size: 13px;
  color: hsl(var(--card-foreground));
  background: transparent;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  outline: none;
}

.conv-item__input:focus {
  border-color: hsl(var(--primary));
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
  color: hsl(var(--muted-foreground));
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 6px;
  transition:
    color 0.15s ease,
    background-color 0.15s ease;
}

.row-op:hover {
  color: hsl(var(--primary));
  background: hsl(var(--accent-hover));
}

.row-op--danger:hover {
  color: hsl(var(--destructive));
  background: hsl(var(--destructive) / 12%);
}

/* ── 消息区 ───────────────────────────────────────────────────────── */
.msg-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* 正文与输入框同宽，形成一条居中的阅读柱 */
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
  color: hsl(var(--primary));
  background: hsl(var(--primary) / 10%);
  border-radius: 50%;
}

.chat-empty__title {
  font-size: 16px;
  font-weight: 600;
  color: hsl(var(--card-foreground));
}

.chat-empty__desc {
  max-width: 26rem;
  font-size: 13px;
  line-height: 1.6;
  color: hsl(var(--muted-foreground));
}

.msg {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.msg--user {
  flex-direction: row-reverse;
}

.msg--user .msg__main {
  align-items: flex-end;
}

.msg--user .msg__meta {
  flex-direction: row-reverse;
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
  color: hsl(var(--primary));
  background: hsl(var(--primary) / 10%);
}

.msg--user .msg__avatar {
  color: hsl(var(--muted-foreground));
  background: hsl(var(--muted));
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
  color: hsl(var(--muted-foreground));
}

.msg__who {
  font-weight: 600;
  color: hsl(var(--muted-foreground));
}

.msg__bubble {
  padding: 9px 13px;
  font-size: 14px;
  line-height: 1.7;
  overflow-wrap: break-word;
  border-radius: 12px;
}

.msg__bubble--user {
  color: hsl(var(--primary-foreground));
  white-space: pre-wrap;
  background: hsl(var(--primary));
  border-top-right-radius: 4px;
}

/* 主色 5% 叠底：暗色下是蓝黑调的微提亮（hsl 中性灰的 --muted 与蓝黑底冷暖打架），
   亮色下≈规范的气泡灰 */
.msg__bubble--ai {
  min-width: 0;
  color: hsl(var(--card-foreground));
  background: hsl(var(--primary) / 5%);
  border: 1px solid hsl(var(--border));
  border-top-left-radius: 4px;
}

.msg__foot {
  display: flex;
  gap: 4px;
  align-items: center;
  margin-top: 6px;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: hsl(var(--muted-foreground));
}

.typing {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  height: 20px;
}

.typing i {
  width: 6px;
  height: 6px;
  background: hsl(var(--muted-foreground));
  border-radius: 50%;
  animation: typing-bounce 1.2s ease-in-out infinite;
}

.typing i:nth-child(2) {
  animation-delay: 0.15s;
}

.typing i:nth-child(3) {
  animation-delay: 0.3s;
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

.is-spinning {
  display: inline-flex;
  animation: typing-spin 1s linear infinite;
}

@keyframes typing-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ── 输入区 ───────────────────────────────────────────────────────── */
.composer {
  flex-shrink: 0;
  padding: 12px 24px 16px;
  background: hsl(var(--card));
  border-top: 1px solid hsl(var(--border));
}

.composer__box {
  max-width: 56rem;
  padding: 6px 10px 8px;
  margin: 0 auto;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--input));
  border-radius: 12px;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.composer__box:hover {
  border-color: hsl(var(--primary) / 40%);
}

.composer__box:focus-within {
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 2px hsl(var(--primary) / 12%);
}

/* borderless 文本域：与 composer 外框只保留一圈边框 */
.composer__box :deep(.composer__input.ant-input),
.composer__box :deep(.composer__input textarea) {
  padding: 4px 2px;
  font-size: 14px;
  line-height: 1.6;
  color: hsl(var(--card-foreground));
  background: transparent !important;
  border: none !important;
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
  color: hsl(var(--muted-foreground));
}

.composer__tool-label {
  flex-shrink: 0;
  font-size: 12px;
}

/* 窄屏下可收缩：固定 width 会被 flex 压缩到 placeholder 都放不下 */
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
  color: hsl(var(--muted-foreground));
}

/* ── Markdown 正文 ────────────────────────────────────────────────── */
/* tailwind preflight 会清掉列表符号/标题字号/引用缩进，
   LLM 输出以列表与标题为主，必须在气泡内逐项还原，否则回复糊成一片纯文本。 */
.markdown-body :deep(p) {
  margin: 0 0 0.5em;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4),
.markdown-body :deep(h5),
.markdown-body :deep(h6) {
  margin: 0.8em 0 0.4em;
  font-weight: 600;
  line-height: 1.5;
  color: hsl(var(--card-foreground));
}

.markdown-body :deep(h1:first-child),
.markdown-body :deep(h2:first-child),
.markdown-body :deep(h3:first-child),
.markdown-body :deep(h4:first-child) {
  margin-top: 0;
}

.markdown-body :deep(h1) {
  font-size: 1.25em;
}

.markdown-body :deep(h2) {
  font-size: 1.15em;
}

.markdown-body :deep(h3) {
  font-size: 1.05em;
}

.markdown-body :deep(h4),
.markdown-body :deep(h5),
.markdown-body :deep(h6) {
  font-size: 1em;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 1.5em;
  margin: 0.4em 0;
}

.markdown-body :deep(ul:last-child),
.markdown-body :deep(ol:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(ul) {
  list-style: disc;
}

.markdown-body :deep(ul ul) {
  list-style: circle;
}

.markdown-body :deep(ol) {
  list-style: decimal;
}

.markdown-body :deep(li) {
  margin: 0.2em 0;
}

.markdown-body :deep(li > p) {
  margin: 0;
}

.markdown-body :deep(a) {
  color: hsl(var(--primary));
  text-decoration: none;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

.markdown-body :deep(blockquote) {
  padding: 2px 0 2px 12px;
  margin: 0.5em 0;
  color: hsl(var(--muted-foreground));
  border-left: 3px solid hsl(var(--border));
}

.markdown-body :deep(hr) {
  margin: 0.8em 0;
  border: none;
  border-top: 1px solid hsl(var(--border));
}

.markdown-body :deep(strong) {
  font-weight: 600;
}

.markdown-body :deep(img) {
  max-width: 100%;
  border-radius: 8px;
}

.markdown-body :deep(code) {
  padding: 1px 5px;
  font-size: 0.92em;
  background: hsl(var(--muted));
  border: 1px solid hsl(var(--border));
  border-radius: 4px;
}

.markdown-body :deep(pre) {
  max-width: 100%;
  padding: 12px;
  margin: 0.5em 0;
  overflow-x: auto;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
}

.markdown-body :deep(pre:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(pre code) {
  padding: 0;
  font-size: 12.5px;
  line-height: 1.6;
  background: transparent;
  border: none;
}

.markdown-body :deep(table) {
  display: block;
  max-width: 100%;
  margin: 0.5em 0;
  overflow-x: auto;
  border-collapse: collapse;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: 5px 10px;
  border: 1px solid hsl(var(--border));
}

.markdown-body :deep(th) {
  font-weight: 600;
  background: hsl(var(--muted));
}

/* 工具调用可见化：ai_chat_tool 帧的展示条（模型调工具期间的中间过程，流式结束即清） */
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
  border: 1px dashed hsl(var(--border));
  border-radius: 8px;
  background: hsl(var(--muted));
}

.tool-call__head {
  color: hsl(var(--muted-foreground));
}

.tool-call__result {
  color: hsl(var(--foreground));
  word-break: break-all;
}
</style>
