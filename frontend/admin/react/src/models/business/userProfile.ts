import {useState, useCallback} from 'react';
import {
  createUserProfileServiceClient,
} from '@/api/admin/service/v1';
import {requestApi} from '@/transport/rest';
import type {IUser} from '../types';

/**
 * 用户档案 Model
 * 管理用户档案信息的获取、更新和清除
 */
export default function UserProfileModel() {
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [userProfile, setUserProfile] = useState<IUser | null>(null);

  // 创建服务客户端
  const userProfileService = createUserProfileServiceClient(requestApi);

  /**
   * 获取用户档案信息
   */
  const fetchUserProfile = useCallback(async (): Promise<IUser | null> => {
    try {
      setLoading(true);
      setError(null);
      
      const profile = await userProfileService.GetUser({});
      setUserProfile(profile as unknown as IUser);
      
      return profile as unknown as IUser;
    } catch (err: any) {
      setError(err?.message || '获取用户信息失败');
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  /**
   * 清除用户档案信息
   */
  const clearUserProfile = useCallback(() => {
    setUserProfile(null);
    setError(null);
  }, []);

  /**
   * 清除错误
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  /**
   * 重置状态
   */
  const resetState = useCallback(() => {
    setLoading(false);
    setError(null);
    setUserProfile(null);
  }, []);

  return {
    // 状态
    loading,
    error,
    userProfile,

    // 方法
    fetchUserProfile,
    clearUserProfile,
    clearError,
    resetState,
  };
}