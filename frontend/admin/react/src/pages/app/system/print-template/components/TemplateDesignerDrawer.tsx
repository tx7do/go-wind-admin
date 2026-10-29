import { useEffect, useRef, useState } from 'react';
import {
  Button,
  Col,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  App,
} from 'antd';
import { EyeOutlined, PrinterOutlined, SaveOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
// registerAll 必须在任何 mount 之前执行：未升级元素上的 property 赋值会以自有属性
// 遮蔽升级后的类访问器（Custom Element 经典坑，docs/print_template.md §8.1）。
import { registerAll, createDefaultTemplate } from '@go-wind/print-common';
import { renderHtmlPages, browserCodeRenderer } from '@go-wind/print-core/browser';
import { writeHtmlToFrame } from './previewRender';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';
import { useCreatePrintTemplate, useUpdatePrintTemplate } from '@/api/hooks/print-template';
import { usePreferencesStore } from '@/core/preferences';
import { applyPrintDesignerTheme } from '../printDesignerTheme';
import {
  extractRedundant,
  fieldsFromSample,
  templateTitle,
  type PrintTemplateMode,
} from '../constants';
import type { printservicev1_PrintTemplate as PrintTemplate } from '@/api/generated/admin/service/v1';

// 模块加载即注册（全应用一次）；import 求值先于组件渲染，天然先于 mount。
registerAll();

interface TemplateDesignerDrawerProps {
  open: boolean;
  mode: PrintTemplateMode;
  data?: PrintTemplate;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 打印模板设计器抽屉：上部基础信息 + 示例数据，下部 go-wind-print 设计器画布。
 * 画布在 Shadow DOM 内自成样式体系，不受三端主题管辖（隔离是特性）。
 */
const TemplateDesignerDrawer: React.FC<TemplateDesignerDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('printTemplate');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [form] = Form.useForm();

  // 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
  const { guardClose, markDirty } = useDrawerCloseGuardForDesigner(open, onClose);

  const designerRef = useRef<any>(null);
  const previewFrameRef = useRef<HTMLIFrameElement>(null);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // 暗色主题：preferences.theme.mode（auto 跟系统）；CSS 变量穿透 shadow 边界，宿主覆盖即生效
  const themeMode = usePreferencesStore((s) => s.preferences.theme.mode);
  const isDark =
    themeMode === 'dark' ||
    (themeMode !== 'light' && window.matchMedia('(prefers-color-scheme: dark)').matches);

  useEffect(() => {
    applyPrintDesignerTheme(designerRef.current, isDark);
  }, [isDark, open]);

  const createMutation = useCreatePrintTemplate({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listPrintTemplates'] });
    },
    onError: (error: Error) => message.error(error.message || t('createFailed')),
  });

  const updateMutation = useUpdatePrintTemplate({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listPrintTemplates'] });
    },
    onError: (error: Error) => message.error(error.message || t('updateFailed')),
  });

  // 打开时回填表单 + 注入画布（注入走 effect，不置脏）
  useEffect(() => {
    if (!open) return;
    const initial =
      mode === 'create'
        ? { isEnabled: true, sortOrder: 100 }
        : {
            name: mode === 'copy' ? `${data?.name || ''} ${t('copySuffix')}` : data?.name,
            code: mode === 'copy' ? undefined : data?.code,
            resourceKey: data?.resourceKey || undefined,
            paperSize: data?.paperSize || 'A4',
            orientation: data?.orientation || 'portrait',
            isEnabled: data?.isEnabled ?? true,
            sortOrder: data?.sortOrder ?? 100,
          };
    form.setFieldsValue(initial);

      // 等自定义元素挂载完成再注入（对象入参必须走 property，不能写 JSX 属性）
    const timer = window.setTimeout(() => {
      const el = designerRef.current;
      if (!el) return;
      applyPrintDesignerTheme(el, isDark);
      let tpl: unknown;
      try {
        tpl = mode === 'create' ? createDefaultTemplate() : JSON.parse(data?.payload || '');
      } catch {
        tpl = createDefaultTemplate();
      }
      el.template = tpl;
      el.fields = fieldsFromSample(parseSample(form.getFieldValue('sampleData')));
    }, 120);
    return () => window.clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, mode]);

  const currentTemplateJson = (): string | null => {
    const el = designerRef.current;
    if (!el) return null;
    try {
      return el.getTemplateJson();
    } catch {
      return null;
    }
  };

  const currentSample = (): Record<string, unknown> => parseSample(form.getFieldValue('sampleData'));

  /** 打开预览：两遍渲染（测量→分页→HTML）写入同源 iframe */
  const handlePreview = async () => {
    const json = currentTemplateJson();
    if (!json) {
      message.warning(t('designerNotReady'));
      return;
    }
    setPreviewOpen(true);
    setPreviewLoading(true);
    try {
      const result = await renderHtmlPages(
        JSON.parse(json),
        currentSample(),
        window.location.origin,
        browserCodeRenderer,
      );
      // 等 iframe 挂载完成再写文档
      window.setTimeout(() => {
        writeHtmlToFrame(previewFrameRef.current, result.html);
        setPreviewLoading(false);
      }, 80);
    } catch (e) {
      setPreviewLoading(false);
      message.error((e as Error).message || t('renderFailed'));
    }
  };

  const handlePreviewPrint = () => {
    previewFrameRef.current?.contentWindow?.print();
  };

  /** 保存：画布 JSON → payload（冗余列同步提取），复制/新建走 Create、编辑走 Update */
  const handleSave = async () => {
    const values = await form.validateFields();
    const json = currentTemplateJson();
    if (!json) {
      message.warning(t('designerNotReady'));
      return;
    }
    const redundant = extractRedundant(json);

    setSubmitting(true);
    try {
      if (mode === 'edit' && data?.id) {
        await updateMutation.mutateAsync({
          id: data.id,
          values: {
            name: values.name,
            code: values.code,
            resourceKey: values.resourceKey || '',
            payload: json,
            paperSize: redundant.paperSize || values.paperSize,
            orientation: redundant.orientation || values.orientation,
            isEnabled: values.isEnabled ?? true,
            sortOrder: values.sortOrder ?? 100,
          },
        });
      } else {
        await createMutation.mutateAsync({
          data: {
            name: values.name,
            code: values.code,
            resourceKey: values.resourceKey || '',
            payload: json,
            paperSize: redundant.paperSize || values.paperSize,
            orientation: redundant.orientation || values.orientation,
            isEnabled: values.isEnabled ?? true,
            sortOrder: values.sortOrder ?? 100,
            isBuiltIn: false,
          } as any,
        });
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Drawer
        title={templateTitle(t, mode)}
        open={open}
        onClose={guardClose}
        width="100%"
        destroyOnHidden
        footer={
          <Space style={{ float: 'right' }}>
            <Button onClick={guardClose}>{t('common:button.cancel')}</Button>
            <Button icon={<EyeOutlined />} onClick={handlePreview}>
              {t('preview')}
            </Button>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={submitting}
              onClick={handleSave}
            >
              {t('common:button.save')}
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical" onValuesChange={markDirty}>
          <Row gutter={16}>
            <Col span={6}>
              <Form.Item
                name="name"
                label={t('name')}
                rules={[{ required: true, message: `${t('name')}${t('requiredSuffix')}` }]}
              >
                <Input placeholder={t('name')} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="code"
                label={t('code')}
                rules={[{ required: true, message: `${t('code')}${t('requiredSuffix')}` }]}
              >
                <Input placeholder="biz.order.a4" disabled={mode === 'edit'} />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item name="paperSize" label={t('paperSize')}>
                <Select
                  options={['A4', 'A3', 'A5', 'Letter', 'Legal'].map((v) => ({ value: v, label: v }))}
                />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item name="orientation" label={t('orientation')}>
                <Select
                  options={[
                    { value: 'portrait', label: t('orientation.portrait') },
                    { value: 'landscape', label: t('orientation.landscape') },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={2}>
              <Form.Item name="isEnabled" label={t('isEnabled')} valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col span={2}>
              <Form.Item name="sortOrder" label={t('sortOrder')}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="resourceKey" label={t('resourceKey')}>
                <Input placeholder="system/user" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="sampleData"
                label={t('sampleData')}
                tooltip={t('sampleDataTip')}
              >
                <Input.TextArea
                  rows={2}
                  placeholder='{"order":{"no":"SO-001","items":[{"sku":"A1","qty":2}]}}'
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>

        {/* 画布容器：任何指针交互视为脏（Shadow DOM 内的编辑以捕获态兜住） */}
        <div
          style={{ border: '1px solid #d9d9d9', borderRadius: 8, overflow: 'hidden', height: 'calc(100vh - 320px)', minHeight: 420 }}
          onPointerDownCapture={markDirty}
          onKeyDownCapture={markDirty}
        >
          {open && <print-designer ref={designerRef} is-edit="true" show-help="false" style={{ display: 'block', height: '100%' }} />}
        </div>
      </Drawer>

      <Modal
        title={t('previewTitle')}
        open={previewOpen}
        onCancel={() => setPreviewOpen(false)}
        width="82%"
        footer={
          <Space>
            <Button onClick={() => setPreviewOpen(false)}>{t('common:button.close')}</Button>
            <Button type="primary" icon={<PrinterOutlined />} onClick={handlePreviewPrint}>
              {t('print')}
            </Button>
          </Space>
        }
      >
        <div style={{ height: '68vh', background: '#f5f5f5', overflow: 'auto' }}>
          {previewLoading && <div style={{ padding: 24 }}>{t('rendering')}</div>}
          <iframe
            ref={previewFrameRef}
            title="print-preview"
            style={{ width: '100%', height: '68vh', border: 'none' }}
          />
        </div>
      </Modal>
    </>
  );
};

function parseSample(text?: string): Record<string, unknown> {
  if (!text || !text.trim()) return {};
  try {
    const v = JSON.parse(text);
    return v && typeof v === 'object' && !Array.isArray(v) ? v : {};
  } catch {
    return {};
  }
}

/** 设计器版关闭守卫：复用通用守卫的脏标记与确认框语义，
 *  但暴露 markDirty 给画布的指针/键盘捕获态使用（表单侧走 Form onValuesChange）。 */
function useDrawerCloseGuardForDesigner(open: boolean, onClose: () => void) {
  const guard = useDrawerCloseGuard(open, onClose);
  return { guardClose: guard.guardClose, markDirty: guard.handleValuesChange };
}

export default TemplateDesignerDrawer;
