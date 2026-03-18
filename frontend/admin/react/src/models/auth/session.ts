import {useState, useEffect} from 'react';

import {StorageManager} from '@/caches/storage-manager';
import {appNamespace} from '@/caches';

import type {ISessionState} from '../types';

const storage = new StorageManager({prefix: `${appNamespace}-session`, storageType: 'localStorage'});

const initialState: ISessionState = {
  isAccessChecked: false,
  loginExpired: false,
  lastLoginTime: undefined,
};

/**
 * 会话管理 Model
 * 管理用户登录会话状态（是否检查过权限、登录是否过期等）
 */
export default function SessionModel() {
  const [state, setState] = useState<ISessionState>(() => {
    // 服务端环境返回初始状态
    if (typeof window === 'undefined') {
      return initialState;
    }

    // 客户端环境从 localStorage 读取
    const isAccessChecked = storage.getItem<boolean>('isAccessChecked', false);
    const loginExpired = storage.getItem<boolean>('loginExpired', false);
    const lastLoginTime = storage.getItem<number>('lastLoginTime', undefined);

    return {
      isAccessChecked: isAccessChecked ?? false,
      loginExpired: loginExpired ?? false,
      lastLoginTime: lastLoginTime ?? undefined,
    };
  });

  // 持久化会话状态
  useEffect(() => {
    if (typeof window !== 'undefined') {
      storage.setItem('isAccessChecked', state.isAccessChecked);
      storage.setItem('loginExpired', state.loginExpired);
      if (state.lastLoginTime) {
        storage.setItem('lastLoginTime', state.lastLoginTime);
      } else {
        storage.removeItem('lastLoginTime');
      }
    }
  }, [state]);

  return {
    ...state,
    // 标记权限已检查
    setIsAccessChecked: (isAccessChecked: boolean) => {
      setState(prev => ({...prev, isAccessChecked}));
    },

    // 设置登录过期状态
    setLoginExpired: (loginExpired: boolean) => {
      setState(prev => ({...prev, loginExpired}));
    },

    // 设置最后登录时间
    setLastLoginTime: (time: number) => {
      setState(prev => ({...prev, lastLoginTime: time}));
    },

    // 重置会话状态
    resetSession: () => {
      setState(initialState);
    },
  };
}
