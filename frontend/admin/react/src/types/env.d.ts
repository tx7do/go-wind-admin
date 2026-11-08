/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_API_URL: string
    readonly VITE_SSE_URL: string

    readonly VITE_APP_TITLE: string
    readonly VITE_APP_NAMESPACE: string
    readonly VITE_APP_VERSION: string

    readonly VITE_AES_KEY: string

    readonly VITE_ENV: string

    /** 错误上报收集端点（留空 = 未接入监控，见 ErrorFallback/utils/report.ts） */
    readonly VITE_ERROR_REPORT_ENDPOINT: string

    readonly VITE_MOCK: boolean
    readonly VITE_ANALYZE: boolean
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}

// React Router handle 类型增强
declare module 'react-router' {
    interface RouteMatch {
        handle: RouteHandle;
    }
}

// 兼容 process.env
declare namespace NodeJS {
    interface ProcessEnv extends ImportMetaEnv {
    }
}
