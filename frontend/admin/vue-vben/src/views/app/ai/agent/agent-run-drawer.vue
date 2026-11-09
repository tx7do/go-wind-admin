<script lang="ts" setup>
import { reactive, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';

import { globalSSEClient, SSE_EVENT } from '#/transport/sse';
import { fetchAiAgentApp, runAiAgentApp } from '#/api';

const emit = defineEmits<{ success: [] }>();

function onFrame(data: any) {
  if (!data || typeof data !== 'object' || data.appId !== app.value.id) return;
  if (data.phase === 'node_end') {
    liveSteps.value.push(`${data.nodeId} ok`);
  }
}

const [Drawer, drawerApi] = useVbenDrawer({
  class: 'w-[520px]',
  onConfirm: async () => {
    for (const v of inputVars.value) {
      if (v.required && !runForm[v.name!]) {
        message.error(`${v.label ?? v.name} ${$t('page.aiAgent.runs.requiredInput')}`);
        return;
      }
    }
    drawerApi.setState({ loading: true });
    try {
      await runAiAgentApp(app.value.id, { ...runForm });
      message.success($t('page.aiAgent.runs.testSent'));
      drawerApi.close();
      emit('success');
    } catch (error: any) {
      console.error('[ai-agent] run failed', error);
      message.error(error?.message || $t('page.aiAgent.runs.testFailed'));
    } finally {
      drawerApi.setState({ loading: false });
    }
  },
  onOpenChange(isOpen) {
    // 关闭时注销订阅（反复开关会累加回调），并清空上一次的实时帧
    if (!isOpen) {
      offFrame?.();
      offFrame = undefined;
      liveSteps.value = [];
      return;
    }
    const data = drawerApi.getData<{ row: any }>() ?? { row: {} };
    app.value = data.row ?? {};
    Object.keys(runForm).forEach((k) => delete runForm[k]);
    liveSteps.value = [];
    globalSSEClient.on(SSE_EVENT.AgentRun, onFrame);
    offFrame = () => globalSSEClient.off(SSE_EVENT.AgentRun, onFrame);
    // start 节点 inputVars 声明解析（graph JSON 里取）
    const parseInputVars = (graph?: string) => {
      try {
        const g = graph ? JSON.parse(graph) : {};
        const start = (g.nodes ?? []).find((n: any) => n.type === 'start');
        inputVars.value = start?.data?.inputVars ?? [];
      } catch {
        inputVars.value = [];
      }
    };
    // 先以列表行占位，再经 Get 拉全量（行字段可能被裁剪/滞后，graph 的事实源是 Get）
    parseInputVars(app.value.graph);
    if (app.value?.id) {
      fetchAiAgentApp(app.value.id)
        .then((resp: any) => {
          app.value = resp;
          parseInputVars(resp?.graph);
        })
        .catch((error: any) => {
          console.error('[ai-agent-run-drawer] load app detail failed', error);
        });
    }
  },
});

const app = ref<any>({});
const liveSteps = ref<string[]>([]);
let offFrame: (() => void) | undefined;
const runForm = reactive<Record<string, string>>({});
const inputVars = ref<Array<{ name?: string; label?: string; required?: boolean }>>([]);
</script>

<template>
  <Drawer :title="$t('page.aiAgent.runs.testTitle', { name: app?.name || '' })">
    <a-form layout="vertical" class="pt-2">
      <a-form-item
        v-for="v in inputVars"
        :key="v.name"
        :label="v.label || v.name"
        :required="v.required"
      >
        <a-textarea v-model:value="runForm[v.name!]" :rows="2" />
      </a-form-item>
      <div v-if="inputVars.length === 0" class="text-xs text-muted-foreground pb-2">
        {{ $t('page.aiAgent.runs.noInputs') }}
      </div>
      <div v-if="liveSteps.length > 0" class="pb-2 text-xs text-muted-foreground">
        {{ liveSteps.join(' · ') }}
      </div>
    </a-form>
  </Drawer>
</template>
