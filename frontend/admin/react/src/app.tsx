import type {Settings as LayoutSettings} from '@ant-design/pro-components';
import '@ant-design/v5-patch-for-react-19';
import type {RunTimeLayoutConfig} from '@umijs/max';
import {history} from '@umijs/max';

import defaultSettings from '../config/defaultSettings';
import {layout as proLayoutConfig} from './layouts/ProLayoutConfig';
import type {IUser} from "@/models/types";
import {useUserProfileModel} from '@/models';

const loginPath = '/user/login';

/**
 * @see https://umijs.org/docs/api/runtime-config#getinitialstate
 * */
export async function getInitialState(): Promise<{
  settings?: Partial<LayoutSettings>;
  currentUser?: IUser;
  loading?: boolean;
  fetchUserInfo?: () => Promise<IUser | undefined>;
}> {
  const fetchUserInfo = async () => {
    try {
      // 使用 UserProfileModel 获取用户信息
      const userProfileModel = useUserProfileModel();
      const user = await userProfileModel.fetchUserProfile();
      return user || undefined;
    } catch (_error) {
      history.push(loginPath);
    }
    return undefined;
  };
  // 如果不是登录页面，执行
  const {location} = history;
  if (
    ![loginPath, '/user/register', '/user/register-result'].includes(
      location.pathname,
    )
  ) {
    const currentUser = await fetchUserInfo();
    return {
      fetchUserInfo,
      currentUser,
      settings: defaultSettings as Partial<LayoutSettings>,
    };
  }
  return {
    fetchUserInfo,
    settings: defaultSettings as Partial<LayoutSettings>,
  };
}

/**
 * @name layout 配置
 * @description ProLayout 的运行时配置
 * @doc https://umijs.org/docs/max/layout
 */
export const layout: RunTimeLayoutConfig = proLayoutConfig;
