/**
 * Request Token 配置
 *
 * 用于在应用初始化时设置 AccessModel 的 token获取方法
 */

import {setGetTokenCallback} from '@/transport/rest';
import React, {useEffect} from 'react';

/**
 * AppInitializer 内部组件
 * 使用 useModel 来获取 access token
 */
function AppInitializerInner({children}: { children: React.ReactNode }) {
  // 延迟导入 useModel，确保在正确的上下文中使用
  const {useModel} = require('@umijs/max');
  const access = useModel('auth.access');

  useEffect(() => {
    // 设置获取 Token 的回调函数
    setGetTokenCallback(() => {
      return access.accessToken?.value || null;
    });

    console.log('Request client initialized with access token getter');
  }, [access.accessToken]);

  return <>{children}</>;
}

/**
 * AppInitializer 主组件
 * 包装整个应用，设置 token获取回调
 */
export function AppInitializer({children}: { children: React.ReactNode }) {
  return <AppInitializerInner>{children}</AppInitializerInner>;
}
