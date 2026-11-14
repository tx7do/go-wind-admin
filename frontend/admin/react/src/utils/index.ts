// 通用工具（merge/inference/tree/color/url/date/diff/nprogress 已移入 @go-wind/admin-react，
// 经包 barrel 导出）。本文件只保留含业务/宿主环境依赖的实现：
export * from './crypto'; // 口令加密（依赖 VITE_AES_KEY / VITE_PASSWORD_RSA_PUB，包内零 env 直读故留宿主）

export {default as cloneDeep} from "lodash.clonedeep";
