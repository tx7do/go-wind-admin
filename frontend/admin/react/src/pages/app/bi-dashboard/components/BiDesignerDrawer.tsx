import { useEffect, useMemo, useRef, useState } from 'react';
import { App, Button, Drawer, Form, Input, Space, Switch } from 'antd';
import { SaveOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { BiDesignerReact } from '@go-wind/bi-react';
import '@go-wind/bi-vue/style.css';
import type { BiDashboardSchema } from '@go-wind/bi-core';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';
import { useCreateBiDashboard, useUpdateBiDashboard } from '@/api/hooks/bi-dashboard';
import type { biservicev1_BiDashboard as BiDashboard } from '@/api/generated/admin/service/v1';

export type BiDesignerMode = 'create' | 'edit';

interface BiDesignerDrawerProps {
  open: boolean;
  mode: BiDesignerMode;
  /** edit 态的目标行（create 态忽略） */
  data?: BiDashboard;
  onClose: () => void;
  onSuccess: () => void;
}

/** payload 里冗余的列表列：保存时随 schema 一起提取 */
function extractRedundant(schema: Partial<BiDashboardSchema>) {
  return {
    chartCount: schema.charts?.length ?? 0,
    themeKey: schema.theme?.key ?? '',
  };
}

/**
 * BI 设计器抽屉（全屏）：上部基础信息，下部 BiDesignerReact 画布。
 * edit 态载入行 payload；保存整份覆盖 schema，edit 走 version 乐观锁。
 * 桥接壳的 initialSchema 是挂载时快照——画布节点 key 里带上行 id，
 * 换行编辑时强制重挂载。
 */
const BiDesignerDrawer: React.FC<BiDesignerDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('biDashboard');
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { guardClose, handleValuesChange } = useDrawerCloseGuard(open, onClose);
  const [form] = Form.useForm();

  const [submitting, setSubmitting] = useState(false);
  // 画布抛出的最新 schema（保存时取；变更即置脏）
  const schemaRef = useRef<BiDashboardSchema | null>(null);

  useEffect(() => {
    if (!open) return;
    if (mode === 'edit' && data) {
      form.setFieldsValue({
        name: data.name,
        code: data.code,
        remark: data.remark,
        isEnabled: data.isEnabled ?? true,
        isPublic: data.isPublic ?? false,
      });
    } else {
      form.resetFields();
    }
    schemaRef.current = null;
  }, [open, mode, data, form]);

  const initialSchema = useMemo<Partial<BiDashboardSchema> | undefined>(() => {
    if (mode !== 'edit' || !data?.payload) return undefined;
    try {
      return JSON.parse(data.payload);
    } catch (error) {
      console.error('[bi-designer-drawer] payload 解析失败，按空白画布打开', error);
      return undefined;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, data?.id]);

  const createMutation = useCreateBiDashboard({
    onSuccess: () => {
      message.success(t('createSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listBiDashboards'] });
      onSuccess();
    },
    onError: (error: Error) => message.error(error.message || t('createFailed')),
  });

  const updateMutation = useUpdateBiDashboard({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listBiDashboards'] });
      queryClient.invalidateQueries({ queryKey: ['biDashboard'] });
      onSuccess();
    },
    onError: (error: Error) => {
      // 后端乐观锁冲突：CONFLICT reason / 409
      if (error.message.includes('modified by another session') || error.message.includes('CONFLICT')) {
        message.error(t('saveConflict'));
      } else {
        message.error(error.message || t('updateFailed'));
      }
    },
  });

  const handleSave = async () => {
    const values = await form.validateFields();
    const schema = schemaRef.current;
    if (!schema) {
      message.warning(t('designerNotReady') || t('invalidSchema'));
      return;
    }
    const payload = JSON.stringify(schema);
    const redundant = extractRedundant(schema);

    setSubmitting(true);
    try {
      if (mode === 'edit' && data?.id) {
        await updateMutation.mutateAsync({
          id: data.id,
          values: {
            name: values.name,
            code: values.code,
            remark: values.remark,
            isEnabled: values.isEnabled ?? true,
            isPublic: values.isPublic ?? false,
            // 乐观锁：把打开时读到的版本回传，冲突由后端 409 + 上方提示兜住
            version: data.version ?? 1,
            payload,
            chartCount: redundant.chartCount,
            themeKey: redundant.themeKey,
          },
        });
      } else {
        await createMutation.mutateAsync({
          data: {
            name: values.name,
            code: values.code,
            kind: 'DASHBOARD',
            remark: values.remark,
            payload,
            chartCount: redundant.chartCount,
            themeKey: redundant.themeKey,
            isEnabled: values.isEnabled ?? true,
            isPublic: values.isPublic ?? false,
            isBuiltIn: false,
            sortOrder: 100,
          } as any,
        });
      }
    } finally {
      setSubmitting(false);
    }
  };

  const title = mode === 'edit'
    ? t('editTitle')
    : t('createTitle');

  return (
    <Drawer
      title={title}
      open={open}
      onClose={guardClose}
      width="100%"
      destroyOnHidden
      styles={{ body: { display: 'flex', flexDirection: 'column', overflow: 'hidden' } }}
      footer={
        <Space style={{ float: 'right' }}>
          <Button onClick={guardClose}>{t('common:button.cancel')}</Button>
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
      <Form form={form} layout="vertical" onValuesChange={handleValuesChange} style={{ flexShrink: 0 }}>
        <div style={{ display: 'flex', gap: 16, marginBottom: 12 }}>
          <Form.Item
            name="name"
            label={t('name')}
            style={{ flex: 1, minWidth: 180 }}
            rules={[{ required: true, message: `${t('name')}不能为空` }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="code"
            label={t('code')}
            style={{ flex: 1, minWidth: 180 }}
            rules={[{ required: true, message: `${t('code')}不能为空` }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="remark" label={t('remark')} style={{ flex: 2, minWidth: 220 }}>
            <Input />
          </Form.Item>
          <Form.Item name="isEnabled" label={t('isEnabled')} valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
          <Form.Item name="isPublic" label={t('isPublic')} valuePropName="checked" initialValue={false}>
            <Switch />
          </Form.Item>
        </div>
      </Form>

      {/* flex:1 吃满抽屉剩余高度（表单收缩、画布撑满），不再用 calc 魔法数 */}
      <div style={{ flex: 1, minHeight: 0 }}>
        <BiDesignerReact
          // key 带行 id：换行编辑强制重挂载（initialSchema 是挂载快照，不回灌）
          key={`${mode}-${data?.id ?? 'new'}`}
          demo={mode === 'create'}
          initialSchema={initialSchema}
          onChange={(schema) => {
            schemaRef.current = schema;
            handleValuesChange();
          }}
          style={{ height: '100%' }}
        />
      </div>
    </Drawer>
  );
};

export default BiDesignerDrawer;
