import { useCallback, useEffect, useMemo, useState } from 'react';
// ReactFlow 样式必须引入：handle/controls/minimap 全靠它——漏引时 handle 塌成 0 高，
// 连线的命中判定永远失败（e2e 实证：真实鼠标在 handle 中心 down/move/up 仍连不出边）
import '@xyflow/react/dist/style.css';
import {
  Background,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  type Connection,
  type Edge,
  type Node,
  type NodeProps,
  useEdgesState,
  useNodesState,
} from '@xyflow/react';
import { Drawer, Button, Space, Form, Input, InputNumber, Select, Switch, App, Tag, Timeline, Descriptions, Popconfirm, Typography } from 'antd';
import { SaveOutlined, PlayCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { aiservicev1_AiAgentApp as AiAgentApp } from '@/api/generated/admin/service/v1';
import { useListAiKnowledgeBases } from '@/api/hooks/ai-knowledge';
import { useListAiProviders } from '@/api/hooks/ai-provider';
import { useRunAiAgentApp, useUpdateAiAgentApp } from '@/api/hooks/ai-agent';
import { useAgentRunStream } from './agent-runs-drawer';
import { PaginationQuery } from '@/core';

/**
 * Agent 画布编辑器（M1）：
 * - 图 DSL 与后端 docs/ai_agent_orchestration.md §2 逐字对齐：node{id,type,position,data} +
 *   edge{id,source,target,branch?}；节点 id 必须是合法标识符（条件表达式的变量名就是它）。
 * - 出度规则在连线时就地拦（condition 恰两条带 true/false 分支、其余至多一条），
 *   全图校验（唯一 start/可达/无环）交给保存时后端强校验——一处真理源，前端不复制第二份。
 * - 试运行面板就地展示 output + trace 时间线 + token 用量；运行台账浏览归 M2。
 */

type AgentNodeType = 'start' | 'knowledge' | 'llm' | 'http' | 'condition' | 'end';

const NODE_TYPE_ORDER: AgentNodeType[] = ['start', 'knowledge', 'llm', 'http', 'condition', 'end'];

// 节点色走 styles/semantic-text.css 的主题档：原先写死的 #52c41a/#1677ff/#8c8c8c
// 是"填充档"，当图标文字用在暗色节点底上只有 2~3:1，且 #1677ff 是已退役的旧主色。
const NODE_META: Record<AgentNodeType, { color: string; icon: string }> = {
  start: { color: 'var(--gowind-node-start)', icon: '▶' },
  knowledge: { color: 'var(--gowind-node-knowledge)', icon: '📚' },
  llm: { color: 'var(--gowind-primary-text)', icon: '✦' },
  http: { color: 'var(--gowind-node-http, #4a8dff)', icon: '⇋' },
  condition: { color: 'var(--gowind-node-condition)', icon: '⑃' },
  end: { color: 'var(--gowind-node-end)', icon: '■' },
};

/** 画布节点 data：nodeType + 图节点的 data 原文（配置抽屉直接编辑它）。 */
type CanvasNodeData = {
  nodeType: AgentNodeType;
  label: string;
  config: Record<string, any>;
};

function AgentFlowNode({ data, selected }: NodeProps) {
  const d = data as CanvasNodeData;
  const meta = NODE_META[d.nodeType] ?? NODE_META.llm;
  return (
    <div
      style={{
        minWidth: 150,
        padding: '8px 12px',
        borderRadius: 8,
        border: `2px solid ${selected ? meta.color : 'transparent'}`,
        background: 'var(--ant-color-bg-container, #fff)',
        boxShadow: '0 1px 4px rgba(0,0,0,.12)',
      }}
    >
      <Handle type="target" position={Position.Top} />
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{ color: meta.color }}>{meta.icon}</span>
        <span style={{ fontWeight: 600 }}>{d.label}</span>
      </div>
      <div style={{ fontSize: 12, color: 'var(--ant-color-text-secondary, #888)', marginTop: 4 }}>
        {configSummary(d.nodeType, d.config)}
      </div>
      <Handle type="source" position={Position.Bottom} />
    </div>
  );
}

function configSummary(nodeType: AgentNodeType, config: Record<string, any> | undefined): string {
  if (!config) return '';
  switch (nodeType) {
    case 'start':
      return (config.inputVars ?? []).map((v: any) => v.name).join(', ');
    case 'knowledge':
      return config.baseId ? `kb#${config.baseId} topK=${config.topK ?? 4}` : '';
    case 'llm':
      return config.model || (config.providerId ? `provider#${config.providerId}` : '');
    case 'http':
      return config.url ? `${config.method ?? 'GET'} ${String(config.url).slice(0, 36)}` : '';
    case 'condition':
      return config.expression || '';
    case 'end':
      return config.output ? String(config.output).slice(0, 40) : '';
    default:
      return '';
  }
}

const nodeTypes = { agent: AgentFlowNode };

/** 图 DSL ↔ ReactFlow 状态的互转（唯一两处，别处不得各写一份）。 */
function graphToFlow(graphJson: string): { nodes: Node[]; edges: Edge[] } {
  if (!graphJson) return { nodes: [], edges: [] };
  try {
    const g = JSON.parse(graphJson);
    return {
      nodes: (g.nodes ?? []).map((n: any) => ({
        id: n.id,
        type: 'agent',
        position: n.position ?? { x: 0, y: 0 },
        data: {
          nodeType: n.type,
          label: `${n.id}`,
          config: n.data ?? {},
        } satisfies CanvasNodeData,
      })),
      edges: (g.edges ?? []).map((e: any, i: number) => ({
        id: e.id ?? `edge_${i}`,
        source: e.source,
        target: e.target,
        label: e.branch,
        animated: e.branch === 'true',
      })),
    };
  } catch {
    return { nodes: [], edges: [] };
  }
}

function flowToGraph(nodes: Node[], edges: Edge[]): string {
  return JSON.stringify({
    nodes: nodes.map((n) => ({
      id: n.id,
      type: (n.data as CanvasNodeData).nodeType,
      position: { x: n.position.x, y: n.position.y },
      data: (n.data as CanvasNodeData).config,
    })),
    edges: edges.map((e, i) => ({
      id: e.id ?? `edge_${i}`,
      source: e.source,
      target: e.target,
      ...(e.label ? { branch: String(e.label) } : {}),
    })),
  });
}

export interface AgentCanvasProps {
  open: boolean;
  app: AiAgentApp | undefined;
  onClose: () => void;
}

export default function AgentCanvas({ open, app, onClose }: AgentCanvasProps) {
  const { t } = useTranslation('ai-agent');
  const { message } = App.useApp();

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedId, setSelectedId] = useState<string | undefined>();
  const [saving, setSaving] = useState(false);
  const [testOpen, setTestOpen] = useState(false);

  useEffect(() => {
    if (open && app) {
      const { nodes: n, edges: e } = graphToFlow(app.graph ?? '');
      setNodes(n);
      setEdges(e);
      setSelectedId(undefined);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, app?.id]);

  const selectedNode = useMemo(
    () => nodes.find((n) => n.id === selectedId),
    [nodes, selectedId],
  );

  const nextNodeId = useCallback(
    (nodeType: AgentNodeType): string => {
      const prefix = `${nodeType}_`;
      let seq = 1;
      while (nodes.some((n) => n.id === `${prefix}${seq}`)) seq++;
      return `${prefix}${seq}`;
    },
    [nodes],
  );

  const addNode = useCallback(
    (nodeType: AgentNodeType) => {
      const id = nextNodeId(nodeType);
      const max = nodes.reduce((m, n) => Math.max(m, n.position.y), 0);
      const node: Node = {
        id,
        type: 'agent',
        position: { x: 120 + (nodes.length % 3) * 60, y: max + 90 },
        data: { nodeType, label: id, config: defaultConfig(nodeType) } satisfies CanvasNodeData,
      };
      setNodes((ns) => ns.concat(node));
      setSelectedId(id);
    },
    [nodes, nextNodeId, setNodes],
  );

  const onConnect = useCallback(
    (connection: Connection) => {
      const { source, target } = connection;
      if (!source || !target || source === target) return;

      const sourceNode = nodes.find((n) => n.id === source);
      if (!sourceNode) return;
      const nodeType = (sourceNode.data as CanvasNodeData).nodeType;
      const existing = edges.filter((e) => e.source === source);

      if (nodeType === 'end') {
        message.error(t('canvas.error.endNoOut'));
        return;
      }
      if (nodeType === 'condition') {
        if (existing.length >= 2) {
          message.error(t('canvas.error.conditionTwoOut'));
          return;
        }
        const branch = existing.length === 0 ? 'true' : 'false';
        setEdges((es) =>
          es.concat({
            id: `edge_${source}_${target}_${branch}_${Date.now()}`,
            source,
            target,
            label: branch,
            animated: branch === 'true',
          }),
        );
        return;
      }
      if (existing.length >= 1) {
        message.error(t('canvas.error.singleOut'));
        return;
      }
      setEdges((es) =>
        es.concat({ id: `edge_${source}_${target}_${Date.now()}`, source, target }),
      );
    },
    [nodes, edges, setEdges, message, t],
  );

  const updateSelectedConfig = useCallback(
    (config: Record<string, any>) => {
      if (!selectedId) return;
      setNodes((ns) =>
        ns.map((n) =>
          n.id === selectedId
            ? { ...n, data: { ...(n.data as CanvasNodeData), config } }
            : n,
        ),
      );
    },
    [selectedId, setNodes],
  );

  const updateMutation = useUpdateAiAgentApp();
  const runMutation = useRunAiAgentApp();

  const handleSave = async () => {
    if (!app?.id) return;
    setSaving(true);
    try {
      await updateMutation.mutateAsync({
        id: app.id,
        data: { id: app.id, graph: flowToGraph(nodes, edges) },
        updateMask: 'graph',
      });
      message.success(t('canvas.saveSuccess'));
    } catch (error: any) {
      // 后端强校验的报错原文在这里最有用（唯一 start/可达性……），直接透出
      message.error(error?.message || t('canvas.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const startNode = nodes.find((n) => (n.data as CanvasNodeData).nodeType === 'start');
  const inputVars: any[] = startNode ? ((startNode.data as CanvasNodeData).config.inputVars ?? []) : [];

  return (
    <Drawer
      title={`${t('canvas.title')} · ${app?.name ?? ''}`}
      open={open}
      onClose={onClose}
      width="100%"
      // 全屏画布不需要遮罩：mask 会以同级 z-index 压住工具条（真实点击被拦，e2e 实证）
      mask={false}
      destroyOnHidden
      styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column' } }}
    >
      <div style={{ padding: '8px 16px', borderBottom: '1px solid var(--ant-color-border-secondary, #f0f0f0)' }}>
        <Space wrap>
          {NODE_TYPE_ORDER.map((nodeType) => (
            <Button key={nodeType} size="small" icon={<PlusOutlined />} onClick={() => addNode(nodeType)}>
              {t(`canvas.nodeType.${nodeType}`)}
            </Button>
          ))}
          <Button type="primary" size="small" icon={<SaveOutlined />} loading={saving} onClick={handleSave}>
            {t('canvas.save')}
          </Button>
          <Button size="small" icon={<PlayCircleOutlined />} onClick={() => setTestOpen(true)}>
            {t('canvas.testRun')}
          </Button>
        </Space>
      </div>

      <div style={{ flex: 1, minHeight: 0 }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          nodeTypes={nodeTypes}
          onNodeClick={(_, node) => setSelectedId(node.id)}
          onPaneClick={() => setSelectedId(undefined)}
          fitView
        >
          <Background />
          <Controls />
          <MiniMap pannable />
        </ReactFlow>
      </div>

      {/* 节点配置抽屉 */}
      <NodeConfigDrawer
        key={selectedId ?? 'none'}
        node={selectedNode}
        open={!!selectedNode}
        onClose={() => setSelectedId(undefined)}
        onConfigChange={updateSelectedConfig}
        onDelete={() => {
          if (!selectedId) return;
          setNodes((ns) => ns.filter((n) => n.id !== selectedId));
          setEdges((es) => es.filter((e) => e.source !== selectedId && e.target !== selectedId));
          setSelectedId(undefined);
        }}
      />

      {/* 试运行面板 */}
      <TestRunDrawer
        open={testOpen}
        appId={app?.id}
        inputVars={inputVars}
        onClose={() => setTestOpen(false)}
        runMutation={runMutation}
      />
    </Drawer>
  );
}

function defaultConfig(nodeType: AgentNodeType): Record<string, any> {
  switch (nodeType) {
    case 'start':
      return { inputVars: [{ name: 'question', label: 'question', required: true }] };
    case 'knowledge':
      return { topK: 4 };
    case 'llm':
      return { temperature: 0.3 };
    case 'http':
      return { method: 'GET', timeoutSeconds: 15 };
    case 'condition':
      return { expression: '' };
    case 'end':
      return { output: '' };
    default:
      return {};
  }
}


/** http 节点 headers 键值对编辑（存 [{key,value}] 数组，引擎侧两种形态都收） */
function HeadersList() {
  const { t } = useTranslation('ai-agent');
  return (
    <Form.List name="headers">
      {(fields, { add, remove }) => (
        <>
          {fields.map((field) => (
            <Space key={field.key} style={{ display: 'flex', marginBottom: 4 }} align="baseline">
              <Form.Item name={[field.name, 'key']} style={{ marginBottom: 0 }}>
                <Input placeholder="Content-Type" style={{ width: 150 }} />
              </Form.Item>
              <Form.Item name={[field.name, 'value']} style={{ marginBottom: 0 }}>
                <Input placeholder="application/json" style={{ width: 180 }} />
              </Form.Item>
              <Button type="text" danger onClick={() => remove(field.name)}>{t('canvas.remove')}</Button>
            </Space>
          ))}
          <Button type="dashed" size="small" icon={<PlusOutlined />} onClick={() => add({ key: '', value: '' })}>
            {t('canvas.httpAddHeader')}
          </Button>
        </>
      )}
    </Form.List>
  );
}

// ============ 节点配置抽屉 ============

function NodeConfigDrawer({
  node,
  open,
  onClose,
  onConfigChange,
  onDelete,
}: {
  node: Node | undefined;
  open: boolean;
  onClose: () => void;
  onConfigChange: (config: Record<string, any>) => void;
  onDelete: () => void;
}) {
  const { t } = useTranslation('ai-agent');
  const [form] = Form.useForm();
  const knowledgeQuery = useListAiKnowledgeBases(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }));
  const providerQuery = useListAiProviders(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }));

  const nodeType = node ? (node.data as CanvasNodeData).nodeType : 'llm';
  const config = node ? (node.data as CanvasNodeData).config : {};

  const handleValuesChange = () => {
    // 表单任意改动即时回写画布节点（保存时才落库）
    const values = form.getFieldsValue();
    onConfigChange(values);
  };

  return (
    <Drawer
      title={`${t('canvas.config')} · ${node?.id ?? ''}`}
      open={open}
      onClose={onClose}
      width={420}
      // 画布内侧板不带 mask：配置时仍可直接操作画布与工具条（带 mask 会全屏拦截，e2e 实证）
      mask={false}
      destroyOnHidden
      extra={
        nodeType !== 'start' ? (
          <Popconfirm title={t('canvas.deleteNodeConfirm')} onConfirm={onDelete}>
            <Button danger size="small">{t('canvas.deleteNode')}</Button>
          </Popconfirm>
        ) : undefined
      }
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={config}
        onValuesChange={handleValuesChange}
      >
        {nodeType === 'start' && (
          <Form.List name="inputVars" initialValue={config.inputVars ?? []}>
            {(fields, { add, remove }) => (
              <>
                {fields.map((field) => (
                  <Space key={field.key} align="baseline" style={{ display: 'flex' }}>
                    <Form.Item
                      name={[field.name, 'name']}
                      rules={[{ required: true, message: t('canvas.requiredVarName') }, { pattern: /^[a-zA-Z_][a-zA-Z0-9_]*$/, message: t('canvas.invalidVarName') }]}
                    >
                      <Input placeholder={t('canvas.varName')} style={{ width: 140 }} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'label']}>
                      <Input placeholder={t('canvas.varLabel')} style={{ width: 120 }} />
                    </Form.Item>
                    <Form.Item name={[field.name, 'required']} valuePropName="checked">
                      <Switch size="small" />
                    </Form.Item>
                    <Button type="text" danger onClick={() => remove(field.name)}>{t('canvas.remove')}</Button>
                  </Space>
                ))}
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ name: '', label: '', required: false })}>
                  {t('canvas.addVar')}
                </Button>
              </>
            )}
          </Form.List>
        )}

        {nodeType === 'knowledge' && (
          <>
            <Form.Item name="baseId" label={t('canvas.knowledgeBase')} rules={[{ required: true, message: t('canvas.requiredKnowledgeBase') }]}>
              <Select
                options={(knowledgeQuery.data?.items ?? []).map((k: any) => ({ value: k.id, label: k.name }))}
                allowClear
              />
            </Form.Item>
            <Form.Item name="topK" label="topK">
              <InputNumber min={1} max={20} precision={0} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item name="query" label={t('canvas.knowledgeQuery')} tooltip={t('canvas.templateHint')}>
              <Input.TextArea rows={2} placeholder="{{start_1.question}}" />
            </Form.Item>
          </>
        )}

        {nodeType === 'llm' && (
          <>
            <Form.Item name="providerId" label={t('canvas.provider')} tooltip={t('canvas.providerHint')}>
              <Select
                options={(providerQuery.data?.items ?? []).map((p: any) => ({ value: p.id, label: p.name }))}
                allowClear
              />
            </Form.Item>
            <Form.Item name="model" label={t('canvas.model')} tooltip={t('canvas.modelHint')}>
              <Input placeholder="deepseek-chat" />
            </Form.Item>
            <Form.Item name="systemPrompt" label={t('canvas.systemPrompt')} tooltip={t('canvas.templateHint')}>
              <Input.TextArea rows={3} />
            </Form.Item>
            <Form.Item name="userPrompt" label={t('canvas.userPrompt')} tooltip={t('canvas.templateHint')} rules={[{ required: true, message: t('canvas.requiredUserPrompt') }]}>
              <Input.TextArea rows={4} placeholder={'资料：{{kb_1.context}}\n问题：{{start_1.question}}'} />
            </Form.Item>
            <Form.Item name="temperature" label="temperature">
              <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
            </Form.Item>
          </>
        )}

        {nodeType === 'condition' && (
          <Form.Item
            name="expression"
            label={t('canvas.expression')}
            tooltip={t('canvas.expressionHint')}
            rules={[{ required: true, message: t('canvas.requiredExpression') }]}
          >
            <Input.TextArea rows={3} placeholder={'llm_1.output contains "退款"'} style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        )}

                {nodeType === 'http' && (
          <>
            <Form.Item name="url" label={t('canvas.httpUrl')} rules={[{ required: true, message: t('canvas.requiredHttpUrl') }]}>
              <Input.TextArea
                rows={2}
                placeholder="https://api.example.com/orders/{{start_1.orderId}}"
                style={{ fontFamily: 'monospace' }}
              />
            </Form.Item>
            <Form.Item name="method" label={t('canvas.httpMethod')} initialValue="GET">
              <Select
                options={['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD'].map((m) => ({ value: m, label: m }))}
              />
            </Form.Item>
            <Form.Item label={t('canvas.httpHeaders')} style={{ marginBottom: 4 }}>
              <HeadersList />
            </Form.Item>
            <Form.Item shouldUpdate={(prev, cur) => prev.method !== cur.method} style={{ marginBottom: 0 }}>
              {({ getFieldValue }) => {
                const method = getFieldValue('method');
                return method !== 'GET' && method !== 'HEAD' ? (
                  <Form.Item name="body" label={t('canvas.httpBody')} tooltip={t('canvas.templateHint')}>
                    <Input.TextArea rows={4} placeholder='{"orderId": "{{start_1.orderId}}"}' style={{ fontFamily: 'monospace' }} />
                  </Form.Item>
                ) : null;
              }}
            </Form.Item>
            <Form.Item name="timeoutSeconds" label={t('canvas.httpTimeout')}>
              <InputNumber min={1} max={60} precision={0} style={{ width: '100%' }} />
            </Form.Item>
            <Typography.Paragraph type="warning" style={{ fontSize: 12, marginBottom: 0 }}>
              {t('canvas.httpSsrHint')}
            </Typography.Paragraph>
          </>
        )}

        {nodeType === 'end' && (
          <Form.Item name="output" label={t('canvas.endOutput')} tooltip={t('canvas.templateHint')}>
            <Input.TextArea rows={3} placeholder="{{llm_1.output}}" />
          </Form.Item>
        )}
      </Form>
      <div style={{ color: 'var(--ant-color-text-tertiary, #aaa)', fontSize: 12 }}>{t('canvas.configAutoSaved')}</div>
    </Drawer>
  );
}

// ============ 试运行面板 ============

type RunResult = {
  status?: string;
  output?: string;
  error?: string;
  trace?: string;
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  durationMs?: number;
};

function TestRunDrawer({
  open,
  appId,
  inputVars,
  onClose,
  runMutation,
}: {
  open: boolean;
  appId: number | undefined;
  inputVars: any[];
  onClose: () => void;
  runMutation: ReturnType<typeof useRunAiAgentApp>;
}) {
  const { t } = useTranslation('ai-agent');
  const [form] = Form.useForm();
  const [result, setResult] = useState<RunResult | undefined>();
  // SSE 实时轨迹帧（ai_agent_run）：Run 同步响应到达前先逐节点呈现，
  // 响应到达后以 result.trace 全量覆盖（尽力而为丢帧可容忍）
  const [liveSteps, setLiveSteps] = useState<Array<Record<string, any>>>([]);

  // useAgentRunStream 以 appId 过滤帧；仅抽屉打开时订阅
  useEffect(() => {
    if (!open || !appId) return;
    return useAgentRunStream(appId, (frame) => {
      if (frame.phase === 'node_end') {
        setLiveSteps((prev) => [
          ...prev,
          { nodeId: frame.nodeId, type: '', status: frame.status ?? 'ok', ms: 0, summary: frame.summary },
        ]);
      } else if (frame.phase === 'error') {
        setLiveSteps((prev) => [
          ...prev,
          { nodeId: '', type: '', status: 'error', ms: 0, error: frame.summary },
        ]);
      }
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, appId]);

  // 打开新一次试运行时清空上一次的实时帧
  useEffect(() => {
    if (open) setLiveSteps([]);
  }, [open]);

  const handleRun = async () => {
    if (!appId) return;
    const values = await form.validateFields().catch(() => null);
    if (values === null) return;
    try {
      const run = await runMutation.mutateAsync({ id: appId, input: values });
      setResult(run as RunResult);
    } catch (error: any) {
      // 图校验/配额类错误原文直接给到面板
      setResult({ status: 'FAILED', error: error?.message || String(error) });
    }
  };

  // 响应未到达时先用 SSE 实时帧呈现；响应到达后以全量 trace 覆盖
  let traceSteps: any[] = result?.trace ? [] : liveSteps;
  if (result?.trace) {
    try {
      traceSteps = JSON.parse(result.trace);
    } catch {
      traceSteps = liveSteps;
    }
  }

  return (
    <Drawer
      title={t('test.title')}
      open={open}
      onClose={() => {
        setResult(undefined);
        onClose();
      }}
      width={480}
      mask={false}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        {inputVars.map((v: any, i: number) => (
          <Form.Item
            key={`${v.name}_${i}`}
            name={v.name}
            label={v.label || v.name}
            rules={v.required ? [{ required: true, message: t('test.requiredInput') }] : []}
          >
            <Input.TextArea rows={2} autoSize />
          </Form.Item>
        ))}
        {inputVars.length === 0 && (
          <div style={{ color: 'var(--ant-color-text-tertiary, #aaa)', marginBottom: 12 }}>
            {t('test.noInputs')}
          </div>
        )}
        <Button type="primary" block loading={runMutation.isPending} onClick={handleRun}>
          {t('test.run')}
        </Button>
      </Form>

      {result && (
        <div style={{ marginTop: 16 }}>
          <Space style={{ marginBottom: 8 }}>
            {result.status === 'SUCCEEDED' ? (
              <Tag color="success">{t('test.succeeded')}</Tag>
            ) : (
              <Tag color="error">{t('test.failed')}</Tag>
            )}
            {result.totalTokens != null && <Tag>{t('test.tokens', { total: result.totalTokens })}</Tag>}
            {result.durationMs != null && <Tag>{result.durationMs}ms</Tag>}
          </Space>

          {result.error && (
            <div style={{ color: 'var(--ant-color-error, #cf1322)', whiteSpace: 'pre-wrap', marginBottom: 8 }}>
              {result.error}
            </div>
          )}

          {result.output != null && (
            <Descriptions size="small" column={1} className="mb-3">
              <Descriptions.Item label={t('test.output')}>
                <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{result.output}</pre>
              </Descriptions.Item>
            </Descriptions>
          )}

          {traceSteps.length > 0 && (
            <>
              <div style={{ fontWeight: 600, margin: '8px 0' }}>{t('test.trace')}</div>
              <Timeline
                items={traceSteps.map((step: any) => ({
                  color: step.status === 'ok' ? 'green' : 'red',
                  children: (
                    <div style={{ fontSize: 12 }}>
                      <div>
                        <code>{step.nodeId}</code> <Tag>{step.type}</Tag> {step.ms}ms
                      </div>
                      {step.summary && (
                        <div style={{ whiteSpace: 'pre-wrap', color: 'var(--ant-color-text-secondary, #888)' }}>
                          {String(step.summary).slice(0, 200)}
                        </div>
                      )}
                      {step.error && <div style={{ color: 'var(--ant-color-error, #cf1322)' }}>{step.error}</div>}
                    </div>
                  ),
                }))}
              />
            </>
          )}
        </div>
      )}
    </Drawer>
  );
}
