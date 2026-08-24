import { useState } from 'react';
import { Alert, Button, Card, Input, Modal, Spin, Tag, App } from 'antd';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import {
  useGetMfaStatus,
  useStartEnrollMfa,
  useConfirmEnrollMfa,
  useDisableMfa,
} from '@/api/hooks/mfa';

/**
 * MFA 管理：展示当前 TOTP 绑定状态，支持绑定（扫码 + 首码验证）与解绑。
 * 本轮仅 TOTP。
 */
const MfaManagement = () => {
  const { t } = useTranslation('profile');
  const { message, modal } = App.useApp();
  const queryClient = useQueryClient();

  const { data: statusData, isLoading } = useGetMfaStatus();
  const enrolledItems = statusData?.enrolled ?? [];
  const hasTotp = enrolledItems.some(
    (m) => m.method === 'TOTP' && m.enabled,
  );

  // 绑定流程状态
  const [enrollOpen, setEnrollOpen] = useState(false);
  const [qrUri, setQrUri] = useState<string>('');
  const [secret, setSecret] = useState<string>('');
  const [opId, setOpId] = useState<string>('');
  const [confirmCode, setConfirmCode] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const startEnroll = useStartEnrollMfa({
    onSuccess: (resp) => {
      const totpResult = resp.totp;
      if (totpResult) {
        setQrUri(totpResult.qrCodeDataUri ?? '');
        setSecret(totpResult.secret ?? '');
        setOpId(resp.operationId ?? '');
        setEnrollOpen(true);
      } else {
        message.error(t('mfa.enrollStartFailed'));
      }
    },
    onError: (err: Error) => message.error(err.message || t('mfa.enrollStartFailed')),
  });

  const confirmEnroll = useConfirmEnrollMfa({
    onSuccess: (resp) => {
      if (resp.success) {
        message.success(t('mfa.bindSuccess'));
        setEnrollOpen(false);
        setQrUri('');
        setSecret('');
        setOpId('');
        setConfirmCode('');
        queryClient.invalidateQueries({ queryKey: ['mfa-status'] });
      } else {
        message.error(t('mfa.bindFailed'));
      }
    },
    onError: (err: Error) => message.error(err.message || t('mfa.bindFailed')),
  });

  const disableMfa = useDisableMfa({
    onSuccess: () => {
      message.success(t('mfa.unbindSuccess'));
      queryClient.invalidateQueries({ queryKey: ['mfa-status'] });
    },
    onError: (err: Error) => message.error(err.message || t('mfa.unbindFailed')),
  });

  const handleStartEnroll = () => {
    setSubmitting(true);
    startEnroll.mutate({ method: 'TOTP' });
    setSubmitting(false);
  };

  const handleConfirmEnroll = async () => {
    if (!confirmCode || !opId) return;
    setSubmitting(true);
    try {
      await confirmEnroll.mutateAsync({
        method: 'TOTP',
        operationId: opId,
        totpCode: confirmCode,
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleUnbind = (credentialId: string) => {
    modal.confirm({
      title: t('mfa.unbindConfirmTitle'),
      content: t('mfa.unbindConfirmContent'),
      okText: t('mfa.unbind'),
      okType: 'danger',
      cancelText: t('mfa.cancel'),
      onOk: () => disableMfa.mutate({ credentialId: credentialId, method: 'TOTP' }),
    });
  };

  if (isLoading) {
    return <Spin />;
  }

  return (
    <div style={{ maxWidth: 540, marginTop: 32 }}>
      <h3 style={{ marginBottom: 16 }}>{t('mfa.title')}</h3>

      {hasTotp ? (
        <Alert
          type="success"
          showIcon
          message={t('mfa.bound')}
          description={
            <div>
              {enrolledItems
                .filter((m) => m.method === 'TOTP' && m.enabled)
                .map((m) => (
                  <Tag key={m.id} color="green">
                    {t('mfa.totpDevice')} #{m.id}
                  </Tag>
                ))}
              <div style={{ marginTop: 12 }}>
                {enrolledItems
                  .filter((m) => m.method === 'TOTP' && m.enabled)
                  .map((m) => (
                    <Button
                      key={m.id}
                      danger
                      onClick={() => handleUnbind(m.id)}
                      loading={disableMfa.isPending}
                    >
                      {t('mfa.unbind')}
                    </Button>
                  ))}
              </div>
            </div>
          }
        />
      ) : (
        <Alert
          type="warning"
          showIcon
          message={t('mfa.notBound')}
          description={
            <div>
              <p>{t('mfa.bindDesc')}</p>
              <Button
                type="primary"
                loading={startEnroll.isPending || submitting}
                onClick={handleStartEnroll}
              >
                {t('mfa.startBind')}
              </Button>
            </div>
          }
        />
      )}

      <Modal
        title={t('mfa.bindTitle')}
        open={enrollOpen}
        onCancel={() => {
          setEnrollOpen(false);
          setQrUri('');
          setSecret('');
          setOpId('');
          setConfirmCode('');
        }}
        footer={null}
        width={480}
      >
        <div style={{ textAlign: 'center' }}>
          <p>{t('mfa.scanQr')}</p>
          {qrUri && (
            <img src={qrUri} alt="TOTP QR" style={{ width: 240, height: 240, margin: '0 auto' }} />
          )}
          <p style={{ marginTop: 8, wordBreak: 'break-all' }}>
            {t('mfa.manualEntry')}: <code>{secret}</code>
          </p>
          <p style={{ marginTop: 16 }}>{t('mfa.enterCode')}</p>
          <Input
            value={confirmCode}
            onChange={(e) => setConfirmCode(e.target.value)}
            placeholder={t('mfa.codePlaceholder')}
            maxLength={6}
            style={{ maxWidth: 200, margin: '0 auto 16px', textAlign: 'center' }}
          />
          <div>
            <Button
              type="primary"
              loading={confirmEnroll.isPending || submitting}
              onClick={handleConfirmEnroll}
            >
              {t('mfa.confirmBind')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default MfaManagement;
