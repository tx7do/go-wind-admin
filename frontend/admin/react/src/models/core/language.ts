import {useState, useEffect} from 'react';

import {StorageManager} from '@/caches/storage-manager';
import {appNamespace} from '@/caches';

const storage = new StorageManager({prefix: `${appNamespace}-lang`, storageType: 'localStorage'});

/**
 * 语言管理 Model
 * 提供国际化语言的切换和持久化
 */
export default function LanguageModel() {
  const [locale, setLocale] = useState<string>(() => {
    // 服务端环境返回默认值
    if (typeof window === 'undefined') {
      return 'zh-CN';
    }

    // 客户端环境从 localStorage 读取
    const stored = storage.getItem<string>('locale', null);
    return stored || 'zh-CN';
  });

  // 持久化语言设置
  useEffect(() => {
    if (typeof window !== 'undefined') {
      storage.setItem('locale', locale);
    }
  }, [locale]);

  return {
    locale,
    setLocale,
  };
}
