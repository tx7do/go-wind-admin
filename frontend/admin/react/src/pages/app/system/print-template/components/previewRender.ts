/** 把渲染产物写入同源 iframe（go-wind-print demo 同款实现） */
export function writeHtmlToFrame(frame: HTMLIFrameElement | null, html: string) {
  const doc = frame?.contentWindow?.document;
  if (!doc) return;
  doc.open();
  doc.write(html);
  doc.close();
}
