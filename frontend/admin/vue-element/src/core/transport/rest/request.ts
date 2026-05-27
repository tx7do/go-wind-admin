import { requestClient } from "./rest-client";

export function requestApi({
  path,
  method,
  body,
}: {
  path: string;
  method: string;
  body: null | string;
}) {
  return requestClient.request(path, {
    method,
    data: body,
  } as never);
}
