import { useRef, useState, useEffect } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormDigit,
  ProFormRadio,
} from '@ant-design/pro-components';
import { App, Select, Form, Table, Input, Button, Space, Spin } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import { PaginationQuery } from '@/core';
import { useCreateDictEntry, useUpdateDictEntry } from '@/api/hooks/dict';
import { fetchListDictTypes } from '@/api/hooks/dict';
import { fetchListLanguages } from '@/api/hooks/language';
import { enableBoolRadioOptions } from './constants';

interface I18nRow {
  key: string;
  languageCode: string;
  languageName: string;
  entryLabel: string;
  description: string;
}

interface DictEntryDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: any;
  typeId: number;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 字典条目编辑/创建抽屉组件
 * 含国际化（i18n）子表格
 */
const DictEntryDrawer: React.FC<DictEntryDrawerProps> = ({
  open,
  mode,
  data,
  typeId,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('dict-entry');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 字典类型下拉数据
  const [typeOptions, setTypeOptions] = useState<{ label: string; value: number }[]>([]);

  // i18n 子表格
  const [i18nData, setI18nData] = useState<I18nRow[]>([]);
  const [i18nLoading, setI18nLoading] = useState(false);
  const [editingKey, setEditingKey] = useState<string>('');

  // 加载字典类型下拉数据
  useEffect(() => {
    if (open) {
      fetchListDictTypes(new PaginationQuery({ formValues: { is_enabled: 'true' } }))
        .then((res) => {
          const options = (res.items || []).map((item: any) => ({
            label: item.typeName,
            value: item.id,
          }));
          setTypeOptions(options);
        })
        .catch(() => setTypeOptions([]));
    }
  }, [open]);

  // 编辑模式下设置表单值
  useEffect(() => {
    if (open && mode === 'edit' && data) {
      setTimeout(() => {
        formRef.current?.setFieldsValue({
          typeId: data.typeId || typeId,
          entryValue: data.entryValue || '',
          numericValue: data.numericValue,
          sortOrder: data.sortOrder ?? 1,
          isEnabled: data.isEnabled ?? true,
        });
      }, 0);
    }
  }, [open, mode, data, typeId]);

  // 加载 i18n 数据（打开时加载语言列表）
  useEffect(() => {
    if (open) {
      setI18nLoading(true);
      fetchListLanguages(new PaginationQuery({ orderBy: ['sortOrder'] }))
        .then((res) => {
          const i18nMap = (data?.i18n ?? {}) as Record<string, { description?: string; entryLabel?: string }>;
          const rows: I18nRow[] = (res.items || [])
            .filter((lang: any) => lang.languageCode)
            .map((lang: any) => ({
              key: lang.languageCode,
              languageCode: lang.languageCode,
              languageName: lang.nativeName || '',
              entryLabel: i18nMap[lang.languageCode]?.entryLabel ?? '',
              description: i18nMap[lang.languageCode]?.description ?? '',
            }));
          setI18nData(rows);
        })
        .catch(() => setI18nData([]))
        .finally(() => setI18nLoading(false));
    }
  }, [open, data?.i18n]);

  // 创建 mutation
  const createMutation = useCreateDictEntry({
    onSuccess: () => {
      message.success(t('createSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listDictEntries'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  // 更新 mutation
  const updateMutation = useUpdateDictEntry({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listDictEntries'] });
      onSuccess();
      onClose();
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  // 提交表单
  const handleSubmit = async (values: Record<string, any>) => {
    try {
      setConfirmLoading(true);

      // 构建 i18n map
      const i18nMap: Record<string, any> = {};
      i18nData.forEach((row) => {
        if (row.languageCode) {
          i18nMap[row.languageCode] = {
            languageCode: row.languageCode,
            entryLabel: row.entryLabel,
            description: row.description,
          };
        }
      });

      const payload = {
        ...values,
        i18n: i18nMap,
      };

      if (mode === 'edit' && data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values: payload });
      } else {
        await createMutation.mutateAsync({ data: payload });
      }
      return true;
    } catch {
      return false;
    } finally {
      setConfirmLoading(false);
    }
  };

  // i18n 表格可编辑单元格
  const isEditing = (record: I18nRow) => record.key === editingKey;

  const edit = (record: I18nRow) => {
    setEditingKey(record.key);
  };

  const cancel = () => {
    setEditingKey('');
  };

  const save = (_key: string) => {
    setEditingKey('');
  };

  const updateI18nRow = (key: string, field: string, value: string) => {
    setI18nData((prev) =>
      prev.map((row) => (row.key === key ? { ...row, [field]: value } : row)),
    );
  };

  // i18n 表格列定义
  const i18nColumns = [
    {
      title: t('languageCode'),
      dataIndex: 'languageCode',
      width: 100,
    },
    {
      title: t('languageName'),
      dataIndex: 'languageName',
      width: 100,
    },
    {
      title: t('entryLabel'),
      dataIndex: 'entryLabel',
      render: (_: any, record: I18nRow) => {
        if (isEditing(record)) {
          return (
            <Input
              value={record.entryLabel}
              onChange={(e) => updateI18nRow(record.key, 'entryLabel', e.target.value)}
              onPressEnter={() => save(record.key)}
              size="small"
            />
          );
        }
        return record.entryLabel || '-';
      },
    },
    {
      title: t('description'),
      dataIndex: 'description',
      render: (_: any, record: I18nRow) => {
        if (isEditing(record)) {
          return (
            <Input
              value={record.description}
              onChange={(e) => updateI18nRow(record.key, 'description', e.target.value)}
              onPressEnter={() => save(record.key)}
              size="small"
            />
          );
        }
        return record.description || '-';
      },
    },
    {
      title: t('action'),
      dataIndex: 'action',
      width: 120,
      render: (_: any, record: I18nRow) => {
        if (isEditing(record)) {
          return (
            <Space size={4}>
              <Button type="link" size="small" onClick={() => save(record.key)}>
                {t('common:button.ok')}
              </Button>
              <Button type="link" size="small" onClick={cancel}>
                {t('common:button.cancel')}
              </Button>
            </Space>
          );
        }
        return (
          <Button type="link" size="small" onClick={() => edit(record)}>
            {t('common:button.edit')}
          </Button>
        );
      },
    },
  ];

  return (
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? t('create') : t('edit')}
      open={open}
      onOpenChange={(visible) => {
        if (!visible) {
          formRef.current?.resetFields();
          setEditingKey('');
          onClose();
        }
      }}
      initialValues={{
        typeId,
        sortOrder: 1,
        isEnabled: true,
      }}
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading || createMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: { onClick: onClose },
      }}
      drawerProps={{ destroyOnClose: true, onClose, size: 800 }}
    >
      {/* 所属类型 - Select */}
      <Form.Item
        name="typeId"
        label={t('typeId')}
        rules={[{ required: true, message: t('requiredTypeId') }]}
      >
        <Select
          options={typeOptions}
          placeholder={t('typeIdPlaceholder')}
          allowClear={false}
          showSearch
        />
      </Form.Item>

      <ProFormText
        name="entryValue"
        label={t('entryValue')}
        placeholder={t('entryValuePlaceholder')}
        rules={[{ required: true, message: t('requiredEntryValue') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormDigit
        name="numericValue"
        label={t('numericValue')}
        placeholder={t('numericValuePlaceholder')}
        fieldProps={{ precision: 0 }}
      />

      <ProFormDigit
        name="sortOrder"
        label={t('sortOrder')}
        placeholder={t('sortOrderPlaceholder')}
        fieldProps={{ precision: 0, min: 0 }}
      />

      <ProFormRadio.Group
        name="isEnabled"
        label={t('status')}
        rules={[{ required: true, message: t('requiredStatus') }]}
        options={enableBoolRadioOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />

      {/* 国际化子表格 */}
      <div style={{ marginBottom: 24 }}>
        <label style={{ display: 'block', marginBottom: 8, color: 'var(--ant-color-text)' }}>
          {t('i18nTitle')}
        </label>
        <Spin spinning={i18nLoading}>
          <Table
            dataSource={i18nData}
            columns={i18nColumns}
            size="small"
            pagination={false}
            bordered
            scroll={{ y: 300 }}
            style={{
              border: '1px solid var(--ant-color-border)',
              borderRadius: 6,
            }}
          />
        </Spin>
      </div>
    </DrawerForm>
  );
};

export default DictEntryDrawer;
