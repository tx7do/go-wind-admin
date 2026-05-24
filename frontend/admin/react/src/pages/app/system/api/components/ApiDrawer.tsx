import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import { DrawerForm, ProFormText, ProFormSelect } from '@ant-design/pro-components';
import { App } from 'antd';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { resourceservicev1_Api as Api } from '@/api/generated/admin/service/v1';
import { createApi, updateApi } from '@/api/service/api';

/**
 * HTTP 方法列表
 */
const methodList = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'HEAD', value: 'HEAD' },
  { label: 'OPTIONS', value: 'OPTIONS' },
];

interface ApiDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: Api;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * API 编辑/创建抽屉组件
 */
const ApiDrawer: React.FC<ApiDrawerProps> = ({ open, mode, data, onClose, onSuccess }) => {
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  // 创建 API
  const createMutation = useMutation({
    mutationFn: (req: { data: Api }) => createApi(req),
    onSuccess: () => {
      message.success('创建成功');
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || '创建失败');
    },
  });

  // 更新 API
  const updateMutation = useMutation({
    mutationFn: (req: { id: number; data: Api }) => updateApi(req),
    onSuccess: () => {
      message.success('更新成功');
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listApis'] });
    },
    onError: (error: Error) => {
      message.error(error.message || '更新失败');
    },
  });

  // 提交表单
  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);

    try {
      if (mode === 'create') {
        await createMutation.mutateAsync({ data: values });
      } else if (data?.id) {
        await updateMutation.mutateAsync({ id: data.id, data: values });
      }
    } finally {
      setConfirmLoading(false);
    }
  };

  return (
    <DrawerForm<Api>
      formRef={formRef}
      open={open}
      title={mode === 'create' ? '新建 API' : '编辑 API'}
      width={600}
      initialValues={mode === 'edit' ? data : undefined}
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: '确定',
          resetText: '取消',
        },
        submitButtonProps: {
          loading: confirmLoading || createMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: {
          onClick: onClose,
        },
      }}
      drawerProps={{
        destroyOnClose: true,
        onClose,
      }}
    >
      <ProFormText
        name="description"
        label="API描述"
        placeholder="请输入API描述"
        rules={[
          { required: false },
          { max: 200, message: '最多 200 个字符' },
        ]}
      />

      <ProFormText
        name="module"
        label="模块"
        placeholder="请输入模块名称"
        rules={[
          { required: false },
          { max: 100, message: '最多 100 个字符' },
        ]}
      />

      <ProFormText
        name="moduleDescription"
        label="模块描述"
        placeholder="请输入模块描述"
        rules={[
          { required: false },
          { max: 200, message: '最多 200 个字符' },
        ]}
      />

      <ProFormText
        name="path"
        label="API路径"
        placeholder="例如：/api/v1/users"
        rules={[
          { required: true, message: '请输入API路径' },
          { max: 500, message: '最多 500 个字符' },
        ]}
      />

      <ProFormSelect
        name="method"
        label="请求方法"
        placeholder="请选择请求方法"
        options={methodList}
        rules={[
          { required: true, message: '请选择请求方法' },
        ]}
        fieldProps={{
          showSearch: true,
          filterOption: (input, option) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
        }}
      />
    </DrawerForm>
  );
};

export default ApiDrawer;
