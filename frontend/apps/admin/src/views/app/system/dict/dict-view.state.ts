import { defineStore } from 'pinia';

import {
  type dictservicev1_ListDictEntryResponse as ListDictEntryResponse,
  type dictservicev1_ListDictTypeResponse as ListDictTypeResponse,
} from '#/generated/api/admin/service/v1';
import { useDictStore } from '#/stores';

const dictStore = useDictStore();

/**
 * 字典视图状态接口
 */
interface DictViewState {
  loading: boolean; // 加载状态

  currentTypeId: null | number; // 当前选中的字典类型ID
  typeList: ListDictTypeResponse; // 字典类型列表
  entryList: ListDictEntryResponse; // 字典条目列表
}

/**
 * 字典视图状态
 */
export const useDictViewStore = defineStore('dict-view', {
  state: (): DictViewState => ({
    currentTypeId: null,
    loading: false,
    typeList: { items: [], total: 0 },
    entryList: { items: [], total: 0 },
  }),

  actions: {
    /**
     * 获取字典类型列表
     */
    async fetchTypeList(
      currentPage: number,
      pageSize: number,
      formValues: any,
    ) {
      this.loading = true;
      try {
        this.typeList = await dictStore.listDictType(
          {
            page: currentPage,
            pageSize,
          },
          formValues,
        );

        await this.setCurrentTypeId(null);

        return this.typeList;
      } catch (error) {
        console.error('获取字典类型失败:', error);
        this.resetTypeList();
      } finally {
        this.loading = false;
      }

      return this.typeList;
    },

    /**
     * 根据字典类型ID获取字典条目列表
     * @param typeId 字典类型ID
     * @param currentPage
     * @param pageSize
     * @param formValues
     */
    async fetchEntryList(
      typeId: null | number,
      currentPage: number,
      pageSize: number,
      formValues: any,
    ) {
      if (!typeId) {
        this.resetEntryList(); // 无字典类型ID时清空子列表
        return this.entryList;
      }

      this.loading = true;
      try {
        this.entryList = await dictStore.listDictEntry(
          {
            page: currentPage,
            pageSize,
          },
          {
            ...formValues,
            type_id: typeId.toString(),
          },
        );
      } catch (error) {
        console.error(`获取字典类型[${typeId}]的条目失败:`, error);
        this.resetEntryList();
      } finally {
        this.loading = false;
      }

      return this.entryList;
    },

    /**
     * 点击字典类型时触发：设置当前字典类型ID + 刷新字典条目列表
     * @param typeId 字典类型ID
     */
    async setCurrentTypeId(typeId: null | number) {
      this.currentTypeId = typeId; // 更新当前选中的字典类型ID
    },

    resetTypeList() {
      this.typeList = { items: [], total: 0 };
    },

    resetEntryList() {
      this.entryList = { items: [], total: 0 };
    },
  },
});
