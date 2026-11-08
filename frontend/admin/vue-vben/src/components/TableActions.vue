<script lang="ts" setup>
import type { VNode } from 'vue';
import { computed } from 'vue';
import { LucideChevronDown } from '@vben/icons';
import { Modal } from 'ant-design-vue';
import { $t } from '#/locales';

/**
 * 列表操作列统一组件（设计语言 §4「列表操作列」定式的代码化）：
 * 行内 link 弱化钮「图标+文本」，判活后 >3 个只留前 3 个，其余收进「更多」下拉。
 * 「更多」里放不下 Popconfirm，带 confirm 的项自动降级 Modal.confirm（danger 项确认键同款 danger）。
 *
 * 用法：#action 插槽里以描述符数组声明动作，条件按钮用 hidden 剔除
 * （在计溢出前过滤，权限/角色隐藏后不足 3 个就不出「更多」）：
 *
 *   <template #action="{ row }">
 *     <TableActions :row="row" :actions="buildActions(row)" />
 *   </template>
 *
 *   function buildActions(row: any): TableAction[] {
 *     return [
 *       { key: 'edit', label: $t('common.edit'), icon: h(LucideFilePenLine), onClick: openEdit },
 *       { key: 'delete', label: $t('common.delete'), icon: h(LucideTrash2), danger: true,
 *         confirm: { title: $t('ui.text.do_you_want_delete') }, onClick: handleDelete },
 *     ];
 *   }
 *
 * 语义色：danger 红墨；success 走 --gowind-success-text 文字档（填充级 success 对白底
 * 对比度不足，见设计语言 §2.1；scoped 样式用 0-4-0 特异性压过 framework 的 0-3-0 蓝）。
 * 「更多」菜单项仅识别 danger（antdv 菜单无 success 语义）。
 * 按钮点击一律 @click.stop，行内可展开/可选中行的页面无需各自处理。
 */
export interface TableActionConfirm {
  title: string;
  description?: string;
  /** 确认键 loading（如强制下线这类带请求的确认） */
  okLoading?: boolean;
}

export interface TableAction {
  key: string;
  label: string;
  icon?: VNode;
  onClick: (row: any) => void;
  danger?: boolean;
  success?: boolean;
  confirm?: TableActionConfirm;
  /** 行内按钮的 Tooltip 提示（仅行内；溢出进「更多」菜单后不渲染 Tooltip） */
  tooltip?: string;
  hidden?: boolean;
}

const props = withDefaults(
  defineProps<{
    row: any;
    actions: TableAction[];
    /** 行内最多按钮数，超出收「更多」；默认 3（§4 定式值） */
    maxInline?: number;
  }>(),
  { maxInline: 3 },
);

const visible = computed(() => (props.actions ?? []).filter((item) => !item.hidden));
const inline = computed(() => visible.value.slice(0, props.maxInline));
const overflow = computed(() => visible.value.slice(props.maxInline));

function run(item: TableAction) {
  item.onClick(props.row);
}

/* 「更多」下拉里放不下 Popconfirm，确认类操作降级 Modal.confirm */
function confirmInMenu(item: TableAction) {
  Modal.confirm({
    title: item.confirm?.title,
    content: item.confirm?.description,
    okText: $t('ui.button.ok'),
    cancelText: $t('ui.button.cancel'),
    okType: item.danger ? 'danger' : 'primary',
    okButtonProps: item.confirm?.okLoading
      ? { loading: item.confirm.okLoading }
      : undefined,
    onOk: () => run(item),
  });
}

function menuItems() {
  return overflow.value.map((item) => ({
    key: item.key,
    icon: item.icon,
    label: item.label,
    danger: item.danger,
    onClick: () => (item.confirm ? confirmInMenu(item) : run(item)),
  }));
}
</script>

<template>
  <template v-for="item in inline" :key="item.key">
    <a-popconfirm
      v-if="item.confirm"
      :title="item.confirm.title"
      :description="item.confirm.description"
      :ok-text="$t('ui.button.ok')"
      :cancel-text="$t('ui.button.cancel')"
      :ok-button-props="
        item.confirm.okLoading ? { loading: item.confirm.okLoading } : undefined
      "
      @confirm="run(item)"
    >
      <a-tooltip v-if="item.tooltip" :title="item.tooltip">
        <a-button
          type="link"
          size="small"
          :danger="item.danger"
          :class="{ 'success-link': item.success }"
          :icon="item.icon"
          @click.stop
        >
          {{ item.label }}
        </a-button>
      </a-tooltip>
      <a-button
        v-else
        type="link"
        size="small"
        :danger="item.danger"
        :class="{ 'success-link': item.success }"
        :icon="item.icon"
        @click.stop
      >
        {{ item.label }}
      </a-button>
    </a-popconfirm>
    <template v-else>
      <a-tooltip v-if="item.tooltip" :title="item.tooltip">
        <a-button
          type="link"
          size="small"
          :danger="item.danger"
          :class="{ 'success-link': item.success }"
          :icon="item.icon"
          @click.stop="run(item)"
        >
          {{ item.label }}
        </a-button>
      </a-tooltip>
      <a-button
        v-else
        type="link"
        size="small"
        :danger="item.danger"
        :class="{ 'success-link': item.success }"
        :icon="item.icon"
        @click.stop="run(item)"
      >
        {{ item.label }}
      </a-button>
    </template>
  </template>
  <a-dropdown v-if="overflow.length > 0">
    <a-tooltip :title="$t('ui.button.more')">
      <a-button type="link" size="small" @click.prevent>
        <LucideChevronDown class="size-4" />
      </a-button>
    </a-tooltip>
    <template #overlay>
      <a-menu :items="menuItems()" />
    </template>
  </a-dropdown>
</template>

<style scoped>
/* 链接型按钮的墨取 §2.1 文字档：hsl(var(--success)) 是填充级 #57D188，对白底只有 1.92:1，
   当链接文字用等于看不见。hover 从文字档再往白走 10%，浅色 4.74、暗色只会更亮，两态都不掉出 4.5。 */
/* 0-4-0 特异性压过 framework 全局 .ant-btn.ant-btn-link:not(.ant-btn-dangerous)（0-3-0 !important）
   的主色蓝——否则这里的 success 文字档被整体改写，success 钮一直是蓝的。 */
:deep(.ant-btn.success-link.success-link) {
  color: var(--gowind-success-text) !important;
}

:deep(.ant-btn.success-link.success-link:hover) {
  color: color-mix(in srgb, var(--gowind-success-text) 90%, white) !important;
}
</style>
