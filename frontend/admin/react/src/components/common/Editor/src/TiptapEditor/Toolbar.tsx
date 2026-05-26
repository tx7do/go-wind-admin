import { useTranslation } from 'react-i18next';

interface ToolbarProps {
  editor: any;
  isActive: (name: string, options?: any) => boolean;
  actions: any;
  isFullscreen?: boolean;
}

export const Toolbar: React.FC<ToolbarProps> = ({ editor, isActive, actions, isFullscreen }) => {
  const { t } = useTranslation();

  return (
    <div className="tiptap-toolbar">
      {/* Heading & Paragraph */}
      <div className="toolbar-group">
        <button
          type="button"
          className={`toolbar-btn${isActive('heading', { level: 1 }) ? ' active' : ''}`}
          onClick={() => actions.toggleHeading(1)}
          title="H1"
        >
          H1
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('heading', { level: 2 }) ? ' active' : ''}`}
          onClick={() => actions.toggleHeading(2)}
          title="H2"
        >
          H2
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('heading', { level: 3 }) ? ' active' : ''}`}
          onClick={() => actions.toggleHeading(3)}
          title="H3"
        >
          H3
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('paragraph') ? ' active' : ''}`}
          onClick={actions.setParagraph}
          title="P"
        >
          P
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Text formatting */}
      <div className="toolbar-group">
        <button
          type="button"
          className={`toolbar-btn${isActive('bold') ? ' active' : ''}`}
          onClick={actions.toggleBold}
          title={t('editor:bold', '粗体')}
        >
          <b>B</b>
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('italic') ? ' active' : ''}`}
          onClick={actions.toggleItalic}
          title={t('editor:italic', '斜体')}
        >
          <i>I</i>
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('strike') ? ' active' : ''}`}
          onClick={actions.toggleStrike}
          title={t('editor:strike', '删除线')}
        >
          <s>S</s>
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('underline') ? ' active' : ''}`}
          onClick={actions.toggleUnderline}
          title={t('editor:underline', '下划线')}
        >
          <u>U</u>
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('code') ? ' active' : ''}`}
          onClick={actions.toggleCode}
          title={t('editor:code', '代码')}
        >
          &lt;/&gt;
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Subscript, Superscript, Colors */}
      <div className="toolbar-group">
        <button
          type="button"
          className={`toolbar-btn${isActive('subscript') ? ' active' : ''}`}
          onClick={actions.toggleSubscript}
          title={t('editor:subscript', '下标')}
        >
          X₂
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('superscript') ? ' active' : ''}`}
          onClick={actions.toggleSuperscript}
          title={t('editor:superscript', '上标')}
        >
          X²
        </button>
        <select
          className="toolbar-select"
          title={t('editor:fontSize', '字体大小')}
          onChange={(e) => actions.setFontSize(e.target.value)}
          defaultValue="16px"
        >
          <option value="12px">12px</option>
          <option value="14px">14px</option>
          <option value="16px">16px</option>
          <option value="18px">18px</option>
          <option value="20px">20px</option>
          <option value="24px">24px</option>
          <option value="28px">28px</option>
          <option value="32px">32px</option>
        </select>
        <input
          type="color"
          className="toolbar-color-picker"
          title={t('editor:textColor', '文字颜色')}
          onChange={(e) => actions.setTextColor(e.target.value)}
        />
        <input
          type="color"
          className="toolbar-color-picker"
          title={t('editor:highlightColor', '高亮颜色')}
          onChange={(e) => actions.setHighlight(e.target.value)}
        />
      </div>

      <div className="toolbar-divider" />

      {/* Lists & Blocks */}
      <div className="toolbar-group">
        <button
          type="button"
          className={`toolbar-btn${isActive('bulletList') ? ' active' : ''}`}
          onClick={actions.toggleBulletList}
          title={t('editor:bulletList', '无序列表')}
        >
          •≡
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('orderedList') ? ' active' : ''}`}
          onClick={actions.toggleOrderedList}
          title={t('editor:orderedList', '有序列表')}
        >
          1.
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('taskList') ? ' active' : ''}`}
          onClick={actions.toggleTaskList}
          title={t('editor:taskList', '任务列表')}
        >
          ☑
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('blockquote') ? ' active' : ''}`}
          onClick={actions.toggleBlockquote}
          title={t('editor:blockquote', '引用')}
        >
          ❝
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('codeBlock') ? ' active' : ''}`}
          onClick={actions.insertCodeBlock}
          title={t('editor:insertCodeBlock', '代码块')}
        >
          {'{}'}
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.insertTable}
          title={t('editor:insertTable', '插入表格')}
        >
          ⊞
        </button>
        {isActive('table') && (
          <button
            type="button"
            className="toolbar-btn"
            style={{ color: '#ef4444' }}
            onClick={actions.deleteTable}
            title={t('editor:deleteTable', '删除表格')}
          >
            🗑
          </button>
        )}
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.insertHorizontalRule}
          title={t('editor:insertHorizontalRule', '水平线')}
        >
          —
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Indent/Outdent */}
      <div className="toolbar-group">
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.indent}
          title={t('editor:indent', '增加缩进')}
        >
          ⇥
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.outdent}
          title={t('editor:outdent', '减少缩进')}
        >
          ⇤
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Alignment */}
      <div className="toolbar-group">
        <button
          type="button"
          className={`toolbar-btn${isActive('textAlign', { textAlign: 'left' }) ? ' active' : ''}`}
          onClick={() => actions.setAlign('left')}
          title={t('editor:left', '左对齐')}
        >
          ⇐
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('textAlign', { textAlign: 'center' }) ? ' active' : ''}`}
          onClick={() => actions.setAlign('center')}
          title={t('editor:center', '居中')}
        >
          ⇔
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('textAlign', { textAlign: 'right' }) ? ' active' : ''}`}
          onClick={() => actions.setAlign('right')}
          title={t('editor:right', '右对齐')}
        >
          ⇒
        </button>
        <button
          type="button"
          className={`toolbar-btn${isActive('textAlign', { textAlign: 'justify' }) ? ' active' : ''}`}
          onClick={() => actions.setAlign('justify')}
          title={t('editor:justify', '两端对齐')}
        >
          ☰
        </button>
      </div>

      {/* Table operations */}
      {isActive('table') && (
        <>
          <div className="toolbar-divider" />
          <div className="toolbar-group">
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.addRowBefore}
              title={t('editor:insertRowBefore', '上方插入行')}
            >
              R+
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.addRowAfter}
              title={t('editor:insertRowAfter', '下方插入行')}
            >
              +R
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.deleteRow}
              title={t('editor:deleteRow', '删除行')}
            >
              R-
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.addColumnBefore}
              title={t('editor:insertColBefore', '左侧插入列')}
            >
              C+
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.addColumnAfter}
              title={t('editor:insertColAfter', '右侧插入列')}
            >
              +C
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.deleteColumn}
              title={t('editor:deleteCol', '删除列')}
            >
              C-
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.mergeCells}
              title={t('editor:mergeCells', '合并单元格')}
            >
              M
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.splitCell}
              title={t('editor:splitCell', '拆分单元格')}
            >
              S
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.toggleHeaderRow}
              title={t('editor:toggleHeaderRow', '切换表头行')}
            >
              HR
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.toggleHeaderColumn}
              title={t('editor:toggleHeaderColumn', '切换表头列')}
            >
              HC
            </button>
            <button
              type="button"
              className="toolbar-btn"
              onClick={actions.toggleHeaderCell}
              title={t('editor:toggleHeaderCell', '切换表头单元格')}
            >
              H
            </button>
          </div>
        </>
      )}

      <div className="toolbar-divider" />

      {/* Undo/Redo/Clear */}
      <div className="toolbar-group">
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.undo}
          disabled={!editor?.can().undo()}
          title={t('editor:undo', '撤销')}
        >
          ↶
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.redo}
          disabled={!editor?.can().redo()}
          title={t('editor:redo', '重做')}
        >
          ↷
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.clearFormatting}
          title={t('editor:clearFormatting', '清除格式')}
        >
          A̶
        </button>
        <button
          type="button"
          className="toolbar-btn"
          style={{ color: '#ef4444' }}
          onClick={actions.clearContent}
          title={t('editor:clearContent', '清空内容')}
        >
          🗑
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Insert Media */}
      <div className="toolbar-group">
        <button
          type="button"
          className={`toolbar-btn${isActive('link') ? ' active' : ''}`}
          onClick={() =>
            isActive('link')
              ? editor?.chain().focus().unsetLink().run()
              : actions.setLinkModalVisible?.(true)
          }
          title={
            isActive('link') ? t('editor:removeUrl', '移除链接') : t('editor:insertUrl', '插入链接')
          }
        >
          🔗
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.uploadImage}
          title={t('editor:uploadImage', '上传图片')}
        >
          🖼
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.insertVideo}
          title={t('editor:insertVideo', '插入视频')}
        >
          🎬
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.insertIframe}
          title={t('editor:insertIframe', '插入iframe')}
        >
          ⧉
        </button>
        <button
          type="button"
          className="toolbar-btn"
          onClick={actions.importMarkdown}
          title={t('editor:importMarkdown', '导入Markdown')}
        >
          MD
        </button>
        <button
          type="button"
          className={`toolbar-btn${isFullscreen ? ' active' : ''}`}
          onClick={actions.toggleFullscreen}
          title={
            isFullscreen
              ? t('editor:exitFullscreen', '退出全屏')
              : t('editor:enterFullscreen', '全屏')
          }
        >
          {isFullscreen ? '⤓' : '⤢'}
        </button>
      </div>
    </div>
  );
};
