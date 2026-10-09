import { RequestClient } from './request-client';

export function requestApi({
  path,
  method,
  body,
  headers,
}: {
  body: null | string;
  headers?: Record<string, string>;
  method: string;
  path: string;
}) {
  return RequestClient.getInstance().request(path, {
    method,
    data: body,
    headers,
  } as never);
}
