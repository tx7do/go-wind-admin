import dayjs from 'dayjs';

/**
 * 将日期选择器提交的值转为 proto Timestamp（google.protobuf.Timestamp）兼容的 RFC3339 字符串。
 * ProFormDateTimePicker 提交的是 'YYYY-MM-DD HH:mm:ss' 本地时间串，
 * protojson 只接受 RFC3339（如 2026-09-04T15:00:21.000Z），直接提交会 400。
 */
export function toProtoTimestamp(value?: string | null): string | undefined {
  if (!value) return undefined;
  const d = dayjs(value);
  return d.isValid() ? d.toISOString() : undefined;
}
