import { preferences } from '@vben/preferences';

import { defineStore } from 'pinia';

import {
  type dictservicev1_DictEntry as DictEntry,
  type dictservicev1_DictType as DictType,
  type dictservicev1_ListDictEntryResponse as ListDictEntryResponse,
  type dictservicev1_ListDictTypeResponse as ListDictTypeResponse,
  type dictservicev1_ListLanguageResponse as ListLanguageResponse,
} from '#/generated/api/admin/service/v1';
import { useDictStore, useLanguageStore } from '#/stores';

const dictStore = useDictStore();
const languageStore = useLanguageStore();

/**
 * 字典视图状态接口
 */
interface DictViewState {
  loading: boolean; // 加载状态

  currentTypeId: null | number; // 当前选中的字典类型ID
  typeList: ListDictTypeResponse; // 字典类型列表
  entryList: ListDictEntryResponse; // 字典项列表
  languageList: ListLanguageResponse; // 语言列表
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
    languageList: { items: [], total: 0 },
  }),

  actions: {
    /**
     * 获取语言列表
     * @param currentPage
     * @param pageSize
     * @param formValues
     */
    async fetchLanguageList(
      currentPage: number,
      pageSize: number,
      formValues: any,
    ) {
      this.loading = true;
      try {
        this.languageList = await languageStore.listLanguage(
          {
            page: currentPage,
            pageSize,
          },
          formValues,
        );

        await this.setCurrentTypeId(null);

        return this.languageList;
      } catch (error) {
        console.error('获取语言列表失败:', error);
        this.resetTypeList();
      } finally {
        this.loading = false;
      }

      return this.languageList;
    },

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
        console.error('获取字典类型列表失败:', error);
        this.resetTypeList();
      } finally {
        this.loading = false;
      }

      return this.typeList;
    },

    /**
     * 根据字典类型ID获取字典项列表
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
        console.error(`获取字典类型[${typeId}]的字典项列表失败:`, error);
        this.resetEntryList();
      } finally {
        this.loading = false;
      }

      return this.entryList;
    },

    /**
     * 点击字典类型时触发：设置当前字典类型ID + 刷新字典项列表
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

export function getTypeName(row: DictType) {
  const currentI18n = row.i18n?.[preferences.app.locale];
  if (currentI18n === undefined) {
    return '';
  }
  return currentI18n.typeName;
}

export function getEntryLabel(row: DictEntry) {
  const currentI18n = row.i18n?.[preferences.app.locale];
  if (currentI18n === undefined) {
    return '';
  }
  return currentI18n.entryLabel;
}
