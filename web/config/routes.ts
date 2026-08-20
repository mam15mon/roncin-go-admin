export default [
  {
    path: '/user',
    layout: false,
    routes: [
      {
        name: '登录',
        path: '/user/login',
        component: './user/login',
      },
    ],
  },
  {
    path: '/welcome',
    name: '工作台',
    icon: 'dashboard',
    access: 'canAccessPlatform',
    component: './Welcome',
  },
  {
    path: '/partners',
    name: '客户与供应商',
    icon: 'contacts',
    access: 'canReadPartners',
    component: './partners',
  },
  {
    path: '/',
    redirect: '/welcome',
  },
  {
    component: './NotFound',
    layout: false,
    path: './*',
  },
];
