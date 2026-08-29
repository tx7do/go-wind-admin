import {useEffect, useState, useRef} from 'react';
import dayjs from 'dayjs';
import {useTranslation} from 'react-i18next';

// 时间格式随界面语言切换：中文“年月日”，其他语言用 ISO 风格
const timeFormat = (language: string | undefined) =>
    (language ?? '').startsWith('zh') ? 'YYYY年MM月DD日 HH:mm:ss' : 'YYYY-MM-DD HH:mm:ss';

/**
 * @description 获取本地时间
 */
export const useTimes = () => {
    const timer = useRef<NodeJS.Timeout | null>(null);
    const {i18n} = useTranslation();
    const [time, setTime] = useState(() => dayjs().format(timeFormat(i18n.language)));

    useEffect(() => {
        setTime(dayjs().format(timeFormat(i18n.language)));

        timer.current = setInterval(() => {
            setTime(dayjs().format(timeFormat(i18n.language)));
        }, 1000);

        return () => {
            if (timer.current) {
                clearInterval(timer.current);
                timer.current = null;
            }
        };
    }, [i18n.language]);

    return {
        time,
    };
};
