import {
  createLanguageServiceClient,
  type dictservicev1_CreateLanguageRequest,
  type dictservicev1_DeleteLanguageRequest,
  type dictservicev1_GetLanguageRequest,
  type dictservicev1_UpdateLanguageRequest,
  type dictservicev1_BatchCreateLanguagesRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createLanguageServiceClient> | null = null;

export function getLanguageService() {
  if (!_instance) {
    _instance = createLanguageServiceClient(requestApi);
  }
  return _instance;
}

export async function listLanguages(query: PaginationQuery) {
  const params = query.toRawParams();
  return getLanguageService().List(params);
}

export async function getLanguage(request: dictservicev1_GetLanguageRequest) {
  return getLanguageService().Get(request);
}

export async function createLanguage(request: dictservicev1_CreateLanguageRequest) {
  return getLanguageService().Create(request);
}

export async function updateLanguage(request: dictservicev1_UpdateLanguageRequest) {
  return getLanguageService().Update(request);
}

export async function deleteLanguage(request: dictservicev1_DeleteLanguageRequest) {
  return getLanguageService().Delete(request);
}

export async function batchCreateLanguages(request: dictservicev1_BatchCreateLanguagesRequest) {
  return getLanguageService().BatchCreate(request);
}
