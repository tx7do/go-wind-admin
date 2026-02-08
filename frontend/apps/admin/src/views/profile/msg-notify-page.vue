<script lang="ts" setup>
import { Page } from '@vben/common-ui';

import { List, Switch } from 'ant-design-vue';

const ListItem = List.Item;
const ListItemMeta = List.Item.Meta;

const msgNotifyList = [
  {
    key: '1',
    title: '账户密码',
    description: '其他用户的消息将以站内信的形式通知',
  },
  {
    key: '2',
    title: '系统消息',
    description: '系统消息将以站内信的形式通知',
  },
  {
    key: '3',
    title: '待办任务',
    description: '待办任务将以站内信的形式通知',
  },
];
</script>

<template>
  <Page :title="$t('page.user.profile.tab.notification')">
    <List>
      <template v-for="item in msgNotifyList" :key="item.key">
        <ListItem>
          <ListItemMeta>
            <template #title>
              {{ item.title }}
              <Switch
                class="extra"
                checked-children="开"
                un-checked-children="关"
                default-checked
              />
            </template>
            <template #description>
              <div>{{ item.description }}</div>
            </template>
          </ListItemMeta>
        </ListItem>
      </template>
    </List>
  </Page>
</template>

<style lang="less" scoped>
.extra {
  float: right;
  margin-top: 10px;
  margin-right: 30px;
}
</style>
