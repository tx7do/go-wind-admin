<script lang="ts" setup>
import { ref } from 'vue';

import { useVbenModal } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { useUserListStore } from '#/stores';

const data = ref();

const userListStore = useUserListStore();

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  // 所有表单项共用，可单独在表单内覆盖
  commonConfig: {
    // 所有表单项
    componentProps: {
      class: 'w-full',
    },
  },
  schema: [
    {
      component: 'VbenInputPassword',
      fieldName: 'new_password',
      label: $t('page.user.form.newPassword'),
      componentProps: {
        passwordStrength: true,
        placeholder: $t('ui.placeholder.input'),
      },
      rules: 'required',
    },
    {
      component: 'VbenInputPassword',
      fieldName: 'confirm_password',
      label: $t('page.user.form.confirmPassword'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
      },
      rules: 'required',
    },
  ],
});

const [Modal, modalApi] = useVbenModal({
  onCancel() {
    modalApi.close();
  },

  async onConfirm() {
    console.log('onConfirm');

    // 校验输入的数据
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();

    if (values.new_password !== values.confirm_password) {
      notification.error({
        message: $t('page.notification.password_mismatch'),
      });

      setLoading(false);
      modalApi.close();
      return;
    }

    try {
      await userListStore.editUserPassword(
        data.value?.userId,
        values.new_password,
      );

      setLoading(false);
      modalApi.close();

      notification.success({
        message: $t('ui.notification.update_status_success'),
      });
    } catch {
      setLoading(false);

      notification.error({
        message: $t('ui.notification.update_status_failed'),
      });
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      // 获取传入的数据
      data.value = modalApi.getData<any>();

      setLoading(false);

      console.log('onOpenChange', data.value?.create, data.value?.userId);
    }
  },
});

function setLoading(loading: boolean) {
  modalApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Modal :title="$t('page.user.button.editPassword')">
    <BaseForm />
  </Modal>
</template>
