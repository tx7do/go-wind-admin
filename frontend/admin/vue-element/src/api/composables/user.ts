import { computed } from "vue";
import type { TagType } from "./shared";
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  identityservicev1_User,
  identityservicev1_ListUserResponse,
  identityservicev1_GetUserRequest,
  identityservicev1_UserExistsRequest,
  identityservicev1_EditUserPasswordRequest,
  identityservicev1_UserExistsResponse,
  identityservicev1_User_Gender as User_Gender,
  identityservicev1_User_Status as User_Status,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";
import { encryptPasswordEnvelope } from "@/utils";

const t = i18n.global.t;

// ==============================
// 获取用户列表
// ==============================
export function useListUsers(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListUserResponse, Error>
) {
  return useQuery({
    queryKey: ["listUsers", query],
    queryFn: () => {
      const params = query.toRawParams();
      return apiClient.userService.List({
        ...params,
        sorting: undefined,
        offset: undefined,
        limit: undefined,
        token: undefined,
        filter: undefined,
        filterExpr: undefined,
      });
    },
    ...options,
  });
}

// ==============================================
// 获取用户列表 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchListUsers(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listUsers", params],
    queryFn: () => {
      const reqParams = params.toRawParams();
      return apiClient.userService.List({
        ...reqParams,
        sorting: undefined,
        offset: undefined,
        limit: undefined,
        token: undefined,
        filter: undefined,
        filterExpr: undefined,
      });
    },
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 获取单个用户
// ==============================
export function useGetUser(
  req: identityservicev1_GetUserRequest,
  options?: UseQueryOptions<identityservicev1_User, Error>
) {
  return useQuery({
    queryKey: ["getUser", req],
    queryFn: () => apiClient.userService.Get(req),
    ...options,
  });
}

// ==============================================
// 获取单个用户 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchUser(params: identityservicev1_GetUserRequest) {
  return queryClient.fetchQuery({
    queryKey: ["getUser", params],
    queryFn: () => apiClient.userService.Get(params),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 创建用户
// ==============================
export function useCreateUser(
  options?: UseMutationOptions<{}, Error, { data: identityservicev1_User; password?: string }>
) {
  return useMutation({
    mutationFn: ({ data, password }) => apiClient.userService.Create({ data, password }),
    ...options,
  });
}

// ==============================
// 删除用户
// ==============================
export function useDeleteUser(options?: UseMutationOptions<{}, Error, string>) {
  return useMutation({
    mutationFn: (id) => apiClient.userService.Delete({ id }),
    ...options,
  });
}

// ==============================
// 更新用户
// ==============================
export function useUpdateUser(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
) {
  return useMutation({
    // proto UpdateUserRequest.password 是顶层字段且要求 AES 密文：
    // 留在 data/updateMask 里后端读不到，会静默跳过重置密码（200 但密码未变）
    mutationFn: async ({ id, values }: { id: string; values: Record<string, any> }) => {
      const { password, ...data } = values ?? {};
      return apiClient.userService.Update({
        id,
        data: data as any,
        password: password ? await encryptPasswordEnvelope(String(password)) : undefined,
        updateMask: makeUpdateMask(Object.keys(data)),
      });
    },
    ...options,
  });
}

// ==============================
// 检查用户是否存在
// ==============================
export function useUserExists(
  options?: UseMutationOptions<
    identityservicev1_UserExistsResponse,
    Error,
    identityservicev1_UserExistsRequest
  >
) {
  return useMutation({
    mutationFn: (data) => apiClient.userService.UserExists(data),
    ...options,
  });
}

// ==============================
// 修改用户密码（管理员）
// ==============================
export function useEditUserPassword(
  options?: UseMutationOptions<{}, Error, identityservicev1_EditUserPasswordRequest>,
) {
  return useMutation({
    // 后端 NeedDecrypt 要求 AES 密文传输（与登录同规），明文会被当密文解密导致校验必败
    mutationFn: async (data) =>
      apiClient.userService.EditUserPassword({
        ...data,
        newPassword: data.newPassword ? await encryptPasswordEnvelope(data.newPassword) : data.newPassword,
      }),
    ...options,
  });
}

// ==============================
// 用户枚举与工具函数
// ==============================

export const userStatusList = computed(() => [
  { value: "NORMAL", label: t("enum.user.status.NORMAL") },
  { value: "DISABLED", label: t("enum.user.status.DISABLED") },
  { value: "PENDING", label: t("enum.user.status.PENDING") },
  { value: "LOCKED", label: t("enum.user.status.LOCKED") },
  { value: "EXPIRED", label: t("enum.user.status.EXPIRED") },
  { value: "CLOSED", label: t("enum.user.status.CLOSED") },
]);

// 启用/禁用 → tag type，NORMAL 为启用（success），其余统一为 info
export function userStatusToType(status: User_Status): TagType {
  return status === "NORMAL" ? "success" : "info";
}

export function userStatusToName(status?: User_Status) {
  const values = userStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

export const genderList = computed(() => [
  { value: "SECRET", label: t("enum.gender.SECRET") },
  { value: "MALE", label: t("enum.gender.MALE") },
  { value: "FEMALE", label: t("enum.gender.FEMALE") },
]);

export function genderToName(gender?: User_Gender) {
  const values = genderList.value;
  const matchedItem = values.find((item) => item.value === gender);
  return matchedItem ? matchedItem.label : "";
}

export function genderToType(gender?: User_Gender): TagType {
  switch (gender) {
    case "FEMALE":
      return "danger";
    case "MALE":
      return "primary";
    case "SECRET":
      return "info";
    default:
      return "info";
  }
}
