// framework/ 已并入单包（原 packages/* 26 个 workspace 包的源码），
// 但 @vue/compiler-sfc 等非 vite 链路的模块解析不读 vite alias / tsconfig paths，
// 仍按 node 规则从源文件向上找 node_modules。此脚本在 framework/node_modules
// 下按原包名重建 junction，为这类解析提供兼容层。pnpm install 后自动执行。
import { existsSync, mkdirSync, rmSync, symlinkSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const nm = join(root, 'framework', 'node_modules');

// 原包名 → framework 下现路径（与 vite alias / tsconfig paths 的映射一致）
const links = {
  '@vben-core/shared': 'base/shared',
  '@vben-core/typings': 'base/typings',
  '@vben-core/icons': 'base/icons',
  '@vben-core/design': 'base/design',
  '@vben-core/composables': 'core/composables',
  '@vben-core/preferences': 'core/preferences',
  '@vben-core/shadcn-ui': 'ui-kit/shadcn-ui',
  '@vben-core/form-ui': 'ui-kit/form-ui',
  '@vben-core/popup-ui': 'ui-kit/popup-ui',
  '@vben-core/menu-ui': 'ui-kit/menu-ui',
  '@vben-core/layout-ui': 'ui-kit/layout-ui',
  '@vben-core/tabs-ui': 'ui-kit/tabs-ui',
  '@vben/access': 'effects/access',
  '@vben/common-ui': 'effects/common-ui',
  '@vben/hooks': 'effects/hooks',
  '@vben/layouts': 'effects/layouts',
  '@vben/plugins': 'effects/plugins',
  '@vben/request': 'effects/request',
  '@vben/constants': 'constants',
  '@vben/icons': 'icons',
  '@vben/locales': 'locales',
  '@vben/preferences': 'preferences',
  '@vben/stores': 'stores',
  '@vben/styles': 'styles',
  '@vben/types': 'types',
  '@vben/utils': 'utils',
};

let created = 0;
let skipped = 0;
for (const [name, target] of Object.entries(links)) {
  const linkPath = join(nm, name);
  const realTarget = join(root, 'framework', target);
  if (!existsSync(realTarget)) {
    throw new Error(`framework target missing: ${realTarget}`);
  }
  if (existsSync(linkPath)) {
    skipped++;
    continue;
  }
  mkdirSync(dirname(linkPath), { recursive: true });
  // junction：Windows 免管理员权限；非 Windows 用目录 symlink
  symlinkSync(realTarget, linkPath, process.platform === 'win32' ? 'junction' : 'dir');
  created++;
}
console.log(
  `[rebuild-framework-links] total=${Object.keys(links).length} created=${created} exists=${skipped}`,
);
