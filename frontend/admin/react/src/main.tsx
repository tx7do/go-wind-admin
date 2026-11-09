import React from 'react';
import ReactDOM from 'react-dom/client';
import { App as AntdApp } from 'antd';

import { QueryClientProvider } from '@tanstack/react-query';

// Devtools 仅在开发环境导入，生产构建时自动 tree-shake
const ReactQueryDevtools = import.meta.env.DEV
  ? await import('@tanstack/react-query-devtools').then((m) => m.ReactQueryDevtools)
  : () => null;

import { ThemeProvider } from '@/core/preferences/components/ThemeProvider';
import { queryClient } from './core';
import { bootstrap } from '@/bootstrap.ts';

// 样式
import 'nprogress/nprogress.css';

import './styles/global.css';
import './styles/tailwind.css';
import './styles/skeleton-force-dark.css';
import './styles/pro-layout-overrides.css';
import './styles/pro-components-dark.css';
import './styles/semantic-text.css';
import './styles/list-table.css';

import './index.css';
import App from './app.tsx';

// 执行全局初始化
bootstrap()
  .then(() => {
    const root = ReactDOM.createRoot(document.getElementById('root')!);

    root.render(
      <React.StrictMode>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider>
            <AntdApp>
              <App />
            </AntdApp>
          </ThemeProvider>

          <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
      </React.StrictMode>,
    );
  })
  .catch((e) => {
    // bootstrap 失败（如 i18n 资源加载失败）时 React 永远不会挂载，页面会停在
    // 启动加载页上。必须带出原始错误（不吞错），并在加载页给出可见提示避免无限空白。
    // 此时 i18n 不一定可用，提示用静态双语短句。
    console.error('[Bootstrap] 应用启动初始化失败:', e);
    const title = document.querySelector('.app-loading-title');
    if (title) {
      title.textContent = '应用加载失败，请刷新重试 / Failed to load, please refresh';
    }
    const spinner = document.querySelector('.app-loading-spinner');
    if (spinner) {
      (spinner as HTMLElement).style.display = 'none';
    }
  });
