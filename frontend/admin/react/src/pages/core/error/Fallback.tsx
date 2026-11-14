import {useMemo} from 'react';
import {useNavigate} from 'react-router-dom';
import {Button} from 'antd';
import {ArrowLeftOutlined, ReloadOutlined} from '@ant-design/icons';

import type {FallbackProps} from './fallback';
import {useI18n} from '@go-wind/admin-react';
import {usePreferencesStore} from '@go-wind/admin-react';

// 导入图标组件
import {Icon401} from './icons/Icon401';
import Icon403 from './icons/Icon403';
import Icon404 from './icons/Icon404';
import Icon500 from './icons/Icon500';
import {IconComingSoon} from './icons/IconComingSoon';
import {IconOffline} from './icons/IconOffline';

// 导入样式
import './Fallback.css';

/**
 * Fallback 错误页面组件
 */
const Fallback = ({
    description = '',
    homePath = '/',
    image = '',
    status = 'coming-soon',
    title = '',
}: FallbackProps) => {
    const {t} = useI18n('common');
    const navigate = useNavigate();
    // 全局动效开关（设计语言 §2.7）：关闭时不播进场/悬浮动画
    const transitionEnabled = usePreferencesStore((state) => state.preferences.transition.enable);

    // 计算标题文本
    const titleText = useMemo(() => {
        if (title) {
            return title;
        }

        switch (status) {
            case '401':
                return t('fallback.unauthorized');
            case '403':
                return t('fallback.forbidden');
            case '404':
                return t('fallback.pageNotFound');
            case '500':
                return t('fallback.internalError');
            case 'coming-soon':
                return t('fallback.comingSoon');
            case 'offline':
                return t('fallback.offlineError');
            default:
                return '';
        }
    }, [title, status, t]);

    // 计算描述文本
    const descText = useMemo(() => {
        if (description) {
            return description;
        }
        switch (status) {
            case '401':
                return t('fallback.unauthorizedDesc');
            case '403':
                return t('fallback.forbiddenDesc');
            case '404':
                return t('fallback.pageNotFoundDesc');
            case '500':
                return t('fallback.internalErrorDesc');
            case 'offline':
                return t('fallback.offlineErrorDesc');
            default:
                return '';
        }
    }, [description, status, t]);

    // 计算要显示的图标组件
    const fallbackIcon = useMemo(() => {
        switch (status) {
            case '401':
                return <Icon401 />;
            case '403':
                return <Icon403 />;
            case '404':
                return <Icon404 />;
            case '500':
                return <Icon500 />;
            case 'coming-soon':
                return <IconComingSoon />;
            case 'offline':
                return <IconOffline />;
            default:
                return null;
        }
    }, [status]);

    // 是否显示返回按钮
    const showBack = status === '401' || status === '403' || status === '404';

    // 是否显示刷新按钮
    const showRefresh = status === '500' || status === 'offline';

    // 返回首页
    const handleBack = () => {
        navigate(homePath);
    };

    // 刷新页面
    const handleRefresh = () => {
        window.location.reload();
    };

    return (
        <div
            className="fallback-container"
            data-animate={transitionEnabled ? 'on' : 'off'}
        >
            {image ? (
                <img src={image} className="fallback-image" alt="error" />
            ) : fallbackIcon ? (
                <div className="fallback-image">{fallbackIcon}</div>
            ) : null}
            <div className="fallback-content">
                {titleText && <p className="fallback-title">{titleText}</p>}
                {descText && <p className="fallback-desc">{descText}</p>}
                {showBack && (
                    <Button
                        type="primary"
                        size="large"
                        icon={<ArrowLeftOutlined />}
                        onClick={handleBack}
                    >
                        {t('button.back')}
                    </Button>
                )}
                {showRefresh && (
                    <Button
                        type="primary"
                        size="large"
                        icon={<ReloadOutlined />}
                        onClick={handleRefresh}
                    >
                        {t('refresh')}
                    </Button>
                )}
            </div>
        </div>
    );
};

export default Fallback;
