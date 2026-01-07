import { defineStore } from 'pinia';

import { createOssServiceClient } from '#/generated/api/admin/service/v1';
import { requestClientRequestHandler } from '#/utils/request';

export const useOssStore = defineStore('oss', () => {
  const service = createOssServiceClient(requestClientRequestHandler);

  async function getOssUploadUrl() {
    return await service.OssUploadUrl({});
  }

  async function uploadFile() {}

  function $reset() {}

  return {
    $reset,
    getOssUploadUrl,
    uploadFile,
  };
});
