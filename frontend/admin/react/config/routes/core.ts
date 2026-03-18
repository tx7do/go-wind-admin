const coreRoutes = [
  {
    path: '/user',
    layout: false, // 关键：登录页不显示侧边栏
    routes: [
      {path: '/user/login', component: './auth/login', name: '登录'},
      {path: '/user/register', component: './auth/register', name: '注册'},
    ],
  },
  {
    path: '/exception',
    layout: false,
    routes: [
      {path: '/exception/404', component: './exception/404'},
      {path: '/exception/500', component: './exception/500'},
    ],
  },
];

export default coreRoutes;
