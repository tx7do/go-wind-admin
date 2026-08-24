import { useMutation, type UseMutationOptions } from "@tanstack/vue-query";
import { apiClient } from "@/api/client";
import { RequestClient } from "@/core/transport/rest";

/**
 * 从MinIO下载文件
 */
async function downloadFile(bucketName: string, objectName: string, preferPresignedUrl: boolean) {
  if (preferPresignedUrl) {
    const resp = await apiClient.fileTransferService.DownloadFile({
      storageObject: { bucketName, objectName },
      preferPresignedUrl,
    });

    const url = (resp as any).downloadUrl || "";
    if (!url) return;

    const a = document.createElement("a");
    a.href = url;
    a.target = "_blank";
    a.download = objectName || "download";
    document.body.append(a);
    a.click();
    a.remove();
    return;
  }

  const resp = await apiClient.fileTransferService.DownloadFile({
    storageObject: { bucketName, objectName },
    preferPresignedUrl,
  });

  const contentType = (resp as any).contentType || "application/octet-stream";
  const payload: ArrayBuffer | Blob | string | Uint8Array | undefined =
    (resp as any).file ?? (resp as any).data ?? (resp as any).payload ?? resp;

  function normalizeBase64(s: string): string {
    let str = s.replaceAll(/\s+/g, "");
    str = str.replaceAll("-", "+").replaceAll("_", "/");
    while (str.length % 4 !== 0) str += "=";
    return str;
  }

  function toBlob(data: any, type = contentType): Blob {
    if (!data) return new Blob([], { type });
    if (data instanceof Blob) return data;
    if (data instanceof ArrayBuffer) return new Blob([data], { type });
    if (ArrayBuffer.isView(data)) return new Blob([data as BufferSource], { type });

    if (typeof data === "string") {
      const maybeBase64 = data.includes("base64,") ? data.split("base64,")[1] : data;
      const base64 = normalizeBase64(maybeBase64 ?? "");

      let binary: string;
      try {
        binary = atob(base64);
      } catch {
        return new Blob([], { type });
      }

      const len = binary.length;
      const arr = new Uint8Array(len);
      for (let i = 0; i < len; i++) {
        arr[i] = (binary.codePointAt(i) ?? 0) & 0xff;
      }
      return new Blob([arr], { type });
    }

    return new Blob([data], { type });
  }

  const blob = toBlob(payload, contentType);
  const objectUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = objectUrl;
  a.download = objectName || "download";
  document.body.append(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(objectUrl);
}

/**
 * 上传文件到MinIO
 */
async function uploadFile(
  bucketName: string,
  fileDirectory: string,
  fileData: File,
  method: "post" | "put" = "post",
  onUploadProgress?: (progressEvent: any) => void
) {
  const storageObject = JSON.stringify({
    bucketName,
    fileDirectory,
  });

  await RequestClient.getInstance().upload(
    "admin/v1/file/upload",
    {
      file: fileData,
      storageObject,
      sourceFileName: fileData.name,
      mime: fileData.type,
      size: fileData.size,
      method,
    },
    { onUploadProgress }
  );
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
  >
) {
  return useMutation({
    mutationFn: async ({ bucketName, objectName, preferPresignedUrl = false }) => {
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
      fileDirectory: string;
      file: File;
      method?: "post" | "put";
      onUploadProgress?: (progress: any) => void;
    }
  >
) {
  return useMutation({
    mutationFn: async ({ bucketName, fileDirectory, file, method = "post", onUploadProgress }) => {
      return uploadFile(bucketName, fileDirectory, file, method, onUploadProgress);
    },
    ...options,
  });
}
