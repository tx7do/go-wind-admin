import {LinkOutlined} from '@ant-design/icons';
import {SettingDrawer} from '@ant-design/pro-components';
import type {RunTimeLayoutConfig} from '@umijs/max';
import {history, Link} from '@umijs/max';
import React from 'react';
import {AvatarDropdown, AvatarName, Footer, Question, SelectLang, ThemeToggle} from '@/components';

const isDev = process.env.NODE_ENV === 'development';
const loginPath = '/user/login';

/**
 * 布局配置组件
 * 负责渲染 ProLayout 的各项配置
 * @doc https://procomponents.ant.design/components/layout
 */
export const layout: RunTimeLayoutConfig = ({initialState, setInitialState}) => {
  return {
    // 顶部操作区域渲染
    actionsRender: () => [
      <ThemeToggle key="theme" />,
      <Question key="question" />,
      <SelectLang key="SelectLang" />,
    ],

    // 头像配置
    avatarProps: {
      src: initialState?.currentUser?.avatar,
      title: <AvatarName/>,
      render: (_, avatarChildren) => {
        return <AvatarDropdown>{avatarChildren}</AvatarDropdown>;
      },
    },

    // 水印配置
    waterMarkProps: {
      content: initialState?.currentUser?.name,
    },

    // 页脚渲染
    footerRender: () => <Footer/>,

    // 页面切换时的权限检查
    onPageChange: () => {
      const {location} = history;
      // 如果没有登录，重定向到登录页
      if (!initialState?.currentUser && location.pathname !== loginPath) {
        history.push(loginPath);
      }
    },

    // 背景图片配置
    bgLayoutImgList: [
      {
        src: 'https://mdn.alipayobjects.com/yuyan_qk0oxh/afts/img/D2LWSqNny4sAAAAAAAAAAAAAFl94AQBr',
        left: 85,
        bottom: 100,
        height: '303px',
      },
      {
        src: 'https://mdn.alipayobjects.com/yunan_qk0oxh/afts/img/C2TWRpJpiC0AAAAAAAAAAAAAFl94AQBr',
        bottom: -68,
        right: -45,
        height: '303px',
      },
      {
        src: 'https://mdn.alipayobjects.com/yuyan_qk0oxh/afts/img/F6vSTbj8KpYAAAAAAAAAAAAAFl94AQBr',
        bottom: 0,
        left: 0,
        width: '331px',
      },
    ],

    // 链接列表 (仅开发环境显示)
    links: isDev
      ? [
        <Link key="openapi" to="/umi/plugin/openapi" target="_blank">
          <LinkOutlined/>
          <span>OpenAPI 文档</span>
        </Link>,
      ]
      : [],

    // 菜单头部渲染
    menuHeaderRender: undefined,

    // 自定义 403 页面 (可选)
    // unAccessible: <div>unAccessible</div>,

    // 子节点渲染 (增加 loading 状态和设置面板)
    childrenRender: (children) => {
      return (
        <>
          {children}
          {isDev && (
            <SettingDrawer
              disableUrlParams
              enableDarkTheme
              settings={initialState?.settings}
              onSettingChange={(settings) => {
                setInitialState((preInitialState) => ({
                  ...preInitialState,
                  settings,
                }));
              }}
            />
          )}
        </>
      );
    },

    // 合并用户设置
    ...initialState?.settings,
  };
};
