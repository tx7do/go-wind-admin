<script lang="ts" setup>
import { ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message, Modal } from 'ant-design-vue';

import { listAiAgentRevisions, restoreAiAgentRevision } from '#/api';

const app = ref<any>({});
const revisions = ref<any[]>([]);
const loading = ref(false);
const expandedId = ref<number | undefined>();

const [Drawer, drawerApi] = useVbenDrawer({
  class: 'w-[640px]',
  footer: false,
  onOpenChange(isOpen) {
    if (!isOpen) return;
    const data = drawerApi.getData<{ row: any }>() ?? { row: {} };
    app.value = data.row ?? {};
    expandedId.value = undefined;
    loading.value = true;
    listAiAgentRevisions(app.value.id)
      .then((resp: any) => {
        revisions.value = resp.items ?? [];
      })
      .catch((error: any) => {
        console.error('[ai-agent] list revisions failed', error);
      })
      .finally(() => {
        loading.value = false;
      });
  },
});

function handleRestore(rev: any) {
  Modal.confirm({
    title: $t('page.aiAgent.revisions.restoreConfirm'),
    okText: $t('ui.button.ok'),
    cancelText: $t('ui.button.cancel'),
    onOk: () => {
      restoreAiAgentRevision(rev.id)
        .then(() => {
          message.success($t('page.aiAgent.revisions.restored'));
          // 重拉快照列表：恢复动作本身会以 RESTORE 来源留底当前版本
          const data = drawerApi.getData<{ row: any }>() ?? { row: {} };
          app.value = data.row ?? app.value;
          loading.value = true;
          listAiAgentRevisions(app.value.id)
            .then((resp: any) => {
              revisions.value = resp.items ?? [];
            })
            .catch((error: any) => {
              console.error('[ai-agent] list revisions failed', error);
            })
            .finally(() => {
              loading.value = false;
            });
        })
        .catch((error: any) => {
          console.error('[ai-agent] restore revision failed', error);
          message.error(error?.message || $t('page.aiAgent.revisions.restoreFailed'));
        });
    },
  });
}
</script>

<template>
  <Drawer :title="`${$t('page.aiAgent.openRevisions')} · ${app?.name || ''}`">
    <div v-if="revisions.length === 0 && !loading" class="pb-2 text-xs text-muted-foreground">
      {{ $t('page.aiAgent.revisions.empty') }}
    </div>
    <div v-if="loading" class="pb-2 text-xs text-muted-foreground">
      {{ $t('ui.loading') }}
    </div>

    <div
      v-for="rev in revisions"
      :key="rev.id"
      class="mb-3 rounded-lg border border-border-light p-3"
    >
      <div class="flex flex-wrap items-center gap-2">
        <a-tag v-if="rev.source === 'RESTORE'" color="processing">
          {{ $t('page.aiAgent.revisions.sourceRestore') }}
        </a-tag>
        <a-tag v-else>{{ $t('page.aiAgent.revisions.sourceSave') }}</a-tag>
        <span class="text-xs text-muted-foreground">#{{ rev.id }}</span>
        <span class="text-xs text-muted-foreground">{{ rev.createdAt }}</span>
        <span class="ml-auto flex items-center gap-1.5">
          <a @click="expandedId = expandedId === rev.id ? undefined : rev.id" class="cursor-pointer text-xs">
            {{ expandedId === rev.id ? $t('page.aiAgent.revisions.collapse') : $t('page.aiAgent.revisions.viewGraph') }}
          </a>
          <a-button size="small" @click="handleRestore(rev)">
            {{ $t('page.aiAgent.revisions.restore') }}
          </a-button>
        </span>
      </div>
      <pre
        v-if="expandedId === rev.id"
        class="mt-2 max-h-[240px] overflow-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap break-all"
      >{{ rev.graph }}</pre>
    </div>
  </Drawer>
</template>
