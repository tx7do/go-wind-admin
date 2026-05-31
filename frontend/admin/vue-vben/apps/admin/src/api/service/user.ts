import {
  createUserServiceClient,
  type identityservicev1_CreateUserRequest,
  type identityservicev1_DeleteUserRequest,
  type identityservicev1_EditUserPasswordRequest,
  type identityservicev1_GetUserRequest,
  type identityservicev1_UpdateUserRequest,
  type identityservicev1_UserExistsRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createUserServiceClient> = null;

export function getUserService() {
  if (!_instance) {
    _instance = createUserServiceClient(requestApi);
  }
  return _instance;
}

export async function listUsers(query: PaginationQuery) {
  const params = query.toRawParams();

  const req = {
    ...params,
    sorting: undefined,
    offset: undefined,
    limit: undefined,
    token: undefined,
    filter: undefined,
    filterExpr: undefined,
  };

  return getUserService().List(req);
}

export async function getUser(request: identityservicev1_GetUserRequest) {
  return getUserService().Get(request);
}

export async function createUser(request: identityservicev1_CreateUserRequest) {
  return getUserService().Create(request);
}

export async function updateUser(request: identityservicev1_UpdateUserRequest) {
  return getUserService().Update(request);
}

export async function deleteUser(request: identityservicev1_DeleteUserRequest) {
  return getUserService().Delete(request);
}

export async function userExists(request: identityservicev1_UserExistsRequest) {
  return getUserService().UserExists(request);
}

export async function editUserPassword(
  request: identityservicev1_EditUserPasswordRequest,
) {
  return getUserService().EditUserPassword(request);
}
