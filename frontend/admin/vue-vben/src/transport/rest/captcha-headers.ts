/**
 * 临时存放登录请求需要携带的验证码 Header。
 *
 * 后端登录通过 HTTP Header（X-Captcha-Id / X-Captcha-Value）校验验证码，
 * 而生成的 ApiClient 无法携带 per-request header。此处用一个模块级持有器
 * 作为旁路：由 auth store 在调用 loginMutation.execute 前设置，
 * transport.unary 消费后即清除（一次性）。
 */
let pendingCaptchaHeaders: Record<string, string> | null = null;

export function setCaptchaHeaders(id: string, value: string) {
  pendingCaptchaHeaders = {
    'X-Captcha-Id': id,
    'X-Captcha-Value': value,
  };
}

export function consumeCaptchaHeaders(): Record<string, string> | null {
  const headers = pendingCaptchaHeaders;
  pendingCaptchaHeaders = null;
  return headers;
}
