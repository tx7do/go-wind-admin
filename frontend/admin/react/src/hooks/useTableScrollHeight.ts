import { useState, useEffect, useCallback } from 'react';
import { TABLE, LAYOUT_HEIGHTS } from '@/config/constants';

/**
 * 动态计算表格高度的 Hook
 * 根据视口高度和排除的区域高度，自动计算表格内容区域的滚动高度
 * 
 * @param options - 配置选项
 * @param options.excludeHeight - 需要排除的固定高度（如顶栏、面包屑等），默认使用 LAYOUT_HEIGHTS 计算
 * @param options.offset - 额外偏移量，默认使用 TABLE.SCROLL_CONFIG.OFFSET
 * @param options.minHeight - 最小高度限制，默认使用 TABLE.SCROLL_CONFIG.MIN_HEIGHT
 * @param options.includeToolbar - 是否包含工具栏高度（默认 true）
 * @param options.includePagination - 是否包含分页器高度（默认 true）
 * @param options.includeSearchForm - 是否包含搜索表单高度（默认 false，搜索栏固定时设为 true）
 * 
 * @returns 表格内容区域高度字符串，如 '600px'
 * 
 * @example
 * // 基础用法（搜索栏固定）
 * const tableScrollY = useTableScrollHeight();
 * 
 * // 搜索栏随表格一起滚动
 * const tableScrollY = useTableScrollHeight({ includeSearchForm: true });
 * 
 * <ProTable
 *   scroll={{ y: tableScrollY }}
 *   // ...
 * />
 */
export function useTableScrollHeight(options: {
  excludeHeight?: number;
  offset?: number;
  minHeight?: number;
  includeToolbar?: boolean;
  includePagination?: boolean;
  includeSearchForm?: boolean;
} = {}): string {
  const {
    excludeHeight,
    offset = TABLE.SCROLL_CONFIG.OFFSET,
    minHeight = TABLE.SCROLL_CONFIG.MIN_HEIGHT,
    includeToolbar = true,
    includePagination = true,
    includeSearchForm = false, // 默认搜索栏固定
  } = options;

  // 计算外部排除高度（如果没有手动指定）
  const calculatedExcludeHeight = excludeHeight ?? (
    LAYOUT_HEIGHTS.HEADER +
    LAYOUT_HEIGHTS.TABS +
    LAYOUT_HEIGHTS.BREADCRUMB +
    LAYOUT_HEIGHTS.CONTENT_PADDING
  );

  const [viewportHeight, setViewportHeight] = useState(window.innerHeight);

  // 更新视口高度
  const updateViewportHeight = useCallback(() => {
    setViewportHeight(window.innerHeight);
  }, []);

  // 监听窗口大小变化
  useEffect(() => {
    updateViewportHeight();
    window.addEventListener('resize', updateViewportHeight);
    return () => {
      window.removeEventListener('resize', updateViewportHeight);
    };
  }, [updateViewportHeight]);

  // 动态计算表格内容区域高度
  const calculateHeight = useCallback(() => {
    // ProTable 内部组件高度（从常量获取）
    const toolbarHeight = includeToolbar ? TABLE.COMPONENT_HEIGHTS.TOOLBAR : 0;
    const paginationHeight = includePagination ? TABLE.COMPONENT_HEIGHTS.PAGINATION : 0;
    const tableHeaderHeight = TABLE.COMPONENT_HEIGHTS.HEADER;
    const searchFormHeight = includeSearchForm ? TABLE.COMPONENT_HEIGHTS.SEARCH_FORM : 0;
    
    // 内容区域高度 = 视口高度 - 外部排除高度 - 内部组件高度 - 偏移量
    const contentHeight = viewportHeight 
      - calculatedExcludeHeight 
      - toolbarHeight 
      - paginationHeight 
      - tableHeaderHeight 
      - searchFormHeight 
      - offset;
    
    // 确保不低于最小高度
    const finalHeight = Math.max(contentHeight, minHeight);
    
    return `${finalHeight}px`;
  }, [
    viewportHeight, 
    calculatedExcludeHeight, 
    offset, 
    minHeight, 
    includeToolbar, 
    includePagination,
    includeSearchForm,
  ]);

  return calculateHeight();
}

/**
 * 简化版的表格高度 Hook
 * 自动估算常见布局的排除高度
 * 
 * @param extraOffset - 额外偏移量
 * @returns 表格高度字符串
 */
export function useAutoTableHeight(extraOffset: number = 0): string {
  const [excludeHeight, setExcludeHeight] = useState(0);
  const headerRef = useRef<HTMLElement | null>(null);
  const breadcrumbRef = useRef<HTMLElement | null>(null);
  const toolbarRef = useRef<HTMLElement | null>(null);

  // 动态测量各区域高度
  useEffect(() => {
    const calculateExcludeHeight = () => {
      let height = 0;
      
      // 测量顶栏高度（如果存在）
      if (headerRef.current) {
        height += headerRef.current.offsetHeight;
      }
      
      // 测量面包屑高度（如果存在）
      if (breadcrumbRef.current) {
        height += breadcrumbRef.current.offsetHeight;
      }
      
      // 测量工具栏高度（如果存在）
      if (toolbarRef.current) {
        height += toolbarRef.current.offsetHeight;
      }

      setExcludeHeight(height);
    };

    // 初次测量
    calculateExcludeHeight();

    // 使用 ResizeObserver 监听高度变化
    const observer = new ResizeObserver(calculateExcludeHeight);
    
    if (headerRef.current) observer.observe(headerRef.current);
    if (breadcrumbRef.current) observer.observe(breadcrumbRef.current);
    if (toolbarRef.current) observer.observe(toolbarRef.current);

    return () => {
      observer.disconnect();
    };
  }, []);

  return useTableScrollHeight({
    excludeHeight,
    offset: extraOffset,
  });
}
