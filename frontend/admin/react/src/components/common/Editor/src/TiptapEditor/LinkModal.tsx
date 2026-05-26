import { Modal, Input } from 'antd';
import { useTranslation } from 'react-i18next';

interface LinkModalProps {
  open: boolean;
  url: string;
  onOk: () => void;
  onCancel: () => void;
  onUrlChange: (url: string) => void;
}

export const LinkModal: React.FC<LinkModalProps> = ({ open, url, onOk, onCancel, onUrlChange }) => {
  const { t } = useTranslation();

  return (
    <Modal
      open={open}
      title={t('editor:insert_url', '插入链接')}
      onOk={onOk}
      onCancel={onCancel}
      okText={t('common.confirm', '确定')}
      cancelText={t('common.cancel', '取消')}
      mask={{ closable: false }}
    >
      <Input
        value={url}
        onChange={(e) => onUrlChange(e.target.value)}
        placeholder={t('editor:input_url', '请输入URL')}
        onPressEnter={onOk}
        allowClear
      />
    </Modal>
  );
};
