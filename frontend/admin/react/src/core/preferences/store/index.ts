import {create} from 'zustand';
import {persist} from 'zustand/middleware';

import {defaultPreferences} from '../config/default';
import type {DeepPartial, Preferences} from '../types';
import {mergeDeep} from "../utils/merge";


export interface PreferencesState {
    preferences: Preferences;
    setPreferences: (overrides: DeepPartial<Preferences>) => void;
    resetPreferences: () => void;
    getPreference: <K extends keyof Preferences>(key: K) => Preferences[K];
}

export const usePreferencesStore = create<PreferencesState>()(
    persist(
        (set, get) => ({
            preferences: defaultPreferences,

            setPreferences: (overrides) => {
                set((state) => ({
                    preferences: mergeDeep(state.preferences, overrides),
                }));
            },

            resetPreferences: () => {
                set({preferences: defaultPreferences});
            },

            getPreference: (key) => {
                return get().preferences[key];
            },
        }),
        {
            name: 'app-preferences',
            // v0 → v1：品牌主色统一为 #3B82F6（科技蓝）。
            // 旧版本默认 colorPrimary 为 "hsl(212 100% 45%)"（≈#006BE6），会被
            // persist 快照留在 localStorage 里压过新默认值，导致 antd 组件用旧蓝、
            // 硬编码 CSS 用新蓝的"两种蓝"错乱。仅当主题仍为内置 default 时迁移，
            // 用户自选的主题色不受影响。
            version: 1,
            migrate: (persistedState, version) => {
                const state = (persistedState ?? {}) as { preferences?: Preferences };
                if (version < 1) {
                    const themePref = state.preferences?.theme;
                    if (
                        themePref &&
                        themePref.builtinType === 'default' &&
                        themePref.colorPrimary === 'hsl(212 100% 45%)'
                    ) {
                        themePref.colorPrimary = '#3B82F6';
                        themePref.radius = '6';
                        console.info(
                            '[Preferences] 迁移 v0→v1：默认主色 hsl(212 100% 45%) → #3B82F6'
                        );
                    }
                }
                return state as { preferences: Preferences };
            },
            partialize: (state) => ({preferences: state.preferences}),
        }
    )
);
