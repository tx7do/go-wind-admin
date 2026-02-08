import { defineStore } from 'pinia';

import { createUserProfileServiceClient } from '#/generated/api/admin/service/v1';
import { makeUpdateMask, omit } from '#/utils/query';
import { requestClientRequestHandler } from '#/utils/request';

export const useUserProfileStore = defineStore('user-profile', () => {
  const userProfileService = createUserProfileServiceClient(
    requestClientRequestHandler,
  );

  /**
   * 获取当前用户
   */
  async function getMe() {
    try {
      return await userProfileService.GetUser({});
    } catch (error) {
      console.error('getMe failed:', error);
      return null;
    }
  }

  /**
   * 更新当前用户
   */
  async function updateUser(values: Record<string, any> = {}) {
    const password = values.password ?? null;
    const cleaned = omit(values, 'password');
    const updateMask = makeUpdateMask(Object.keys(cleaned ?? []));
    return await userProfileService.UpdateUser({
      // @ts-ignore proto generated code is error.
      data: {
        ...cleaned,
      },
      password,
      // @ts-ignore proto generated code is error.
      updateMask,
    });
  }

  /**
   * 修改用户密码
   * @param oldPassword 用户旧密码
   * @param newPassword 用户新密码
   */
  async function changePassword(oldPassword: string, newPassword: string) {
    return await userProfileService.ChangePassword({
      oldPassword,
      newPassword,
    });
  }

  /**
   * 上传用户头像（Base64）
   * @param imageBase64 图片Base64字符串
   */
  async function uploadAvatarBase64(imageBase64: string) {
    return await userProfileService.UploadAvatar({
      imageBase64,
    });
  }

  /**
   * 上传用户头像（图片URL）
   * @param imageUrl 图片URL
   */
  async function uploadAvatarUrl(imageUrl: string) {
    return await userProfileService.UploadAvatar({
      imageUrl,
    });
  }

  /**
   * 删除用户头像
   */
  async function deleteAvatar() {
    return await userProfileService.DeleteAvatar({});
  }

  /**
   * 绑定手机号
   * @param phone 手机号
   * @param code 验证码
   */
  async function bindPhone(phone: string, code: string) {
    return await userProfileService.BindContact({
      phone: { phone, code },
    });
  }

  /**
   * 绑定邮箱
   * @param email 邮箱
   * @param verificationCode 验证码
   */
  async function bindEmail(email: string, verificationCode: string) {
    return await userProfileService.BindContact({
      email: { email, verificationCode },
    });
  }

  /**
   * 验证手机号
   * @param phone 手机号码，带国家码
   * @param code 短信验证码
   * @param verificationId 服务端生成的验证码会话ID
   */
  async function verifyPhone(
    phone: string,
    code: string,
    verificationId?: string,
  ) {
    return await userProfileService.VerifyContact({
      phone: { phone, code },
      verificationId,
    });
  }

  /**
   * 验证邮箱
   * @param email 邮箱地址
   * @param code 邮箱验证码
   * @param verificationId 服务端生成的验证码会话ID
   */
  async function verifyEmail(
    email: string,
    code: string,
    verificationId?: string,
  ) {
    return await userProfileService.VerifyContact({
      email: { email, code },
      verificationId,
    });
  }

  function $reset() {}

  return {
    $reset,
    getMe,
    updateUser,
    changePassword,
    uploadAvatarBase64,
    uploadAvatarUrl,
    deleteAvatar,
    bindPhone,
    bindEmail,
    verifyPhone,
    verifyEmail,
  };
});
