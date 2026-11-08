import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import type { HTMLAttributes, Key, MutableRefObject } from 'react';
import { ProTable } from '@ant-design/pro-components';
import type { ActionType, ProColumns, ProTableProps } from '@ant-design/pro-components';
import { Alert, Button, Empty } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { usePreferencesStore } from '@/core/preferences';
import { useI18n } from '@/core/i18n';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import { TABLE } from '@/config/constants';
import { MIN_COLUMN_WIDTH, ResizableHeaderCell } from './ResizableHeaderCell';
import type { ResizableCellProps } from './ResizableHeaderCell';
import { TableSkeleton } from './TableSkeleton';

/**
 * 加载态出场阈值（ms）。本地接口常在 100ms 内返回，任何早于阈值就出现的加载态（骨架屏或 Spin）
 * 都比「什么都不显示」更闪，所以首屏骨架与表内刷新的 Spin 共用这一个阈值。
 */
const LOADING_DELAY = 250;

/** pagination=false 的页面没有 pageSize 可依据，骨架行数取一个接近满屏的常量 */
const FALLBACK_SKELETON_ROWS = 10;

/** 页面列允许的扩展字段：resizable=false 可对单列关掉拖拽把手 */
type ListTableColumn<RecordType> = ProColumns<RecordType> & { resizable?: boolean };

/** 与 antd getColumnKey 同规则的稳定列标识（key → dataIndex → 序号），列宽状态按它挂靠 */
const getColumnKey = (column: { key?: Key; dataIndex?: unknown }, fallbackIndex: number): string => {
  if (column.key !== undefined && column.key !== null) return String(column.key);
  const { dataIndex } = column;
  if (typeof dataIndex === 'string' || typeof dataIndex === 'number') return String(dataIndex);
  if (Array.isArray(dataIndex) && dataIndex.length > 0) return dataIndex.join('-');
  return `col-${fallbackIndex}`;
};

/** 列宽状态：widths 挂全部叶子列的锁定宽度，flexKey 是不锁宽、吸收增减的弹性列（null = 全列定宽） */
interface ResizeState {
  widths: Record<string, number>;
  flexKey: string | null;
  /** 锁定瞬间的滚动容器宽度，拖动中用它判定弹性列是否被压穿 */
  containerWidth: number;
}

type ListTableProps<
  RecordType extends Record<string, any>,
  Params extends Record<string, any> = Record<string, any>,
> = ProTableProps<RecordType, Params>;

type LoadPhase = 'initial' | 'ready' | 'failed';

/**
 * 列表页统一表格：在 ProTable 之上补四件页面各自都没有的事——首屏骨架屏（按页面 columns 生成，
 * 受偏好设置 transition.loading 控制）、快响应不闪的 Spin（delay）、请求失败的错误态与重试、
 * 以及**高度自动撑满**。
 *
 * 高度契约（2026-10-22 起）：页面不传 scroll.y 时，本组件自动接管撑满——内部测量容器
 * （.listtable-fill）+ useProTableScrollY 把分页器钉在容器底，要求外层是 ContentContainer
 * 这类 flex 列容器（非 flex 父级时自动退回自然高度，Drawer 内嵌表等场景不误伤）。
 * 页面显式传了 scroll.y 则完全尊重（历史页面的手写样板照常工作，新页面不要再写）。
 * 页面只需：<ContentContainer><ListTable …/></ContentContainer>。
 *
 * 页面把 `<ProTable>` 换成 `<ListTable>` 即可，request / columns / actionRef 用法不变。
 */
export function ListTable<
  RecordType extends Record<string, any>,
  Params extends Record<string, any> = Record<string, any>,
>(props: ListTableProps<RecordType, Params> & { striped?: boolean }) {
  const {
    request,
    columns,
    actionRef,
    pagination,
    size,
    bordered,
    scroll,
    loading,
    rowClassName,
    components,
    tableLayout,
    striped = true,
    ...restProps
  } = props;
  const { t } = useI18n('common');
  const skeletonEnabled = usePreferencesStore((state) => state.preferences.transition.loading);

  // 高度自动撑满：页面未传 scroll.y 时接管（传了则页面自管，行为与历史完全一致）。
  const autoFill = scroll?.y === undefined;
  const fillRef = useRef<HTMLDivElement>(null);
  const fillScrollY = useProTableScrollY(fillRef, { enabled: autoFill, requireFlexParent: true });

  const [phase, setPhase] = useState<LoadPhase>('initial');
  /** 请求抛错时的原始文案；页面自己 catch 后返回 success:false 的情况拿不到原因，退化为通用文案 */
  const [errorReason, setErrorReason] = useState<string | null>(null);
  const [showSkeleton, setShowSkeleton] = useState(false);
  /** 列宽拖拽：null = 用户从未拖过（保持自适应布局），一旦有值即切换 fixed 布局 + 显式列宽 */
  const [resizeState, setResizeState] = useState<ResizeState | null>(null);
  const [resizingKey, setResizingKey] = useState<string | null>(null);

  // 表体纵向滚动条宽度（0 = 不显示）实时下传 rc-table：它只在挂载时测一次滚动条宽度，
  // 而表体的滚动条随内容显隐（useProTableScrollY）和全局自定义滚动条样式都会让那次测量
  // 失真——信念过期时表头的补偿列与固定列 sticky 偏移全部错位。scrollbarSize prop 由
  // patches/@rc-component__table@1.10.0.patch 暴露。
  const tableDomId = useId();
  const [scrollbarWidth, setScrollbarWidth] = useState<number | null>(null);

  useEffect(() => {
    const root = document.getElementById(tableDomId);
    if (!root) return undefined;
    let body: HTMLElement | null = null;
    let bodyResizeObserver: ResizeObserver | null = null;
    let innerTableResizeObserver: ResizeObserver | null = null;
    let styleAttrObserver: MutationObserver | null = null;
    const measure = () => {
      if (!body) return;
      // 纵向滚动条占宽 = 边框盒 − 内容盒 − 左右边框；overflow-y 隐藏/未溢出时为 0。
      // Chrome 对「纯滚动条显隐」不触发 ResizeObserver，故 style 属性变化由
      // styleAttrObserver 兜底；内层 table 的 RO 兜住内容/容器尺寸变化引发的显隐。
      const cs = getComputedStyle(body);
      const borderX = (parseFloat(cs.borderLeftWidth) || 0) + (parseFloat(cs.borderRightWidth) || 0);
      const width = body.offsetWidth - body.clientWidth - borderX;
      setScrollbarWidth((prev) => (prev === width ? prev : width));
    };
    const syncBody = () => {
      const next = root.querySelector('.ant-table-body') as HTMLElement | null;
      if (next === body) return;
      bodyResizeObserver?.disconnect();
      innerTableResizeObserver?.disconnect();
      styleAttrObserver?.disconnect();
      body = next;
      if (body) {
        bodyResizeObserver = new ResizeObserver(measure);
        bodyResizeObserver.observe(body);
        innerTableResizeObserver = new ResizeObserver(measure);
        const innerTable = body.querySelector(':scope > table');
        if (innerTable) innerTableResizeObserver.observe(innerTable);
        styleAttrObserver = new MutationObserver(measure);
        styleAttrObserver.observe(body, { attributes: true, attributeFilter: ['style'] });
        measure();
      }
    };
    syncBody();
    // 骨架屏 ↔ 真实表格互换会替换 .ant-table-body：按 DOM 变更重新挂观察
    const mutationObserver = new MutationObserver(syncBody);
    mutationObserver.observe(root, { childList: true, subtree: true });
    return () => {
      mutationObserver.disconnect();
      bodyResizeObserver?.disconnect();
      innerTableResizeObserver?.disconnect();
      styleAttrObserver?.disconnect();
    };
  }, [tableDomId]);

  const loadedOnceRef = useRef(false);
  const skeletonTimerRef = useRef<number | null>(null);
  const ownActionRef = useRef<ActionType | undefined>(undefined);

  /** 可拖拽的叶子列（组表头的组列没有把手，只有叶子列参与锁定） */
  const leafColumns = useMemo(() => {
    const source = (columns ?? []) as ListTableColumn<RecordType>[];
    const leaves: { column: ListTableColumn<RecordType>; key: string }[] = [];
    const collect = (cols: ListTableColumn<RecordType>[]) => {
      cols.forEach((column, index) => {
        if (Array.isArray(column.children) && column.children.length > 0) {
          collect(column.children as ListTableColumn<RecordType>[]);
          return;
        }
        leaves.push({ column, key: getColumnKey(column, index) });
      });
    };
    collect(source);
    return leaves;
  }, [columns]);

  const handleResizeStart = useCallback(
    ({
      key,
      width,
      allWidths,
      containerWidth,
    }: {
      key: string;
      width: number;
      allWidths: Record<string, number>;
      containerWidth: number;
    }) => {
      setResizingKey(key);
      setResizeState((prev) => {
        const next = Math.max(MIN_COLUMN_WIDTH, width);
        if (prev) {
          return {
            widths: { ...prev.widths, [key]: next },
            // 拖的就是弹性列：它从此有了确定宽度，弹性位退役（全列定宽后表格按总和伸缩）
            flexKey: prev.flexKey === key ? null : prev.flexKey,
            containerWidth: prev.containerWidth,
          };
        }
        // 首拖锁定：整行列宽锁成当前渲染值，锁定瞬间（自适应布局 → fixed 布局）渲染结果
        // 与锁前一模一样，零跳变。仅当列宽总和不超出容器时留一根非冻结列做弹性列吸收
        // 增减；本就溢出的页面（页面自带列宽且无 scroll.x）全部锁宽——留给弹性列的
        // 剩余空间是负数，会把它压瘪成 0 宽（用户页实测）。
        const widths: Record<string, number> = { ...allWidths, [key]: next };
        const measuredSum = Object.values(allWidths).reduce((sum, w) => sum + w, 0);
        let flexKey: string | null = null;
        if (measuredSum <= containerWidth + 1) {
          for (let i = leafColumns.length - 1; i >= 0; i -= 1) {
            const leaf = leafColumns[i];
            if (!leaf.column.fixed && leaf.key !== key) {
              flexKey = leaf.key;
              break;
            }
          }
        }
        if (flexKey) delete widths[flexKey];
        return { widths, flexKey, containerWidth };
      });
    },
    [leafColumns],
  );

  const handleResize = useCallback((key: string, width: number) => {
    setResizeState((prev) => {
      if (!prev) return prev;
      const widths = { ...prev.widths, [key]: Math.max(MIN_COLUMN_WIDTH, Math.round(width)) };
      if (!prev.flexKey) return { ...prev, widths };
      const { flexKey, containerWidth } = prev;
      const othersSum = Object.entries(widths)
        .filter(([k]) => k !== flexKey)
        .reduce((sum, [, w]) => sum + w, 0);
      const flexWidth = containerWidth - othersSum;
      // 拖到弹性列被压穿：把它冻结在当前值（不低于最小宽），此后全列定宽、继续拖宽走横向滚动
      if (flexWidth >= MIN_COLUMN_WIDTH) return { ...prev, widths };
      return {
        widths: { ...widths, [flexKey]: Math.max(MIN_COLUMN_WIDTH, Math.floor(flexWidth)) },
        flexKey: null,
        containerWidth,
      };
    });
  }, []);

  const handleResizeEnd = useCallback(() => setResizingKey(null), []);

  const handleResizeReset = useCallback(() => {
    setResizeState(null);
    setResizingKey(null);
  }, []);

  // 注入拖拽参数与锁定宽度：组列与 resizable=false 的列不带把手；
  // 弹性列显式去宽（width: undefined），由 fixed 布局分它剩余空间
  const resizableColumns = useMemo(() => {
    const source = (columns ?? []) as ListTableColumn<RecordType>[];
    return source.map((column, index) => {
      const isGroup = Array.isArray(column.children) && column.children.length > 0;
      if (isGroup || column.resizable === false) return column;
      const key = getColumnKey(column, index);
      const width =
        resizeState && !column.hideInTable
          ? key === resizeState.flexKey
            ? undefined
            : (resizeState.widths[key] ?? column.width)
          : column.width;
      const handleProps: ResizableCellProps = {
        'data-resize-key': key,
        onResizeStart: handleResizeStart,
        onResize: handleResize,
        onResizeEnd: handleResizeEnd,
        onResizeReset: handleResizeReset,
        resizing: resizingKey === key,
        resizeTitle: t('table.columnResizeHint'),
      };
      return {
        ...column,
        width,
        onHeaderCell: () => handleProps as unknown as HTMLAttributes<HTMLTableCellElement>,
      };
    });
  }, [
    columns,
    resizeState,
    resizingKey,
    handleResizeStart,
    handleResize,
    handleResizeEnd,
    handleResizeReset,
    t,
  ]);

  // 页面传进来的 request 是内联箭头函数（每次渲染都是新引用）。走 ref 取值，给 ProTable 的
  // request 才能保持恒定引用，否则包装层自己会变成重新取数的原因。
  const requestRef = useRef(request);
  requestRef.current = request;

  // ProTable 只接受一个 actionRef：这里收下它，再把同一个 action 对象转交给页面的 ref。
  // 子组件 effect 先于父组件 effect 运行，所以本页的 actionRef 在页面首次 reload 前就已就位。
  useEffect(() => {
    if (!actionRef) return;
    if (typeof actionRef === 'function') {
      actionRef(ownActionRef.current);
      return;
    }
    (actionRef as MutableRefObject<ActionType | undefined>).current = ownActionRef.current;
  });

  // 斑马纹：默认开启，对齐 ele/vben 共享表格的 stripe 默认值（值表见 docs/design-language.md §4 表格条）。
  // 页面自带 rowClassName 时叠加而非覆盖；占位行（暂无数据）由 CSS 侧排除。
  const composedRowClassName = useCallback(
    (record: RecordType, index: number, indent: number) => {
      const pageCls =
        typeof rowClassName === 'function'
          ? (rowClassName(record, index, indent) ?? '')
          : (rowClassName ?? '');
      if (!striped || index % 2 !== 1) return pageCls;
      return pageCls ? `${pageCls} ant-table-row-striped` : 'ant-table-row-striped';
    },
    [rowClassName, striped],
  );

  const stopSkeleton = useCallback(() => {
    if (skeletonTimerRef.current !== null) {
      window.clearTimeout(skeletonTimerRef.current);
      skeletonTimerRef.current = null;
    }
    setShowSkeleton(false);
  }, []);

  useEffect(() => stopSkeleton, [stopSkeleton]);

  const wrappedRequest = useCallback<NonNullable<ListTableProps<RecordType, Params>['request']>>(
    async (params, sort, filter) => {
      setErrorReason(null);
      if (!loadedOnceRef.current && skeletonEnabled) {
        if (skeletonTimerRef.current !== null) window.clearTimeout(skeletonTimerRef.current);
        skeletonTimerRef.current = window.setTimeout(() => setShowSkeleton(true), LOADING_DELAY);
      }
      try {
        const result = await requestRef.current?.(params, sort, filter);
        if (!result || result.success === false) {
          setPhase('failed');
          return result ?? { data: [], success: false };
        }
        loadedOnceRef.current = true;
        setPhase('ready');
        return result;
      } catch (error) {
        console.error('[ListTable] 列表数据请求失败', error);
        setPhase('failed');
        // message 可能为 undefined/空串（异常 rejection 对象缺 message 字段），
        // 空值兜底到通用失败文案，避免把字面量 "undefined" 渲染进错误态
        setErrorReason((error as Error)?.message || t('loading.failed'));
        return { data: [], total: 0, success: false };
      } finally {
        stopSkeleton();
      }
    },
    [skeletonEnabled, stopSkeleton],
  );

  const retry = useCallback(() => {
    setPhase('initial');
    ownActionRef.current?.reload?.();
  }, []);

  const skeletonRows = useMemo(() => {
    if (pagination === false) return FALLBACK_SKELETON_ROWS;
    const configured = pagination?.defaultPageSize ?? pagination?.pageSize;
    return configured && configured > 0 ? configured : TABLE.DEFAULT_PAGE_SIZE;
  }, [pagination]);

  // antd（rc-table）以 scroll.x 是否存在作为表格可横向滚动（horizonScroll）的开关：
  // 冻结列的分割线 shadow 类、scrollLeft 监听全被它门控，而 sticky 冻结本身是纯 CSS——
  // 缺 x 时列照样冻、能横向滚，分割线却永远不画（脚本页操作列实测）。
  // 列里带 fixed 时补 '100%'：表格仍自适应撑满容器（不会像固定像素那样撑出空白列，
  // 即 4bae15a3 批量去掉 scroll.x 的原因），真溢出时 antd 的滚动与分割线链路照常工作。
  // 列宽锁定后同理：拖宽的总和一旦超过容器就靠它出横向滚动条，未超出时表格依旧铺满。
  const hasFixedColumn = !!columns?.some((column) => !!column.fixed);
  const resizeLocked = resizeState !== null;
  const mergedScroll = useMemo(() => {
    const pageX = typeof scroll === 'object' ? scroll?.x : undefined;
    if (pageX || (!hasFixedColumn && !resizeLocked)) return scroll;
    return { ...(typeof scroll === 'object' ? scroll : {}), x: '100%' };
  }, [scroll, hasFixedColumn, resizeLocked]);

  // 自动撑满时的 scroll：y 走内部测量的 fillScrollY（初始 '100px' 仅为了让 antd 建
  // .ant-table-body 结构，真实高度由 hook 直写 style）；固定列仍需 x:'100%' 画分割线。
  const effectiveScroll = useMemo(() => {
    if (!autoFill) return mergedScroll;
    return {
      ...(hasFixedColumn && !resizeLocked ? { x: '100%' as const } : {}),
      y: fillScrollY,
    };
  }, [autoFill, mergedScroll, hasFixedColumn, resizeLocked, fillScrollY]);

  // 锁定后列宽是显式像素值，必须切 fixed 布局才生效；未锁定时不动 tableLayout，
  // 页面自带 tableLayout 的（如有）优先
  const mergedTableLayout = tableLayout ?? (resizeLocked ? 'fixed' : undefined);

  const mergedComponents = useMemo(
    () => ({
      ...components,
      header: { ...components?.header, cell: ResizableHeaderCell },
    }),
    [components],
  );

  // 骨架继承页面的两向 scroll：scroll.x 保证列宽与真实表格逐列对齐，scroll.y 保证两态都有
  // 同一层限高 .ant-table-body —— 分页器的落点由它决定，骨架缺这层就会在数据到位时整段跳动。
  const scrollX = typeof effectiveScroll === 'object' ? effectiveScroll?.x : undefined;
  const scrollY = typeof effectiveScroll === 'object' ? effectiveScroll?.y : undefined;

  const tableViewRender = useCallback<NonNullable<ListTableProps<RecordType, Params>['tableViewRender']>>(
    (_tableProps, defaultDom) => {
      // 首屏就失败：一行数据都没有，"暂无数据"会被读成查询成功但结果为空，必须换成错误态
      if (phase === 'failed' && !loadedOnceRef.current) {
        return (
          <div className="flex flex-col items-center justify-center gap-3 py-12">
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={errorReason ?? t('loading.failed')} />
            <Button icon={<ReloadOutlined />} onClick={retry}>
              {t('button.retry')}
            </Button>
          </div>
        );
      }
      if (showSkeleton) {
        return (
          <TableSkeleton<RecordType>
            columns={resizableColumns}
            rows={skeletonRows}
            size={size}
            bordered={bordered}
            scrollX={scrollX}
            scrollY={scrollY}
            pagination={pagination}
          />
        );
      }
      // 已有数据时刷新失败：旧数据留着可读，只在表格上方补一条带重试入口的提示
      if (phase === 'failed') {
        return (
          <>
            <Alert
              className="mb-2"
              type="error"
              showIcon
              title={errorReason ?? t('loading.failed')}
              action={
                <Button size="small" icon={<ReloadOutlined />} onClick={retry}>
                  {t('button.retry')}
                </Button>
              }
            />
            {defaultDom}
          </>
        );
      }
      return defaultDom;
    },
    [phase, showSkeleton, errorReason, retry, resizableColumns, skeletonRows, size, bordered, scrollX, scrollY, pagination, t],
  );

  const tableNode = (
    <ProTable<RecordType, Params>
      {...restProps}
      id={tableDomId}
      scrollbarSize={scrollbarWidth ?? undefined}
      columns={resizableColumns}
      // 这六个 prop 上面为了喂骨架/算骨架行数/接管列宽拖拽而被解构出来，解构即从 restProps
      // 里摘走，不在此显式转交就会被 ProTable 静默丢掉（scroll.y 丢 → 无粘性表头与限高 body，
      // scroll.x 丢 → 列被压到容器宽度，pagination 丢 → 页面关不掉分页器，
      // components/tableLayout 丢 → 拖拽把手与 fixed 布局失效）。
      pagination={pagination}
      size={size}
      bordered={bordered}
      scroll={effectiveScroll}
      components={mergedComponents}
      tableLayout={mergedTableLayout}
      actionRef={ownActionRef}
      request={request ? wrappedRequest : undefined}
      rowClassName={composedRowClassName}
      loading={{ ...(typeof loading === 'object' ? loading : {}), delay: LOADING_DELAY }}
      tableViewRender={tableViewRender}
    />
  );

  // 自动撑满时套一层测量容器（flex 列父级里 flex:1 撑满；hook 据此测高并直写表体高度）。
  // 页面自己传了 scroll.y 的历史页面不包，DOM 结构与迁移前逐字节一致。
  if (!autoFill) return tableNode;
  return <div ref={fillRef} className="listtable-fill">{tableNode}</div>;
}

export default ListTable;
