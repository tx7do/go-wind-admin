<template>
  <div class="layout-panel">
    <!-- 布局选择 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.modes.title") }}</h3>
      <div class="layout-grid">
        <div
          v-for="option in LAYOUT_OPTIONS"
          :key="option.value"
          class="layout-wrapper"
          @click="updatePreferences({ app: { layout: option.value as any } })"
        >
          <div class="layout-item" :class="{ active: preferences.app.layout === option.value }">
            <div class="layout-preview">
              <!-- sidebar-nav -->
              <div v-if="option.value === 'sidebar-nav'" class="layout-visual sidebar-nav">
                <div class="browser-frame">
                  <div class="browser-header">
                    <div class="browser-dots">
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                  </div>
                  <div class="browser-body">
                    <div class="sidebar">
                      <div class="menu-item"></div>
                      <div class="menu-item"></div>
                      <div class="menu-item"></div>
                      <div class="menu-item"></div>
                      <div class="menu-item"></div>
                    </div>
                    <div class="main-content">
                      <div class="content-blocks">
                        <div class="block-row">
                          <div class="block"></div>
                          <div class="block"></div>
                        </div>
                        <div class="block-full"></div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <!-- sidebar-mixed-nav -->
              <div
                v-else-if="option.value === 'sidebar-mixed-nav'"
                class="layout-visual sidebar-mixed-nav"
              >
                <div class="browser-frame">
                  <div class="browser-header">
                    <div class="browser-dots">
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                  </div>
                  <div class="browser-body">
                    <div class="sidebar-thin primary">
                      <div class="mini-item"></div>
                      <div class="mini-item"></div>
                      <div class="mini-item"></div>
                      <div class="mini-item"></div>
                    </div>
                    <div class="sidebar-thin gray">
                      <div class="mini-item"></div>
                      <div class="mini-item"></div>
                      <div class="mini-item"></div>
                    </div>
                    <div class="main-content">
                      <div class="content-blocks">
                        <div class="block-row">
                          <div class="block"></div>
                          <div class="block"></div>
                        </div>
                        <div class="block-full"></div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <!-- header-nav -->
              <div v-else-if="option.value === 'header-nav'" class="layout-visual header-nav">
                <div class="browser-frame">
                  <div class="browser-header">
                    <div class="browser-dots">
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                  </div>
                  <div class="browser-body column">
                    <div class="header-nav-bar">
                      <div class="nav-item"></div>
                      <div class="nav-item"></div>
                      <div class="nav-item"></div>
                      <div class="nav-item"></div>
                      <div class="nav-item"></div>
                    </div>
                    <div class="content-blocks">
                      <div class="block-row">
                        <div class="block"></div>
                        <div class="block"></div>
                      </div>
                      <div class="block-full"></div>
                    </div>
                  </div>
                </div>
              </div>
              <!-- mixed-nav -->
              <div v-else-if="option.value === 'mixed-nav'" class="layout-visual mixed-nav">
                <div class="browser-frame">
                  <div class="browser-header">
                    <div class="browser-dots">
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                  </div>
                  <div class="browser-body column">
                    <div class="header-nav-bar">
                      <div class="nav-item"></div>
                      <div class="nav-item"></div>
                      <div class="nav-item"></div>
                    </div>
                    <div class="mixed-body">
                      <div class="sidebar-thin gray">
                        <div class="mini-item"></div>
                        <div class="mini-item"></div>
                        <div class="mini-item"></div>
                      </div>
                      <div class="content-blocks">
                        <div class="block-row">
                          <div class="block"></div>
                          <div class="block"></div>
                        </div>
                        <div class="block-full"></div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <!-- full-content -->
              <div v-else-if="option.value === 'full-content'" class="layout-visual full-content">
                <div class="browser-frame">
                  <div class="browser-header">
                    <div class="browser-dots">
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                  </div>
                  <div class="browser-body">
                    <div class="content-blocks full">
                      <div class="block-row">
                        <div class="block"></div>
                        <div class="block"></div>
                      </div>
                      <div class="block-full"></div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="layout-label">
            <span>{{ t(`preferences.${option.label}`) }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 内容宽度 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.contentModes.title") }}</h3>
      <div class="content-compact-grid">
        <div
          v-for="option in CONTENT_COMPACT_OPTIONS"
          :key="option.value"
          class="content-compact-wrapper"
          @click="updatePreferences({ app: { contentCompact: option.value as any } })"
        >
          <div
            class="content-compact-item"
            :class="{ active: preferences.app.contentCompact === option.value }"
          >
            <div class="content-preview">
              <div class="browser-frame-small">
                <div class="browser-header-small">
                  <div class="browser-dots-small">
                    <span></span>
                    <span></span>
                    <span></span>
                  </div>
                </div>
                <div class="browser-body-small">
                  <div
                    class="content-area-small"
                    :class="option.value === 'compact' ? 'narrow' : 'full'"
                  >
                    <div class="small-block-row">
                      <div class="small-block"></div>
                      <div class="small-block"></div>
                    </div>
                    <div class="small-block-full"></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="content-label">
            <span>{{ t(`preferences.${option.label}`) }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 侧边栏 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.sidebar") }}</h3>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showSidebar") }}</span>
        <ElSwitch
          :model-value="preferences.sidebar.enable"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ sidebar: { enable: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.collapseMenu") }}</span>
        <ElSwitch
          :model-value="preferences.sidebar.collapsed"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ sidebar: { collapsed: v as boolean } })
          "
        />
      </div>
      <div class="preference-item" :class="{ disabled: !preferences.sidebar.collapsed }">
        <span>{{ t("preferences.layout.collapseShowTitle") }}</span>
        <ElSwitch
          :disabled="!preferences.sidebar.collapsed"
          :model-value="preferences.sidebar.collapsedShowTitle"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ sidebar: { collapsedShowTitle: v as boolean } })
          "
        />
      </div>
      <div class="preference-item width-control">
        <span>{{ t("preferences.layout.width") }}</span>
        <div class="width-control-group">
          <ElButton
            size="small"
            :icon="Minus"
            :disabled="preferences.sidebar.width <= 180"
            @click="handleSidebarWidthChange(preferences.sidebar.width - 8)"
          />
          <ElInputNumber
            size="small"
            :min="180"
            :max="320"
            :model-value="preferences.sidebar.width"
            style="width: 70px"
            controls-position="right"
            @change="handleSidebarWidthChange"
          />
          <ElButton
            size="small"
            :icon="Plus"
            :disabled="preferences.sidebar.width >= 320"
            @click="handleSidebarWidthChange(preferences.sidebar.width + 8)"
          />
        </div>
      </div>
    </section>

    <!-- 顶栏 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.header") }}</h3>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showHeader") }}</span>
        <ElSwitch
          :model-value="preferences.header.enable"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ header: { enable: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.mode") }}</span>
        <ElSegmented
          :model-value="preferences.header.mode"
          :options="headerModeOptions"
          @change="(v: string) => updatePreferences({ header: { mode: v as any } })"
        />
      </div>
    </section>

    <!-- 导航菜单 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.navigationMenu") }}</h3>
      <div class="preference-item">
        <span>{{ t("preferences.layout.menuStyle") }}</span>
        <ElSegmented
          :model-value="preferences.navigation.styleType"
          :options="menuStyleOptions"
          @change="(v: string) => updatePreferences({ navigation: { styleType: v as any } })"
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.menuSplit") }}</span>
        <ElSwitch
          :model-value="preferences.navigation.split"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ navigation: { split: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.accordionMode") }}</span>
        <ElSwitch
          :model-value="preferences.navigation.accordion"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ navigation: { accordion: v as boolean } })
          "
        />
      </div>
    </section>

    <!-- 面包屑 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.breadcrumb") }}</h3>
      <div class="preference-item">
        <span>{{ t("preferences.layout.enableBreadcrumb") }}</span>
        <ElSwitch
          :model-value="preferences.breadcrumb.enable"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ breadcrumb: { enable: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.hideWhenOnlyOne") }}</span>
        <ElSwitch
          :model-value="preferences.breadcrumb.hideOnlyOne"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ breadcrumb: { hideOnlyOne: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showIcon") }}</span>
        <ElSwitch
          :model-value="preferences.breadcrumb.showIcon"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ breadcrumb: { showIcon: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showHomeButton") }}</span>
        <ElSwitch
          :model-value="preferences.breadcrumb.showHome"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ breadcrumb: { showHome: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.breadcrumbStyle") }}</span>
        <ElSegmented
          :model-value="preferences.breadcrumb.styleType"
          :options="breadcrumbStyleOptions"
          @change="(v: string) => updatePreferences({ breadcrumb: { styleType: v as any } })"
        />
      </div>
    </section>

    <!-- 标签栏 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.tabbar") }}</h3>
      <div class="preference-item">
        <span>{{ t("preferences.layout.enableTabbar") }}</span>
        <ElSwitch
          :model-value="preferences.tabbar.enable"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ tabbar: { enable: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.persistTabs") }}</span>
        <ElSwitch
          :model-value="preferences.tabbar.persist"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ tabbar: { persist: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.draggableSort") }}</span>
        <ElSwitch
          :model-value="preferences.tabbar.draggable"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ tabbar: { draggable: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showTabbarIcon") }}</span>
        <ElSwitch
          :model-value="preferences.tabbar.showIcon"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ tabbar: { showIcon: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showMoreButton") }}</span>
        <ElSwitch
          :model-value="preferences.tabbar.showMore"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ tabbar: { showMore: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showMaximizeButton") }}</span>
        <ElSwitch
          :model-value="preferences.tabbar.showMaximize"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ tabbar: { showMaximize: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.tabbarStyle") }}</span>
        <ElSelect
          :model-value="preferences.tabbar.styleType"
          style="width: 160px"
          @change="(v: string) => updatePreferences({ tabbar: { styleType: v as any } })"
        >
          <ElOption
            v-for="opt in tabbarStyleOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </ElSelect>
      </div>
    </section>

    <!-- 小部件 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.widgets") }}</h3>
      <div v-for="item in widgetItems" :key="item.key" class="preference-item">
        <span>{{ t(`preferences.layout.${item.key}`) }}</span>
        <ElSwitch
          :model-value="(preferences.widget as any)[item.prop]"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ widget: { [item.prop]: v as boolean } })
          "
        />
      </div>
      <div class="preference-item">
        <span>{{ t("preferences.layout.preferencesPosition") }}</span>
        <ElSelect
          :model-value="preferences.app.preferencesButtonPosition"
          style="width: 160px"
          @change="
            (v: string) => updatePreferences({ app: { preferencesButtonPosition: v as any } })
          "
        >
          <ElOption
            v-for="opt in positionOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </ElSelect>
      </div>
    </section>

    <!-- 底栏 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.footer") }}</h3>
      <div class="preference-item">
        <span>{{ t("preferences.layout.showFooter") }}</span>
        <ElSwitch
          :model-value="preferences.footer.enable"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ footer: { enable: v as boolean } })
          "
        />
      </div>
      <div class="preference-item" :class="{ disabled: !preferences.footer.enable }">
        <span>{{ t("preferences.layout.fixedAtBottom") }}</span>
        <ElSwitch
          :disabled="!preferences.footer.enable"
          :model-value="preferences.footer.fixed"
          @change="
            (v: string | number | boolean) => updatePreferences({ footer: { fixed: v as boolean } })
          "
        />
      </div>
    </section>

    <!-- 版权 -->
    <section class="layout-section">
      <h3 class="section-title">{{ t("preferences.layout.copyright") }}</h3>
      <div class="preference-item">
        <span>{{ t("preferences.layout.enableCopyright") }}</span>
        <ElSwitch
          :model-value="preferences.copyright.enable"
          @change="
            (v: string | number | boolean) =>
              updatePreferences({ copyright: { enable: v as boolean } })
          "
        />
      </div>
      <div
        v-for="field in copyrightFields"
        :key="field.prop"
        class="preference-item"
        :class="{ disabled: !preferences.copyright.enable }"
      >
        <span>{{ t(`preferences.layout.${field.key}`) }}</span>
        <ElInput
          style="width: 200px"
          :disabled="!preferences.copyright.enable"
          :model-value="(preferences.copyright as any)[field.prop]"
          @input="(v: string) => updatePreferences({ copyright: { [field.prop]: v } })"
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import {
  ElSwitch,
  ElButton,
  ElInputNumber,
  ElInput,
  ElSelect,
  ElOption,
  ElSegmented,
} from "element-plus";
import { Minus, Plus } from "@element-plus/icons-vue";
import { useI18n } from "@/i18n";
import { preferences, updatePreferences } from "../../index";

const { t } = useI18n();

const LAYOUT_OPTIONS = [
  { label: "layout.modes.sidebarNav", value: "sidebar-nav" },
  { label: "layout.modes.sidebarMixedNav", value: "sidebar-mixed-nav" },
  { label: "layout.modes.headerNav", value: "header-nav" },
  { label: "layout.modes.mixedNav", value: "mixed-nav" },
  { label: "layout.modes.fullContent", value: "full-content" },
];

const CONTENT_COMPACT_OPTIONS = [
  { label: "layout.contentModes.wide", value: "wide" },
  { label: "layout.contentModes.compact", value: "compact" },
];

const headerModeOptions = computed(() => [
  { label: t("preferences.layout.headerModes.fixed"), value: "fixed" },
  { label: t("preferences.layout.headerModes.auto"), value: "auto" },
  { label: t("preferences.layout.headerModes.static"), value: "static" },
]);

const menuStyleOptions = computed(() => [
  { label: t("preferences.layout.menuStyles.rounded"), value: "rounded" },
  { label: t("preferences.layout.menuStyles.plain"), value: "plain" },
]);

const breadcrumbStyleOptions = computed(() => [
  { label: t("preferences.layout.breadcrumbStyles.normal"), value: "normal" },
  { label: t("preferences.layout.breadcrumbStyles.background"), value: "background" },
]);

const tabbarStyleOptions = computed(() => [
  { label: t("preferences.layout.tabbarStyles.chrome"), value: "chrome" },
  { label: t("preferences.layout.tabbarStyles.card"), value: "card" },
  { label: t("preferences.layout.tabbarStyles.brisk"), value: "brisk" },
  { label: t("preferences.layout.tabbarStyles.plain"), value: "plain" },
]);

const positionOptions = computed(() => [
  { label: t("preferences.layout.positions.auto"), value: "auto" },
  { label: t("preferences.layout.positions.fixed"), value: "fixed" },
  { label: t("preferences.layout.positions.hidden"), value: "hidden" },
]);

const widgetItems = [
  { key: "globalSearch", prop: "globalSearch" },
  { key: "themeToggle", prop: "themeToggle" },
  { key: "languageToggle", prop: "languageToggle" },
  { key: "fullscreen", prop: "fullscreen" },
  { key: "notification", prop: "notification" },
  { key: "lockScreen", prop: "lockScreen" },
  { key: "sidebarToggle", prop: "sidebarToggle" },
  { key: "refresh", prop: "refresh" },
];

const copyrightFields = [
  { key: "companyName", prop: "companyName" },
  { key: "companySite", prop: "companySiteLink" },
  { key: "date", prop: "date" },
  { key: "icpNumber", prop: "icp" },
  { key: "icpLink", prop: "icpLink" },
];

function handleSidebarWidthChange(width: number | null | undefined) {
  if (width && width >= 180 && width <= 320) {
    updatePreferences({ sidebar: { width } });
  }
}
</script>

<style src="./LayoutPanel.scss" scoped lang="scss"></style>
