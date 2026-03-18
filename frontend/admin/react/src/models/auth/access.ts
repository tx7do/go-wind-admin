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

export function loadAccessToken(): TokenPayload | null {
  return storage.getItem<TokenPayload>('accessToken', null);
}

export function loadRefreshToken(): TokenPayload | null {
  return storage.getItem<TokenPayload>('refreshToken', null);
}

export function saveAccessToken(accessToken: TokenPayload) {
  storage.setItem('accessToken', accessToken);
}

export function saveRefreshToken(refreshToken: TokenPayload) {
  storage.setItem('refreshToken', refreshToken);
}

export function clearAccessToken() {
  storage.removeItem('accessToken');
}

export function clearRefreshToken() {
  storage.removeItem('refreshToken');
}

export function clearAccess() {
  clearAccessToken();
  clearRefreshToken();
}

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
    const accessToken = loadAccessToken();
    const refreshToken = loadRefreshToken();

    return {
      permissions: permissions || [],
      accessToken: accessToken || null,
      refreshToken: refreshToken || null,
    };
  });

  // 持久化 state
  useEffect(() => {
    if (typeof window !== 'undefined') {
      console.log('[AccessModel] useEffect triggered, persisting state:', state);

      storage.setItem('permissions', state.permissions);
      if (state.accessToken) {
        saveAccessToken(state.accessToken);
        console.log('[AccessModel] Saved accessToken to localStorage');
      } else {
        storage.removeItem('accessToken');
        console.log('[AccessModel] Removed accessToken from localStorage');
      }
      if (state.refreshToken) {
        saveRefreshToken(state.refreshToken);
        console.log('[AccessModel] Saved refreshToken to localStorage');
      } else {
        storage.removeItem('refreshToken');
        console.log('[AccessModel] Removed refreshToken from localStorage');
      }

      // 验证是否真的保存了
      const savedAccessToken = storage.getItem('accessToken', null);
      console.log('[AccessModel] Verify - saved accessToken from localStorage:', savedAccessToken);
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
      console.log('[AccessModel] setTokens called:', payload);

      // 直接更新 state
      const newState = {
        ...state,
        accessToken: payload.accessToken,
        refreshToken: payload.refreshToken,
      };
      setState(newState);

      // 立即保存到 localStorage
      if (typeof window !== 'undefined') {
        console.log('[AccessModel] Immediately saving tokens to localStorage');
        if (payload.accessToken) {
          saveAccessToken(payload.accessToken);
          console.log('[AccessModel] Saved accessToken');
        }
        if (payload.refreshToken) {
          saveRefreshToken(payload.refreshToken);
          console.log('[AccessModel] Saved refreshToken');
        }
      }
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
