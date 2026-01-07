import { defineStore } from 'pinia';

import { createUEditorServiceClient } from '#/generated/api/admin/service/v1';
import { requestClientRequestHandler } from '#/utils/request';

export const useUEditorStore = defineStore('ueditor', () => {
  const service = createUEditorServiceClient(requestClientRequestHandler);

  async function config() {
    return await service.UEditorAPI({});
  }

  async function uploadFile() {}

  function $reset() {}

  return {
    $reset,
    config,
    uploadFile,
  };
});
