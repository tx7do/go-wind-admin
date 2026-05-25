import { useState, useEffect, useRef, type RefObject } from 'react';

/**
 * 动态计算 ProTable 的 scroll.y，使表格撑满容器，分页器置底。
 *
 * 核心原理：
 *   1. 初始值必须是像素值（不能用 'auto'），否则 antd 不会创建 .ant-table-body
 *   2. 用 getBoundingClientRect 测量 .ant-table-body 上方占用的空间（自动含所有 margin）
 *   3. 直接测量分页器的 offsetHeight + margin
 *   4. scroll.y = 容器高度 - 上方空间 - 下方空间 - 安全边距
 *
 * @param containerRef - .page-container-content 容器 div 的 ref（无 padding/border）
 * @param options.buffer - 安全边距（像素），默认 8
 * @param options.minHeight - scroll.y 最小值，默认 100
 */
export function useProTableScrollY(
  containerRef: RefObject<HTMLElement | null>,
  options: { buffer?: number; minHeight?: number } = {},
): string {
  const { buffer = 8, minHeight = 100 } = options;

  // 关键：初始值必须是像素值，触发 antd 创建 .ant-table-body
  const [scrollY, setScrollY] = useState<string>(() => {
    const estimate = window.innerHeight - 350;
    return `${Math.max(estimate, minHeight)}px`;
  });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const measure = () => {
      const containerHeight = container.clientHeight;
      if (containerHeight <= 0) return;

      // 查找 .ant-table-body（scroll.y 为像素值时 antd 会创建此元素）
      const tableBody = container.querySelector('.ant-table-body');
      if (!tableBody) return;

      const containerRect = container.getBoundingClientRect();
      const bodyRect = tableBody.getBoundingClientRect();

      // 上方空间：从容器顶部到表体顶部（搜索表单 + 工具栏 + 表头 + 所有间距）
      const aboveSpace = bodyRect.top - containerRect.top;

      // 下方空间：分页器高度 + margin
      const pagination = container.querySelector('.ant-pagination');
      let belowSpace = 0;
      if (pagination) {
        const pagStyle = getComputedStyle(pagination);
        const marginTop = parseFloat(pagStyle.marginTop) || 0;
        const marginBottom = parseFloat(pagStyle.marginBottom) || 0;
        belowSpace = pagination.offsetHeight + marginTop + marginBottom;
      } else {
        // 分页器可能还没渲染，用默认估算
        belowSpace = 56;
      }

      const available = containerHeight - aboveSpace - belowSpace - buffer;
      const result = Math.max(available, minHeight);
      setScrollY(`${result}px`);
    };

    // 多次重试测量，确保 ProTable 异步渲染完成
    const timers = [
      setTimeout(measure, 50),
      setTimeout(measure, 200),
      setTimeout(measure, 500),
      setTimeout(measure, 1200),
    ];

    const resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(measure);
    });
    resizeObserver.observe(container);

    const mutationObserver = new MutationObserver(() => {
      requestAnimationFrame(measure);
    });
    mutationObserver.observe(container, { childList: true, subtree: true });

    return () => {
      timers.forEach(clearTimeout);
      resizeObserver.disconnect();
      mutationObserver.disconnect();
    };
  }, [containerRef, buffer, minHeight]);

  return scrollY;
}
