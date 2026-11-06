import { mergeAttributes, Node } from '@tiptap/core';

/**
 * P2 修复：校验 iframe src（对齐 react 端 sanitizeIframeSrc 口径）。
 * 允许同源相对路径或 https 绝对 URL；拒绝 javascript:/data:/http:/协议相对
 * `//` 等——富文本内容可能来自他人，`javascript:` src 会在编辑者浏览器执行。
 */
function sanitizeIframeSrc(raw: unknown): string {
  if (typeof raw !== 'string' || !raw) return '';
  const trimmed = raw.trim();
  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return trimmed;
  }
  try {
    const u = new URL(trimmed);
    if (u.protocol === 'https:') return trimmed;
  } catch {
    // 非合法 URL
  }
  return '';
}


/**
 * 自定义 Iframe 扩展
 * - 支持嵌入外部页面
 * - 自定义宽、高、标题、全屏属性
 */
export const CustomIframe = Node.create({
  name: 'iframe',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: null,
        parseHTML: (element) => sanitizeIframeSrc(element.getAttribute('src')) || null,
        renderHTML: (attributes) => {
          const safe = sanitizeIframeSrc(attributes.src);
          if (!safe) return {};
          return { src: safe };
        },
      },
      width: {
        default: '100%',
        parseHTML: (element) => element.style.width || '100%',
        renderHTML: (attributes) => {
          return { style: `width: ${attributes.width}` };
        },
      },
      height: {
        default: '500px',
        parseHTML: (element) => element.style.height || '500px',
        renderHTML: (attributes) => {
          return { style: `height: ${attributes.height}` };
        },
      },
      title: {
        default: null,
        parseHTML: (element) => element.getAttribute('title'),
        renderHTML: (attributes) => {
          if (!attributes.title) return {};
          return { title: attributes.title };
        },
      },
      allowfullscreen: {
        default: true,
        parseHTML: (element) => element.hasAttribute('allowfullscreen'),
        renderHTML: (attributes) => {
          if (!attributes.allowfullscreen) return {};
          return { allowfullscreen: 'allowfullscreen' };
        },
      },
    };
  },

  parseHTML() {
    return [{ tag: 'iframe' }];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'iframe',
      mergeAttributes(HTMLAttributes, {
        frameborder: '0',
        style:
          'max-width: 100%; border: 1px solid var(--border); border-radius: 6px; margin: 12px 0;',
      }),
    ];
  },

  addCommands() {
    return {
      setIframe:
        (options: {
          allowfullscreen?: boolean;
          height?: string;
          src: string;
          title?: string;
          width?: string;
        }) =>
        ({ commands }: { commands: any }) => {
          return commands.insertContent({ type: this.name, attrs: options });
        },
    } as any;
  },
});
