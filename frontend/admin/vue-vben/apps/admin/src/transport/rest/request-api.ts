import { RequestClient } from './request-client';

export function requestApi({
  path,
  method,
  body,
}: {
  body: null | string;
  method: string;
  path: string;
}) {
  return RequestClient.getInstance().request(path, {
    method,
    data: body,
  } as never);
}
