import type { AxiosRequestConfig } from "axios";

import type { RequestResponse } from "../types";
import { RequestClient } from "..";

class FileDownloader {
  private client: RequestClient;

  constructor(client: RequestClient) {
    this.client = client;
  }

  public async download(url: string, config?: AxiosRequestConfig): Promise<RequestResponse<Blob>> {
    const finalConfig: AxiosRequestConfig = {
      ...config,
      responseType: "blob",
    };

    return await this.client.get<RequestResponse<Blob>>(url, finalConfig);
  }
}

export { FileDownloader };
