import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(relativeTime);

const dateUtil = dayjs;

// 统一以上海时区展示时间。注意：setDefault 仅影响 dayjs.tz(input) 的解析，
// 不影响 dayjs(input)（按浏览器本地时区）。因此 formatDate 内必须使用
// dateUtil.tz(...) 才能让该默认时区真正生效，否则跨时区客户端会显示偏差。
const DEFAULT_TZ = 'Asia/Shanghai';
dayjs.tz.setDefault(DEFAULT_TZ);
dateUtil.tz.setDefault(DEFAULT_TZ);

export { dateUtil };

export function formatDate(time: number | string, format = 'YYYY-MM-DD') {
  if (time === null || time === undefined || time === '') {
    return '';
  }
  if (isDate(time)) {
    return dateUtil.tz(time, DEFAULT_TZ).format(format);
  }

  try {
    // 使用 tz 解析：后端返回的 UTC 时间（如 2026-08-06T12:00:00Z）
    // 会被统一转换为上海时区展示，避免随浏览器本地时区漂移。
    const date = dateUtil.tz(time, DEFAULT_TZ);
    if (!date.isValid()) {
      // throw new Error('Invalid date');
      return '';
    }
    return date.format(format);
  } catch (error) {
    console.error(`Error formatting date: ${error}`);
    return time;
  }
}

export function formatDateTime(time: number | string) {
  if (time === null || time === undefined || time === '') {
    return '';
  }
  return formatDate(time, 'YYYY-MM-DD HH:mm:ss');
}

export function isDate(value: any): value is Date {
  return value instanceof Date;
}

export function isDayjsObject(value: any): value is dayjs.Dayjs {
  return dateUtil.isDayjs(value);
}
