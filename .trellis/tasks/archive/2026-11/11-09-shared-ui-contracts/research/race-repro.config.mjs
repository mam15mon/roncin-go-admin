// 从 pnpm --dir web 的工作目录定位依赖；配置随任务归档后只需更新 --config 路径。
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = process.cwd();
export default {
  root: webRoot,
  resolve: {
    alias: [
      { find: 'research-testing', replacement: resolve(webRoot, 'node_modules/@testing-library/react/dist/index.js') },
      { find: 'research-react', replacement: resolve(webRoot, 'node_modules/react/index.js') },
      { find: 'research-vitest', replacement: resolve(webRoot, 'node_modules/vitest/dist/index.js') },
      { find: 'research-pro-fetch', replacement: resolve(webRoot, 'node_modules/@ant-design/pro-components/es/table/useFetchData.js') },
    ],
  },
  test: {
    environment: 'happy-dom',
    include: [fileURLToPath(new URL('./race-repro.test.mjs', import.meta.url))],
    testTimeout: 10000,
  },
};
