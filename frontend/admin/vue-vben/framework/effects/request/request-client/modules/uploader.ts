import type { AxiosRequestConfig, AxiosResponse } from 'axios';

import type { RequestClient } from '../request-client';

class FileUploader {
  private client: RequestClient;

  constructor(client: RequestClient) {
    this.client = client;
  }

  public async upload(
    url: string,
    data: { file: Blob | File } & Record<string, any>,
    config?: AxiosRequestConfig,
  ): Promise<AxiosResponse> {
    const formData = new FormData();

    Object.entries(data).forEach(([key, value]) => {
      formData.append(key, value);
    });

    // 注意：不要手动设置 Content-Type 为 'multipart/form-data'。
    // 浏览器在发送 FormData 时会自动生成带 boundary 的
    // 'multipart/form-data; boundary=----WebKitFormBoundaryXXX'，
    // 手动写死会导致 boundary 丢失，后端无法解析请求体。
    // 这里显式删除调用方可能误传的 Content-Type，确保由浏览器自动设置。
    const headers: Record<string, any> = { ...config?.headers };
    delete headers['Content-Type'];
    delete headers['content-type'];
    const finalConfig: AxiosRequestConfig = { ...config, headers };

    return this.client.post(url, formData, finalConfig);
  }
}

export { FileUploader };
