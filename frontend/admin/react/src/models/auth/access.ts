import {useState, useEffect} from 'react';

import {StorageManager} from '@/caches/storage-manager';
import {appNamespace} from '@/caches';

import type {IAccessState, TokenPayload} from '../types';

const storage = new StorageManager({prefix: `${appNamespace}-access`, storageType: 'localStorage'});

const initialState: IAccessState = {
  permissions: [],
  accessToken: null,
  refreshToken: null,
};

/**
 * 权限管理 Model
 * 提供访问令牌、权限码的管理和持久化
 */
export default function AccessModel() {
  const [state, setState] = useState<IAccessState>(() => {
    // 服务端环境返回初始状态
    if (typeof window === 'undefined') {
      return initialState;
    }

    // 客户端环境从 localStorage 读取
    const permissions = storage.getItem<string[]>('permissions', []);
    const accessToken = storage.getItem<TokenPayload>('accessToken', null);
    const refreshToken = storage.getItem<TokenPayload>('refreshToken', null);

    return {
      permissions: permissions || [],
      accessToken: accessToken || null,
      refreshToken: refreshToken || null,
    };
  });

  // 持久化 state
  useEffect(() => {
    if (typeof window !== 'undefined') {
      storage.setItem('permissions', state.permissions);
      if (state.accessToken) {
        storage.setItem('accessToken', state.accessToken);
      } else {
        storage.removeItem('accessToken');
      }
      if (state.refreshToken) {
        storage.setItem('refreshToken', state.refreshToken);
      } else {
        storage.removeItem('refreshToken');
      }
    }
  }, [state]);

  return {
    ...state,
    // 设置权限列表
    setPermissions: (permissions: string[]) => {
      setState(prev => ({...prev, permissions}));
    },

    // 添加单个权限
    addPermission: (permission: string) => {
      setState(prev => ({
        ...prev,
        permissions: prev.permissions.includes(permission)
          ? prev.permissions
          : [...prev.permissions, permission],
      }));
    },

    // 移除权限
    removePermission: (permission: string) => {
      setState(prev => ({
        ...prev,
        permissions: prev.permissions.filter(p => p !== permission),
      }));
    },

    // 设置访问令牌
    setAccessToken: (accessToken: TokenPayload | null) => {
      setState(prev => ({...prev, accessToken}));
    },

    // 设置刷新令牌
    setRefreshToken: (refreshToken: TokenPayload | null) => {
      setState(prev => ({...prev, refreshToken}));
    },

    // 同时设置两个令牌
    setTokens: (payload: { accessToken: TokenPayload | null; refreshToken: TokenPayload | null }) => {
      setState(prev => ({...prev, ...payload}));
    },

    // 清除访问令牌（但保留权限信息）
    clearTokens: () => {
      setState(prev => ({
        ...prev,
        accessToken: null,
        refreshToken: null,
      }));
    },

    // 重置所有访问状态
    resetAccess: () => {
      setState(initialState);
    },
  };
}
