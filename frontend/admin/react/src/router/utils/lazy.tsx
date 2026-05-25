import { Suspense, lazy, type ComponentType } from 'react';
import { Spin } from 'antd';

/**
 * 创建懒加载路由元素
 * 自动包裹 Suspense + Spin fallback
 *
 * @param loader Vite 动态导入函数，如 () => import('@/pages/dashboard')
 * @returns 可直接赋值给 route.element 的 ReactNode
 */
export const createLazyRoute = (loader: () => Promise<{ default: ComponentType<any> }>) => {
  const LazyComponent = lazy(loader);

  return (
    <Suspense fallback={<Spin size="large" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }} />}>
      <LazyComponent />
    </Suspense>
  );
};
