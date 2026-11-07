import { useState } from 'react';
import { App, Form, Input, Modal, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import type { printservicev1_PrintTemplate as PrintTemplate } from '@/api/generated/admin/service/v1';
import { useGeneratePrintTemplate } from '@/api/hooks/print-template';

interface AiGenerateModalProps {
  open: boolean;
  onClose: () => void;
  /** 生成成功（后端护栏校验通过）：把产物包装成伪模板交给设计器抽屉（ai-generate 模式） */
  onGenerated: (template: PrintTemplate) => void;
}

/** 描述文本截取为模板预填名（首行取 30 字符足够辨识，完整名用户在设计器里改） */
function nameFromDescription(description: string): string {
  const firstLine = description.split('\n')[0]?.trim() ?? '';
  return firstLine.length > 30 ? `${firstLine.slice(0, 30)}…` : firstLine;
}

/**
 * AI 一句话生成模板弹窗（M4-1）：描述单据要素 → 后端 LLM 产出 TemplateData
 * （护栏校验 + AI_TOKENS 配额在服务端）。产物不经本弹窗入库——确认交给设计器
 * 抽屉预览，用户点保存才走既有 Create 链。
 */
const AiGenerateModal: React.FC<AiGenerateModalProps> = ({ open, onClose, onGenerated }) => {
  const { t } = useTranslation('printTemplate');
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  const generateMutation = useGeneratePrintTemplate({
    onError: (error: Error) => {
      message.error(error.message || t('aiGenerateFailed'));
    },
  });

  const handleGenerate = async () => {
    try {
      await form.validateFields();
    } catch {
      return;
    }
    const description = (form.getFieldValue('description') as string || '').trim();
    if (!description) {
      message.warning(t('aiDescriptionRequired'));
      return;
    }
    const paperSize = form.getFieldValue('paperSize') || undefined;
    const orientation = form.getFieldValue('orientation') || undefined;
    const resourceKey = (form.getFieldValue('resourceKey') as string || '').trim() || undefined;

    setSubmitting(true);
    try {
      const resp = await generateMutation.mutateAsync({
        description,
        paperSize,
        orientation,
        resourceKey,
      });
      message.success(t('aiGenerateSuccess'));
      form.resetFields();
      onClose();
      onGenerated({
        name: nameFromDescription(description),
        payload: resp.payload,
        paperSize,
        orientation,
        resourceKey,
      } as PrintTemplate);
    } catch (e) {
      // 用户可见的错误提示已在 mutation 的 onError 出过；此处带出原始错误对象
      // 防静默吞错（本仓铁律 1）
      console.error('ai generate template failed:', e);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('aiGenerateTitle')}
      open={open}
      onCancel={onClose}
      onOk={handleGenerate}
      confirmLoading={submitting}
      okText={t('aiGenerate')}
      maskClosable={false}
      destroyOnHidden
      width={560}
    >
      <Form form={form} layout="vertical" initialValues={{ paperSize: 'A4', orientation: 'portrait' }}>
        <Form.Item
          name="description"
          label={t('aiDescription')}
          rules={[{ required: true, message: t('aiDescriptionRequired') }]}
        >
          <Input.TextArea
            rows={4}
            maxLength={2000}
            showCount
            placeholder={t('aiDescriptionPlaceholder')}
          />
        </Form.Item>
        <div style={{ display: 'flex', gap: 12 }}>
          <Form.Item name="paperSize" label={t('paperSize')} style={{ flex: 1 }}>
            <Select
              options={[
                { value: 'A4', label: 'A4' },
                { value: 'A5', label: 'A5' },
                { value: 'A3', label: 'A3' },
                { value: 'B5', label: 'B5' },
              ]}
            />
          </Form.Item>
          <Form.Item name="orientation" label={t('orientation')} style={{ flex: 1 }}>
            <Select
              options={[
                { value: 'portrait', label: t('portrait') },
                { value: 'landscape', label: t('landscape') },
              ]}
            />
          </Form.Item>
        </div>
        <Form.Item name="resourceKey" label={t('aiResourceKey')}>
          <Input maxLength={100} />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default AiGenerateModal;
