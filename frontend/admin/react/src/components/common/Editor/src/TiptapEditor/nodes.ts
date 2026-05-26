import { mergeAttributes, Node } from '@tiptap/core';

// Custom Video extension
export const CustomVideo = Node.create({
  name: 'video',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() {
    return {
      src: { default: null, parseHTML: (el: HTMLElement) => el.getAttribute('src') },
      width: {
        default: '100%',
        parseHTML: (el: HTMLElement) => (el as HTMLElement).style.width || '100%',
      },
      height: {
        default: 'auto',
        parseHTML: (el: HTMLElement) => (el as HTMLElement).style.height || 'auto',
      },
    };
  },
  parseHTML() {
    return [{ tag: 'video' }];
  },
  renderHTML({ HTMLAttributes }: { HTMLAttributes: Record<string, any> }) {
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
        ({ commands }: { commands: any }) =>
          commands.insertContent({ type: this.name, attrs: options }),
    } as any;
  },
});

// Custom Iframe extension
export const CustomIframe = Node.create({
  name: 'iframe',
  group: 'block',
  atom: true,
  draggable: true,
  addAttributes() {
    return {
      src: { default: null, parseHTML: (el: HTMLElement) => el.getAttribute('src') },
      width: {
        default: '100%',
        parseHTML: (el: HTMLElement) => (el as HTMLElement).style.width || '100%',
      },
      height: {
        default: '500px',
        parseHTML: (el: HTMLElement) => (el as HTMLElement).style.height || '500px',
      },
      title: { default: null, parseHTML: (el: HTMLElement) => el.getAttribute('title') },
      allowfullscreen: {
        default: true,
        parseHTML: (el: HTMLElement) => el.hasAttribute('allowfullscreen'),
      },
    };
  },
  parseHTML() {
    return [{ tag: 'iframe' }];
  },
  renderHTML({ HTMLAttributes }: { HTMLAttributes: Record<string, any> }) {
    return [
      'iframe',
      mergeAttributes(HTMLAttributes, {
        frameborder: '0',
        style: 'max-width: 100%; border: 1px solid #e5e7eb; border-radius: 6px; margin: 12px 0;',
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
        ({ commands }: { commands: any }) =>
          commands.insertContent({ type: this.name, attrs: options }),
    } as any;
  },
});
