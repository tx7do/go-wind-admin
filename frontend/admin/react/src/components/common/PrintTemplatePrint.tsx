import { useRef, useState } from 'react';
import { App, Button, Empty, Modal, Select, Space, Spin } from 'antd';
import { PrinterOutlined, FilePdfOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { printservicev1_PrintTemplate as PrintTemplate } from '@/api/generated/admin/service/v1';
import { apiClient } from '@/api/client';
import { fetchListPrintTemplates } from '@/api/hooks/print-template';
import { renderHtmlPages, browserCodeRenderer } from '@worm-vue3-print/core/browser';
import { getPaperDimensions } from '@worm-vue3-print/core/designer';
import { PaginationQuery } from '@/core';

/**
 * 行打印定式（docs/print_template.md §5）：任意列表行接入打印能力的统一入口。
 *
 * 用法：页面里 `const { node, print } = usePrintTemplateDialog();`，
 * JSX 放 `{node}`，行动作里 `print('system/user', record)`。
 *
 * 模板匹配：优先取 resourceKey 精确相等且启用的模板；没有则回退"通用模板"
 * （resourceKey 为空的行）；仍无则显示空态引导到打印模板页。
 */
export function usePrintTemplateDialog() {
  const { modal } = App.useApp();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [templates, setTemplates] = useState<PrintTemplate[]>([]);
  const [selectedId, setSelectedId] = useState<number | undefined>();
  const [rendering, setRendering] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const lastHtmlRef = useRef<string>('');
  const lastTemplateRef = useRef<PrintTemplate | null>(null);
  const printDataRef = useRef<Record<string, unknown>>({});

  const { t } = useTranslation('printTemplate');

  /** 拉取资源可用模板：精确绑定优先，回退通用（resourceKey 为空） */
  const loadTemplates = async (resourceKey: string) => {
    const resp = await fetchListPrintTemplates(
      new PaginationQuery({ paging: { page: 1, pageSize: 100 } }),
    );
    const enabled = (resp.items || []).filter((item) => item.isEnabled !== false);
    const exact = enabled.filter((item) => item.resourceKey === resourceKey);
    const generic = enabled.filter((item) => !item.resourceKey);
    return [...exact, ...generic];
  };

  /** 渲染当前选中模板到 iframe（两遍渲染：测量→分页→HTML），产物留存供服务端 PDF 复用 */
  const renderPreview = async (tpl: PrintTemplate) => {
    setRendering(true);
    try {
      const result = await renderHtmlPages(
        JSON.parse(tpl.payload || '{}'),
        printDataRef.current,
        window.location.origin,
        browserCodeRenderer,
      );
      lastHtmlRef.current = result.html;
      lastTemplateRef.current = tpl;
      // 等 iframe 挂载完成再写文档
      window.setTimeout(() => {
        const doc = frameRef.current?.contentWindow?.document;
        if (doc) {
          doc.open();
          doc.write(result.html);
          doc.close();
        }
        setRendering(false);
      }, 80);
    } catch (e) {
      setRendering(false);
      modal.error({ title: t('renderFailed'), content: (e as Error).message });
    }
  };

  const print = async (resourceKey: string, rowData: Record<string, unknown>) => {
    printDataRef.current = rowData || {};
    setOpen(true);
    setLoading(true);
    setTemplates([]);
    setSelectedId(undefined);
    try {
      const list = await loadTemplates(resourceKey);
      setTemplates(list);
      if (list.length > 0) {
        setSelectedId(list[0].id);
        await renderPreview(list[0]);
      }
    } catch (e) {
      modal.error({ title: t('renderFailed'), content: (e as Error).message });
    } finally {
      setLoading(false);
    }
  };

  const handleSelect = async (id: number) => {
    setSelectedId(id);
    const tpl = templates.find((item) => item.id === id);
    if (tpl) await renderPreview(tpl);
  };

  const handlePrint = () => {
    frameRef.current?.contentWindow?.print();
  };

  /** 服务端 PDF（M3 输出链）：当前预览 HTML → 后端 Chromium PrintToPDF → 下载。
   *  与浏览器打印同源同引擎（同一份 HTML），纸张按模板 paperSize 结算。 */
  const handleServerPdf = async () => {
    const html = lastHtmlRef.current;
    const tpl = lastTemplateRef.current;
    if (!html || !tpl) return;
    try {
      let payloadTpl: any = {};
      try {
        payloadTpl = JSON.parse(tpl.payload || '{}');
      } catch {
        /* 保底 A4 */
      }
      const dims = getPaperDimensions(payloadTpl);
      const resp = await apiClient.printTemplateService.RenderPdf({
        html,
        filename: (tpl.code || 'print') + '-' + Date.now(),
        paperWidthMm: dims.width,
        paperHeightMm: dims.height,
        landscape: false,
        background: true,
      });
      // protojson：bytes 字段为 base64 字符串
      const bin = atob(resp.pdf as unknown as string);
      const bytes = new Uint8Array(bin.length);
      for (let i = 0; i < bin.length; i += 1) bytes[i] = bin.charCodeAt(i);
      const blob = new Blob([bytes], { type: 'application/pdf' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = resp.filename || 'print.pdf';
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      modal.error({ title: t('serverPdfFailed'), content: (e as Error).message });
    }
  };

  const node = (
    <Modal
      title={t('printTitle')}
      open={open}
      onCancel={() => setOpen(false)}
      width="78%"
      footer={
        <Space>
          {templates.length > 0 && (
            <Select
              value={selectedId}
              onChange={handleSelect}
              style={{ minWidth: 220 }}
              options={templates.map((item) => ({
                value: item.id as number,
                label: item.name || item.code || String(item.id),
              }))}
            />
          )}
          <Button
            type="primary"
            icon={<PrinterOutlined />}
            disabled={templates.length === 0}
            onClick={handlePrint}
          >
            {t('print')}
          </Button>
          <Button
            icon={<FilePdfOutlined />}
            disabled={templates.length === 0 || rendering}
            onClick={handleServerPdf}
          >
            {t('serverPdf')}
          </Button>
        </Space>
      }
    >
      {loading ? (
        <div style={{ padding: 32, textAlign: 'center' }}>
          <Spin />
        </div>
      ) : templates.length === 0 ? (
        <Empty description={t('noTemplate')} style={{ padding: 32 }}>
          <span style={{ color: '#999' }}>{t('noTemplateHint')}</span>
        </Empty>
      ) : (
        <div style={{ height: '64vh', background: '#f5f5f5', overflow: 'auto', position: 'relative' }}>
          {rendering && (
            <div style={{ position: 'absolute', inset: 0, zIndex: 2, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(255,255,255,.6)' }}>
              <Spin />
            </div>
          )}
          <iframe ref={frameRef} title="print-preview" style={{ width: '100%', height: '64vh', border: 'none' }} />
        </div>
      )}
    </Modal>
  );

  return { node, print };
}
