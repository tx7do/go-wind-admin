import { mergeAttributes, Node } from "@tiptap/core";

/**
 * 自定义 Iframe 扩展
 * - 支持嵌入外部页面
 * - 自定义宽、高、标题、全屏属性
 */
export const CustomIframe = Node.create({
  name: "iframe",
  group: "block",
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: null,
        parseHTML: (element) => element.getAttribute("src"),
        renderHTML: (attributes) => {
          if (!attributes.src) return {};
          return { src: attributes.src };
        },
      },
      width: {
        default: "100%",
        parseHTML: (element) => element.style.width || "100%",
        renderHTML: (attributes) => {
          return { style: `width: ${attributes.width}` };
        },
      },
      height: {
        default: "500px",
        parseHTML: (element) => element.style.height || "500px",
        renderHTML: (attributes) => {
          return { style: `height: ${attributes.height}` };
        },
      },
      title: {
        default: null,
        parseHTML: (element) => element.getAttribute("title"),
        renderHTML: (attributes) => {
          if (!attributes.title) return {};
          return { title: attributes.title };
        },
      },
      allowfullscreen: {
        default: true,
        parseHTML: (element) => element.hasAttribute("allowfullscreen"),
        renderHTML: (attributes) => {
          if (!attributes.allowfullscreen) return {};
          return { allowfullscreen: "allowfullscreen" };
        },
      },
    };
  },

  parseHTML() {
    return [{ tag: "iframe" }];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "iframe",
      mergeAttributes(HTMLAttributes, {
        frameborder: "0",
        style:
          "max-width: 100%; border: 1px solid var(--el-border-color-light); border-radius: 6px; margin: 12px 0;",
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
