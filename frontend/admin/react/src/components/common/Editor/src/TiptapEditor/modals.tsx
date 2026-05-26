import { Modal, Input, Select, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { languages } from './constants';

// Code Block Modal
export interface CodeBlockModalProps {
  open: boolean;
  language: string;
  content: string;
  onOk: () => void;
  onCancel: () => void;
  onLanguageChange: (lang: string) => void;
  onContentChange: (content: string) => void;
}

export const CodeBlockModal: React.FC<CodeBlockModalProps> = ({
  open,
  language,
  content,
  onOk,
  onCancel,
  onLanguageChange,
  onContentChange,
}) => {
  const { t } = useTranslation();

  return (
    <Modal
      open={open}
      title={t('editor:insertCodeBlock', '插入代码块')}
      onOk={onOk}
      onCancel={onCancel}
      okText={t('common.confirm', '确定')}
      cancelText={t('common.cancel', '取消')}
      mask={{ closable: false }}
      width={600}
    >
      <div className="code-block-modal">
        <div className="modal-field">
          <label className="field-label">{t('editor:codeLanguage', '编程语言')}</label>
          <Select
            value={language}
            onChange={onLanguageChange}
            options={languages}
            style={{ width: '100%' }}
            showSearch
            placeholder={t('editor:selectLanguage', '选择语言')}
          />
        </div>
        <div className="modal-field">
          <label className="field-label">{t('editor:codeContent', '代码内容')}</label>
          <textarea
            value={content}
            onChange={(e) => onContentChange(e.target.value)}
            className="code-textarea"
            rows={10}
            placeholder={t('editor:codeContentPlaceholder', '请输入代码')}
          />
        </div>
      </div>
    </Modal>
  );
};

// Video Modal
export interface VideoModalProps {
  open: boolean;
  url: string;
  width: string;
  onOk: () => void;
  onCancel: () => void;
  onUrlChange: (url: string) => void;
  onWidthChange: (width: string) => void;
}

export const VideoModal: React.FC<VideoModalProps> = ({
  open,
  url,
  width,
  onOk,
  onCancel,
  onUrlChange,
  onWidthChange,
}) => {
  const { t } = useTranslation();

  return (
    <Modal
      open={open}
      title={t('editor:insertVideo', '插入视频')}
      onOk={onOk}
      onCancel={onCancel}
      okText={t('common.confirm', '确定')}
      cancelText={t('common.cancel', '取消')}
      mask={{ closable: false }}
      width={500}
    >
      <div className="video-modal">
        <div className="modal-field">
          <label className="field-label">{t('editor:videoUrl', '视频URL')}</label>
          <Input
            value={url}
            onChange={(e) => onUrlChange(e.target.value)}
            placeholder={t('editor:videoUrlPlaceholder', '请输入视频URL')}
            onPressEnter={onOk}
            allowClear
          />
        </div>
        <div className="modal-field">
          <label className="field-label">{t('editor:videoWidth', '宽度')}</label>
          <Select
            value={width}
            onChange={onWidthChange}
            options={[
              { value: '100%', label: '100%' },
              { value: '75%', label: '75%' },
              { value: '50%', label: '50%' },
              { value: '640px', label: '640px' },
              { value: '800px', label: '800px' },
            ]}
            style={{ width: '100%' }}
          />
        </div>
      </div>
    </Modal>
  );
};

// Iframe Modal
export interface IframeModalProps {
  open: boolean;
  url: string;
  width: string;
  height: string;
  title: string;
  allowFullscreen: boolean;
  onOk: () => void;
  onCancel: () => void;
  onUrlChange: (url: string) => void;
  onWidthChange: (width: string) => void;
  onHeightChange: (height: string) => void;
  onTitleChange: (title: string) => void;
  onAllowFullscreenChange: (allow: boolean) => void;
}

export const IframeModal: React.FC<IframeModalProps> = ({
  open,
  url,
  width,
  height,
  title,
  allowFullscreen,
  onOk,
  onCancel,
  onUrlChange,
  onWidthChange,
  onHeightChange,
  onTitleChange,
  onAllowFullscreenChange,
}) => {
  const { t } = useTranslation();

  return (
    <Modal
      open={open}
      title={t('editor:insertIframe', '插入Iframe')}
      onOk={onOk}
      onCancel={onCancel}
      okText={t('common.confirm', '确定')}
      cancelText={t('common.cancel', '取消')}
      mask={{ closable: false }}
      width={500}
    >
      <div className="iframe-modal">
        <div className="modal-field">
          <label className="field-label">{t('editor:iframeUrl', 'Iframe URL')}</label>
          <Input
            value={url}
            onChange={(e) => onUrlChange(e.target.value)}
            placeholder={t('editor:iframeUrlPlaceholder', '请输入URL')}
            onPressEnter={onOk}
            allowClear
          />
        </div>
        <div className="modal-field">
          <label className="field-label">{t('editor:iframeWidth', '宽度')}</label>
          <Select
            value={width}
            onChange={onWidthChange}
            options={[
              { value: '100%', label: '100%' },
              { value: '75%', label: '75%' },
              { value: '50%', label: '50%' },
              { value: '640px', label: '640px' },
              { value: '800px', label: '800px' },
            ]}
            style={{ width: '100%' }}
          />
        </div>
        <div className="modal-field">
          <label className="field-label">{t('editor:iframeHeight', '高度')}</label>
          <Select
            value={height}
            onChange={onHeightChange}
            options={[
              { value: '500px', label: '500px' },
              { value: '300px', label: '300px' },
              { value: '100%', label: '100%' },
            ]}
            style={{ width: '100%' }}
          />
        </div>
        <div className="modal-field">
          <label className="field-label">{t('editor:iframeTitle', '标题')}</label>
          <Input
            value={title}
            onChange={(e) => onTitleChange(e.target.value)}
            placeholder={t('editor:iframeTitlePlaceholder', '请输入标题')}
            allowClear
          />
        </div>
        <div className="modal-field">
          <label className="field-label">{t('editor:allowFullscreen', '允许全屏')}</label>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <Switch checked={allowFullscreen} onChange={onAllowFullscreenChange} />
            <span>
              {allowFullscreen
                ? t('editor:allowFullscreenEnabled', '已启用')
                : t('editor:allowFullscreenDisabled', '已禁用')}
            </span>
          </div>
        </div>
      </div>
    </Modal>
  );
};
