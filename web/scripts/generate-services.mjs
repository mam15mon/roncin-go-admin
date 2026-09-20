// OpenAPI 客户端生成入口：直调 @umijs/openapi（与原 max openapi 同引擎，
// 输出与既有 *Service.ts 风格完全一致），仅 requestLibPath 指向自建客户端。
// 输入由 generate-openapi-input.mjs 从 server/openapi.yaml 生成。
import { generateService } from '@umijs/openapi';
import { fileURLToPath } from 'node:url';

// @umijs/openapi 对相对路径按自身包目录解析，必须传绝对路径。
const webRoot = fileURLToPath(new URL('..', import.meta.url));

await generateService({
  schemaPath: `${webRoot}config/openapi.generated.json`,
  serversPath: `${webRoot}src/services`,
  projectName: 'roncin',
  requestLibPath: "import { request } from '@/utils/requestClient'",
});
