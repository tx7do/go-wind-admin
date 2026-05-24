import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type identityservicev1_CreatePositionRequest,
  type identityservicev1_DeletePositionRequest,
  type identityservicev1_GetPositionRequest,
  type identityservicev1_ListPositionResponse,
  type identityservicev1_Position,
  type identityservicev1_UpdatePositionRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listPositions, getPosition, createPosition, updatePosition, deletePosition } from '@/api/service/position';

// ==============================
// 职位管理
// ==============================

export function useListPositions(
  options?: UseMutationOptions<identityservicev1_ListPositionResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listPositions(query),
    ...options,
  });
}

export function useGetPosition(
  options?: UseMutationOptions<
    identityservicev1_Position,
    Error,
    identityservicev1_GetPositionRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getPosition(req),
    ...options,
  });
}

export function useCreatePosition(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreatePositionRequest>,
) {
  return useMutation({
    mutationFn: (data) => createPosition(data),
    ...options,
  });
}

export function useUpdatePosition(
  options?: UseMutationOptions<{}, Error, identityservicev1_UpdatePositionRequest>,
) {
  return useMutation({
    mutationFn: (data) => updatePosition(data),
    ...options,
  });
}

export function useDeletePosition(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePositionRequest>,
) {
  return useMutation({
    mutationFn: (req) => deletePosition(req),
    ...options,
  });
}
