import {
  createPositionServiceClient,
  type identityservicev1_CreatePositionRequest,
  type identityservicev1_DeletePositionRequest,
  type identityservicev1_GetPositionRequest,
  type identityservicev1_UpdatePositionRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _positionInstance: null | ReturnType<typeof createPositionServiceClient> =
  null;

export function getPositionService() {
  if (!_positionInstance) {
    _positionInstance = createPositionServiceClient(requestApi);
  }
  return _positionInstance;
}

// ==============================
// 职位管理
// ==============================

export async function listPositions(query: PaginationQuery) {
  const params = query.toRawParams();
  return getPositionService().List(params);
}

export async function getPosition(
  request: identityservicev1_GetPositionRequest,
) {
  return getPositionService().Get(request);
}

export async function createPosition(
  request: identityservicev1_CreatePositionRequest,
) {
  return getPositionService().Create(request);
}

export async function updatePosition(
  request: identityservicev1_UpdatePositionRequest,
) {
  return getPositionService().Update(request);
}

export async function deletePosition(
  request: identityservicev1_DeletePositionRequest,
) {
  return getPositionService().Delete(request);
}
