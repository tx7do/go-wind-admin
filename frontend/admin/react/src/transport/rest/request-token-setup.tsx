/**
 * Request Token 配置示例
 * 
 * 展示如何在应用初始化时设置 AccessModel 的 token 获取方法
 */

import {setGetTokenCallback} from '@/transport/rest';
import {useModel} from '@umijs/max';
import React, {useEffect} from 'react';

/**
 * 方式一：在应用入口组件中设置（推荐）
 */
export function AppInitializer({children}: {children: React.ReactNode}) {
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
 * 方式二：在 getInitialStates 或 layout 中设置
 * 注意：这需要配合 useModel 使用
 */
export function setupRequestTokenGetter() {
  // 由于 getInitialState 和 layout 不是 React 组件，无法直接使用 useModel
  // 建议在根组件中调用 setGetTokenCallback
  
  // 示例代码（在实际使用时需要根据项目结构调整）：
  /*
  import {useAccessStore} from '@/store/core/access/hooks';
  
  export function RootComponent() {
    const accessStore = useAccessStore();
    
    useEffect(() => {
      setGetTokenCallback(() => {
        return accessStore.access.accessToken?.value || null;
      });
    }, [accessStore.access.accessToken]);
    
    return <App />;
  }
  */
}

/**
 * 方式三：直接在 request 配置中使用（如果不使用 models）
 * 在 src/app.tsx 的 request 配置中
 */
export const requestConfigExample = {
  // baseURL: 'your-api-url',
  requestInterceptors: [
    (url, options) => {
      // 直接从 localStorage 读取（不推荐，但可以使用）
      const tokenKey = 'app-access-token';
      const tokenItem = localStorage.getItem(tokenKey);
      
      if (tokenItem) {
        try {
          const tokenData = JSON.parse(tokenItem);
          if (tokenData.value) {
            return {
              ...options,
              headers: {
                ...options.headers,
                Authorization: `Bearer ${tokenData.value}`,
              },
            };
          }
        } catch (e) {
          console.error('Failed to parse token:', e);
        }
      }
      
      return {...options};
    },
  ],
};

export default AppInitializer;
