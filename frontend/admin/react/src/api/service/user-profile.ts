import { requestApi } from '@/core';
import {
  createUserProfileServiceClient,
  type identityservicev1_BindContactRequest,
  type identityservicev1_ChangePasswordRequest,
  type identityservicev1_UpdateUserRequest,
  type identityservicev1_UploadAvatarRequest,
  type identityservicev1_User,
  type identityservicev1_VerifyContactRequest,
} from '@/api/generated/admin/service/v1';

let _instance: ReturnType<typeof createUserProfileServiceClient> | null = null;

export function getUserProfileService() {
  if (!_instance) {
    _instance = createUserProfileServiceClient(requestApi);
  }
  return _instance;
}

/**
 * 获取当前用户
 */
export async function getMe(): Promise<identityservicev1_User | null> {
  try {
    return await getUserProfileService().GetUser({});
  } catch (error) {
    console.error('getMe failed:', error);
    return null;
  }
}

/**
 * 更新当前用户
 */
export async function updateMyUserInfo(request: identityservicev1_UpdateUserRequest) {
  return await getUserProfileService().UpdateUser(request);
}

/**
 * 修改用户密码
 */
export async function changeMyPassword(request: identityservicev1_ChangePasswordRequest) {
  return await getUserProfileService().ChangePassword(request);
}

export async function uploadMyAvatar(request: identityservicev1_UploadAvatarRequest) {
  return await getUserProfileService().UploadAvatar(request);
}

/**
 * 上传用户头像（Base64）
 * @param imageBase64 图片Base64字符串
 */
export async function uploadMyAvatarBase64(imageBase64: string) {
  return await uploadMyAvatar({
    imageBase64,
  });
}

/**
 * 上传用户头像（图片URL）
 * @param imageUrl 图片URL
 */
export async function uploadMyAvatarUrl(imageUrl: string) {
  return await uploadMyAvatar({
    imageUrl,
  });
}

/**
 * 删除用户头像
 */
export async function deleteMyAvatar() {
  return await getUserProfileService().DeleteAvatar({});
}

export async function bindMyContact(request: identityservicev1_BindContactRequest) {
  return await getUserProfileService().BindContact(request);
}

/**
 * 绑定手机号
 * @param phone 手机号
 * @param code 验证码
 */
export async function bindMyPhone(phone: string, code: string) {
  return await bindMyContact({
    phone: { phone, code },
  });
}

/**
 * 绑定邮箱
 * @param email 邮箱
 * @param verificationCode 验证码
 */
export async function bindMyEmail(email: string, verificationCode: string) {
  return await bindMyContact({
    email: { email, verificationCode },
  });
}

export async function verifyMyContact(request: identityservicev1_VerifyContactRequest) {
  return await getUserProfileService().VerifyContact(request);
}

/**
 * 验证手机号
 * @param phone 手机号码，带国家码
 * @param code 短信验证码
 * @param verificationId 服务端生成的验证码会话ID
 */
export async function verifyMyPhone(phone: string, code: string, verificationId?: string) {
  return await verifyMyContact({
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
export async function verifyMyEmail(email: string, code: string, verificationId?: string) {
  return await verifyMyContact({
    email: { email, code },
    verificationId,
  });
}
