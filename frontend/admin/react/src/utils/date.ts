export function formatDate(dateString: string | undefined) {
    if (!dateString) return ''
    return new Date(dateString).toLocaleString()
}

// 格式化日期时间
export const formatDateTime = (timestamp?: any) => {
    if (!timestamp) return '-';
    try {
        let date: Date;
        if (timestamp.seconds) {
            date = new Date(timestamp.seconds * 1000);
        } else if (typeof timestamp === 'string') {
            date = new Date(timestamp);
        } else {
            return '-';
        }
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
        });
    } catch (error) {
        return '-';
    }
};
