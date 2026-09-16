// https://umijs.org/config/

import { join } from 'node:path';
import { defineConfig } from '@umijs/max';
import defaultSettings from './defaultSettings';
import { getProxyConfig } from './proxy';

import routes from './routes';

const { UMI_ENV = 'dev' } = process.env;

// Compute commit hash: env vars take precedence, fall back to git at build time
const commitHash =
  process.env.COMMIT_HASH ||
  process.env.CF_PAGES_COMMIT_SHA ||
  (() => {
    try {
      return require('node:child_process')
        .execSync('git rev-parse HEAD', {
          stdio: ['ignore', 'pipe', 'ignore'],
          encoding: 'utf-8',
        })
        .trim();
    } catch {
      return '';
    }
  })();

/**
 * @name 使用公共路径
 * @description 部署时的路径，如果部署在非根目录下，需要配置这个变量
 * @doc https://umijs.org/docs/api/config#publicpath
 */
const PUBLIC_PATH: string = '/';

export default defineConfig({
  alias: {
    '@root': join(__dirname, '..'),
  },
  /**
   * @name 开启 hash 模式
   * @description 让 build 之后的产物包含 hash 后缀。通常用于增量发布和避免浏览器加载缓存。
   * @doc https://umijs.org/docs/api/config#hash
   */
  hash: true,

  publicPath: PUBLIC_PATH,

  devtool: 'source-map',

  /**
   * @name 兼容性设置
   * @description 设置 ie11 不一定完美兼容，需要检查自己使用的所有依赖
   * @doc https://umijs.org/docs/api/config#targets
   */
  // targets: {
  //   ie: 11,
  // },
  /**
   * @name 路由的配置，不在路由中引入的文件不会编译
   * @description 只支持 path，component，routes，redirect，wrappers，title 的配置
   * @doc https://umijs.org/docs/guides/routes
   */
  // umi routes: https://umijs.org/docs/routing
  routes,
  /**
   * @name 主题的配置
   * @description 虽然叫主题，但是其实只是 less 的变量设置
   * @doc antd的主题设置 https://ant.design/docs/react/customize-theme-cn
   * @doc umi 的 theme 配置 https://umijs.org/docs/api/config#theme
   */
  // theme: { '@primary-color': '#1DA57A' }
  /**
   * @name moment 的国际化配置
   * @description 如果对国际化没有要求，打开之后能减少js的包大小
   * @doc https://umijs.org/docs/api/config#ignoremomentlocale
   */
  ignoreMomentLocale: true,
  /**
   * @name 代理配置
   * @description 可以让你的本地服务器代理到你的服务器上，这样你就可以访问服务器的数据了
   * @see 要注意以下 代理只能在本地开发时使用，build 之后就无法使用了。
   * @doc 代理介绍 https://umijs.org/docs/guides/proxy
   * @doc 代理配置 https://umijs.org/docs/api/config#proxy
   */
  proxy: getProxyConfig(UMI_ENV),
  /**
   * @name 快速热更新配置
   * @description 一个不错的热更新组件，更新时可以保留 state
   */
  fastRefresh: true,
  /**
   * @name 路由预加载
   * @description 预加载路由资源，提升页面切换速度
   * @doc https://umijs.org/docs/api/config#routePrefetch
   */
  routePrefetch: {},
  /**
   * @name manifest 配置
   * @description 生成资源清单，配合 routePrefetch 使用
   */
  manifest: {},
  //============== 以下都是max的插件配置 ===============
  /**
   * @name 数据流插件
   * @@doc https://umijs.org/docs/max/data-flow
   */
  model: {},
  /**
   * 一个全局的初始数据流，可以用它在插件之间共享数据
   * @description 可以用来存放一些全局的数据，比如用户信息，或者一些全局的状态，全局初始状态在整个 Umi 项目的最开始创建。
   * @doc https://umijs.org/docs/max/data-flow#%E5%85%A8%E5%B1%80%E5%88%9D%E5%A7%8B%E7%8A%B6%E6%80%81
   */
  initialState: {},
  /**
   * @name layout 插件
   * @doc https://umijs.org/docs/max/layout-menu
   */
  title: 'Roncin 货代后台',
  layout: {
    locale: false,
    ...defaultSettings,
  },
  /**
   * @name moment2dayjs 插件
   * @description 将项目中的 moment 替换为 dayjs
   * @doc https://umijs.org/docs/max/moment2dayjs
   */
  moment2dayjs: {
    preset: 'antd',
    plugins: ['duration', 'relativeTime'],
  },
  /**
   * @name 国际化插件
   * @doc https://umijs.org/docs/max/i18n
   */
  locale: {
    // default zh-CN
    default: 'zh-CN',
    antd: true,
    // default true, when it is true, will use `navigator.language` overwrite default
    baseNavigator: true,
  },
  /**
   * @name antd 插件
   * @description 内置了 babel import 插件
   * @doc https://umijs.org/docs/max/antd#antd
   */
  antd: {
    appConfig: {},
    configProvider: {
      theme: {
        token: {
          fontFamily:
            'AlibabaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
          borderRadius: 6,
          borderRadiusSM: 4,
          borderRadiusLG: 8,
          colorPrimary: '#1677ff',
          colorBgLayout: '#f5f7fa',
          colorBgContainer: '#ffffff',
          colorBorder: '#e2e8f0',
          colorBorderSecondary: '#f1f5f9',
          colorText: '#0f172a',
          colorTextSecondary: '#475569',
          colorTextTertiary: '#94a3b8',
          colorTextQuaternary: '#cbd5e1',
          controlHeight: 32,
          controlHeightSM: 24,
          fontSize: 13,
          boxShadow:
            '0 1px 3px 0 rgba(0, 0, 0, 0.04), 0 1px 2px -1px rgba(0, 0, 0, 0.02)',
          boxShadowSecondary:
            '0 4px 6px -1px rgba(0, 0, 0, 0.06), 0 2px 4px -2px rgba(0, 0, 0, 0.04)',
          boxShadowTertiary:
            '0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -4px rgba(0, 0, 0, 0.04)',
        },
        components: {
          Card: {
            paddingLG: 14,
            padding: 12,
            paddingSM: 8,
            headerHeight: 40,
            headerFontSize: 13,
            colorBorderSecondary: '#f1f5f9',
          },
          Table: {
            headerBg: '#f8fafc',
            headerColor: '#475569',
            headerSortActiveBg: '#f1f5f9',
            headerSortHoverBg: '#f1f5f9',
            rowHoverBg: '#f8fafc',
            rowSelectedBg: '#eff6ff',
            rowSelectedHoverBg: '#dbeafe',
            cellPaddingBlock: 8,
            cellPaddingInline: 12,
            cellPaddingBlockSM: 6,
            cellPaddingInlineSM: 8,
            fontSize: 13,
            borderColor: '#f1f5f9',
            headerSplitColor: 'transparent',
          },
          Button: {
            controlHeight: 32,
            controlHeightSM: 24,
            paddingInline: 12,
            paddingInlineSM: 8,
            borderRadius: 6,
            defaultBorderColor: '#e2e8f0',
            defaultColor: '#334155',
            defaultBg: '#ffffff',
            defaultHoverBorderColor: '#cbd5e1',
            defaultHoverColor: '#0f172a',
            defaultHoverBg: '#f8fafc',
          },
          Input: {
            colorBorder: '#e2e8f0',
            hoverBorderColor: '#cbd5e1',
            activeBorderColor: '#1677ff',
            activeShadow: '0 0 0 3px rgba(22, 119, 255, 0.12)',
          },
          Select: {
            colorBorder: '#e2e8f0',
            hoverBorderColor: '#cbd5e1',
          },
          DatePicker: {
            colorBorder: '#e2e8f0',
            hoverBorderColor: '#cbd5e1',
          },
          Tabs: {
            itemColor: '#64748b',
            itemHoverColor: '#0f172a',
            itemSelectedColor: '#1677ff',
            titleFontSize: 13,
            horizontalItemPadding: '8px 12px',
          },
          Tag: {
            defaultBg: '#f1f5f9',
            defaultColor: '#475569',
          },
          Form: {
            itemMarginBottom: 12,
            verticalLabelPadding: '0 0 4px',
          },
          Modal: {
            headerBg: '#ffffff',
            contentBg: '#ffffff',
            borderRadiusLG: 10,
          },
          Drawer: {
            paddingLG: 16,
          },
        },
      },
    },
  },
  /**
   * @name 网络请求配置
   * @description 它基于 axios 和 ahooks 的 useRequest 提供了一套统一的网络请求和错误处理方案。
   * @doc https://umijs.org/docs/max/request
   */
  request: {},
  /**
   * @name 权限插件
   * @description 基于 initialState 的权限插件，必须先打开 initialState
   * @doc https://umijs.org/docs/max/access
   */
  access: {},
  /**
   * @name <head> 中额外的 script
   * @description 配置 <head> 中额外的 script
   */
  headScripts: [
    // 解决首次加载时白屏的问题
    { src: join(PUBLIC_PATH, 'scripts/loading.js'), async: true },
  ],

  //================ pro 插件配置 =================
  plugins: ['@umijs/max-plugin-openapi', '@umijs/request-record'],

  /**
   * @name openAPI 插件的配置
   * @description 基于 openapi 的规范生成serve 和mock，能减少很多样板代码
   * @doc https://pro.ant.design/zh-cn/docs/openapi/
   */
  openAPI: [
    {
      requestLibPath: "import { request } from '@umijs/max'",
      schemaPath: join(__dirname, 'openapi.generated.json'),
      projectName: 'roncin',
      mock: false,
    },
  ],

  tailwindcss: {},

  mock: {
    include: ['src/pages/**/_mock.ts'],
    exclude: ['mock/requestRecord.mock.js'],
  },
  utoopack: {
    module: {
      rules: {
        '*.md': {
          loaders: [{ loader: join(__dirname, 'md-raw-loader.cjs') }],
          as: '*.js',
        },
      },
    },
  },
  requestRecord: {},
  define: {
    'process.env.CI': process.env.CI,
    'process.env.COMMIT_HASH': commitHash,
    // UMI_ENV 供运行时代码推导 Sentry environment（dev/test/prod）。
    'process.env.UMI_ENV': process.env.UMI_ENV ?? '',
    // Sentry 轻量接入：DSN 由部署环境注入，未配置时前端完全跳过初始化。
    'process.env.SENTRY_DSN': process.env.SENTRY_DSN ?? '',
    __APP_VERSION__: require('./../package.json').version,
    __UMI_VERSION__: require('@umijs/max/package.json').version,
  },
});
