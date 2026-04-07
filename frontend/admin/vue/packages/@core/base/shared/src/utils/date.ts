import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(relativeTime);

const dateUtil = dayjs;

dayjs.tz.setDefault('Asia/Shanghai');
dateUtil.tz.setDefault('Asia/Shanghai');

export { dateUtil };

export function formatDate(time: number | string, format = 'YYYY-MM-DD') {
  if (time === null || time === undefined || time === '') {
    return '';
  }
  if (isDate(time)) {
    return dateUtil(time).format(format);
  }

  try {
    const date = dateUtil(time);
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
