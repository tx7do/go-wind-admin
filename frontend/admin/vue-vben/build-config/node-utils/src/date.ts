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
