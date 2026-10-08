import type { AxiosRequestConfig, AxiosResponse } from 'axios';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { FileUploader } from './uploader';

describe('fileUploader', () => {
  let fileUploader: FileUploader;
  // Mock the AxiosInstance
  const mockAxiosInstance = {
    post: vi.fn(),
  } as any;

  beforeEach(() => {
    fileUploader = new FileUploader(mockAxiosInstance);
  });

  it('should create an instance of FileUploader', () => {
    expect(fileUploader).toBeInstanceOf(FileUploader);
  });

  it('should upload a file and return the response', async () => {
    const url = 'https://example.com/upload';
    const file = new File(['file content'], 'test.txt', { type: 'text/plain' });
    const mockResponse: AxiosResponse = {
      config: {} as any,
      data: { success: true },
      headers: {},
      status: 200,
      statusText: 'OK',
    };

    (
      mockAxiosInstance.post as unknown as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce(mockResponse);

    const result = await fileUploader.upload(url, { file });
    expect(result).toEqual(mockResponse);
    expect(mockAxiosInstance.post).toHaveBeenCalledWith(
      url,
      expect.any(FormData),
      {
        // 不应手动设置 Content-Type：浏览器发送 FormData 时会自动生成
        // 带 boundary 的 'multipart/form-data; boundary=...'，
        // 手动写死会导致 boundary 丢失，后端无法解析。
        headers: {},
      },
    );
  });

  it('should merge provided config with default config', async () => {
    const url = 'https://example.com/upload';
    const file = new File(['file content'], 'test.txt', { type: 'text/plain' });
    const mockResponse: AxiosResponse = {
      config: {} as any,
      data: { success: true },
      headers: {},
      status: 200,
      statusText: 'OK',
    };

    (
      mockAxiosInstance.post as unknown as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce(mockResponse);

    const customConfig: AxiosRequestConfig = {
      headers: { 'Custom-Header': 'value' },
    };

    const result = await fileUploader.upload(url, { file }, customConfig);
    expect(result).toEqual(mockResponse);
    expect(mockAxiosInstance.post).toHaveBeenCalledWith(
      url,
      expect.any(FormData),
      {
        // 调用方传入的自定义 header 原样保留，但不应注入 Content-Type
        // （由浏览器自动生成带 boundary 的值）。
        headers: {
          'Custom-Header': 'value',
        },
      },
    );
  });

  it('should strip caller-provided Content-Type so the browser can set the boundary', async () => {
    const url = 'https://example.com/upload';
    const file = new File(['file content'], 'test.txt', { type: 'text/plain' });
    const mockResponse: AxiosResponse = {
      config: {} as any,
      data: { success: true },
      headers: {},
      status: 200,
      statusText: 'OK',
    };

    (
      mockAxiosInstance.post as unknown as ReturnType<typeof vi.fn>
    ).mockResolvedValueOnce(mockResponse);

    // 调用方误传了无 boundary 的 Content-Type，uploader 应剥离它，
    // 否则后端会因找不到 boundary 而解析失败。
    await fileUploader.upload(url, { file }, {
      headers: { 'Content-Type': 'multipart/form-data' },
    } as AxiosRequestConfig);
    const calledArgs = (mockAxiosInstance.post as any).mock.calls[0];
    expect(calledArgs[2].headers['Content-Type']).toBeUndefined();
  });

  it('should handle errors gracefully', async () => {
    const url = 'https://example.com/upload';
    const file = new File(['file content'], 'test.txt', { type: 'text/plain' });
    (
      mockAxiosInstance.post as unknown as ReturnType<typeof vi.fn>
    ).mockRejectedValueOnce(new Error('Network Error'));

    await expect(fileUploader.upload(url, { file })).rejects.toThrow(
      'Network Error',
    );
  });

  it('should handle empty URL gracefully', async () => {
    const url = '';
    const file = new File(['file content'], 'test.txt', { type: 'text/plain' });
    (
      mockAxiosInstance.post as unknown as ReturnType<typeof vi.fn>
    ).mockRejectedValueOnce(new Error('Request failed with status code 404'));

    await expect(fileUploader.upload(url, { file })).rejects.toThrow(
      'Request failed with status code 404',
    );
  });

  it('should handle null URL gracefully', async () => {
    const url = null as unknown as string;
    const file = new File(['file content'], 'test.txt', { type: 'text/plain' });
    (
      mockAxiosInstance.post as unknown as ReturnType<typeof vi.fn>
    ).mockRejectedValueOnce(new Error('Request failed with status code 404'));

    await expect(fileUploader.upload(url, { file })).rejects.toThrow(
      'Request failed with status code 404',
    );
  });
});
