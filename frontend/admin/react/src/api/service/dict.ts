import {
  createDictTypeServiceClient,
  createDictEntryServiceClient,
  type dictservicev1_CreateDictTypeRequest,
  type dictservicev1_DeleteDictTypeRequest,
  type dictservicev1_UpdateDictTypeRequest,
  type dictservicev1_CreateDictEntryRequest,
  type dictservicev1_DeleteDictEntryRequest,
  type dictservicev1_UpdateDictEntryRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _dictTypeInstance: ReturnType<typeof createDictTypeServiceClient> | null = null;
let _dictEntryInstance: ReturnType<typeof createDictEntryServiceClient> | null = null;

export function getDictTypeService() {
  if (!_dictTypeInstance) {
    _dictTypeInstance = createDictTypeServiceClient(requestApi);
  }
  return _dictTypeInstance;
}

export function getDictEntryService() {
  if (!_dictEntryInstance) {
    _dictEntryInstance = createDictEntryServiceClient(requestApi);
  }
  return _dictEntryInstance;
}

// ==============================
// 字典类型管理
// ==============================

export async function listDictTypes(query: PaginationQuery) {
  const params = query.toRawParams();
  return getDictTypeService().List(params);
}

export async function getDictType(id: number) {
  return getDictTypeService().Get({ id });
}

export async function createDictType(request: dictservicev1_CreateDictTypeRequest) {
  return getDictTypeService().Create(request);
}

export async function updateDictType(request: dictservicev1_UpdateDictTypeRequest) {
  return getDictTypeService().Update(request);
}

export async function deleteDictType(request: dictservicev1_DeleteDictTypeRequest) {
  return getDictTypeService().Delete(request);
}

// ==============================
// 字典条目管理
// ==============================

export async function listDictEntries(query: PaginationQuery) {
  const params = query.toRawParams();
  return getDictEntryService().List(params);
}

export async function createDictEntry(request: dictservicev1_CreateDictEntryRequest) {
  return getDictEntryService().Create(request);
}

export async function updateDictEntry(request: dictservicev1_UpdateDictEntryRequest) {
  return getDictEntryService().Update(request);
}

export async function deleteDictEntry(request: dictservicev1_DeleteDictEntryRequest) {
  return getDictEntryService().Delete(request);
}
