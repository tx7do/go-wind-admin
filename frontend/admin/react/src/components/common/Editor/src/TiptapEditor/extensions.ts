import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight';

// Custom CodeBlock with language selector
export const CustomCodeBlockLowlight = CodeBlockLowlight.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      language: {
        default: 'javascript',
        parseHTML: (element: HTMLElement) => element.dataset.language,
        renderHTML: (attributes: Record<string, string>) => ({
          'data-language': attributes.language || 'javascript',
          class: `language-${attributes.language || 'javascript'}`,
        }),
      },
    };
  },
  addNodeView() {
    return ({ node, getPos, editor: editorInstance }) => {
      const dom = document.createElement('pre');
      const contentDOM = document.createElement('code');

      dom.dataset.language = node.attrs.language || 'javascript';
      dom.className = `language-${node.attrs.language || 'javascript'}`;

      // Language selector
      const selectorWrapper = document.createElement('div');
      selectorWrapper.className = 'code-block-language-selector';
      selectorWrapper.contentEditable = 'false';

      const select = document.createElement('select');
      select.contentEditable = 'false';

      const languageOptions = [
        { value: 'javascript', label: 'JavaScript' },
        { value: 'typescript', label: 'TypeScript' },
        { value: 'python', label: 'Python' },
        { value: 'java', label: 'Java' },
        { value: 'cpp', label: 'C++' },
        { value: 'c', label: 'C' },
        { value: 'csharp', label: 'C#' },
        { value: 'go', label: 'Go' },
        { value: 'rust', label: 'Rust' },
        { value: 'php', label: 'PHP' },
        { value: 'ruby', label: 'Ruby' },
        { value: 'swift', label: 'Swift' },
        { value: 'kotlin', label: 'Kotlin' },
        { value: 'html', label: 'HTML' },
        { value: 'css', label: 'CSS' },
        { value: 'scss', label: 'SCSS' },
        { value: 'json', label: 'JSON' },
        { value: 'yaml', label: 'YAML' },
        { value: 'sql', label: 'SQL' },
        { value: 'bash', label: 'Bash' },
        { value: 'shell', label: 'Shell' },
        { value: 'markdown', label: 'Markdown' },
        { value: 'plaintext', label: 'Plain Text' },
      ];

      languageOptions.forEach((lang) => {
        const option = document.createElement('option');
        option.value = lang.value;
        option.textContent = lang.label;
        if (lang.value === node.attrs.language) {
          option.selected = true;
        }
        select.append(option);
      });

      select.addEventListener('change', (e) => {
        const newLanguage = (e.target as HTMLSelectElement).value;
        if (typeof getPos === 'function') {
          const pos = getPos();
          if (typeof pos === 'number') {
            editorInstance.view.dispatch(
              editorInstance.view.state.tr.setNodeMarkup(pos, undefined, {
                ...node.attrs,
                language: newLanguage,
              }),
            );
            dom.dataset.language = newLanguage;
            dom.className = `language-${newLanguage}`;
          }
        }
      });

      selectorWrapper.append(select);
      dom.append(selectorWrapper);
      dom.append(contentDOM);

      return {
        dom,
        contentDOM,
        update: (updatedNode: any) => {
          if (updatedNode.type !== node.type) {
            return false;
          }
          if (updatedNode.attrs.language !== node.attrs.language) {
            select.value = updatedNode.attrs.language;
            dom.dataset.language = updatedNode.attrs.language;
            dom.className = `language-${updatedNode.attrs.language}`;
          }
          return true;
        },
      };
    };
  },
});
