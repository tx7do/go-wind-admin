import {
  createDictEntryServiceClient,
  createDictTypeServiceClient,
  type dictservicev1_CreateDictEntryRequest,
  type dictservicev1_CreateDictTypeRequest,
  type dictservicev1_DeleteDictEntryRequest,
  type dictservicev1_DeleteDictTypeRequest,
  type dictservicev1_GetDictTypeRequest,
  type dictservicev1_UpdateDictEntryRequest,
  type dictservicev1_UpdateDictTypeRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _dictTypeInstance: null | ReturnType<typeof createDictTypeServiceClient> =
  null;
let _dictEntryInstance: null | ReturnType<typeof createDictEntryServiceClient> =
  null;

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

export async function getDictType(request: dictservicev1_GetDictTypeRequest) {
  return getDictTypeService().Get(request);
}

export async function createDictType(
  request: dictservicev1_CreateDictTypeRequest,
) {
  return getDictTypeService().Create(request);
}

export async function updateDictType(
  request: dictservicev1_UpdateDictTypeRequest,
) {
  return getDictTypeService().Update(request);
}

export async function deleteDictType(
  request: dictservicev1_DeleteDictTypeRequest,
) {
  return getDictTypeService().Delete(request);
}

// ==============================
// 字典条目管理
// ==============================

export async function listDictEntries(query: PaginationQuery) {
  const params = query.toRawParams();
  return getDictEntryService().List(params);
}

export async function createDictEntry(
  request: dictservicev1_CreateDictEntryRequest,
) {
  return getDictEntryService().Create(request);
}

export async function updateDictEntry(
  request: dictservicev1_UpdateDictEntryRequest,
) {
  return getDictEntryService().Update(request);
}

export async function deleteDictEntry(
  request: dictservicev1_DeleteDictEntryRequest,
) {
  return getDictEntryService().Delete(request);
}
