import { useEffect, useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { useCreateAiKnowledgeBase, useUpdateAiKnowledgeBase } from '@/api/hooks/ai-knowledge';
import { fetchListAiProviders } from '@/api/hooks/ai-provider';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';

interface KnowledgeDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: AiKnowledgeBase;
  onClose: () => void;
  onSuccess: () => void;
}

export default function KnowledgeDrawer({ open, mode, data, onClose, onSuccess }: KnowledgeDrawerProps) {
  const { t } = useTranslation('aiKnowledge');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const formRef = useRef<ProFormInstance>(null);
  const [providerOptions, setProviderOptions] = useState<{ label: string; value: string }[]>([]);

  // 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
  const { guardClose, handleValuesChange } = useDrawerCloseGuard(open, onClose);

  const createMutation = useCreateAiKnowledgeBase();
  const updateMutation = useUpdateAiKnowledgeBase();

  // provider 下拉（启用项）
  useEffect(() => {
    if (!open) return;
    fetchListAiProviders(new PaginationQuery({ paging: { page: 1, pageSize: 100 }, formValues: { isEnabled: true } }))
      .then((res) => {
        setProviderOptions(
          (res.items ?? []).map((p) => ({ label: `${p.name ?? ''} / ${p.modelName ?? ''}`, value: p.id! })),
        );
      })
      .catch((error: Error) => console.error('fetch providers for knowledge drawer failed:', error));
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const timer = setTimeout(() => {
      if (mode === 'edit' && data) {
        formRef.current?.setFieldsValue({
          name: data.name,
          description: data.description,
          providerId: data.providerId,
          embeddingModel: data.embeddingModel,
        });
      }
    }, 0);
    return () => clearTimeout(timer);
  }, [open, mode, data]);

  const handleSubmit = async (values: Record<string, any>) => {
    try {
      if (mode === 'create') {
        await createMutation.mutateAsync({ data: values as any });
        message.success(t('createSuccess'));
      } else {
        await updateMutation.mutateAsync({ id: data!.id!, values });
        message.success(t('updateSuccess'));
      }
      queryClient.invalidateQueries({ queryKey: ['listAiKnowledgeBases'] });
      onSuccess();
      onClose();
      return true;
    } catch (error) {
      console.error('save ai knowledge base failed:', error);
      message.error((error as Error).message || t('fetchFailed'));
      return false;
    }
  };

  return (
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? t('common:modal.create', { moduleName: t('moduleName') }) : t('common:modal.update', { moduleName: t('moduleName') })}
      open={open}
      onValuesChange={handleValuesChange}
      onFinish={handleSubmit}
      submitter={{ submitButtonProps: { loading: createMutation.isPending || updateMutation.isPending }, resetButtonProps: { onClick: guardClose } }}
      drawerProps={{ destroyOnHidden: true, onClose: guardClose, size: 600 }}
      layout="horizontal"
      labelCol={{ flex: '0 0 120px' }}
      wrapperCol={{ flex: 'auto' }}
    >
      <ProFormText
        name="name"
        label={t('name')}
        rules={[{ required: true, message: t('requiredName') }]}
        placeholder={t('namePlaceholder')}
      />
      <ProFormTextArea
        name="description"
        label={t('description')}
        fieldProps={{ rows: 2 }}
      />
      <ProFormSelect
        name="providerId"
        label={t('provider')}
        options={providerOptions}
        rules={[{ required: true, message: t('requiredProvider') }]}
        tooltip={t('providerTooltip')}
      />
      <ProFormText
        name="embeddingModel"
        label={t('embeddingModel')}
        rules={[{ required: true, message: t('requiredEmbeddingModel') }]}
        placeholder="text-embedding-3-small / bge-m3"
      />
    </DrawerForm>
  );
}
