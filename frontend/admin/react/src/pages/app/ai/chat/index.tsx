import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { App, Button, Input, Popconfirm, Select, Tooltip } from 'antd';
import {
  BookOutlined,
  DeleteOutlined,
  EditOutlined,
  InboxOutlined,
  LoadingOutlined,
  MessageOutlined,
  NumberOutlined,
  PlusOutlined,
  RobotOutlined,
  SendOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { aiservicev1_AiConversation, aiservicev1_AiMessage } from '@/api/generated/admin/service/v1';
import { PaginationQuery, globalSSEClient, SSE_EVENT } from '@/core';
import {
  useDeleteAiConversation,
  useListAiConversations,
  useListAiMessages,
  useSendChat,
  useUpdateAiConversation,
} from '@/api/hooks/ai-chat';
import { fetchListAiKnowledgeBases } from '@/api/hooks/ai-knowledge';
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from '@/api/generated/admin/service/v1';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import './chat-page.css';

interface ChatChunk {
  // conversationId 为 uint64（protojson/SSE 帧下是字符串）
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

export default function AiChatPage() {
  const { t } = useTranslation('aiChat');
  const { message: antdMessage } = App.useApp();
  const queryClient = useQueryClient();

  // ── 会话列表 ──────────────────────────────────────────────────────
  const conversationsQuery = useListAiConversations(
    new PaginationQuery({ paging: { page: 1, pageSize: 100 }, orderBy: ['-last_message_at'] }),
  );
  const conversations = useMemo(
    () => (conversationsQuery.data?.items ?? []) as aiservicev1_AiConversation[],
    [conversationsQuery.data],
  );

  const [activeId, setActiveId] = useState<string | undefined>(undefined);

  // 仅在首次拉到列表时自动选中最近会话：放在 loadConversations 里会让「新建对话」
  // 立刻被重新选中顶回去，空态与"新会话"标题都渲染不出来
  const autoSelected = useRef(false);
  useEffect(() => {
    if (!autoSelected.current && conversations.length > 0) {
      autoSelected.current = true;
      setActiveId(conversations[0].id);
    }
  }, [conversations]);

  // 右栏标题：未选中会话时是"新会话"，选中后取标题，无标题回落到 ID
  const activeTitle = useMemo(() => {
    if (activeId === undefined) return t('untitled');
    const conv = conversations.find((c) => c.id === activeId);
    return conv?.title || `#${conv?.id}`;
  }, [activeId, conversations, t]);

  // ── 消息列表 ──────────────────────────────────────────────────────
  const messagesQuery = useListAiMessages(
    new PaginationQuery({
      paging: { page: 1, pageSize: 200 },
      formValues: activeId ? { conversation_id: activeId } : undefined,
      orderBy: ['id'],
    }),
    { enabled: activeId !== undefined },
  );
  const messages = useMemo(
    () => (messagesQuery.data?.items ?? []) as aiservicev1_AiMessage[],
    [messagesQuery.data],
  );

  // ── 会话重命名（内联编辑） ────────────────────────────────────────
  const [renamingId, setRenamingId] = useState<string | undefined>(undefined);
  const [renameText, setRenameText] = useState('');
  const updateConvMutation = useUpdateAiConversation();

  const startRename = (conv: aiservicev1_AiConversation) => {
    setRenamingId(conv.id!);
    setRenameText(conv.title ?? '');
  };

  const commitRename = (conv: (typeof conversations)[number]) => {
    const title = renameText.trim();
    setRenamingId(undefined);
    if (!title || title === (conv.title ?? '')) return;
    updateConvMutation.mutate(
      { id: conv.id!, values: { title } },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: ['listAiConversations'] });
        },
        onError: (error: Error) => antdMessage.error(error.message || t('fetchFailed')),
      },
    );
  };

  // ── 知识库选择（RAG：发送时携带 knowledgeBaseId） ─────────────────
  const [knowledgeBases, setKnowledgeBases] = useState<AiKnowledgeBase[]>([]);
  const [knowledgeBaseId, setKnowledgeBaseId] = useState<string | undefined>(undefined);

  // 头部知识库标签：未选则为空串（不渲染标签）
  const selectedKb = knowledgeBases.find((k) => k.id === knowledgeBaseId);
  const activeKbName = knowledgeBaseId === undefined ? '' : (selectedKb?.name || `#${knowledgeBaseId}`);

  useEffect(() => {
    fetchListAiKnowledgeBases(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }))
      .then((res) => setKnowledgeBases((res.items ?? []) as AiKnowledgeBase[]))
      .catch((error: Error) => console.error('fetch ai knowledge bases failed:', error));
  }, []);

  // ── 流式累积 ──────────────────────────────────────────────────────
  const [streamingText, setStreamingText] = useState('');
  // 工具调用可见化：ai_chat_tool 帧按到达顺序累积，随流式气泡一并渲染
  const [toolCalls, setToolCalls] = useState<ChatToolEvent[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = useCallback(() => {
    requestAnimationFrame(() => {
      scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
    });
  }, []);

  // SSE 订阅：本用户流上的 ai_chat_chunk，按会话归组累积
  useEffect(() => {
    const handler = (data: unknown) => {
      const chunk = data as ChatChunk;
      if (!chunk || typeof chunk !== 'object' || !chunk.conversationId) return;
      setStreamingText((prev) => (chunk.delta ? prev + chunk.delta : prev));
    };
    globalSSEClient.on(SSE_EVENT.AIChatChunk, handler);
    return () => {
      globalSSEClient.off(SSE_EVENT.AIChatChunk, handler);
    };
  }, []);

  // SSE 订阅：ai_chat_tool —— 模型每次本地工具执行完成后的一帧
  useEffect(() => {
    const handler = (data: unknown) => {
      const ev = data as ChatToolEvent;
      if (!ev || typeof ev !== 'object' || !ev.name) return;
      setToolCalls((prev) => [...prev, ev]);
    };
    globalSSEClient.on(SSE_EVENT.AIChatTool, handler);
    return () => {
      globalSSEClient.off(SSE_EVENT.AIChatTool, handler);
    };
  }, []);

  // 内容变化时滚到底
  useEffect(() => {
    scrollToBottom();
  }, [messages.length, streamingText, toolCalls.length, scrollToBottom]);

  // ── 发送 ──────────────────────────────────────────────────────────
  const [input, setInput] = useState('');
  const sendMutation = useSendChat({
    onSuccess: (resp) => {
      setInput('');
      setStreamingText('');
      setToolCalls([]);
      if (resp.conversation?.id) {
        setActiveId(resp.conversation.id);
      }
      queryClient.invalidateQueries({ queryKey: ['listAiConversations'] });
      queryClient.invalidateQueries({ queryKey: ['listAiMessages'] });
    },
    onError: (error: Error) => {
      setStreamingText('');
      setToolCalls([]);
      antdMessage.error(error.message || t('chatFailed'));
    },
  });

  const handleSend = () => {
    const content = input.trim();
    if (!content || sendMutation.isPending) return;
    // 先本地渲染用户气泡与占位：响应到达后由列表刷新接管
    setStreamingText('');
    // conversationId/knowledgeBaseId/providerId 均为 uint64（字符串），未选传 undefined（后端按 0 处理）
    sendMutation.mutate({ conversationId: activeId, providerId: undefined, content, knowledgeBaseId });
  };

  // ── 删除会话 ──────────────────────────────────────────────────────
  const deleteConversationMutation = useDeleteAiConversation({
    onSuccess: () => {
      antdMessage.success(t('deleteSuccess'));
      setStreamingText('');

      queryClient.invalidateQueries({ queryKey: ['listAiConversations'] });
      queryClient.invalidateQueries({ queryKey: ['listAiMessages'] });
    },
    onError: (error: Error) => antdMessage.error(error.message || t('fetchFailed')),
  });

  const handleDeleteConversation = (id: string) => {
    deleteConversationMutation.mutate({ id });
    if (activeId === id) {
      setActiveId(undefined);
    }
  };

  const handleNewConversation = () => {
    setActiveId(undefined);
    setStreamingText('');
    messagesQuery.refetch();
  };

  // ── 渲染 ──────────────────────────────────────────────────────────
  const isStreaming = sendMutation.isPending;

  return (
    <ContentContainer heightMode="fixed">
      <div className="ai-chat-page flex h-full min-h-0 gap-3">
        {/* 左栏：会话列表 */}
        <aside className="chat-card flex w-60 shrink-0 flex-col">
          <div className="panel-head">
            <MessageOutlined />
            <span className="panel-head__title">{t('conversations')}</span>
            {conversations.length > 0 && <span className="panel-head__badge">{conversations.length}</span>}
          </div>
          <div className="px-3 pt-2">
            <Button className="new-conv-btn" icon={<PlusOutlined />} onClick={handleNewConversation}>
              {t('newConversation')}
            </Button>
          </div>
          <div className="conv-list">
            {conversations.length === 0 ? (
              <div className="conv-list__empty">
                <InboxOutlined style={{ fontSize: 26 }} />
                <span>{t('noConversations')}</span>
              </div>
            ) : (
              conversations.map((conv) => (
                <div
                  key={conv.id}
                  role="button"
                  tabIndex={0}
                  className={`conv-item${activeId === conv.id ? ' is-active' : ''}`}
                  onClick={() => setActiveId(conv.id)}
                  onDoubleClick={() => startRename(conv)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') setActiveId(conv.id);
                  }}
                >
                  <div className="conv-item__text">
                    {renamingId === conv.id ? (
                      <Input
                        size="small"
                        autoFocus
                        value={renameText}
                        onClick={(e) => e.stopPropagation()}
                        onChange={(e) => setRenameText(e.target.value)}
                        onPressEnter={() => commitRename(conv)}
                        onBlur={() => commitRename(conv)}
                      />
                    ) : (
                      <>
                        <span className="conv-item__title">{conv.title || `#${conv.id}`}</span>
                        {conv.createdAt && (
                          <span className="conv-item__time">{dayjs(conv.createdAt).format('MM-DD HH:mm')}</span>
                        )}
                      </>
                    )}
                  </div>
                  <div className="conv-item__ops">
                    <Tooltip title={t('rename')}>
                      <button
                        type="button"
                        className="row-op"
                        aria-label={t('rename')}
                        onClick={(e) => {
                          e.stopPropagation();
                          startRename(conv);
                        }}
                      >
                        <EditOutlined style={{ fontSize: 14 }} />
                      </button>
                    </Tooltip>
                    <Popconfirm
                      title={t('deleteConversation')}
                      description={t('deleteConversationConfirm')}
                      onConfirm={(e) => {
                        e?.stopPropagation();
                        handleDeleteConversation(conv.id!);
                      }}
                      onCancel={(e) => e?.stopPropagation()}
                    >
                      <Tooltip title={t('deleteConversation')}>
                        <button
                          type="button"
                          className="row-op row-op--danger"
                          aria-label={t('deleteConversation')}
                          onClick={(e) => e.stopPropagation()}
                        >
                          <DeleteOutlined style={{ fontSize: 14 }} />
                        </button>
                      </Tooltip>
                    </Popconfirm>
                  </div>
                </div>
              ))
            )}
          </div>
        </aside>

        {/* 右栏：消息区 + 输入区 */}
        <section className="chat-card flex min-h-0 min-w-0 flex-1 flex-col">
          <div className="panel-head">
            <span className="panel-head__title" title={activeTitle}>
              {activeTitle}
            </span>
            {activeKbName && (
              <span className="panel-head__tag">
                <BookOutlined style={{ fontSize: 12 }} />
                {activeKbName}
              </span>
            )}
            {messages.length > 0 && <span className="panel-head__badge">{messages.length}</span>}
          </div>

          <div ref={scrollRef} className="msg-scroll">
            <div className="msg-list">
              {messages.length === 0 && !isStreaming ? (
                <div className="chat-empty">
                  <div className="chat-empty__icon">
                    <RobotOutlined style={{ fontSize: 24 }} />
                  </div>
                  <div className="chat-empty__title">{t('emptyTitle')}</div>
                  <div className="chat-empty__desc">{t('emptyDesc')}</div>
                </div>
              ) : (
                <>
                  {messages.map((msg) => (
                    <MessageBubble key={msg.id} message={msg} />
                  ))}
                  {isStreaming && (
                    <>
                      {toolCalls.map((tc, i) => (
                        <div key={`${i}-${tc.name}`} className="tool-call">
                          <span className="tool-call__head">🛠 {tc.name}({tc.arguments})</span>
                          <span className="tool-call__result">{tc.result}</span>
                        </div>
                      ))}
                      <StreamingBubble text={streamingText} />
                    </>
                  )}
                </>
              )}
            </div>
          </div>

          {/* 输入区：文本域与工具行同框，与正文同宽对齐阅读柱 */}
          <div className="composer">
            <div className="composer__box">
              <Input.TextArea
                className="composer__input"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder={t('inputPlaceholder')}
                autoSize={{ minRows: 2, maxRows: 6 }}
                onPressEnter={(e) => {
                  if (!e.shiftKey) {
                    e.preventDefault();
                    handleSend();
                  }
                }}
                disabled={isStreaming}
              />
              <div className="composer__bar">
                <div className="composer__tool">
                  <BookOutlined style={{ fontSize: 13 }} />
                  <span className="composer__tool-label">{t('knowledgeBase')}</span>
                  <Select
                    value={knowledgeBaseId}
                    onChange={(v) => setKnowledgeBaseId(v)}
                    allowClear
                    size="small"
                    className="composer__kb"
                    placeholder={t('knowledgeBasePlaceholder')}
                    options={knowledgeBases.map((kb) => ({ label: kb.name ?? `#${kb.id}`, value: kb.id! }))}
                  />
                </div>
                <div className="composer__actions">
                  <span className="composer__hint">{t('sendHint')}</span>
                  <Button
                    type="primary"
                    shape="round"
                    icon={<SendOutlined />}
                    loading={isStreaming}
                    onClick={handleSend}
                    disabled={!input.trim()}
                  >
                    {isStreaming ? t('sending') : t('send')}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </ContentContainer>
  );
}

// ── 子组件 ──────────────────────────────────────────────────────────

function Avatar({ role }: { role: 'assistant' | 'user' }) {
  const { t } = useTranslation('aiChat');
  return (
    <div className="msg__avatar">
      {role === 'assistant' ? <RobotOutlined /> : <UserOutlined />}
      <span className="sr-only">{role === 'assistant' ? t('assistant') : t('you')}</span>
    </div>
  );
}

function MessageBubble({ message: msg }: { message: aiservicev1_AiMessage }) {
  const { t } = useTranslation('aiChat');
  const isUser = msg.role === 'USER';

  if (isUser) {
    return (
      <div className="msg msg--user">
        <Avatar role="user" />
        <div className="msg__main">
          <div className="msg__meta">
            <span className="msg__who">{t('you')}</span>
            {msg.createdAt && <span>{dayjs(msg.createdAt).format('HH:mm:ss')}</span>}
          </div>
          <div className="msg__bubble msg__bubble--user">{msg.content}</div>
        </div>
      </div>
    );
  }

  return (
    <div className="msg msg--ai">
      <Avatar role="assistant" />
      <div className="msg__main">
        <div className="msg__meta">
          <span className="msg__who">{t('assistant')}</span>
          {msg.createdAt && <span>{dayjs(msg.createdAt).format('HH:mm:ss')}</span>}
        </div>
        <div className="msg__bubble msg__bubble--ai">
          <MarkdownContent content={msg.content ?? ''} />
          {(msg.promptTokens || msg.completionTokens) && (
            <Tooltip title={`${msg.modelName ?? ''} · ${msg.durationMs ?? 0}ms`}>
              <div className="msg__foot">
                <NumberOutlined style={{ fontSize: 11 }} />
                {t('tokensUsage', { prompt: msg.promptTokens ?? 0, completion: msg.completionTokens ?? 0 })}
              </div>
            </Tooltip>
          )}
        </div>
      </div>
    </div>
  );
}

/** 流式占位气泡：片段到达前用打点动画，到达后原地渲染 markdown */
function StreamingBubble({ text }: { text: string }) {
  const { t } = useTranslation('aiChat');
  return (
    <div className="msg msg--ai">
      <Avatar role="assistant" />
      <div className="msg__main">
        <div className="msg__meta">
          <span className="msg__who">{t('assistant')}</span>
        </div>
        <div className="msg__bubble msg__bubble--ai">
          {text ? <MarkdownContent content={text} /> : <span className="typing"><i /><i /><i /></span>}
          <div className="msg__foot">
            <LoadingOutlined className="is-spinning" style={{ fontSize: 11 }} />
            {t('streaming')}
          </div>
        </div>
      </div>
    </div>
  );
}

function MarkdownContent({ content }: { content: string }) {
  return (
    <div className="markdown-body break-words">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
    </div>
  );
}
