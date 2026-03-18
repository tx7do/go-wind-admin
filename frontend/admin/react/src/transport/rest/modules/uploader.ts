import type {AxiosRequestConfig, AxiosResponse} from 'axios';

import type {RequestClient} from '@/transport/rest';

class FileUploader {
    private client: RequestClient;

    constructor(client: RequestClient) {
        this.client = client;
    }

    public async upload(
        url: string,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        data: { file: Blob | File } & Record<string, any>,
        config?: AxiosRequestConfig,
    ): Promise<AxiosResponse> {
        const formData = new FormData();

        Object.entries(data).forEach(([key, value]) => {
            formData.append(key, value);
        });

        const finalConfig: AxiosRequestConfig = {
            ...config,
            headers: {
                'Content-Type': 'multipart/form-data',
                ...config?.headers,
            },
        };

        // 使用 request 方法代替 post 方法，避免类型推断问题
        return this.client.request(url, {
            method: 'POST',
            data: formData,
            ...finalConfig,
        });
    }
}

export {FileUploader};
