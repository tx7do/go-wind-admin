/**
 * SSE 事件类型（区分默认事件和自定义事件）
 */
export type SSEEventName = 'error' | 'message' | 'open' | string; // 'message' 是默认事件，其他为自定义事件

/**
 * SSE 事件回调函数类型（T 为数据类型）
 */
export type SSEEventHandler<T = unknown> = (
  data: T,
  event: MessageEvent,
) => void;

/**
 * SSE 连接实现方式
 */
export type SSETransport = 'event-source' | 'fetch-event-source';

/**
 * SSE 客户端配置
 */
export interface SSEClientConfig {
  url: string; // SSE 服务器端点 URL
  withCredentials?: boolean; // 是否携带跨域凭证（cookie 等）
  reconnectDelay?: number; // 断开后重连延迟（毫秒，默认 3000）
  autoParseJson?: boolean; // 是否自动解析 JSON 格式的数据（默认 true）
  headers?: Record<string, string>; // 可选的 HTTP 请求头
  transport?: SSETransport; // 连接实现方式（默认 fetch-event-source）
}

/**
 * SSE 连接状态
 */
export type SSEConnectionStatus =
  | 'connected'
  | 'connecting'
  | 'disconnected'
  | 'error';
