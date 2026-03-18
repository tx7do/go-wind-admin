import {useState, useCallback} from 'react';
import type {AxiosProgressEvent} from 'axios';
import {
  createFileTransferServiceClient,
} from '@/api/admin/service/v1';
import {requestApi, requestClient} from '@/transport/rest';

export interface DownloadFileParams {
  bucketName?: string;
  objectName?: string;
  fileDirectory?: string;
  fileId?: number;
  downloadUrl?: string;
  preferPresignedUrl?: boolean;
  presignExpireSeconds?: number;
  rangeStart?: number;
  rangeEnd?: number;
  disposition?: string;
  acceptMime?: string;
}

export interface UploadFileParams {
  bucketName: string;
  fileDirectory: string;
  fileData: File;
  method?: 'post' | 'put';
  onUploadProgress?: (progressEvent: AxiosProgressEvent) => void;
}

export interface FileTransferResponse {
  success: boolean;
  data?: any;
}

/**
 * 文件传输 Model
 * 管理文件上传、下载等文件传输操作
 */
export default function FileTransferModel() {
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [fileDetail, setFileDetail] = useState<any | null>(null);

  // 创建服务客户端
  const fileTransferService = createFileTransferServiceClient(requestApi);

  /**
   * 下载文件
   */
  const downloadFile = useCallback(
    async (params: DownloadFileParams): Promise<any> => {
      try {
        setLoading(true);
        setError(null);

        const response = await fileTransferService.DownloadFile({
          storageObject: {
            bucketName: params.bucketName || '',
            objectName: params.objectName || '',
            fileDirectory: params.fileDirectory || '',
          },
          preferPresignedUrl: params.preferPresignedUrl,
          presignExpireSeconds: params.presignExpireSeconds,
          rangeStart: params.rangeStart,
          rangeEnd: params.rangeEnd,
          disposition: params.disposition,
          acceptMime: params.acceptMime,
        });

        setFileDetail(response);
        return response;
      } catch (err: any) {
        setError(err?.message || '下载文件失败');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  /**
   * 上传文件
   */
  const uploadFile = useCallback(
    async (params: UploadFileParams): Promise<FileTransferResponse> => {
      try {
        setLoading(true);
        setError(null);

        const {bucketName, fileDirectory, fileData, method = 'post', onUploadProgress} = params;
        const storageObject = JSON.stringify({bucketName, fileDirectory});

        // 使用 requestClient.upload 方法，它会自动处理 FormData
        await requestClient.upload(
          'admin/v1/file/upload',
          {
            file: fileData,
            storageObject,
            sourceFileName: fileData.name,
            mime: fileData.type,
            size: String(fileData.size),
            method,
          },
          {onUploadProgress},
        );

        return {success: true};
      } catch (err: any) {
        setError(err?.message || '上传文件失败');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  /**
   * 清除文件详情
   */
  const clearFileDetail = useCallback(() => {
    setFileDetail(null);
    setError(null);
  }, []);

  /**
   * 重置状态
   */
  const resetState = useCallback(() => {
    setLoading(false);
    setError(null);
    setFileDetail(null);
  }, []);

  /**
   * 清除错误
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    // 状态
    loading,
    error,
    fileDetail,

    // 方法
    downloadFile,
    uploadFile,
    clearFileDetail,
    resetState,
    clearError,
  };
}
