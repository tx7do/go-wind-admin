<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :close-guard="drawer.closeGuard"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{
      component: 'drawer',
      drawer: { size: drawer.drawerWidth },
    }"
  >
    <ElForm
      ref="formRef"
      :model="drawer.formData"
      :rules="formRules"
      label-width="140px"
      class="drawer-form"
    >
      <ElFormItem :label="t('pages.notification_channel.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="t('pages.notification_channel.requiredName')" clearable />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_channel.type')" prop="type">
        <!-- teleported=false：浮层渲染在抽屉 DOM 内，位置必跟随抽屉；
             默认 teleport 到 body 时 popper 对抽屉动画中的触发元素定位会算错（浮层飞到屏幕左上） -->
        <ElSelect
          v-model="drawer.formData.type"
          :disabled="!isCreate"
          filterable
          :teleported="false"
        >
          <ElOption :label="t('pages.notification_channel.typeEmail')" value="EMAIL" />
          <ElOption :label="t('pages.notification_channel.typeWebhook')" value="WEBHOOK" />
          <ElOptionGroup
            v-for="group in extTypeGroups"
            :key="group.group"
            :label="group.label"
          >
            <ElOption
              v-for="opt in group.options"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </ElOptionGroup>
        </ElSelect>
      </ElFormItem>

      <!-- 三类渠道各自的配置项分组显示：表单里同时摆几组，管理员要自己猜哪一组生效。
           编辑态 type 不可改（改类型等于换一条渠道实现），所以几组不会在同一个会话里来回切。 -->
      <template v-if="drawer.formData.type === 'WEBHOOK'">
        <ElFormItem :label="t('pages.notification_channel.webhookUrl')" prop="webhookUrl">
          <ElInput
            v-model="drawer.formData.webhookUrl"
            placeholder="https://example.com/hooks/notification"
            clearable
          />
          <div class="field-tip">{{ t("pages.notification_channel.webhookUrlHint") }}</div>
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.webhookSignStyle')" prop="webhookSignStyle">
          <ElSelect
            v-model="drawer.formData.webhookSignStyle"
            :teleported="false"
          >
            <ElOption :label="t('pages.notification_channel.signStyleCustom')" value="CUSTOM" />
            <ElOption :label="t('pages.notification_channel.signStyleNone')" value="NONE" />
            <ElOption :label="t('pages.notification_channel.signStyleDingtalk')" value="DINGTALK" />
            <ElOption :label="t('pages.notification_channel.signStyleFeishu')" value="FEISHU" />
            <ElOption :label="t('pages.notification_channel.signStyleWecom')" value="WECOM" />
          </ElSelect>
          <div class="field-tip">
            {{ t("pages.notification_channel.webhookSignStyleHint") }}
          </div>
        </ElFormItem>

        <!-- 留空 = 用该风格的内置默认正文（内置形状由后端 webhook_style.go 持有，
             这里不复制一份，否则两处会漂）。 -->
        <ElFormItem
          :label="t('pages.notification_channel.webhookPayloadTemplate')"
          prop="webhookPayloadTemplate"
        >
          <ElInput
            v-model="drawer.formData.webhookPayloadTemplate"
            type="textarea"
            :rows="3"
            :placeholder="WEBHOOK_TEMPLATE_EXAMPLE"
          />
          <div class="field-tip">
            {{
              t("pages.notification_channel.webhookPayloadTemplateHint", {
                vars: WEBHOOK_TEMPLATE_VARS,
              })
            }}
          </div>
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.webhookSecret')" prop="webhookSecret">
          <ElInput
            v-model="drawer.formData.webhookSecret"
            type="password"
            show-password
            :placeholder="
              isCreate
                ? t('pages.notification_channel.webhookSecretPlaceholder')
                : t('pages.notification_channel.webhookSecretKeepHint')
            "
          />
          <div class="field-tip">{{ t("pages.notification_channel.webhookSecretHint") }}</div>
        </ElFormItem>
      </template>

      <template v-else-if="isExtType(drawer.formData.type)">
        <div v-if="extSchema?.targetLabel" class="field-tip" style="margin-bottom: 8px">
          {{ t("pages.notification_channel.ext.targetLabel", { label: extSchema.targetLabel }) }}
        </div>
        <ElFormItem
          v-for="field in extSchema?.fields ?? []"
          :key="field.name"
          :label="field.label ?? field.name"
          :prop="field.name"
        >
          <ElInputNumber
            v-if="field.type === 'number'"
            v-model="drawer.formData.settings[field.name!]"
            :min="0"
            class="!w-full"
          />
          <ElSelect
            v-else-if="field.type === 'select'"
            v-model="drawer.formData.settings[field.name!]"
            clearable
          >
            <ElOption
              v-for="opt in field.options ?? []"
              :key="opt"
              :label="opt || t('pages.notification_channel.ext.optionDefault')"
              :value="opt"
            />
          </ElSelect>
          <ElInput
            v-else-if="field.type === 'textarea'"
            v-model="drawer.formData.settings[field.name!] as string"
            type="textarea"
            :rows="3"
            :placeholder="field.placeholder"
          />
          <ElInput
            v-else
            v-model="drawer.formData.settings[field.name!] as string"
            :type="field.type === 'password' || field.sensitive ? 'password' : 'text'"
            show-password
            :placeholder="
              isCreate
                ? (field.placeholder ?? t('pages.notification_channel.ext.placeholder'))
                : t('pages.notification_channel.ext.secretKeepHint')
            "
          />
          <div v-if="field.hint" class="field-tip">{{ field.hint }}</div>
        </ElFormItem>
      </template>

      <template
        v-else-if="
          drawer.formData.type === 'SMS' ||
          String(drawer.formData.type).startsWith('SMS@')
        "
      >
        <ElFormItem :label="t('pages.notification_channel.smsProvider')" prop="smsProvider">
          <ElSelect
          :teleported="false" v-model="drawer.formData.smsProvider">
            <ElOption :label="t('pages.notification_channel.providerAliyun')" value="ALIYUN" />
            <ElOption :label="t('pages.notification_channel.providerTencent')" value="TENCENT" />
          </ElSelect>
          <div class="field-tip">{{ t("pages.notification_channel.smsProviderHint") }}</div>
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smsAccessKeyId')" prop="smsAccessKeyId">
          <ElInput v-model="drawer.formData.smsAccessKeyId" clearable />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smsAccessSecret')" prop="smsAccessSecret">
          <ElInput
            v-model="drawer.formData.smsAccessSecret"
            type="password"
            show-password
            :placeholder="
              isCreate
                ? t('pages.notification_channel.smsAccessSecretPlaceholder')
                : t('pages.notification_channel.smsAccessSecretKeepHint')
            "
          />
          <div class="field-tip">{{ t("pages.notification_channel.smsAccessSecretHint") }}</div>
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smsSignName')" prop="smsSignName">
          <ElInput v-model="drawer.formData.smsSignName" clearable />
          <div class="field-tip">{{ t("pages.notification_channel.smsSignNameHint") }}</div>
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smsTemplateCode')" prop="smsTemplateCode">
          <ElInput
            v-model="drawer.formData.smsTemplateCode"
            :placeholder="t('pages.notification_channel.smsTemplateCodePlaceholder')"
            clearable
          />
          <div class="field-tip">{{ t("pages.notification_channel.smsTemplateCodeHint") }}</div>
        </ElFormItem>

        <!-- 腾讯云两项必填、阿里云留空：必留必填差在 field-tip 里说清，
             不按服务商切 required——选错服务商时让后端报错点名缺哪个字段，比模糊的红星更可排障。 -->
        <ElFormItem :label="t('pages.notification_channel.smsAppId')" prop="smsAppId">
          <ElInput v-model="drawer.formData.smsAppId" placeholder="1400000000" clearable />
          <div class="field-tip">{{ t("pages.notification_channel.smsAppIdHint") }}</div>
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smsRegion')" prop="smsRegion">
          <ElInput v-model="drawer.formData.smsRegion" placeholder="ap-guangzhou" clearable />
          <div class="field-tip">{{ t("pages.notification_channel.smsRegionHint") }}</div>
        </ElFormItem>
      </template>

      <template v-else>
        <ElFormItem :label="t('pages.notification_channel.smtpHost')" prop="smtpHost">
          <ElInput v-model="drawer.formData.smtpHost" placeholder="smtp.example.com" clearable />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpPort')" prop="smtpPort">
          <ElInputNumber
            v-model="drawer.formData.smtpPort"
            :min="1"
            :max="65535"
            :precision="0"
            controls-position="right"
          />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpUsername')" prop="smtpUsername">
          <ElInput v-model="drawer.formData.smtpUsername" clearable />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.password')" prop="password">
          <ElInput
            v-model="drawer.formData.password"
            type="password"
            show-password
            :placeholder="isCreate ? t('pages.notification_channel.passwordPlaceholder') : t('pages.notification_channel.passwordKeepHint')"
          />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpFrom')" prop="smtpFrom">
          <ElInput v-model="drawer.formData.smtpFrom" placeholder="noreply@example.com" clearable />
        </ElFormItem>
        <ElFormItem :label="t('pages.notification_channel.smtpFromName')" prop="smtpFromName">
          <ElInput v-model="drawer.formData.smtpFromName" placeholder="GoWind" clearable />
        </ElFormItem>

        <ElFormItem :label="t('pages.notification_channel.smtpTls')" prop="smtpTls">
          <ElSelect
          :teleported="false" v-model="drawer.formData.smtpTls">
            <ElOption :label="t('pages.notification_channel.tlsNone')" value="NONE" />
            <ElOption :label="t('pages.notification_channel.tlsStartTls')" value="START_TLS" />
            <ElOption :label="t('pages.notification_channel.tlsSsl')" value="SSL" />
          </ElSelect>
        </ElFormItem>
      </template>

      <ElFormItem :label="t('pages.notification_channel.enabled')" prop="enabled">
        <ElSwitch v-model="drawer.formData.enabled" />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_channel.remark')" prop="remark">
        <ElInput v-model="drawer.formData.remark" type="textarea" :rows="2" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="drawer.guard.requestClose(drawer.close)">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton
          type="primary"
          :loading="drawer.submitLoading.value"
          @click="drawer.handleSubmit(formRef, () => emit('success'))"
        >
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElSwitch,
} from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import { computed, ref } from "vue";
import { useI18n } from "@/core/i18n";
import {
  createNotificationChannel,
  fetchListChannelSchemas,
  updateNotificationChannel,
  type NotificationChannelSchema,
} from "@/api/composables";

/**
 * 扩展渠道 schema（ListChannelSchemas）与分组下拉/动态表单的联动。
 * 必须是 ref：setup 期发起请求，模板与分组计算都依赖它就绪后的数据。
 */
const extSchemas = ref<NotificationChannelSchema[]>([]);
fetchListChannelSchemas()
  .then((items) => {
    extSchemas.value = items;
  })
  .catch(() => {
    // schema 拉取失败不阻塞渠道 CRUD：EXT 类型下拉会缺项，错误由请求层日志带出
  });

/** 是否扩展渠道类型（EXT_*：配置整包走 settings）。 */
function isExtType(type?: string): boolean {
  return !!type && extSchemas.value.some((s) => s.type === type);
}

function extSchemaOf(type?: string): NotificationChannelSchema | undefined {
  return isExtType(type) ? extSchemas.value.find((s) => s.type === type) : undefined;
}

const GROUP_LABELS: Record<string, string> = {
  IM: "IM / 群聊",
  PUSH: "个人推送",
  MOBILE: "移动推送",
  SMS: "短信 / 消息",
  MAIL: "邮件 API",
  AUTO: "效率自动化",
  OPS: "运维告警",
};

/** 分组下拉：按 schema 分组（阿里云/腾讯云短信是老三样，平铺在分组外不在此列）。 */
const extTypeGroups = computed(() => {
  const acc: { group: string; label: string; options: { label: string; value: string }[] }[] = [];
  for (const s of extSchemas.value) {
    const group = s.group ?? "EXT";
    let node = acc.find((g) => g.group === group);
    if (!node) {
      node = { group, label: GROUP_LABELS[group] ?? group, options: [] };
      acc.push(node);
    }
    node.options.push({ label: s.label ?? s.type ?? "", value: s.type ?? "" });
  }
  // 阿里云/腾讯云拆成两个一级选项（与华为云并列）；合成键 SMS@provider 提交/回显时互转
  const smsGroup = acc.find((g) => g.group === "SMS");
  if (smsGroup) {
    smsGroup.options.unshift(
      {
        label: t("pages.notification_channel.smsAliyun"),
        value: "SMS@ALIYUN",
      },
      {
        label: t("pages.notification_channel.smsTencent"),
        value: "SMS@TENCENT",
      },
    );
  }
  return acc;
});

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const formRef = ref();
const isCreate = ref(true);

/**
 * 载荷模板的可引用变量清单。不写进词条，两个原因：这些名字是接口契约的一部分
 * （三种语言都不译），且 vue-i18n 会把消息里的花括号当插值语法解析。
 */
const WEBHOOK_TEMPLATE_VARS = [
  "title",
  "content",
  "event_type",
  "timestamp",
  "sign",
  "nonce",
  "recipient_user_id",
  "related_id",
  "delivered_at",
]
  .map((name) => `{{${name}}}`)
  .join(" ");

// 输入框里的示例取自钉钉那一档的内置形状：留空时后端就发这个形状，示例只是提示可以改写。
const WEBHOOK_TEMPLATE_EXAMPLE = '{"msgtype":"text","text":{"content":"{{title}}"}}';

const drawer = useDrawerForm({
  moduleKey: "pages.notification_channel.moduleName",
  defaults: {
    name: "",
    type: "EMAIL",
    smtpHost: "",
    smtpPort: 587,
    smtpUsername: "",
    password: "",
    smtpFrom: "",
    smtpFromName: "",
    smtpTls: "START_TLS",
    webhookUrl: "",
    webhookSignStyle: "CUSTOM",
    webhookPayloadTemplate: "",
    webhookSecret: "",
    smsProvider: "ALIYUN",
    smsAccessKeyId: "",
    settings: {} as Record<string, any>,
    smsAccessSecret: "",
    smsSignName: "",
    smsTemplateCode: "",
    smsAppId: "",
    smsRegion: "",
    enabled: true,
    remark: "",
  },
  // password / webhookSecret / smsAccessSecret 是请求级字段，与 data 并列发出去：读视图只有
  // hasPassword / hasWebhookSecret / hasSmsAccessSecret 三个布尔，所以它们既不进 data、也不进 updateMask。
  createFn: async (values: Record<string, any>) => {
    const { password, webhookSecret, smsAccessSecret, ...data } =
      splitSmsComposite(values);
    return createNotificationChannel(data, password, webhookSecret, smsAccessSecret);
  },
  updateFn: (id: string, values: Record<string, any>) => {
    const { password, webhookSecret, smsAccessSecret, ...data } =
      splitSmsComposite(values);
    return updateNotificationChannel(
      id,
      data,
      password || undefined,
      webhookSecret || undefined,
      smsAccessSecret || undefined,
    );
  },
});

/**
 * 拆分选项合成键还原：SMS@ALIYUN / SMS@TENCENT → type=SMS + smsProvider。
 * 下拉里阿里云/腾讯云是两个一级选项，数据面仍是同一条 SMS 渠道行（sms_provider 列区分）。
 */
function splitSmsComposite(values: Record<string, any>): Record<string, any> {
  const { ...data } = values;
  if (String(data.type ?? "").startsWith("SMS@")) {
    data.smsProvider = String(data.type).split("@")[1];
    data.type = "SMS";
  }
  return data;
}

/** 当前类型对应的扩展渠道 schema（非 EXT 类型为 undefined）。 */
const extSchema = computed(() =>
  isExtType(drawer.formData.type)
    ? extSchemaOf(drawer.formData.type)
    : undefined,
);

const formRules = computed(() => ({
  name: [{ required: true, message: t("pages.notification_channel.requiredName"), trigger: "blur" }],
  // SMTP 密码不填等于建了个发不出信的渠道，故创建时必填；签名密钥始终可选（不签名是合法
  // 配置），所以 WEBHOOK 组里只有 webhookUrl 在创建时必填。SMS 组的四个必填同理：
  // 缺一项就是一条发不出验证码的渠道。各类规则按当前 type 生效。
  password:
    isCreate.value && drawer.formData.type === "EMAIL"
      ? [{ required: true, message: t("pages.notification_channel.requiredPassword"), trigger: "blur" }]
      : [],
  webhookUrl:
    isCreate.value && drawer.formData.type === "WEBHOOK"
      ? [
          {
            required: true,
            message: t("pages.notification_channel.requiredWebhookUrl"),
            trigger: "blur",
          },
        ]
      : [],
  smsProvider:
    drawer.formData.type === "SMS"
      ? [{ required: true, message: t("pages.notification_channel.requiredSmsProvider"), trigger: "change" }]
      : [],
  smsAccessKeyId:
    drawer.formData.type === "SMS"
      ? [{ required: true, message: t("pages.notification_channel.requiredSmsAccessKeyId"), trigger: "blur" }]
      : [],
  smsAccessSecret:
    isCreate.value && drawer.formData.type === "SMS"
      ? [{ required: true, message: t("pages.notification_channel.requiredSmsAccessSecret"), trigger: "blur" }]
      : [],
  smsSignName:
    drawer.formData.type === "SMS"
      ? [{ required: true, message: t("pages.notification_channel.requiredSmsSignName"), trigger: "blur" }]
      : [],
  smsTemplateCode:
    drawer.formData.type === "SMS"
      ? [{ required: true, message: t("pages.notification_channel.requiredSmsTemplateCode"), trigger: "blur" }]
      : [],
}));

// 包装 open：追踪创建/编辑模式（切换类型字段禁用态与密钥必填规则），
// 并在编辑时显式回填行数据（useDrawerForm 不做默认填充；hasPassword /
// hasWebhookSecret 为服务端计算字段、两处密钥均不回显，故不进表单）。
function open(options: { create: boolean; row?: any }) {
  isCreate.value = options.create;
  drawer.open(options, (row: any) => {
    Object.assign(drawer.formData, {
      name: row.name || "",
      type:
        row.type === "SMS"
          ? `SMS@${row.smsProvider ?? "ALIYUN"}`
          : row.type || "EMAIL",
      // settings 读回显是脱敏 JSON 串：parse 成对象供动态表单回填（sensitive 空串占位）
      settings: row.settings ? parseSettingsSafe(row.settings) : {},
      smtpHost: row.smtpHost || "",
      smtpPort: row.smtpPort ?? 587,
      smtpUsername: row.smtpUsername || "",
      smtpFrom: row.smtpFrom || "",
      smtpFromName: row.smtpFromName || "",
      smtpTls: row.smtpTls || "START_TLS",
      webhookUrl: row.webhookUrl || "",
      // 存量行这两列是空的，而空值的实际行为就是 CUSTOM（后端 resolveWebhookStyle 的归一化）：
      // 表单里摆成 CUSTOM，管理员才改得动，也不会以为"没选=没签名"。
      webhookSignStyle: row.webhookSignStyle || "CUSTOM",
      webhookPayloadTemplate: row.webhookPayloadTemplate || "",
      smsProvider: row.smsProvider || "ALIYUN",
      smsAccessKeyId: row.smsAccessKeyId || "",
      smsSignName: row.smsSignName || "",
      smsTemplateCode: row.smsTemplateCode || "",
      smsAppId: row.smsAppId || "",
      smsRegion: row.smsRegion || "",
      enabled: !!row.enabled,
      remark: row.remark || "",
    });
  });
}

defineExpose({ open });

/** settings 读回显 JSON 容错解析：坏数据返回空对象，动态表单按全空处理。 */
function parseSettingsSafe(raw: string): Record<string, any> {
  try {
    const parsed = JSON.parse(raw);
    return typeof parsed === "object" && parsed !== null ? parsed : {};
  } catch {
    return {};
  }
}
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.field-tip {
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
