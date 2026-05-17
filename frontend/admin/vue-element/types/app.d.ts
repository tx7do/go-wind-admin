declare global {
  /**
   * 登录过期模式
   * modal 弹窗模式
   * page 页面模式
   */
  type LoginExpiredModeType = "modal" | "page";

  /**
   * 权限模式
   * backend 后端权限模式
   * frontend 前端权限模式
   */
  type AccessModeType = "backend" | "frontend";

  /**
   * 主题模式
   * auto 自动
   * dark 暗色
   * light 亮色
   */
  type ThemeModeType = "auto" | "dark" | "light";
}
