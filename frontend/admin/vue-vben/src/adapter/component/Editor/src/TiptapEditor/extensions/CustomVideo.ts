import { mergeAttributes, Node } from '@tiptap/core';

/**
 * 自定义视频扩展
 * - 支持视频插入、拖拽
 * - 自定义宽高属性
 */
export const CustomVideo = Node.create({
  name: 'video',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: null,
        parseHTML: (element) => element.getAttribute('src'),
        renderHTML: (attributes) => {
          if (!attributes.src) return {};
          return { src: attributes.src };
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
        default: 'auto',
        parseHTML: (element) => element.style.height || 'auto',
        renderHTML: (attributes) => {
          return { style: `height: ${attributes.height}` };
        },
      },
    };
  },

  parseHTML() {
    return [{ tag: 'video' }];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'video',
      mergeAttributes(HTMLAttributes, {
        controls: 'controls',
        style: 'max-width: 100%; height: auto;',
      }),
    ];
  },

  addCommands() {
    return {
      setVideo:
        (options: { height?: string; src: string; width?: string }) =>
        ({ commands }: { commands: any }) => {
          return commands.insertContent({ type: this.name, attrs: options });
        },
    } as any;
  },
});
