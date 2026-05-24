import {
  createFileServiceClient,
  type storageservicev1_CreateFileRequest,
  type storageservicev1_DeleteFileRequest,
  type storageservicev1_GetFileRequest,
  type storageservicev1_UpdateFileRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createFileServiceClient> | null = null;

export function getFileService() {
  if (!_instance) {
    _instance = createFileServiceClient(requestApi);
  }
  return _instance;
}

export async function listFiles(query: PaginationQuery) {
  const params = query.toRawParams();
  return getFileService().List(params);
}

export async function getFile(request: storageservicev1_GetFileRequest) {
  return getFileService().Get(request);
}

export async function createFile(request: storageservicev1_CreateFileRequest) {
  return getFileService().Create(request);
}

export async function updateFile(request: storageservicev1_UpdateFileRequest) {
  return getFileService().Update(request);
}

export async function deleteFile(request: storageservicev1_DeleteFileRequest) {
  return getFileService().Delete(request);
}
