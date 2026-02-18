import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createFileServiceClient,
  type storageservicev1_OSSProvider as OSSProvider,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useFileStore = defineStore('file', () => {
  const service = createFileServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询文件列表
   */
  async function listFile(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await service.List({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取文件
   */
  async function getFile(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建文件
   */
  async function createFile(values: Record<string, any> = {}) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新文件
   */
  async function updateFile(id: number, values: Record<string, any> = {}) {
    return await service.Update({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除文件
   */
  async function deleteFile(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listFile,
    getFile,
    createFile,
    updateFile,
    deleteFile,
  };
});

export const ossProviderList = computed(() => [
  {
    value: 'LOCAL',
    label: $t('enum.ossProvider.LOCAL'),
  },
  {
    value: 'MINIO',
    label: $t('enum.ossProvider.MINIO'),
  },
  {
    value: 'ALIYUN',
    label: $t('enum.ossProvider.ALIYUN'),
  },
  {
    value: 'QINIU',
    label: $t('enum.ossProvider.QINIU'),
  },
  {
    value: 'TENCENT',
    label: $t('enum.ossProvider.TENCENT'),
  },
  {
    value: 'BAIDU',
    label: $t('enum.ossProvider.BAIDU'),
  },
  {
    value: 'HUAWEI',
    label: $t('enum.ossProvider.HUAWEI'),
  },
  {
    value: 'AWS',
    label: $t('enum.ossProvider.AWS'),
  },
  {
    value: 'AZURE',
    label: $t('enum.ossProvider.AZURE'),
  },
  {
    value: 'GOOGLE',
    label: $t('enum.ossProvider.GOOGLE'),
  },
]);

export function ossProviderLabel(value: OSSProvider): string {
  const values = ossProviderList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const OSS_PROVIDER_COLOR_MAP = {
  // 本地存储 - 清新绿（适配本地服务视觉，柔和不刺眼）
  LOCAL: '#36D399',
  // MinIO - 深蓝（贴合官方品牌蓝，沉稳专业）
  MINIO: '#2563EB',
  // 七牛云 - 品牌紫（官方视觉核心色，高辨识度）
  QINIU: '#722ED1',
  // 阿里云 - 橙红（官方品牌代表色，一眼识别）
  ALIYUN: '#FF6A00',
  // 腾讯云 - 天青蓝（官方视觉色，与 MinIO 深蓝区分）
  TENCENT: '#12B7F5',
  // 百度智能云 - 浅湖蓝（柔和色调，差异化所有蓝色系）
  BAIDU: '#4080FF',
  // 华为云 - 砖红（官方品牌色，鲜明有记忆点）
  HUAWEI: '#E64340',
  // AWS - 经典橙（AWS 官方品牌核心色，全球通用识别）
  AWS: '#FF9900',
  // 微软 Azure - 科技蓝（官方标准品牌色）
  AZURE: '#0078D4',
  // 谷歌云 - 蓝绿（Google Cloud 官方视觉色）
  GOOGLE: '#4285F4',
  // 默认兜底 - 中性灰（不抢色，适配未定义厂商）
  DEFAULT: '#C9CDD4',
} as const satisfies Record<'DEFAULT' | OSSProvider, string>;

export function ossProviderColor(type: OSSProvider): string {
  return OSS_PROVIDER_COLOR_MAP[type] || OSS_PROVIDER_COLOR_MAP.DEFAULT;
}
