import type { EventSourceMessage } from '@microsoft/fetch-event-source/lib/cjs/parse';

import type {
  SSEClientConfig,
  SSEConnectionStatus,
  SSEEventHandler,
  SSEEventName,
  SSETransport,
} from './types';

import { fetchEventSource } from '@microsoft/fetch-event-source';

/**
 * SSE客户端
 */
export class SSEClient {
  private abortController: AbortController | null = null;
  private config: Required<SSEClientConfig>;
  private eventSource: EventSource | null = null;
  // 存储事件监听器（键：事件名，值：回调数组）
  private handlers = new Map<SSEEventName, SSEEventHandler[]>();
  private status: SSEConnectionStatus = 'disconnected';
  private readonly transport: SSETransport;

  constructor(config: SSEClientConfig) {
    // 合并默认配置
    this.config = {
      withCredentials: false,
      reconnectDelay: 3000,
      autoParseJson: true,
      headers: {},
      transport: 'fetch-event-source',
      ...config,
    };
    this.transport = this.config.transport;
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
    event: Event | EventSourceMessage,
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
   * 通过 EventSource 建立 SSE 连接
   * @param url 连接 URL
   * @private
   */
  _connectByEventSource(url: string): void {
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
  }

  /**
   * 通过 fetchEventSource 建立 SSE 连接（支持更灵活的配置和控制）
   * @param url 连接 URL
   * @private
   */
  async _connectByFetchEventSource(url: string): Promise<void> {
    // 用于手动关闭
    this.abortController = new AbortController();

    await fetchEventSource(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
        ...this.config.headers,
      },
      credentials: this.config.withCredentials ? 'include' : 'same-origin',
      signal: this.abortController.signal,
      openWhenHidden: true, // 后台页保持连接（可选）

      // 连接成功
      onopen: async (response) => {
        if (response.ok) {
          this.status = 'connected';
          this.triggerHandler('open', undefined, new Event('open'));
        } else {
          throw new Error(`SSE 连接失败 ${response.status}`);
        }
      },

      // 接收消息
      onmessage: (event: EventSourceMessage) => {
        const data = this.parseData(event.data);

        // 自定义事件名（event: xxx）
        if (event.event && event.event !== 'message') {
          this.triggerHandler(event.event as SSEEventName, data, event);
        }

        // 默认 message 事件
        this.triggerHandler('message', data, event);
      },

      // 错误 & 重连
      onerror: (err) => {
        this.status = 'error';
        this.triggerHandler('error', err, new Event('error'));

        // 手动关闭不重连
        if (this.abortController?.signal.aborted) {
          throw err;
        }

        console.log(`[SSE] ${this.config.reconnectDelay}ms 后重试...`);
      },

      // 关闭
      onclose: () => {
        this.status = 'disconnected';
        throw new Error('SSE 连接关闭，准备重连');
      },
    });
  }

  /**
   * 关闭 SSE 连接
   */
  close(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
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

    const targetUrl = url === undefined || url === '' ? this.config.url : url;

    this.status = 'connecting';
    if (this.transport === 'event-source') {
      this._connectByEventSource(targetUrl);
    } else {
      this._connectByFetchEventSource(targetUrl);
    }

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

  /**
   * 关闭当前连接并使用新的 URL / headers 重新连接
   * 适用于 token 刷新后需要携带新凭证的场景
   * @param url 可选的新连接 URL，默认沿用配置中的 URL
   */
  reconnect(url?: string): void {
    this.close();
    this.connect(url);
  }

  /**
   * 动态设置/更新请求头（不会影响正在进行中的连接，下次 connect 时生效）
   * 注意：EventSource 模式不支持自定义请求头，此方法仅对 fetch-event-source 模式生效
   * @param headers 键值对，会浅合并到当前 headers 配置中
   */
  setHeaders(headers: Record<string, string>): void {
    this.config.headers = { ...this.config.headers, ...headers };
  }
}
