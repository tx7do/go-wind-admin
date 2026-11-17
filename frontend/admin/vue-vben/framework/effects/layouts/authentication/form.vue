<script setup lang="ts">
defineOptions({
  name: 'AuthenticationFormView',
});
</script>

<template>
  <div
    class="flex-col-center dark:bg-background-deep bg-background relative px-6 py-10 lg:flex-initial lg:px-8"
  >
    <slot></slot>
    <!-- Router View with Transition and KeepAlive -->
    <RouterView v-slot="{ Component }">
      <!-- 显式 duration：与 basic/content.vue 同理（见 b99c0be1）——部分内嵌
           webview 不派发 transitionend/animationend，纯 CSS 事件判定会把
           out-in 的 enter 永久卡死。毫秒值与 slide-right CSS 时长对齐。
           不给组件加 :key="route.fullPath"（KeepAlive 内注释都算子节点，说明写在
           这里）：同组件仅 query 变化（如守卫追加 redirect 参数）时换 key 会让
           out-in+KeepAlive 在 leave 完成后跳过同型新 key 的挂载——登录表单整块
           消失、面板只剩版权行，硬刷新才恢复（真实 Chromium 可复现，401 登出链路
           必踩）。Login 不读 fullPath 渲染；mfa/sso 是不同路由记录不同组件，按
           组件身份换入即可。 -->
      <Transition :duration="300" appear mode="out-in" name="slide-right">
        <KeepAlive :include="['Login']">
          <component
            :is="Component"
            class="enter-x mt-6 w-full sm:mx-auto md:max-w-md"
          />
        </KeepAlive>
      </Transition>
    </RouterView>

    <!-- Footer Copyright -->

    <div
      class="text-muted-foreground absolute bottom-3 flex text-center text-xs"
    >
      <slot name="copyright"> </slot>
    </div>
  </div>
</template>
