<template>
  <ElDrawer
    v-model="visible"
    size="100%"
    :modal="false"
    :destroy-on-close="true"
    :title="canvasTitle"
    class="ai-agent-canvas-drawer"
    :append-to-body="true"
  >
    <div class="canvas-body">
      <div class="canvas-toolbar">
        <ElButton
          v-for="nt in NODE_TYPE_ORDER"
          :key="nt"
          size="small"
          :icon="Plus"
          @click="addNode(nt)"
        >
          {{ t(`pages.ai_agent.canvas.nodeType.${nt}`) }}
        </ElButton>
        <ElButton type="primary" size="small" :icon="DocumentChecked" :loading="saving" @click="handleSave">
          {{ t("pages.ai_agent.canvas.save") }}
        </ElButton>
        <ElButton size="small" :icon="VideoPlay" @click="openTest">
          {{ t("pages.ai_agent.canvas.testRun") }}
        </ElButton>
      </div>

      <!-- 等抽屉动画结束（尺寸稳定）再挂 VueFlow：动画期挂载会触发 fitView 前视口未初始化的告警 -->
      <div v-if="flowReady" class="canvas-flow-wrap">
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

    <!-- 节点配置侧板：不带遮罩，配置时画布仍可操作（删除按钮放表单底部——EP 的
         Popconfirm 触发器放进 #header slot 会被 ElOnlyChild 判成注释节点整块吞掉） -->
    <ElDrawer
      v-model="configOpen"
      size="420px"
      :modal="false"
      :destroy-on-close="true"
      :title="configTitle"
      :append-to-body="true"
    >
      <div v-if="selectedNode" :key="selectedNode.id" class="config-form">
        <!-- start：输入变量列表 -->
        <template v-if="selectedNode.data.nodeType === 'start'">
          <div
            v-for="(v, i) in selectedNode.data.config.inputVars"
            :key="i"
            class="input-var-row"
          >
            <ElInput
              v-model="v.name"
              size="small"
              style="width: 140px"
              :placeholder="t('pages.ai_agent.canvas.varName')"
            />
            <ElInput
              v-model="v.label"
              size="small"
              style="width: 120px"
              :placeholder="t('pages.ai_agent.canvas.varLabel')"
            />
            <ElCheckbox v-model="v.required" size="small" />
            <ElButton size="small" text type="danger" @click="removeInputVar(Number(i))">
              {{ t("pages.ai_agent.canvas.remove") }}
            </ElButton>
          </div>
          <ElButton size="small" dashed style="width: 100%" :icon="Plus" @click="addInputVar">
            {{ t("pages.ai_agent.canvas.addVar") }}
          </ElButton>
        </template>

        <!-- knowledge -->
        <template v-if="selectedNode.data.nodeType === 'knowledge'">
          <ElFormItem :label="t('pages.ai_agent.canvas.knowledgeBase')" required>
            <ElSelect
              v-model="selectedNode.data.config.baseId"
              clearable
              filterable
              style="width: 100%"
            >
              <ElOption
                v-for="k in knowledgeBases"
                :key="k.id"
                :value="k.id"
                :label="k.name"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="topK">
            <ElInputNumber
              v-model="selectedNode.data.config.topK"
              :min="1"
              :max="20"
              :precision="0"
              style="width: 100%"
            />
          </ElFormItem>
          <ElFormItem :label="t('pages.ai_agent.canvas.knowledgeQuery')">
            <ElInput
              v-model="selectedNode.data.config.query"
              type="textarea"
              :rows="2"
              placeholder="{{start_1.question}}"
            />
          </ElFormItem>
        </template>

        <!-- llm -->
        <template v-if="selectedNode.data.nodeType === 'llm'">
          <ElFormItem :label="t('pages.ai_agent.canvas.provider')">
            <ElSelect
              v-model="selectedNode.data.config.providerId"
              clearable
              filterable
              style="width: 100%"
            >
              <ElOption
                v-for="p in providers"
                :key="p.id"
                :value="p.id"
                :label="p.name"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem :label="t('pages.ai_agent.canvas.model')">
            <ElInput v-model="selectedNode.data.config.model" placeholder="deepseek-chat" />
          </ElFormItem>
          <ElFormItem :label="t('pages.ai_agent.canvas.systemPrompt')">
            <ElInput v-model="selectedNode.data.config.systemPrompt" type="textarea" :rows="3" />
          </ElFormItem>
          <ElFormItem :label="t('pages.ai_agent.canvas.userPrompt')" required>
            <ElInput
              v-model="selectedNode.data.config.userPrompt"
              type="textarea"
              :rows="4"
              placeholder="资料：{{kb_1.context}}
问题：{{start_1.question}}"
            />
          </ElFormItem>
          <ElFormItem label="temperature">
            <ElInputNumber
              v-model="selectedNode.data.config.temperature"
              :min="0"
              :max="2"
              :step="0.1"
              style="width: 100%"
            />
          </ElFormItem>
        </template>

        <!-- condition -->
        <template v-if="selectedNode.data.nodeType === 'condition'">
          <ElFormItem :label="t('pages.ai_agent.canvas.expression')" required>
            <ElInput
              v-model="selectedNode.data.config.expression"
              type="textarea"
              :rows="3"
              style="font-family: monospace"
              placeholder='llm_1.output contains "退款"'
            />
          </ElFormItem>
        </template>

        <!-- http -->
        <template v-if="selectedNode.data.nodeType === 'http'">
          <ElFormItem :label="t('pages.ai_agent.canvas.httpUrl')" required>
            <ElInput
              v-model="selectedNode.data.config.url"
              type="textarea"
              :rows="2"
              style="font-family: monospace"
              placeholder="https://api.example.com/orders/{{start_1.orderId}}"
            />
          </ElFormItem>
          <ElFormItem :label="t('pages.ai_agent.canvas.httpMethod')">
            <ElSelect v-model="selectedNode.data.config.method">
              <ElOption v-for="m in ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD']" :key="m" :label="m" :value="m" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem :label="t('pages.ai_agent.canvas.httpHeaders')">
            <div style="width: 100%">
              <div
                v-for="(h, i) in httpHeaderRows"
                :key="i"
                style="display: flex; gap: 6px; margin-bottom: 6px"
              >
                <ElInput v-model="h.key" placeholder="Content-Type" style="width: 150px" />
                <ElInput v-model="h.value" placeholder="application/json" style="flex: 1" />
                <ElButton type="danger" text @click="removeHttpHeader(i)">
                  {{ t("pages.ai_agent.canvas.remove") }}
                </ElButton>
              </div>
              <ElButton dashed style="width: 100%" @click="addHttpHeader()">
                {{ t("pages.ai_agent.canvas.httpAddHeader") }}
              </ElButton>
            </div>
          </ElFormItem>
          <ElFormItem v-if="selectedNode.data.config.method !== 'GET' && selectedNode.data.config.method !== 'HEAD'" :label="t('pages.ai_agent.canvas.httpBody')">
            <ElInput
              v-model="selectedNode.data.config.body"
              type="textarea"
              :rows="4"
              style="font-family: monospace"
              placeholder='{"orderId": "{{start_1.orderId}}"}'
            />
          </ElFormItem>
          <ElFormItem :label="t('pages.ai_agent.canvas.httpTimeout')">
            <ElInputNumber v-model="selectedNode.data.config.timeoutSeconds" :min="1" :max="60" style="width: 100%" controls-position="right" />
          </ElFormItem>
          <div class="config-tip">{{ t("pages.ai_agent.canvas.httpSsrHint") }}</div>
        </template>

        <!-- end -->
        <!-- v-if 显式非空：render 函数顺序收窄会因前面兄弟块内的表达式失效，
             补一层让本块重新成为收窄起点（语义等价，外层 div v-if 本就保证非空） -->
        <template v-if="selectedNode && selectedNode.data.nodeType === 'end'">
          <ElFormItem :label="t('pages.ai_agent.canvas.endOutput')">
            <ElInput
              v-model="selectedNode.data.config.output"
              type="textarea"
              :rows="3"
              placeholder="{{llm_1.output}}"
            />
          </ElFormItem>
        </template>

        <div class="config-tip">{{ t("pages.ai_agent.canvas.configAutoSaved") }}</div>

        <div
          v-if="selectedNode && selectedNode.data.nodeType !== 'start'"
          class="config-delete"
        >
          <ElPopconfirm
            :title="t('pages.ai_agent.canvas.deleteNodeConfirm')"
            @confirm="deleteSelectedNode"
          >
            <!-- EP 2.14 起 Popconfirm 触发器是具名 #reference 插槽，默认插槽被无视（渲染成注释） -->
            <template #reference>
              <ElButton size="small" type="danger" plain>
                {{ t("pages.ai_agent.canvas.deleteNode") }}
              </ElButton>
            </template>
          </ElPopconfirm>
        </div>
      </div>
    </ElDrawer>

    <!-- 试运行面板 -->
    <ElDrawer
      v-model="testOpen"
      size="480px"
      :modal="false"
      :destroy-on-close="true"
      :title="t('pages.ai_agent.runs.testTitle', { name: app?.name || '' })"
      :append-to-body="true"
      @close="runResult = undefined"
    >
      <ElForm label-position="top">
        <ElFormItem
          v-for="(v, i) in inputVars"
          :key="`${v.name}_${i}`"
          :label="v.label || v.name"
          :required="v.required"
        >
          <ElInput v-model="testForm[v.name!]" type="textarea" :rows="2" autosize />
        </ElFormItem>
        <div v-if="inputVars.length === 0" class="config-tip">
          {{ t("pages.ai_agent.runs.noInputs") }}
        </div>
        <ElButton type="primary" style="width: 100%" :loading="running" @click="handleRun">
          {{ t("pages.ai_agent.runs.run") }}
        </ElButton>
      </ElForm>

      <div v-if="runResult" class="run-result">
        <div class="run-meta">
          <ElTag v-if="runResult.status === 'SUCCEEDED'" type="success" size="small">
            {{ t("pages.ai_agent.runs.succeeded") }}
          </ElTag>
          <ElTag v-else type="danger" size="small">
            {{ t("pages.ai_agent.runs.failed") }}
          </ElTag>
          <ElTag v-if="runResult.totalTokens != null" size="small">
            {{ t("pages.ai_agent.runs.tokens", { total: runResult.totalTokens }) }}
          </ElTag>
          <ElTag v-if="runResult.durationMs != null" size="small">
            {{ runResult.durationMs }}ms
          </ElTag>
        </div>

        <div v-if="runResult.error" class="run-error">{{ runResult.error }}</div>

        <div v-if="runResult.output != null" class="run-output">
          <div class="run-section-title">{{ t("pages.ai_agent.runs.output") }}</div>
          <pre>{{ runResult.output }}</pre>
        </div>

        <template v-if="traceSteps.length > 0">
          <div class="run-section-title">{{ t("pages.ai_agent.runs.trace") }}</div>
          <ElTimeline>
            <ElTimelineItem
              v-for="(step, i) in traceSteps"
              :key="i"
              :type="step.status === 'ok' ? 'success' : 'danger'"
            >
              <div class="trace-step">
                <code>{{ step.nodeId }}</code>
                <ElTag size="small">{{ step.type }}</ElTag>
                <span>{{ step.ms }}ms</span>
              </div>
              <div v-if="step.summary" class="trace-summary">{{ String(step.summary).slice(0, 200) }}</div>
              <div v-if="step.error" class="run-error">{{ step.error }}</div>
            </ElTimelineItem>
          </ElTimeline>
        </template>
      </div>
    </ElDrawer>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, markRaw, reactive, ref, watch } from "vue";
import { ElButton, ElCheckbox, ElDrawer, ElForm, ElFormItem, ElInput, ElInputNumber, ElMessage, ElOption, ElPopconfirm, ElSelect, ElTag, ElTimeline, ElTimelineItem } from "element-plus";
import { DocumentChecked, Plus, VideoPlay } from "@element-plus/icons-vue";
// vue-flow 样式必须引入：handle/controls/minimap 全靠它——漏引时 handle 塌成 0 高、连线命中永远失败
import "@vue-flow/core/dist/style.css";
import "@vue-flow/core/dist/theme-default.css";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import { MiniMap } from "@vue-flow/minimap";
import { VueFlow, useVueFlow } from "@vue-flow/core";

import { useI18n } from "@/core/i18n";
import { PaginationQuery } from "@/core/transport/rest";
import {
  fetchListAiKnowledgeBases,
  fetchListAiProviders,
  runAiAgentApp,
  updateAiAgentApp,
} from "@/api/composables";

import AiAgentFlowNode from "./ai-agent-flow-node.vue";

/**
 * Agent 画布编辑器（vue-flow 版，与 react 基准 agent-canvas.tsx 行为同构）：
 * - 图 DSL 与后端 docs/ai_agent_orchestration.md §2 逐字对齐：node{id,type,position,data} +
 *   edge{id,source,target,branch?}；节点 id 必须是合法标识符（条件表达式的变量名就是它）。
 * - 出度规则在连线时就地拦（condition 恰两条带 true/false 分支、其余至多一条、end 无出边），
 *   全图校验（唯一 start/可达/无环）交给保存时后端强校验——一处真理源。
 * - 试运行面板就地展示 output + trace 时间线 + token 用量。
 */

type AgentNodeType = "start" | "knowledge" | "llm" | "http" | "condition" | "end";

const NODE_TYPE_ORDER: AgentNodeType[] = ["start", "knowledge", "llm", "http", "condition", "end"];

// markRaw 防 nodeTypes 进响应式 store 后对组件做深度响应式代理（Vue warn 且拖性能）；
// NodeComponent 类型要求完整 NodeProps 形状，自定义节点只声明用到的字段，再断言收编
const nodeTypes = { agent: markRaw(AiAgentFlowNode) as any };

const { t } = useI18n();

const vueFlow = useVueFlow();
const { nodes, edges } = vueFlow;

const visible = ref(false);
const flowReady = ref(false);
const app = ref<any>();

// 抽屉开启动画期间挂载 VueFlow 会触发 fitView 前视口未初始化的告警；
// 等 400ms（动画结束、尺寸稳定）再挂载。EP 的 @opened 事件依赖 transitionend
// 信号，部分环境不触发，这里用定时器兜底。
watch(visible, (v) => {
  if (v) {
    window.setTimeout(() => {
      flowReady.value = true;
      // fitViewOnInit 在视口尺寸未测完时会静默不跑（宽屏真机偶发），挂载后显式补一次
      window.setTimeout(() => {
        vueFlow.fitView({ padding: 0.1, maxZoom: 1.5 });
      }, 150);
    }, 400);
  } else {
    flowReady.value = false;
  }
});

const selectedId = ref<string>();
const saving = ref(false);

const configOpen = ref(false);
const testOpen = ref(false);
const running = ref(false);
const testForm = reactive<Record<string, string>>({});
const runResult = ref<any>();

const knowledgeBases = ref<any[]>([]);
const providers = ref<any[]>([]);

const canvasTitle = computed(() => `${t("pages.ai_agent.canvas.title")} · ${app.value?.name || ""}`);

const selectedNode = computed(() => (selectedId.value ? nodes.value.find((n) => n.id === selectedId.value) : undefined));

/** http 节点 headers 行（模板深链收口到 computed，v-model 绑迭代变量） */
const httpHeaderRows = computed<any[]>(() => ((selectedNode.value?.data.config as any)?.headers ?? []));

const configTitle = computed(() => `${t("pages.ai_agent.canvas.config")} · ${selectedId.value ?? ""}`);

const startNode = computed(() => nodes.value.find((n: any) => n.data?.nodeType === "start"));

const inputVars = computed(() => startNode.value?.data?.config?.inputVars ?? []);

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
        type: "agent",
        position: n.position ?? { x: 0, y: 0 },
        data: { nodeType: n.type, label: `${n.id}`, config: n.data ?? {} },
      })),
      edges: (g.edges ?? []).map((e: any, i: number) => ({
        id: e.id ?? `edge_${i}`,
        source: e.source,
        target: e.target,
        label: e.branch,
        animated: e.branch === "true",
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
    case "start": {
      return { inputVars: [{ name: "question", label: "question", required: true }] };
    }
    case "knowledge": {
      return { topK: 4 };
    }
    case "llm": {
      return { temperature: 0.3 };
    }
    case "http": {
      return { method: "GET", timeoutSeconds: 15 };
    }
    case "condition": {
      return { expression: "" };
    }
    case "end": {
      return { output: "" };
    }
    default: {
      return {};
    }
  }
}

/** http 节点 headers 行操作（@click 内联表达式拿不到 v-if 的非空收窄，集中到方法里判空） */
function addHttpHeader() {
  const node = selectedNode.value;
  if (!node) return;
  const config = node.data.config as Record<string, any>;
  config.headers = [...(config.headers || []), { key: "", value: "" }];
}

function removeHttpHeader(index: number | string) {
  selectedNode.value?.data.config?.headers?.splice(Number(index), 1);
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
    type: "agent",
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

  if (nodeType === "end") {
    ElMessage.error(t("pages.ai_agent.canvas.error.endNoOut"));
    return;
  }
  if (nodeType === "condition") {
    if (existing.length >= 2) {
      ElMessage.error(t("pages.ai_agent.canvas.error.conditionTwoOut"));
      return;
    }
    const branch = existing.length === 0 ? "true" : "false";
    vueFlow.addEdges({
      id: `edge_${source}_${target}_${branch}_${Date.now()}`,
      source,
      target,
      label: branch,
      animated: branch === "true",
    });
    return;
  }
  if (existing.length >= 1) {
    ElMessage.error(t("pages.ai_agent.canvas.error.singleOut"));
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
  config.inputVars.push({ name: "", label: "", required: false });
}

function removeInputVar(index: number) {
  const config = selectedNode.value?.data?.config;
  if (!config?.inputVars) return;
  config.inputVars.splice(index, 1);
}

function open(row: any) {
  app.value = row;
  const flow = graphToFlow(row?.graph);
  vueFlow.setNodes(flow.nodes);
  vueFlow.setEdges(flow.edges);
  selectedId.value = undefined;
  configOpen.value = false;
  testOpen.value = false;
  runResult.value = undefined;
  visible.value = true;
  // 知识库/模型提供商下拉数据（保存期后端强校验，这里只为选择项）
  fetchListAiKnowledgeBases(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }))
    .then((resp: any) => {
      knowledgeBases.value = resp.items ?? [];
    })
    .catch((error: any) => {
      console.error("[ai-agent-canvas] list knowledge bases failed:", error);
    });
  fetchListAiProviders(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }))
    .then((resp: any) => {
      providers.value = resp.items ?? [];
    })
    .catch((error: any) => {
      console.error("[ai-agent-canvas] list providers failed:", error);
    });
}

async function handleSave() {
  if (!app.value?.id) return;
  saving.value = true;
  try {
    await updateAiAgentApp(app.value.id, { graph: flowToGraph() }, "graph");
    ElMessage.success(t("pages.ai_agent.canvas.saveSuccess"));
  } catch (error: any) {
    // 后端强校验的报错原文在这里最有用（唯一 start/可达性……），直接透出
    console.error("[ai-agent-canvas] save failed:", error);
    ElMessage.error(error?.message || t("pages.ai_agent.canvas.saveFailed"));
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
      ElMessage.error(`${v.label ?? v.name} ${t("pages.ai_agent.runs.requiredInput")}`);
      return;
    }
  }
  running.value = true;
  try {
    const run = await runAiAgentApp(app.value.id, { ...testForm });
    runResult.value = run;
  } catch (error: any) {
    // 图校验/配额类错误原文直接给到面板
    console.error("[ai-agent-canvas] run failed:", error);
    runResult.value = { status: "FAILED", error: error?.message || String(error) };
  } finally {
    running.value = false;
  }
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.canvas-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.canvas-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 8px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.canvas-flow-wrap {
  flex: 1;
  min-height: 0;
}

.config-delete {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.config-form {
  padding-right: 4px;
}

.input-var-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}

.config-tip {
  margin-top: 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.run-result {
  margin-top: 16px;
}

.run-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.run-error {
  margin-bottom: 8px;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  color: var(--el-color-danger);
}

.run-section-title {
  margin: 8px 0;
  font-weight: 600;
}

.run-output pre {
  margin: 0;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}

.trace-step {
  display: flex;
  gap: 6px;
  align-items: center;
  font-size: 12px;
}

.trace-summary {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  color: var(--el-text-color-secondary);
}
</style>

<style lang="scss">
/* 全屏画布抽屉：去掉默认 body 内边距，让 flow 撑满 */
.ai-agent-canvas-drawer {
  .el-drawer__body {
    padding: 0;
    overflow: hidden;
  }
}

/* vue-flow 内置主题写死浅色（边标签/控制条/小地图），暗色模式按 EP token 覆盖 */
html.dark {
  .vue-flow__edge-textbg {
    fill: var(--el-bg-color-overlay);
  }

  .vue-flow__edge-text {
    fill: var(--el-text-color-primary);
  }

  .vue-flow__controls {
    box-shadow: 0 0 2px 1px rgb(0 0 0 / 40%);
  }

  .vue-flow__controls-button {
    background: var(--el-bg-color-overlay);
    border-bottom-color: var(--el-border-color-darker);
    color: var(--el-text-color-primary);

    &:hover {
      background: var(--el-fill-color);
    }
  }

  .vue-flow__minimap {
    background-color: var(--el-bg-color-overlay);
  }

  .vue-flow__minimap .vue-flow__minimap-node {
    fill: var(--el-fill-color-darker);
  }

  .vue-flow__minimap .vue-flow__minimap-mask {
    fill: rgb(255 255 255 / 8%);
  }
}
</style>
