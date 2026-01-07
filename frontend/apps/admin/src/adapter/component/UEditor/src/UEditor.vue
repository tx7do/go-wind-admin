<script setup lang="ts">
import { computed, ref, unref, watch } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  formatSaveShowContent,
  full_toolbars,
  getCurrentLang,
  isDarkMode,
} from './utils';

const props = defineProps({
  value: { type: String, default: '' },
  editorId: { type: String, default: 'editor' },
  maximumWords: { type: Number, default: 10_000 },
  contentHeight: { type: Number, default: 320 },
  toolbars: { type: Array, default: full_toolbars },
});

const emits = defineEmits(['editContent']);

const accessStore = useAccessStore();

const baseUrl = import.meta.env.VITE_GLOB_API_URL;
const contentRef: any = ref(formatSaveShowContent('show', props.value));

const editorInst = ref<any>(null);
const initedRef = ref<boolean>(false);

const ueditorHomeUrl = ref('/static/UEditorPlus/');

const getUeditorConfigUrl = ref(`${baseUrl}/admin/v1/ueditor`);

const getToken = computed(() => {
  console.log('getToken');
  return accessStore.accessToken;
});
const getServerHeaders = computed(() => {
  return {
    Authorization: `Bearer ${getToken.value}`,
  };
});

// const getImageUploadUrl = computed(() => {
//   return import.meta.env.VITE_GLOB_UPLOAD_URL;
// });
const getImageDownloadUrl = computed(() => {
  return import.meta.env.VITE_GLOB_DOWNLOAD_URL;
});

/* {filename} 会替换成原文件名,配置这项需要注意中文乱码问题 */
/* {rand:6} 会替换成随机数,后面的数字是随机数的位数 */
/* {time} 会替换成时间戳 */
/* {yyyy} 会替换成四位年份 */
/* {yy} 会替换成两位年份 */
/* {mm} 会替换成两位月份 */
/* {dd} 会替换成两位日期 */
/* {hh} 会替换成两位小时 */
/* {ii} 会替换成两位分钟 */
/* {ss} 会替换成两位秒 */
/* 非法字符 \ : * ? " < > | */
/* 具请体看线上文档: fex.baidu.com/ueditor/#use-format_upload_filename */

// TODO 需要修改js文件，去掉"jsonp"
// TODO 无法动态的改变样式

const theme = computed(() => {
  return isDarkMode() ? 'dark' : 'default';
});

const editorConfig = ref({
  // 后端服务地址，后端处理参考
  // https://open-doc.modstart.com/ueditor-plus/backend.html
  serverUrl: getUeditorConfigUrl.value,

  // 服务器统一请求头信息，会在所有请求中带上该信息
  serverHeaders: getServerHeaders.value,

  // UEditorPlus 的根目录，绝对路径。
  UEDITOR_HOME_URL: ueditorHomeUrl.value,
  // UEditorPlus 的跨域根目录，绝对路径。
  UEDITOR_CORS_URL: ueditorHomeUrl.value,

  // 设置null则宽度自适应
  initialFrameWidth: null,
  initialFrameHeight: props.contentHeight /* 区域内容高度 */,

  catchRemoteImageEnable: false /* 抓取远程图片是否开启,默认true */,

  autoHeightEnabled: false /* 固定高度且带滚动条 */,
  autoSyncData: false,
  autoFloatEnabled: true,

  theme: theme.value,
  themePath: '/static/UEditorPlus/themes/',

  toolbars: props.toolbars,
  maximumWords: props.maximumWords,

  /* 上传图片配置项 */
  imageActionName: 'uploadImage' /* 执行上传图片的action名称 */,
  imageFieldName: 'file' /* 提交的图片表单名称 */,
  imageMaxSize: 2_048_000 /* 上传大小限制，单位B */,
  imageAllowFiles: [
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.bmp',
  ] /* 上传图片格式显示 */,
  imageCompressEnable: true /* 是否压缩图片,默认是true */,
  imageCompressBorder: 1600 /* 图片压缩最长边限制 */,
  imageInsertAlign: 'none' /* 插入的图片浮动方式 */,
  imageUrlPrefix: getImageDownloadUrl.value /* 图片访问路径前缀 */,

  /* 涂鸦图片上传配置项 */
  scrawlActionName: 'uploadScrawl' /* 执行上传涂鸦的action名称 */,
  scrawlFieldName: 'file' /* 提交的图片表单名称 */,
  scrawlMaxSize: 2_048_000 /* 上传大小限制，单位B */,
  scrawlUrlPrefix: getImageDownloadUrl.value /* 图片访问路径前缀 */,
  scrawlInsertAlign: 'none',

  /* 截图工具上传 */
  snapscreenActionName: 'uploadImage' /* 执行上传截图的action名称 */,
  snapscreenUrlPrefix: getImageDownloadUrl.value /* 图片访问路径前缀 */,
  snapscreenInsertAlign: 'none' /* 插入的图片浮动方式 */,

  /* 抓取远程图片配置 */
  catcherLocalDomain: ['127.0.0.1', 'localhost', 'img.baidu.com'],
  catcherActionName: 'catchImage' /* 执行抓取远程图片的action名称 */,
  catcherFieldName: 'file' /* 提交的图片列表表单名称 */,
  catcherUrlPrefix: getImageDownloadUrl.value /* 图片访问路径前缀 */,
  catcherMaxSize: 2_048_000 /* 上传大小限制，单位B */,
  catcherAllowFiles: [
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.bmp',
  ] /* 抓取图片格式显示 */,

  /* 上传视频配置 */
  videoActionName: 'uploadVideo' /* 执行上传视频的action名称 */,
  videoFieldName: 'file' /* 提交的视频表单名称 */,
  videoUrlPrefix: getImageDownloadUrl.value /* 视频访问路径前缀 */,
  videoMaxSize: 102_400_000 /* 上传大小限制，单位B，默认100MB */,
  videoAllowFiles: [
    '.flv',
    '.swf',
    '.mkv',
    '.avi',
    '.rm',
    '.rmvb',
    '.mpeg',
    '.mpg',
    '.ogg',
    '.ogv',
    '.mov',
    '.wmv',
    '.mp4',
    '.webm',
    '.mp3',
    '.wav',
    '.mid',
  ] /* 上传视频格式显示 */,

  /* 上传文件配置 */
  fileActionName: 'uploadFile' /* controller里,执行上传视频的action名称 */,
  fileFieldName: 'file' /* 提交的文件表单名称 */,
  fileUrlPrefix: getImageDownloadUrl.value /* 文件访问路径前缀 */,
  fileMaxSize: 51_200_000 /* 上传大小限制，单位B，默认50MB */,
  fileAllowFiles: [
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.bmp',
    '.flv',
    '.swf',
    '.mkv',
    '.avi',
    '.rm',
    '.rmvb',
    '.mpeg',
    '.mpg',
    '.ogg',
    '.ogv',
    '.mov',
    '.wmv',
    '.mp4',
    '.webm',
    '.mp3',
    '.wav',
    '.mid',
    '.rar',
    '.zip',
    '.tar',
    '.gz',
    '.7z',
    '.bz2',
    '.cab',
    '.iso',
    '.doc',
    '.docx',
    '.xls',
    '.xlsx',
    '.ppt',
    '.pptx',
    '.pdf',
    '.txt',
    '.md',
    '.xml',
  ] /* 上传文件格式显示 */,

  /* 列出指定目录下的图片 */
  imageManagerActionName: 'listImage' /* 执行图片管理的action名称 */,
  imageManagerListSize: 20 /* 每次列出文件数量 */,
  imageManagerUrlPrefix: getImageDownloadUrl.value /* 图片访问路径前缀 */,
  imageManagerInsertAlign: 'none' /* 插入的图片浮动方式 */,
  imageManagerAllowFiles: [
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.bmp',
  ] /* 列出的文件类型 */,

  /* 列出指定目录下的文件 */
  fileManagerActionName: 'listFile' /* 执行文件管理的action名称 */,
  fileManagerUrlPrefix: getImageDownloadUrl.value /* 文件访问路径前缀 */,
  fileManagerListSize: 20 /* 每次列出文件数量 */,
  fileManagerAllowFiles: [
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.bmp',
    '.flv',
    '.swf',
    '.mkv',
    '.avi',
    '.rm',
    '.rmvb',
    '.mpeg',
    '.mpg',
    '.ogg',
    '.ogv',
    '.mov',
    '.wmv',
    '.mp4',
    '.webm',
    '.mp3',
    '.wav',
    '.mid',
    '.rar',
    '.zip',
    '.tar',
    '.gz',
    '.7z',
    '.bz2',
    '.cab',
    '.iso',
    '.doc',
    '.docx',
    '.xls',
    '.xlsx',
    '.ppt',
    '.pptx',
    '.pdf',
    '.txt',
    '.md',
    '.xml',
  ],
});

const ready = (editorInstance: any) => {
  // 获取编辑器实例
  editorInst.value = editorInstance;
  initedRef.value = true;
};

watch(contentRef, (_newVal) => {
  // 向父组件传递保存的内容
  emits('editContent', formatSaveShowContent('save', contentRef.value));
});

watch(
  [() => isDarkMode(), () => initedRef.value],
  ([_val, initialed]) => {
    if (!initialed) {
      return;
    }

    editorInst.value.setOpt({
      theme: theme.value,
    });
    console.log('watch isDarkMode', editorInst.value.getOpt('theme'));
  },
  {
    immediate: true,
  },
);

watch(
  [() => getCurrentLang, () => initedRef.value],
  ([_val, initialed]) => {
    if (!initialed) {
      return;
    }

    // const editor = UE.getEditor(props.editorId);
    // editor.setOpt('lang', unref(getCurrentLang));
    // editorConfig.value.lang = unref(getCurrentLang);

    editorInst.value.setOpt({ lang: unref(getCurrentLang) });

    console.log(
      'watch getCurrentLang',
      editorInst.value.getOpt('lang'),
      unref(getCurrentLang),
    );
  },
  {
    immediate: true,
  },
);
</script>

<template>
  <vue-ueditor-wrap
    :editor-id="editorId"
    v-model="contentRef"
    :config="editorConfig"
    :editor-dependencies="['ueditor.config.js', 'ueditor.all.js']"
    @ready="ready"
  />
</template>

<style scoped lang="scss"></style>
