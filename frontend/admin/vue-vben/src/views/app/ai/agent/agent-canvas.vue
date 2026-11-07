<script lang="ts" setup>
import { computed, markRaw, reactive, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';
// vue-flow 样式必须引入：handle/controls/minimap 全靠它——漏引时 handle 塌成 0 高、连线命中永远失败
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import { MiniMap } from '@vue-flow/minimap';
import { VueFlow, useVueFlow } from '@vue-flow/core';

import {
  fetchListAiKnowledgeBases,
  fetchListAiProviders,
  PaginationQuery,
  runAiAgentApp,
  updateAiAgentApp,
} from '#/api';

import AgentFlowNode from './agent-flow-node.vue';

/**
 * Agent 画布编辑器（vue-flow 版，与 react 基准 agent-canvas.tsx 行为同构）：
 * - 图 DSL 与后端 docs/ai_agent_orchestration.md §2 逐字对齐：node{id,type,position,data} +
 *   edge{id,source,target,branch?}；节点 id 必须是合法标识符（条件表达式的变量名就是它）。
 * - 出度规则在连线时就地拦（condition 恰两条带 true/false 分支、其余至多一条、end 无出边），
 *   全图校验（唯一 start/可达/无环）交给保存时后端强校验——一处真理源。
 * - 试运行面板就地展示 output + trace 时间线 + token 用量。
 */

type AgentNodeType = 'start' | 'knowledge' | 'llm' | 'http' | 'condition' | 'end';

const NODE_TYPE_ORDER: AgentNodeType[] = [
  'start',
  'knowledge',
  'llm',
  'http',
  'condition',
  'end',
];

// markRaw 防 nodeTypes 进响应式 store 后对组件做深度响应式代理（Vue warn 且拖性能）；
// NodeComponent 类型要求完整 NodeProps 形状，自定义节点只声明用到的字段，再断言收编
const nodeTypes = { agent: markRaw(AgentFlowNode) as any };

const emit = defineEmits<{ success: [] }>();

const [Drawer, drawerApi] = useVbenDrawer({
  class: 'w-full',
  footer: false,
  modal: false,
  contentClass: 'p-0 overflow-hidden',
  onOpenChange(isOpen) {
    if (isOpen) {
      const data = drawerApi.getData<{ row: any }>() ?? { row: {} };
      app.value = data.row ?? {};
      const flow = graphToFlow(app.value.graph);
      vueFlow.setNodes(flow.nodes);
      vueFlow.setEdges(flow.edges);
      selectedId.value = undefined;
      configOpen.value = false;
      testOpen.value = false;
      runResult.value = undefined;
      // 抽屉开启动画期间挂载 VueFlow 会触发 fitView 前视口未初始化的告警，等动画结束再挂；
      // fitViewOnInit 在视口尺寸未测完时会静默不跑，挂载后显式补一次
      window.setTimeout(() => {
        flowReady.value = true;
        window.setTimeout(() => {
          vueFlow.fitView({ padding: 0.1, maxZoom: 1.5 });
        }, 150);
      }, 400);
      // 知识库/模型提供商下拉数据（保存期后端强校验，这里只为选择项）
      fetchListAiKnowledgeBases(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }))
        .then((resp: any) => {
          knowledgeBases.value = resp.items ?? [];
        })
        .catch((error: any) => {
          console.error('[ai-agent-canvas] list knowledge bases failed', error);
        });
      fetchListAiProviders(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }))
        .then((resp: any) => {
          providers.value = resp.items ?? [];
        })
        .catch((error: any) => {
          console.error('[ai-agent-canvas] list providers failed', error);
        });
    } else {
      flowReady.value = false;
    }
  },
});

const vueFlow = useVueFlow();
const { nodes, edges } = vueFlow;

const app = ref<any>({});
const flowReady = ref(false);
const selectedId = ref<string>();
const saving = ref(false);

const configOpen = ref(false);
const testOpen = ref(false);
const running = ref(false);
const testForm = reactive<Record<string, string>>({});
const runResult = ref<any>();

const knowledgeBases = ref<any[]>([]);
const providers = ref<any[]>([]);

const canvasTitle = computed(() =>
  `${$t('page.aiAgent.canvas.title')} · ${app.value?.name || ''}`,
);

const selectedNode = computed(() =>
  selectedId.value ? nodes.value.find((n) => n.id === selectedId.value) : undefined,
);

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD'].map((m) => ({ value: m, label: m }));

/** http 节点 headers 行（模板深链收口到 computed，v-model 绑迭代变量） */
const httpHeaderRows = computed<any[]>(() => ((selectedNode.value?.data.config as any)?.headers ?? []));

function addHttpHeader() {
  const node = selectedNode.value;
  if (!node) return;
  const config = node.data.config as Record<string, any>;
  config.headers = [...(config.headers || []), { key: '', value: '' }];
}

function removeHttpHeader(index: number | string) {
  selectedNode.value?.data.config?.headers?.splice(Number(index), 1);
}

const configTitle = computed(() =>
  `${$t('page.aiAgent.canvas.config')} · ${selectedId.value ?? ''}`,
);

const startNode = computed(() =>
  nodes.value.find((n: any) => n.data?.nodeType === 'start'),
);

const inputVars = computed<any[]>(() => startNode.value?.data?.config?.inputVars ?? []);

const traceSteps = computed<any[]>(() => {
  if (!runResult.value?.trace) return [];
  try {
    const parsed = JSON.parse(runResult.value.trace);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
});

/** 图 DSL ↔ vue-flow 状态的互转（唯一两处，别处不得各写一份）。 */
function graphToFlow(graphJson?: string) {
  if (!graphJson) return { nodes: [], edges: [] };
  try {
    const g = JSON.parse(graphJson);
    return {
      nodes: (g.nodes ?? []).map((n: any) => ({
        id: n.id,
        type: 'agent',
        position: n.position ?? { x: 0, y: 0 },
        data: { nodeType: n.type, label: `${n.id}`, config: n.data ?? {} },
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

function flowToGraph(): string {
  return JSON.stringify({
    nodes: nodes.value.map((n: any) => ({
      id: n.id,
      type: n.data.nodeType,
      position: { x: n.position.x, y: n.position.y },
      data: n.data.config,
    })),
    edges: edges.value.map((e: any, i: number) => ({
      id: e.id ?? `edge_${i}`,
      source: e.source,
      target: e.target,
      ...(e.label ? { branch: String(e.label) } : {}),
    })),
  });
}

function defaultConfig(nodeType: AgentNodeType): Record<string, any> {
  switch (nodeType) {
    case 'start': {
      return { inputVars: [{ name: 'question', label: 'question', required: true }] };
    }
    case 'knowledge': {
      return { topK: 4 };
    }
    case 'llm': {
      return { temperature: 0.3 };
    }
    case 'http': {
      return { method: 'GET', timeoutSeconds: 15 };
    }
    case 'condition': {
      return { expression: '' };
    }
    case 'end': {
      return { output: '' };
    }
    default: {
      return {};
    }
  }
}

function nextNodeId(nodeType: AgentNodeType): string {
  const prefix = `${nodeType}_`;
  let seq = 1;
  while (nodes.value.some((n) => n.id === `${prefix}${seq}`)) seq++;
  return `${prefix}${seq}`;
}

function addNode(nodeType: AgentNodeType) {
  const id = nextNodeId(nodeType);
  const maxY = nodes.value.reduce((m, n) => Math.max(m, n.position.y), 0);
  vueFlow.addNodes({
    id,
    type: 'agent',
    position: { x: 120 + (nodes.value.length % 3) * 60, y: maxY + 90 },
    data: { nodeType, label: id, config: defaultConfig(nodeType) },
  });
  selectedId.value = id;
  configOpen.value = true;
}

function onConnect(connection: { source?: string | null; target?: string | null }) {
  const { source, target } = connection;
  if (!source || !target || source === target) return;

  const sourceNode = nodes.value.find((n) => n.id === source);
  if (!sourceNode) return;
  const nodeType = (sourceNode.data as any).nodeType as AgentNodeType;
  const existing = edges.value.filter((e) => e.source === source);

  if (nodeType === 'end') {
    message.error($t('page.aiAgent.canvas.error.endNoOut'));
    return;
  }
  if (nodeType === 'condition') {
    if (existing.length >= 2) {
      message.error($t('page.aiAgent.canvas.error.conditionTwoOut'));
      return;
    }
    const branch = existing.length === 0 ? 'true' : 'false';
    vueFlow.addEdges({
      id: `edge_${source}_${target}_${branch}_${Date.now()}`,
      source,
      target,
      label: branch,
      animated: branch === 'true',
    });
    return;
  }
  if (existing.length >= 1) {
    message.error($t('page.aiAgent.canvas.error.singleOut'));
    return;
  }
  vueFlow.addEdges({ id: `edge_${source}_${target}_${Date.now()}`, source, target });
}

function onNodeClick({ node }: { node: any }) {
  selectedId.value = node.id;
  configOpen.value = true;
}

function deleteSelectedNode() {
  if (!selectedId.value) return;
  vueFlow.removeNodes([selectedId.value], true);
  selectedId.value = undefined;
  configOpen.value = false;
}

function addInputVar() {
  const config = selectedNode.value?.data?.config;
  if (!config) return;
  if (!Array.isArray(config.inputVars)) config.inputVars = [];
  config.inputVars.push({ name: '', label: '', required: false });
}

function removeInputVar(index: number) {
  const config = selectedNode.value?.data?.config;
  if (!config?.inputVars) return;
  config.inputVars.splice(index, 1);
}

async function handleSave() {
  if (!app.value?.id) return;
  saving.value = true;
  try {
    await updateAiAgentApp(app.value.id, { graph: flowToGraph() }, 'graph');
    message.success($t('page.aiAgent.canvas.saveSuccess'));
    emit('success');
  } catch (error: any) {
    // 后端强校验的报错原文在这里最有用（唯一 start/可达性……），直接透出
    console.error('[ai-agent-canvas] save failed', error);
    message.error(error?.message || $t('page.aiAgent.canvas.saveFailed'));
  } finally {
    saving.value = false;
  }
}

function openTest() {
  Object.keys(testForm).forEach((k) => delete testForm[k]);
  runResult.value = undefined;
  testOpen.value = true;
}

async function handleRun() {
  if (!app.value?.id) return;
  for (const v of inputVars.value) {
    if (v.required && !testForm[v.name!]) {
      message.error(`${v.label ?? v.name} ${$t('page.aiAgent.runs.requiredInput')}`);
      return;
    }
  }
  running.value = true;
  try {
    const run = await runAiAgentApp(app.value.id, { ...testForm });
    runResult.value = run;
  } catch (error: any) {
    // 图校验/配额类错误原文直接给到面板
    console.error('[ai-agent-canvas] run failed', error);
    runResult.value = { status: 'FAILED', error: error?.message || String(error) };
  } finally {
    running.value = false;
  }
}
</script>

<template>
  <Drawer :title="canvasTitle">
    <div class="canvas-body flex h-full flex-col overflow-hidden">
      <div class="flex flex-wrap items-center gap-2 border-b border-border px-4 py-2">
        <a-button
          v-for="nt in NODE_TYPE_ORDER"
          :key="nt"
          size="small"
          @click="addNode(nt)"
        >
          {{ $t(`page.aiAgent.canvas.nodeType.${nt}`) }}
        </a-button>
        <a-button type="primary" size="small" :loading="saving" @click="handleSave">
          {{ $t('page.aiAgent.canvas.save') }}
        </a-button>
        <a-button size="small" @click="openTest">
          {{ $t('page.aiAgent.canvas.testRun') }}
        </a-button>
      </div>

      <!-- 等抽屉动画结束（尺寸稳定）再挂 VueFlow：动画期挂载会触发 fitView 前视口未初始化的告警 -->
      <div v-if="flowReady" class="min-h-0 flex-1">
        <VueFlow
          :node-types="nodeTypes"
          fit-view-on-init
          @connect="onConnect"
          @node-click="onNodeClick"
          @pane-click="selectedId = undefined"
        >
          <Background />
          <Controls />
          <MiniMap pannable />
        </VueFlow>
      </div>
    </div>

    <!-- 节点配置侧板：不带遮罩，配置时画布仍可操作（删除按钮放表单底部——antd-vue 的
         Popconfirm 触发器同样要求单元素子节点，放头部易出 Weird 行为） -->
    <a-drawer
      v-model:open="configOpen"
      width="420px"
      :mask="false"
      destroy-on-close
      :title="configTitle"
    >
      <div v-if="selectedNode" :key="selectedNode.id">
        <!-- start：输入变量列表 -->
        <template v-if="selectedNode.data.nodeType === 'start'">
          <div
            v-for="(v, i) in selectedNode.data.config.inputVars"
            :key="i"
            class="mb-2 flex items-center gap-2"
          >
            <a-input
              v-model:value="v.name"
              size="small"
              style="width: 140px"
              :placeholder="$t('page.aiAgent.canvas.varName')"
            />
            <a-input
              v-model:value="v.label"
              size="small"
              style="width: 120px"
              :placeholder="$t('page.aiAgent.canvas.varLabel')"
            />
            <a-switch v-model:checked="v.required" size="small" />
            <a-button size="small" type="text" danger @click="removeInputVar(Number(i))">
              {{ $t('page.aiAgent.canvas.remove') }}
            </a-button>
          </div>
          <a-button size="small" dashed block @click="addInputVar">
            {{ $t('page.aiAgent.canvas.addVar') }}
          </a-button>
        </template>

        <!-- knowledge -->
        <template v-if="selectedNode.data.nodeType === 'knowledge'">
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.knowledgeBase') }}</div>
            <a-select
              v-model:value="selectedNode.data.config.baseId"
              allow-clear
              show-search
              option-filter-prop="label"
              :options="knowledgeBases.map((k) => ({ value: k.id, label: k.name }))"
              class="w-full"
            />
          </div>
          <div class="pb-4">
            <div class="pb-1">topK</div>
            <a-input-number
              v-model:value="selectedNode.data.config.topK"
              :min="1"
              :max="20"
              :precision="0"
              class="w-full"
            />
          </div>
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.knowledgeQuery') }}</div>
            <a-textarea
              v-model:value="selectedNode.data.config.query"
              :rows="2"
              placeholder="{{start_1.question}}"
            />
          </div>
        </template>

        <!-- llm -->
        <template v-if="selectedNode.data.nodeType === 'llm'">
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.provider') }}</div>
            <a-select
              v-model:value="selectedNode.data.config.providerId"
              allow-clear
              show-search
              option-filter-prop="label"
              :options="providers.map((p) => ({ value: p.id, label: p.name }))"
              class="w-full"
            />
          </div>
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.model') }}</div>
            <a-input v-model:value="selectedNode.data.config.model" placeholder="deepseek-chat" />
          </div>
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.systemPrompt') }}</div>
            <a-textarea v-model:value="selectedNode.data.config.systemPrompt" :rows="3" />
          </div>
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.userPrompt') }}</div>
            <a-textarea
              v-model:value="selectedNode.data.config.userPrompt"
              :rows="4"
              placeholder="资料：{{kb_1.context}}
问题：{{start_1.question}}"
            />
          </div>
          <div class="pb-4">
            <div class="pb-1">temperature</div>
            <a-input-number
              v-model:value="selectedNode.data.config.temperature"
              :min="0"
              :max="2"
              :step="0.1"
              class="w-full"
            />
          </div>
        </template>

        <!-- condition -->
        <template v-if="selectedNode.data.nodeType === 'condition'">
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.expression') }}</div>
            <a-textarea
              v-model:value="selectedNode.data.config.expression"
              :rows="3"
              style="font-family: monospace"
              placeholder='llm_1.output contains "退款"'
            />
          </div>
        </template>

        <!-- http -->
        <template v-if="selectedNode.data.nodeType === 'http'">
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.httpUrl') }}</div>
            <a-textarea
              v-model:value="selectedNode.data.config.url"
              :rows="2"
              style="font-family: monospace"
              placeholder="https://api.example.com/orders/{{start_1.orderId}}"
            />
          </div>
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.httpMethod') }}</div>
            <a-select v-model:value="selectedNode.data.config.method" :options="HTTP_METHODS" />
          </div>
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.httpHeaders') }}</div>
            <div
              v-for="(h, i) in httpHeaderRows"
              :key="i"
              class="mb-1.5 flex gap-1.5"
            >
              <a-input v-model:value="h.key" placeholder="Content-Type" class="w-[150px]" />
              <a-input v-model:value="h.value" placeholder="application/json" class="flex-1" />
              <a-button danger type="text" size="small" @click="removeHttpHeader(i)">
                {{ $t('page.aiAgent.canvas.remove') }}
              </a-button>
            </div>
            <a-button dashed size="small" class="w-full" @click="addHttpHeader">
              {{ $t('page.aiAgent.canvas.httpAddHeader') }}
            </a-button>
          </div>
          <div v-if="selectedNode.data.config.method !== 'GET' && selectedNode.data.config.method !== 'HEAD'" class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.httpBody') }}</div>
            <a-textarea
              v-model:value="selectedNode.data.config.body"
              :rows="4"
              style="font-family: monospace"
              placeholder='{"orderId": "{{start_1.orderId}}"}'
            />
          </div>
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.httpTimeout') }}</div>
            <a-input-number v-model:value="selectedNode.data.config.timeoutSeconds" :min="1" :max="60" class="w-full" />
          </div>
          <div class="pt-1 text-xs text-muted-foreground">{{ $t('page.aiAgent.canvas.httpSsrHint') }}</div>
        </template>

        <!-- end -->
        <!-- v-if 显式非空：render 函数顺序收窄会因前面兄弟块内的表达式失效，
             补一层让本块重新成为收窄起点（语义等价，外层 v-if 本就保证非空） -->
        <template v-if="selectedNode && selectedNode.data.nodeType === 'end'">
          <div class="pb-4">
            <div class="pb-1">{{ $t('page.aiAgent.canvas.endOutput') }}</div>
            <a-textarea
              v-model:value="selectedNode.data.config.output"
              :rows="3"
              placeholder="{{llm_1.output}}"
            />
          </div>
        </template>

        <div class="pt-3 text-xs text-muted-foreground">
          {{ $t('page.aiAgent.canvas.configAutoSaved') }}
        </div>

        <div
          v-if="selectedNode && selectedNode.data.nodeType !== 'start'"
          class="mt-4 border-t border-border pt-3"
        >
          <a-popconfirm
            :title="$t('page.aiAgent.canvas.deleteNodeConfirm')"
            @confirm="deleteSelectedNode"
          >
            <a-button size="small" danger>
              {{ $t('page.aiAgent.canvas.deleteNode') }}
            </a-button>
          </a-popconfirm>
        </div>
      </div>
    </a-drawer>

    <!-- 试运行面板 -->
    <a-drawer
      v-model:open="testOpen"
      width="480px"
      :mask="false"
      destroy-on-close
      :title="$t('page.aiAgent.runs.testTitle', { name: app?.name || '' })"
      @close="runResult = undefined"
    >
      <div class="pb-2">
        <div v-for="(v, i) in inputVars" :key="`${v.name}_${i}`" class="pb-3">
          <div class="pb-1">
            {{ v.label || v.name }}
            <span v-if="v.required" class="text-destructive">*</span>
          </div>
          <a-textarea v-model:value="testForm[v.name!]" :rows="2" :auto-size="{ minRows: 2, maxRows: 6 }" />
        </div>
        <div v-if="inputVars.length === 0" class="pb-2 text-xs text-muted-foreground">
          {{ $t('page.aiAgent.runs.noInputs') }}
        </div>
        <a-button type="primary" block :loading="running" @click="handleRun">
          {{ $t('page.aiAgent.runs.run') }}
        </a-button>
      </div>

      <div v-if="runResult" class="pt-4">
        <div class="flex flex-wrap items-center gap-2 pb-2">
          <a-tag v-if="runResult.status === 'SUCCEEDED'" color="success">
            {{ $t('page.aiAgent.runs.succeeded') }}
          </a-tag>
          <a-tag v-else color="error">{{ $t('page.aiAgent.runs.failed') }}</a-tag>
          <a-tag v-if="runResult.totalTokens != null">
            {{ $t('page.aiAgent.runs.tokens', { total: runResult.totalTokens }) }}
          </a-tag>
          <a-tag v-if="runResult.durationMs != null">{{ runResult.durationMs }}ms</a-tag>
        </div>

        <div
          v-if="runResult.error"
          class="mb-2 whitespace-pre-wrap text-xs text-destructive"
        >
          {{ runResult.error }}
        </div>

        <div v-if="runResult.output != null" class="pb-2">
          <div class="py-1 font-semibold">{{ $t('page.aiAgent.runs.output') }}</div>
          <pre class="m-0 whitespace-pre-wrap break-all text-xs">{{ runResult.output }}</pre>
        </div>

        <template v-if="traceSteps.length > 0">
          <div class="py-1 font-semibold">{{ $t('page.aiAgent.runs.trace') }}</div>
          <a-timeline class="pt-1">
            <a-timeline-item
              v-for="(step, i) in traceSteps"
              :key="i"
              :color="step.status === 'ok' ? 'green' : 'red'"
            >
              <div class="flex items-center gap-1.5 text-xs">
                <code>{{ step.nodeId }}</code>
                <a-tag>{{ step.type }}</a-tag>
                <span>{{ step.ms }}ms</span>
              </div>
              <div
                v-if="step.summary"
                class="mt-0.5 whitespace-pre-wrap text-xs text-muted-foreground"
              >
                {{ String(step.summary).slice(0, 200) }}
              </div>
              <div v-if="step.error" class="mt-0.5 text-xs text-destructive">{{ step.error }}</div>
            </a-timeline-item>
          </a-timeline>
        </template>
      </div>
    </a-drawer>
  </Drawer>
</template>

<style lang="scss">
/* vue-flow 内置主题写死浅色（边标签/控制条/小地图），暗色模式按 vben token 覆盖 */
html.dark {
  .vue-flow__edge-textbg {
    fill: hsl(var(--card));
  }

  .vue-flow__edge-text {
    fill: hsl(var(--foreground));
  }

  .vue-flow__controls {
    box-shadow: 0 0 2px 1px rgb(0 0 0 / 40%);
  }

  .vue-flow__controls-button {
    background: hsl(var(--card));
    border-bottom-color: hsl(var(--border));
    color: hsl(var(--foreground));

    &:hover {
      background: hsl(var(--accent));
    }
  }

  .vue-flow__minimap {
    background-color: hsl(var(--card));
  }

  .vue-flow__minimap .vue-flow__minimap-node {
    fill: hsl(var(--secondary));
  }

  .vue-flow__minimap .vue-flow__minimap-mask {
    fill: rgb(255 255 255 / 8%);
  }
}
</style>
