import { preferences } from "@/core/preferences";

import { defineStore } from "pinia";

import type {
  dictservicev1_DictEntry as DictEntry,
  dictservicev1_ListDictEntryResponse as ListDictEntryResponse,
  dictservicev1_ListDictTypeResponse as ListDictTypeResponse,
  dictservicev1_ListLanguageResponse as ListLanguageResponse,
} from "@/api/generated/admin/service/v1";
import { fetchListDictTypes, fetchListDictEntries, fetchListLanguages } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";

/**
 * 字典视图状态接口
 */
interface DictViewState {
  loading: boolean; // 加载状态
  needReloadEntryList: boolean; // 是否需要重新加载字典项列表

  currentTypeId: null | number; // 当前选中的字典类型ID
  typeList: ListDictTypeResponse; // 字典类型列表
  entryList: ListDictEntryResponse; // 字典项列表
  languageList: ListLanguageResponse; // 语言列表
}

/**
 * 字典视图状态
 */
export const useDictViewStore = defineStore("dict-view", {
  state: (): DictViewState => ({
    currentTypeId: null,
    loading: false,
    needReloadEntryList: false,
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
    async fetchLanguageList(currentPage: number, pageSize: number, formValues: any) {
      this.loading = true;
      try {
        this.languageList = await fetchListLanguages(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues,
            orderBy: ["sortOrder"],
          })
        );

        await this.setCurrentTypeId(null);

        return this.languageList;
      } catch (error) {
        console.error("获取语言列表失败:", error);
        this.resetTypeList();
      } finally {
        this.loading = false;
      }

      return this.languageList;
    },

    /**
     * 获取字典类型列表
     */
    async fetchTypeList(currentPage: number, pageSize: number, formValues: any) {
      this.loading = true;
      try {
        this.typeList = await fetchListDictTypes(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues,
          })
        );

        await this.setCurrentTypeId(null);

        return this.typeList;
      } catch (error) {
        console.error("获取字典类型列表失败:", error);
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
      formValues: any
    ) {
      if (!typeId) {
        this.resetEntryList(); // 无字典类型ID时清空子列表
        return this.entryList;
      }

      this.loading = true;
      try {
        this.entryList = await fetchListDictEntries(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues: {
              ...formValues,
              type_id: typeId.toString(),
            },
          })
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
      this.needReloadEntryList = true; // 触发字典项列表刷新
    },

    resetTypeList() {
      this.typeList = { items: [], total: 0 };
    },

    resetEntryList() {
      this.entryList = { items: [], total: 0 };
    },
  },
});

export function getEntryLabel(row: DictEntry) {
  const currentI18n = row.i18n?.[preferences.app.locale];
  if (currentI18n === undefined) {
    return "";
  }
  return currentI18n.entryLabel;
}
