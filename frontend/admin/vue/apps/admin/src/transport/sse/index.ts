import { SSEClient } from './sse_client';

export * from './sse_client';
export * from './types';

export const globalSSEClient = new SSEClient({
  url: '/api/sse',
  withCredentials: false,
  reconnectDelay: 5000,
  autoParseJson: true,
});
