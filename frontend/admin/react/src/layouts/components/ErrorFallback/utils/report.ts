interface ErrorReportPayload {
  message: string;
  stack?: string;
  componentStack?: string;
  url: string;
  userAgent: string;
  timestamp: number;
}

/**
 * 错误上报收集端点（构建期注入，见 .env 的 VITE_ERROR_REPORT_ENDPOINT）。
 * 部署侧把该地址指到任意能收 POST JSON 的收集器即可（自研网关、gossip 类日志服务等）；
 * 留空表示未接入监控，只走控制台输出，不发起网络请求。
 */
const REPORT_ENDPOINT: string = import.meta.env.VITE_ERROR_REPORT_ENDPOINT ?? '';

/**
 * 上报错误到监控系统（未配置端点时仅控制台可见，见 .env 注释）。
 * 上报失败只记日志不打断 UI——错误兜底页自身不能再抛错。
 */
export const reportError = (error: Error, componentStack?: string) => {
  if (import.meta.env.PROD) {
    const payload: ErrorReportPayload = {
      message: error.message,
      stack: error.stack,
      componentStack,
      url: window.location.href,
      userAgent: navigator.userAgent,
      timestamp: Date.now(),
    };

    if (!REPORT_ENDPOINT) {
      console.debug('[ErrorReport] 未配置上报端点（VITE_ERROR_REPORT_ENDPOINT），payload:', payload);
      return;
    }

    fetch(REPORT_ENDPOINT, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }).catch((err) => {
      console.error('[ErrorReport] 错误上报失败:', err);
    });
  }
};
