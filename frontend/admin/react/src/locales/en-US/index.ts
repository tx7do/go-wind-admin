/* eslint-disable @typescript-eslint/no-explicit-any */
// 一次性加载 _core + _modules 下所有 JSON（eager: true 静态）
const allModules = import.meta.glob<Record<string, any>>(
    './_core/*.json',
    { eager: true, import: 'default' },
);

const extModules = import.meta.glob<Record<string, any>>(
    './_modules/*.json',
    { eager: true, import: 'default' },
);

// 路由标题按域拆分在 _core/routes/<域>.json（与 router/modules/ 一一对应），
// 这里合并回单一 routes 命名空间——新增模块只需新增文件，不改共享 routes.json
const routeFiles = import.meta.glob<Record<string, any>>(
    './_core/routes/*.json',
    { eager: true, import: 'default' },
);

// 路径 → 命名空间
const toNs = (path: string): string | null => {
    const match = path.match(/\.\/(?:_core|_modules)\/(.+)\.json$/);
    return match ? match[1] : null;
};

const buildMap = (glob: Record<string, any>) =>
    Object.entries(glob).reduce((map, [path, data]) => {
        const ns = toNs(path);
        if (ns) map[ns] = data?.default ?? data;
        return map;
    }, {} as Record<string, Record<string, any>>);

// 合并核心 + 扩展
const coreMap = buildMap(allModules);
coreMap['routes'] = Object.values(routeFiles).reduce<Record<string, any>>(
    (acc, data) => Object.assign(acc, data?.default ?? data),
    {},
);
const extMap = buildMap(extModules);

export const enUS = {
    ...coreMap,
    ...extMap,
} as const;

// 类型导出
export type EnUSResources = typeof enUS;

// 所有命名空间列表
export const allNamespaces = Object.keys(enUS);
