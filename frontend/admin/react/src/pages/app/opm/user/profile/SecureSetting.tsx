import { useTranslation } from 'react-i18next';
import { Button, Listy } from 'antd';
import MfaManagement from './MfaManagement';

interface SecureSettingProps {
  onSwitchTab: (key: string) => void;
}

/* antd 6 弃用 List（提示改用 Listy），这里用 Listy + 自绘行还原原
   List.Item.Meta + actions 的观感：行距/配色按原 List 样式 token 写死
   （item padding 8px 0、标题下 4px、描述用 colorTextDescription、
   actions 与内容间距 24px），分隔线只在行间（末行无线）。 */
const SecureSetting = ({ onSwitchTab }: SecureSettingProps) => {
  const { t } = useTranslation('profile');

  const secureItems = [
    {
      key: 'password',
      title: t('secureSetting.password'),
      description: t('secureSetting.passwordDesc'),
    },
  ];

  return (
    <>
      <Listy
        items={secureItems}
        rowKey="key"
        styles={{ item: { padding: '8px 0', background: 'transparent' } }}
        itemRender={(item, index) => (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 24,
              borderBottom:
                index < secureItems.length - 1
                  ? '1px solid var(--ant-color-split)'
                  : undefined,
            }}
          >
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: 14,
                  color: 'var(--ant-color-text)',
                  marginBottom: 4,
                }}
              >
                {item.title}
              </div>
              <div style={{ fontSize: 14, color: 'var(--ant-color-text-description)' }}>
                {item.description}
              </div>
            </div>
            <Button
              type="link"
              style={{ flexShrink: 0 }}
              onClick={() => onSwitchTab('2')}
            >
              {t('secureSetting.modify')}
            </Button>
          </div>
        )}
      />
      <div style={{ marginTop: 16 }}>
        <MfaManagement />
      </div>
    </>
  );
};

export default SecureSetting;
