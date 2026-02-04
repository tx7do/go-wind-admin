<script setup lang="ts">
import { computed, h, ref, unref, useAttrs, watch } from 'vue';

import { LucideEllipsisVertical } from '@vben/icons';
import { $t } from '@vben/locales';
import { get, isEqual, isFunction } from '@vben-core/shared/utils';

import { objectOmit } from '@vueuse/core';

import {
  type MenuInfo,
  type OptionsItem,
  type Props,
  ToolbarEnum,
  type TreeEmits,
} from './types';

const props = withDefaults(defineProps<Props>(), {
  title: '',
  toolbar: true,
  checkable: true,
  search: true,
  searchText: '',

  labelField: 'label',
  valueField: 'value',
  childrenField: 'children',
  optionsPropName: 'treeData',
  resultField: 'items',
  visibleEvent: 'onVisibleChange',
  numberToString: false,
  params: () => ({}),
  immediate: true,
  alwaysLoad: false,
  treeDefaultExpandAll: false,
  loadingSlot: 'suffixIcon',
  beforeFetch: undefined,
  afterFetch: undefined,
  modelPropName: 'checked-keys',
  api: undefined,
  options: () => [],
});

const emit = defineEmits<TreeEmits>();

const attrs = useAttrs();

const modelValue = defineModel({ default: [] });

const refOptions = ref<OptionsItem[]>([]);

// 首次是否加载过了
const isFirstLoaded = ref(false);
const loading = ref(false);

const expandedKeys = ref<string[]>();
const selectedKeys = ref<string[]>();
const checkStrictly = ref(false);

const toolbarList = computed(() => {
  const { checkable } = props;
  const defaultToolbarList = [
    { label: $t('ui.tree.expand_all'), value: ToolbarEnum.EXPAND_ALL },
    {
      label: $t('ui.tree.collapse_all'),
      value: ToolbarEnum.UN_EXPAND_ALL,
      divider: checkable,
    },
  ];

  return checkable
    ? [
        { label: $t('ui.tree.select_all'), value: ToolbarEnum.SELECT_ALL },
        {
          label: $t('ui.tree.unselect_all'),
          value: ToolbarEnum.UN_SELECT_ALL,
          divider: checkable,
        },
        ...defaultToolbarList,
        {
          label: $t('ui.tree.hierarchical_association'),
          value: ToolbarEnum.CHECK_STRICTLY,
        },
        {
          label: $t('ui.tree.hierarchical_independence'),
          value: ToolbarEnum.CHECK_UN_STRICTLY,
        },
      ]
    : defaultToolbarList;
});

function handleMenuClick(e: MenuInfo) {
  const { key } = e;
  switch (key) {
    case ToolbarEnum.CHECK_STRICTLY: {
      checkStrictly.value = false;
      break;
    }
    case ToolbarEnum.CHECK_UN_STRICTLY: {
      checkStrictly.value = true;
      break;
    }
    case ToolbarEnum.EXPAND_ALL: {
      expandAll();
      break;
    }
    case ToolbarEnum.SELECT_ALL: {
      checkAll();
      break;
    }
    case ToolbarEnum.UN_EXPAND_ALL: {
      collapseAll();
      break;
    }
    case ToolbarEnum.UN_SELECT_ALL: {
      uncheckAll();
      break;
    }
  }
}

function transformData(data: OptionsItem[]): OptionsItem[] {
  const { labelField, valueField, childrenField, numberToString } = props;

  return data.map((item, index) => {
    const value = get(item, valueField);
    // 校验value是否存在，不存在则警告并生成临时唯一key
    if (value === undefined || value === null) {
      console.warn('节点缺少valueField对应的值', { item, valueField });
      // 生成临时唯一key（避免重复）
      const tempKey = `temp-key-${index}-${Date.now()}`;
      console.warn(`自动生成临时key: ${tempKey}`);
    }
    return {
      ...objectOmit(item, [labelField, valueField, childrenField]),
      title: get(item, labelField),
      // 确保key有效：优先用value，否则用临时key
      key:
        value !== undefined && value !== null
          ? (numberToString ? `${value}` : value)
          : `temp-key-${index}-${Date.now()}`,
      ...(childrenField && item[childrenField]
        ? { children: transformData(item[childrenField]) }
        : {}),
    };
  });
}

const getOptions = computed(() => {
  const refOptionsData = unref(refOptions);

  const data: OptionsItem[] = transformData(refOptionsData);

  return data.length > 0 ? data : props.options;
});

function emitChange() {
  emit('optionsChange', unref(getOptions));
}

async function fetchApi() {
  let { api, beforeFetch, afterFetch, params, resultField } = props;

  if (!api || !isFunction(api) || loading.value) {
    return;
  }

  refOptions.value = [];
  try {
    loading.value = true;

    if (beforeFetch && isFunction(beforeFetch)) {
      params = (await beforeFetch(params)) || params;
    }

    let res = await api(params);

    if (afterFetch && isFunction(afterFetch)) {
      res = (await afterFetch(res)) || res;
    }

    isFirstLoaded.value = true;

    if (Array.isArray(res)) {
      refOptions.value = res;
      emitChange();
      return;
    }

    if (resultField) {
      refOptions.value = get(res, resultField) || [];
    }
    emitChange();
  } catch (error) {
    console.warn(error);
    // reset status
    isFirstLoaded.value = false;
  } finally {
    loading.value = false;
  }
}

function getModalValue() {
  return unref(getOptions).length > 0 ? unref(modelValue) : [];
}

const bindProps = computed(() => {
  return {
    [props.modelPropName]: getModalValue(),
    [props.optionsPropName]: unref(getOptions),
    [`onUpdate:${props.modelPropName}`]: (val: any) => {
      modelValue.value = val;
    },
    ...objectOmit(attrs, ['onUpdate:value']),
    ...(props.visibleEvent
      ? {
          [props.visibleEvent]: handleFetchForVisible,
        }
      : {}),
  };
});

async function handleFetchForVisible(visible: boolean) {
  if (visible) {
    if (props.alwaysLoad) {
      await fetchApi();
    } else if (!props.immediate && !unref(isFirstLoaded)) {
      await fetchApi();
    }
  }
}

function getAllKeys(): never[] {
  const keys: never[] = [];
  function getKeys(data: OptionsItem[]) {
    data.forEach((item) => {
      keys.push(item.key as never);
      if (item.children) {
        getKeys(item.children);
      }
    });
  }
  getKeys(unref(getOptions));
  return keys;
}

/**
 * 展开所有节点
 */
function expandAll() {
  expandedKeys.value = getAllKeys();
}

/**
 * 收起所有节点
 */
function collapseAll() {
  expandedKeys.value = [];
}

/**
 * 全选
 */
function checkAll() {
  modelValue.value = getAllKeys();
}

/**
 * 全不选
 */
function uncheckAll() {
  modelValue.value = [];
}

watch(
  () => props.params,
  (value, oldValue) => {
    if (isEqual(value, oldValue)) {
      return;
    }
    fetchApi();
  },
  { deep: true, immediate: props.immediate },
);
</script>

<template>
  <a-space direction="vertical" v-bind="{ ...$attrs }">
    <a-space>
      <div>{{ props.title }}</div>
      <a-dropdown>
        <a-button type="link" :icon="h(LucideEllipsisVertical)" />
        <template #overlay>
          <a-menu @click="handleMenuClick">
            <template v-for="item in toolbarList" :key="item.value">
              <a-menu-item v-bind="{ key: item.value }">
                {{ item.label }}
              </a-menu-item>
              <a-menu-divider v-if="item.divider" />
            </template>
          </a-menu>
        </template>
      </a-dropdown>
    </a-space>
    <a-tree
      checkable
      v-bind="bindProps"
      v-model:expanded-keys="expandedKeys"
      v-model:selected-keys="selectedKeys"
      :check-strictly="checkStrictly"
    />
  </a-space>
</template>

<style scoped></style>
