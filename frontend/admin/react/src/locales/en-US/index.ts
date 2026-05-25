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

// 路径 → 命名空间
const toNs = (path: string): string | null => {
    const match = path.match(/\.\/(?:_core|_modules)\/(.+)\.json$/);
    return match ? match[1] : null;
};

const buildMap = (glob: Record<string, any>) =>
    Object.entries(glob).reduce((map, [path, data]) => {
        const ns = toNs(path);
        if (ns) map[ns] = data;
        return map;
    }, {} as Record<string, Record<string, any>>);

// 合并核心 + 扩展
const coreMap = buildMap(allModules);
const extMap = buildMap(extModules);

export const enUS = {
    ...coreMap,
    ...extMap,
} as const;

// 类型导出
export type EnUSResources = typeof enUS;

// 所有命名空间列表
export const allNamespaces = Object.keys(enUS);
