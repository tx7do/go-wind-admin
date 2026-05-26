import { useEffect, type RefObject } from 'react';

/**
 * 为 ProTable 的 .ant-table-body 动态设置高度，使分页器固定在容器底部。
 *
 * 核心思路（参考 CSDN 验证可行的方案）：
 *   1. scroll.y 给一个初始像素值，让 antd 创建 .ant-table-body 滚动结构
 *   2. JS 测量容器高度、分页器位置，直接操作 .ant-table-body.style.maxHeight
 *   3. 不依赖 CSS flex 穿透 antd 内部 DOM（因为 antd 内部层级不可控）
 *
 * @param containerRef - .page-container-content 容器 div 的 ref
 */
export function useProTableScrollY(
  containerRef: RefObject<HTMLElement | null>,
): string {
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let rafId = 0;

    const resize = () => {
      cancelAnimationFrame(rafId);
      rafId = requestAnimationFrame(() => {
        const tableBody = container.querySelector('.ant-table-body') as HTMLElement | null;
        if (!tableBody) return;

        const containerHeight = container.clientHeight;
        if (containerHeight <= 0) return;

        const containerRect = container.getBoundingClientRect();
        const bodyRect = tableBody.getBoundingClientRect();

        // 表格体上方的空间（搜索栏 + 工具栏 + 表头 + 所有间距）
        const aboveSpace = bodyRect.top - containerRect.top;

        // 分页器占用的空间（高度 + margin）
        const pagination = container.querySelector('.ant-pagination') as HTMLElement | null;
        let belowSpace = 0;
        if (pagination) {
          const pagStyle = getComputedStyle(pagination);
          belowSpace =
            pagination.offsetHeight +
            (parseFloat(pagStyle.marginTop) || 0) +
            (parseFloat(pagStyle.marginBottom) || 0);
        }

        // 安全边距（给分页器留出视觉呼吸空间，确保表格底线完整可见）
        const buffer = 24;

        // 计算表格体可用高度
        const availableHeight = Math.max(containerHeight - aboveSpace - belowSpace - buffer, 100);
        tableBody.style.maxHeight = `${availableHeight}px`;
        tableBody.style.minHeight = `${availableHeight}px`;

        // 根据内容是否溢出动态切换 overflow-y
        // antd 默认设置 overflow-y: scroll，在 Windows 上会始终显示滚动条轨道
        // 内容不超出时用 hidden 隐藏滚动条，超出时用 auto 按需显示
        if (tableBody.scrollHeight <= tableBody.clientHeight) {
          tableBody.style.overflowY = 'hidden';
        } else {
          tableBody.style.overflowY = 'auto';
        }
      });
    };

    // 多次延迟重试，确保 ProTable 异步渲染完成
    const timers = [
      setTimeout(resize, 50),
      setTimeout(resize, 200),
      setTimeout(resize, 500),
      setTimeout(resize, 1500),
    ];

    // 监听容器尺寸变化
    const resizeObserver = new ResizeObserver(resize);
    resizeObserver.observe(container);

    // 监听 DOM 变化（antd 重新渲染）
    const mutationObserver = new MutationObserver(resize);
    mutationObserver.observe(container, { childList: true, subtree: true });

    return () => {
      cancelAnimationFrame(rafId);
      timers.forEach(clearTimeout);
      resizeObserver.disconnect();
      mutationObserver.disconnect();
    };
  }, [containerRef]);

  // 返回初始像素值，仅用于触发 antd 创建 .ant-table-body
  return '100px';
}
