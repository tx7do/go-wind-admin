import type { Recordable } from '@vben/types';
import type { VxeGridProps, VxeUIExport } from 'vxe-table';

import type { VxeGridApi } from './api';

import { $t } from '@vben/locales';
import { formatDate, formatDateTime, isFunction } from '@vben/utils';

export function extendProxyOptions(
  api: VxeGridApi,
  options: VxeGridProps,
  getFormValues: () => Recordable<any>,
) {
  [
    'query',
    'querySuccess',
    'queryError',
    'queryAll',
    'queryAllSuccess',
    'queryAllError',
  ].forEach((key) => {
    extendProxyOption(key, api, options, getFormValues);
  });
}

function extendProxyOption(
  key: string,
  api: VxeGridApi,
  options: VxeGridProps,
  getFormValues: () => Recordable<any>,
) {
  const { proxyConfig } = options;
  const configFn = (proxyConfig?.ajax as Recordable<any>)?.[key];
  if (!isFunction(configFn)) {
    return options;
  }

  const wrapperFn = async (
    params: Recordable<any>,
    customValues: Recordable<any>,
    ...args: Recordable<any>[]
  ) => {
    const formValues = getFormValues();
    // 只有 query 承载"列表页三态"：加载中标记 + 失败原因要交给外层组件渲染骨架/横幅
    const isQuery = key === 'query';
    if (isQuery) {
      api.queryLoading.value = true;
      api.queryError.value = null;
    }
    try {
      const data = await configFn(
        params,
        {
          /**
           * 开启toolbarConfig.refresh功能
           * 点击刷新按钮 这里的值为PointerEvent 会携带错误参数
           */
          ...(customValues instanceof PointerEvent ? {} : customValues),
          ...formValues,
        },
        ...args,
      );
      if (isQuery) {
        api.hasLoaded.value = true;
      }
      return data;
    } catch (error) {
      if (isQuery) {
        // 不吞错：原始错误对象进控制台；页面上留一条可读原因（传输层已把 message 本地化，
        // 见 apps/admin/src/transport/rest/request-client.ts 的 request catch）
        console.error('[vxe-grid] 列表数据请求失败', error);
        api.queryError.value =
          (error as { message?: string } | null)?.message ||
          $t('common.loadDataFailed');
      }
      // 原样抛出：vxe 自己收尾 tableLoading，并保留页面定义 queryError 钩子的能力
      throw error;
    } finally {
      if (isQuery) {
        api.queryLoading.value = false;
      }
    }
  };
  api.setState({
    gridOptions: {
      proxyConfig: {
        ajax: {
          [key]: wrapperFn,
        },
      },
    },
  });
}

export function extendsDefaultFormatter(vxeUI: VxeUIExport) {
  vxeUI.formats.add('formatDate', {
    tableCellFormatMethod({ cellValue }) {
      return formatDate(cellValue);
    },
  });

  vxeUI.formats.add('formatDateTime', {
    tableCellFormatMethod({ cellValue }) {
      return formatDateTime(cellValue);
    },
  });

  vxeUI.formats.add('formatCent', {
    tableCellFormatMethod({ cellValue }) {
      if (
        cellValue === null ||
        cellValue === undefined ||
        Number.isNaN(cellValue)
      ) {
        return '-';
      }
      // 假设金额以分为单位，转换为元并保留两位小数
      return (cellValue / 100).toLocaleString(undefined, {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      });
    },
  });

  vxeUI.formats.add('formatPercent', {
    tableCellFormatMethod({ cellValue }) {
      if (
        cellValue === null ||
        cellValue === undefined ||
        Number.isNaN(cellValue)
      ) {
        return '-';
      }
      return `${(Number(cellValue) * 100).toFixed(2).replace(/\.00$/, '')}%`;
    },
  });

  vxeUI.formats.add('formatThousand', {
    tableCellFormatMethod({ cellValue }) {
      if (
        cellValue === null ||
        cellValue === undefined ||
        Number.isNaN(cellValue)
      ) {
        return '-';
      }
      return Number(cellValue).toLocaleString();
    },
  });

  vxeUI.formats.add('formatStr', {
    tableCellFormatMethod({ cellValue }) {
      const str = String(cellValue);
      // 如果是数字，前面补零
      if (/^\d+$/.test(str)) {
        return str.padStart(2, '0');
      }
      return str;
    },
  });
}
