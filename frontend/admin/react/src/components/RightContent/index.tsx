import {MoonOutlined, QuestionCircleOutlined, SunOutlined} from '@ant-design/icons';
import {SelectLang as UmiSelectLang, useModel} from '@umijs/max';
import type {MenuProps} from 'antd';
import {Dropdown, Space, Tooltip} from 'antd';
import React, {useMemo, useCallback} from 'react';

export type SiderTheme = 'light' | 'dark';

/**
 * 主题切换按钮组件
 */
export const ThemeToggle: React.FC = () => {
  const {theme, setMode} = useModel('core.theme');

  // 使用 useMemo 缓存主题值，避免不必要的重新渲染
  const currentTheme = useMemo(() => theme || 'light', [theme]);

  const handleToggle = useCallback(() => {
    setMode(currentTheme === 'dark' ? 'light' : 'dark');
  }, [currentTheme, setMode]);

  return (
    <Tooltip title={currentTheme === 'dark' ? '切换到亮色模式' : '切换到暗色模式'}>
      <a
        onClick={handleToggle}
        style={{
          display: 'inline-flex',
          padding: '4px',
          fontSize: '18px',
          color: 'inherit',
          cursor: 'pointer',
        }}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            handleToggle();
          }
        }}
      >
        <Space>
          {currentTheme === 'dark' ? <SunOutlined /> : <MoonOutlined />}
        </Space>
      </a>
    </Tooltip>
  );
};

export const SelectLang: React.FC = () => {
  return (
    <UmiSelectLang
      style={{
        padding: 4,
      }}
    />
  );
};

export const Question: React.FC = () => {
  return (
    <a
      href="https://pro.ant.design/docs/getting-started"
      target="_blank"
      rel="noreferrer"
      style={{
        display: 'inline-flex',
        padding: '4px',
        fontSize: '18px',
        color: 'inherit',
      }}
    >
      <QuestionCircleOutlined />
    </a>
  );
};
