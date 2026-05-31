import type { App } from 'vue';

import {
  Button,
  Card,
  Divider,
  Dropdown,
  Input,
  Layout,
  Menu,
  Popconfirm,
  Select,
  Space,
  Switch,
  Tabs,
  Tag,
  Tree,
} from 'ant-design-vue';

/**
 * 注册全局组件
 * @param app
 */
export function registerGlobComp(app: App) {
  app
    .use(Input)
    .use(Button)
    .use(Layout)
    .use(Space)
    .use(Card)
    .use(Switch)
    .use(Popconfirm)
    .use(Dropdown)
    .use(Tag)
    .use(Tabs)
    .use(Divider)
    .use(Menu)
    .use(Select)
    .use(Tree);
}
