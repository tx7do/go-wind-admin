<template>
  <el-icon
    class="pwd-toggle"
    @click.prevent="visible = !visible"
    @mousedown.prevent
    @mouseup.prevent
  >
    <View v-if="visible" />
    <Hide v-else />
  </el-icon>
</template>

<script setup lang="ts">
/**
 * 密码明文切换眼睛（恒显；与 antd 两端 Input.Password 朝向一致：
 * 隐藏态显示闭眼 Hide、可见态显示睁眼 View）。
 *
 * 不用 el-input 的 show-password——Element Plus 2.14 起其图标渲染含
 * !!nativeInputValue 守卫，输入框为空时不渲染眼睛，与另两端行为不一致。
 * 用法：
 *   <el-input :type="pwdVisible ? 'text' : 'password'" ...>
 *     <template #suffix><PasswordToggle v-model="pwdVisible" /></template>
 *   </el-input>
 */
const visible = defineModel<boolean>({ default: false });
</script>

<style scoped>
.pwd-toggle {
  cursor: pointer;
}
</style>
