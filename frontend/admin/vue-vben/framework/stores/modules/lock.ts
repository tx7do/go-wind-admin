import { defineStore } from 'pinia';

interface AppState {
  /**
   * 是否锁屏状态
   */
  isLockScreen: boolean;
  /**
   * 锁屏密码
   */
  lockScreenPassword?: string;
}

export const useLockStore = defineStore('core-lock', {
  actions: {
    lockScreen(password: string) {
      this.isLockScreen = true;
      this.lockScreenPassword = password;
    },

    unlockScreen() {
      this.isLockScreen = false;
      this.lockScreenPassword = undefined;
    },
  },
  persist: {
    pick: ['isLockScreen', 'lockScreenPassword'],
    // P2 修复：密码改存 sessionStorage——此前随全局默认落 localStorage 持久盘，
    // 共用机器的下一个使用者/任何 XSS 都能直接读走（用户常复用登录密码）。
    // sessionStorage 按标签页隔离、关标签/浏览器即清，刷新（同标签）锁屏态
    // 与密码仍可用，UX 不变。
    storage: sessionStorage,
  },
  state: (): AppState => ({
    isLockScreen: false,
    lockScreenPassword: undefined,
  }),
});
