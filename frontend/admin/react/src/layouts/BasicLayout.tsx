import type {ProLayoutProps} from '@ant-design/pro-components';
import {ProLayout} from '@ant-design/pro-components';
import {history, useIntl, useModel} from '@umijs/max';
import React from 'react';

import {AvatarDropdown, Footer, Question, SelectLang} from '@/components';
import defaultSettings from '../../config/defaultSettings';

export interface BasicLayoutProps extends ProLayoutProps {
  children?: React.ReactNode;
  route?: any;
}

/**
 * BasicLayout 基础布局组件
 * 参考 Vue 版本的 VbenAdminLayout 实现
 * 支持多种导航模式：侧边栏、顶部、混合导航
 */
const BasicLayout: React.FC<BasicLayoutProps> = (props) => {
  const {children} = props;
  const intl = useIntl();
  const {initialState, setInitialState} = useModel('@@initialState');

  return (
    <ProLayout
      {...defaultSettings}
      {...props}
      title={intl.formatMessage({
        id: 'menu.dashboard',
        defaultMessage: 'Go Wind Admin',
      })}
      logo={
        <img alt="logo" src="/logo.svg" style={{height: 32}}/>
      }
      layout="mix"
      mode="inline"
      theme={'light' as const}
      contentStyle={{margin: 0, padding: 24}}
      siderMenuType="group"
      fixSiderbar
      fixedHeader
      splitMenus
      waterMarkProps={{
        content: initialState?.currentUser?.name,
      }}
      avatarProps={{
        src: initialState?.currentUser?.avatar,
        size: 'small',
        title: <span>{initialState?.currentUser?.name}</span>,
        render: (_, avatarChildren) => {
          return <AvatarDropdown>{avatarChildren}</AvatarDropdown>;
        },
      }}
      actionsRender={() => [
        <Question key="question"/>,
        <SelectLang key="selectLang"/>,
      ]}
      footerRender={() => <Footer/>}
      onPageChange={(routeConfig) => {
        // 页面切换时的权限检查
        const {location} = history;
        if (!initialState?.currentUser && location.pathname !== '/user/login') {
          history.push('/user/login');
        }
      }}
      menuDataRender={() => {
        // TODO: 渲染菜单数据
        return props.route?.routes || [];
      }}
      menuItemRender={(item, dom) => (
        <div
          onClick={() => {
            if (item.path) {
              history.push(item.path);
            }
          }}
        >
          {dom}
        </div>
      )}
    >
      {children}
    </ProLayout>
  );
};

export default BasicLayout;
