import type {
  SSEClientConfig,
  SSEConnectionStatus,
  SSEEventHandler,
  SSEEventName,
} from '#/transport/sse/types';

/**
 * SSE客户端
 */
export class SSEClient {
  private config: Required<SSEClientConfig>;
  private eventSource: EventSource | null = null;
  // 存储事件监听器（键：事件名，值：回调数组）
  private handlers = new Map<SSEEventName, SSEEventHandler[]>();
  private status: SSEConnectionStatus = 'disconnected';

  constructor(config: SSEClientConfig) {
    // 合并默认配置
    this.config = {
      withCredentials: false,
      reconnectDelay: 3000,
      autoParseJson: true,
      ...config,
    };
  }

  /**
   * 解析数据（自动处理 JSON 格式）
   * @param rawData
   * @private
   */
  private parseData(rawData: string): unknown {
    if (!this.config.autoParseJson) {
      return rawData;
    }
    try {
      return JSON.parse(rawData);
    } catch {
      // 非 JSON 格式数据直接返回原始字符串
      return rawData;
    }
  }

  /**
   * 触发事件监听器
   * @param eventName
   * @param data
   * @param event
   * @private
   */
  private triggerHandler<T = unknown>(
    eventName: SSEEventName,
    data: T,
    event: Event,
  ): void {
    const handlers = this.handlers.get(eventName);
    if (handlers) {
      handlers.forEach((handler) => {
        try {
          handler(data, event as MessageEvent);
        } catch (error) {
          console.error(`SSE 事件 ${eventName} 处理失败:`, error);
        }
      });
    }
  }

  /**
   * 关闭 SSE 连接
   */
  close(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.status = 'disconnected';
  }

  /**
   * 建立 SSE 连接
   * @param url 可选连接 URL，默认为配置中的 URL
   */
  connect(url?: string): Error | null {
    if (this.status === 'connected' || this.status === 'connecting') {
      console.warn('SSE 连接已存在或正在建立中');
      return new Error('SSE 连接已存在或正在建立中');
    }

    if (url === undefined || url === '') {
      url = this.config.url;
    }

    this.status = 'connecting';
    this.eventSource = new EventSource(url, {
      withCredentials: this.config.withCredentials,
    });

    // 监听连接成功事件
    this.eventSource.addEventListener('open', (event) => {
      this.status = 'connected';
      this.triggerHandler('open', undefined, event);
    });

    // 监听默认消息事件（服务器未指定 event 字段时触发）
    this.eventSource.addEventListener('message', (event: MessageEvent) => {
      const data = this.parseData(event.data);
      this.triggerHandler('message', data, event);
    });

    // 监听错误事件（连接断开、网络异常等）
    this.eventSource.addEventListener('error', (event: Event) => {
      this.status = 'error';
      this.triggerHandler('error', undefined, event as MessageEvent);

      // 连接关闭时尝试重连（排除手动关闭的情况）
      if (this.eventSource?.readyState === EventSource.CLOSED) {
        this.status = 'disconnected';
        setTimeout(() => this.connect(), this.config.reconnectDelay);
      }
    });

    return null;
  }

  /**
   * 获取当前连接状态
   */
  getStatus(): SSEConnectionStatus {
    return this.status;
  }

  /**
   * 移除事件监听器
   * @param eventName
   * @param handler
   */
  off(eventName: SSEEventName, handler?: SSEEventHandler): void {
    const handlers = this.handlers.get(eventName);
    if (!handlers) return;

    if (handler) {
      // 移除指定回调
      this.handlers.set(
        eventName,
        handlers.filter((h) => h !== handler),
      );
    } else {
      // 移除所有回调
      this.handlers.delete(eventName);
    }
  }

  /**
   * 注册事件监听器（支持泛型指定数据类型）
   * @param eventName
   * @param handler
   */
  on<T = unknown>(eventName: SSEEventName, handler: SSEEventHandler<T>): void {
    if (!this.handlers.has(eventName)) {
      this.handlers.set(eventName, []);
    }
    // 存储回调（类型断言确保类型安全）
    this.handlers.get(eventName)?.push(handler as SSEEventHandler);

    // 对自定义事件（非 open/error/message），需要额外注册到 EventSource
    if (!['error', 'message', 'open'].includes(eventName) && this.eventSource) {
      this.eventSource.addEventListener(eventName, (event) => {
        const data = this.parseData((event as MessageEvent).data);
        this.triggerHandler(eventName, data, event as MessageEvent);
      });
    }
  }
}
