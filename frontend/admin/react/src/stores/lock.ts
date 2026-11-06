import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

interface LockState {
  isLockScreen: boolean;
  lockScreenPassword?: string;
  lockScreen: (password: string) => void;
  unlockScreen: () => void;
}

/**
 * 锁屏只是"离开工位遮个屏"的界面遮罩，不是认证边界，真正的鉴权仍由后端令牌负责。
 * P2 修复：密码改存 sessionStorage——此前明文落 localStorage 持久盘，共用机器的
 * 下一个使用者/任何 XSS 都能直接读走（用户常复用登录密码）。sessionStorage 按
 * 标签页隔离、关标签/浏览器即清，刷新（同标签）锁屏态与密码仍可用，UX 不变。
 */
export const useLockStore = create<LockState>()(
  persist(
    (set) => ({
      isLockScreen: false,
      lockScreenPassword: undefined,
      lockScreen: (password) => set({ isLockScreen: true, lockScreenPassword: password }),
      unlockScreen: () => set({ isLockScreen: false, lockScreenPassword: undefined }),
    }),
    {
      name: 'lock-storage',
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({
        isLockScreen: state.isLockScreen,
        lockScreenPassword: state.lockScreenPassword,
      }),
    },
  ),
);
