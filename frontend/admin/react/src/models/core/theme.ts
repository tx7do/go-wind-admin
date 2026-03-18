import {useState, useEffect} from 'react';

import {StorageManager} from '@/caches/storage-manager';
import {appNamespace} from '@/caches';

import type {ThemeMode} from '../types';

const storage = new StorageManager({prefix: `${appNamespace}-theme`, storageType: 'localStorage'});

/**
 * 主题管理 Model
 * 提供主题模式的切换和持久化
 */
export default function ThemeModel() {
  const [mode, setMode] = useState<ThemeMode>(() => {
    // 服务端环境返回默认值
    if (typeof window === 'undefined') {
      return 'system';
    }

    // 客户端环境从 localStorage 读取
    const stored = storage.getItem<ThemeMode>('mode', null);
    return stored || 'system';
  });

  // 持久化主题设置
  useEffect(() => {
    if (typeof window !== 'undefined') {
      storage.setItem('mode', mode);
    }
  }, [mode]);

  return {
    mode,
    setMode,
  };
}
