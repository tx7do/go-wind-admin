import { useMutation, type UseMutationOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { RequestClient } from '#/transport/rest';

/**
 * 从MinIO下载文件
 * @param bucketName 文件桶名称
 * @param objectName 对象名称
 * @param preferPresignedUrl 是否优先使用预签名URL下载
 */
export async function downloadFile(
  bucketName: string,
  objectName: string,
  preferPresignedUrl: boolean,
) {
  if (preferPresignedUrl) {
    const resp = await apiClient.fileTransferService.DownloadFile({
      storageObject: { bucketName, objectName },
      preferPresignedUrl,
    });

    const url = (resp as any).downloadUrl || '';
    if (!url) return;


    const a = document.createElement('a');
    a.href = url;
    a.target = '_blank';
    a.download = objectName || 'download';
    document.body.append(a);
    a.click();
    a.remove();
    return;
  }

  const resp = await apiClient.fileTransferService.DownloadFile({
    storageObject: { bucketName, objectName },
    preferPresignedUrl,
  });

  const contentType = (resp as any).contentType || 'application/octet-stream';
  const payload: ArrayBuffer | Blob | string | Uint8Array | undefined =
    (resp as any).file ?? (resp as any).data ?? (resp as any).payload ?? resp;

  function normalizeBase64(s: string): string {
    let str = s.replaceAll(/\s+/g, '');
    str = str.replaceAll('-', '+').replaceAll('_', '/');
    while (str.length % 4 !== 0) str += '=';
    return str;
  }

  function toBlob(data: any, type = contentType): Blob {
    if (!data) return new Blob([], { type });
    if (data instanceof Blob) return data;
    if (data instanceof ArrayBuffer) return new Blob([data], { type });
    if (ArrayBuffer.isView(data))
      return new Blob([data as BufferSource], { type });

    if (typeof data === 'string') {
      // 支持 data URI 或纯 base64（处理 URL-safe base64）
      const maybeBase64 = data.includes('base64,')
        ? data.split('base64,')[1]
        : data;
      const base64 = normalizeBase64(maybeBase64 ?? '');

      let binary: string;
      try {
        binary = atob(base64);
      } catch {
        // 如果仍然失败，返回空 Blob（也可以改为抛错或走异步 fetch fallback）
        return new Blob([], { type });
      }

      const len = binary.length;
      const arr = new Uint8Array(len);
      for (let i = 0; i < len; i++) {
        arr[i] = (binary.codePointAt(i) ?? 0) & 0xff;
      }
      return new Blob([arr], { type });
    }

    // fallback
    return new Blob([data], { type });
  }

  const blob = toBlob(payload, contentType);
  const objectUrl = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = objectUrl;
  a.download = objectName || 'download';
  document.body.append(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(objectUrl);
}

/**
 * 上传文件到MinIO
 * @param bucketName 文件桶名称
 * @param fileDirectory 远端存储文件目录
 * @param fileData 文件数据
 * @param method 上传方法，支持 'post' 和 'put'
 * @param onUploadProgress 上传进度回调函数
 */
export async function uploadFile(
  bucketName: string,
  fileDirectory: string,
  fileData: File,
  method: 'post' | 'put' = 'post',
  onUploadProgress?: (progressEvent: any) => void,
) {
  // kratos 未注册 form-data codec，multipart 上传是死路径（后端 400
  // unregister Content-Type，go 参照后端同型）。JSON 形态（bytes 走 base64）
  // 才是生成客户端定义的活路；手工组 protojson 同形请求体以保留
  // axios 的 onUploadProgress（生成客户端的 transport.unary 不支持上传进度）。
  // 响应经 ResponseData 拦截器解包，直接是 UploadFileResponse（含 publicUrl）。
  const file = await fileToBase64(fileData);
  const body = {
    file,
    mime: fileData.type,
    size: fileData.size,
    sourceFileName: fileData.name,
    storageObject: { bucketName, fileDirectory },
  };
  return await RequestClient.getInstance().request('admin/v1/file/upload', {
    method: method === 'put' ? 'PUT' : 'POST',
    data: body,
    onUploadProgress,
  });
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error ?? new Error('文件读取失败'));
    reader.onload = () => {
      const dataUrl = String(reader.result ?? '');
      const idx = dataUrl.indexOf('base64,');
      resolve(idx >= 0 ? dataUrl.slice(idx + 'base64,'.length) : '');
    };
    reader.readAsDataURL(file);
  });
}

// -----------------------------------------------------------------------------
// 下载文件 Hook
// -----------------------------------------------------------------------------
export function useDownloadFile(
  options?: UseMutationOptions<
    void,
    Error,
    {
      bucketName: string;
      objectName: string;
      preferPresignedUrl?: boolean;
    }
  >,
) {
  return useMutation({
    mutationFn: async ({
      bucketName,
      objectName,
      preferPresignedUrl = false,
    }) => {
      return downloadFile(bucketName, objectName, preferPresignedUrl);
    },
    ...options,
  });
}

// -----------------------------------------------------------------------------
// 上传文件 Hook（支持进度）
// -----------------------------------------------------------------------------
export function useUploadFile(
  options?: UseMutationOptions<
    void,
    Error,
    {
      bucketName: string;
      file: File;
      fileDirectory: string;
      method?: 'post' | 'put';
      onUploadProgress?: (progressEvent: any) => void;
    }
  >,
) {
  return useMutation({
    mutationFn: async ({
      bucketName,
      fileDirectory,
      file,
      method = 'post',
      onUploadProgress,
    }) => {
      await uploadFile(
        bucketName,
        fileDirectory,
        file,
        method,
        onUploadProgress,
      );
    },
    ...options,
  });
}
