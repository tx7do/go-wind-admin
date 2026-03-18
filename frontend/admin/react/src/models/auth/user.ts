import {useState, useEffect} from 'react';

import {StorageManager} from '@/caches/storage-manager';
import {appNamespace} from '@/caches';

import type {IUser} from '../types';

const storage = new StorageManager({prefix: `${appNamespace}-user`, storageType: 'localStorage'});

/**
 * 用户信息管理 Model
 * 提供用户信息的存储和更新
 */
export default function UserModel() {
  const [user, setUser] = useState<IUser | null>(() => {
    // 服务端环境返回 null
    if (typeof window === 'undefined') {
      return null;
    }

    // 客户端环境从 localStorage 读取
    return storage.getItem<IUser>('info', null);
  });

  // 持久化用户信息
  useEffect(() => {
    if (typeof window !== 'undefined') {
      if (user) {
        storage.setItem('info', user);
      } else {
        storage.removeItem('info');
      }
    }
  }, [user]);

  return {
    user,
    setUser,
    clearUser: () => setUser(null),
    updateUser: (updates: Partial<IUser>) => {
      setUser(prev => prev ? {...prev, ...updates} : null);
    },
  };
}
