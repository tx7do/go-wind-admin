import { ElMessageBox } from "element-plus";
import { $t } from "@go-wind/admin-element";

/**
 * 表单抽屉关闭守卫（规范见 docs/form_drawer_close_guard.md）
 *
 * 表单类 Drawer 的四条关闭路径（遮罩 / ESC / 标题栏 X / 底部取消）统一行为：
 * 无脏数据直接关闭；有脏数据弹确认框，确定后才关闭。提交成功后的关闭不经守卫。
 *
 * 脏判定：打开填充完成后 syncBaseline() 记 JSON 基线快照，关闭尝试时对比当前值
 * （程序化回填不算脏；编辑态超出 formData 的——勾选树等——在 getModel 里合并）。
 *
 * @example
 * ```ts
 * const guard = useDrawerCloseGuard(() => ({ ...formData, menuIds: menuTreeRef.value?.getCheckedKeys() }));
 * // open() 填充完成后：
 * guard.syncBaseline();
 * ```
 * ```vue
 * <ProModal :close-guard="guard.allowClose" ...>
 * <!-- 自定义 footer 的取消按钮 -->
 * <ElButton @click="guard.requestClose(handleClose)">取消</ElButton>
 * ```
 */
export function useDrawerCloseGuard(getModel: () => Record<string, any>) {
  let baseline: string | null = null;

  /** 打开填充完成后调用：记录基线快照 */
  function syncBaseline() {
    baseline = JSON.stringify(getModel() ?? {});
  }

  /** 保存成功后手动清脏（后续关闭不再确认） */
  function markClean() {
    baseline = null;
  }

  function isDirty() {
    return baseline !== null && JSON.stringify(getModel() ?? {}) !== baseline;
  }

  /** 接 ProModal 的 close-guard：返回 true 才允许关闭 */
  async function allowClose(): Promise<boolean> {
    if (!isDirty()) return true;
    try {
      await ElMessageBox.confirm($t("common.drawer.unsavedMessage"), "", {
        confirmButtonText: $t("common.drawer.discardButton"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      });
    } catch {
      // 用户点「取消」是预期分支，非错误
      return false;
    }
    return true;
  }

  /** 自定义 footer 取消按钮等场景的关闭尝试：脏则先确认，通过后执行 close() */
  async function requestClose(close: () => void) {
    if (await allowClose()) close();
  }

  /** 接裸 el-drawer / el-dialog 的 before-close */
  function beforeClose(done: () => void) {
    void requestClose(done);
  }

  return { syncBaseline, markClean, isDirty, allowClose, requestClose, beforeClose };
}
