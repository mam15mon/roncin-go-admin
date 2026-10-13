// OpenAPI 客户端生成入口：直调 @umijs/openapi（与原 max openapi 同引擎，
// 输出与既有 *Service.ts 风格完全一致），仅 requestLibPath 指向自建客户端。
// 输入由 generate-openapi-input.mjs 从 server/openapi.yaml 生成。

import { readdir, readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { generateService } from '@umijs/openapi';

// @umijs/openapi 对相对路径按自身包目录解析，必须传绝对路径。
const webRoot = fileURLToPath(new URL('..', import.meta.url));

await generateService({
  schemaPath: `${webRoot}config/openapi.generated.json`,
  serversPath: `${webRoot}src/services`,
  projectName: 'roncin',
  requestLibPath: "import { request } from '@/utils/requestClient'",
});

// @umijs/openapi 将字面动作后缀（:batchCreate）误识别为路径变量。
// 以契约中声明的参数为界，仅恢复含自定义动作的完整路径；普通路由不参与替换。
const document = JSON.parse(
  await readFile(`${webRoot}config/openapi.generated.json`, 'utf8'),
);
const customVerbPaths = new Map();
for (const [path, pathItem] of Object.entries(document.paths ?? {})) {
  for (const [method, operation] of Object.entries(pathItem)) {
    if (
      !['get', 'post', 'put', 'patch', 'delete', 'head', 'options'].includes(
        method,
      )
    )
      continue;
    const parameters = [
      ...(pathItem.parameters ?? []),
      ...(operation.parameters ?? []),
    ].filter((parameter) => parameter.in === 'path');
    const parameterNames = new Set(
      parameters.map((parameter) => parameter.name),
    );
    const suffixes = [...path.matchAll(/:([a-zA-Z][a-zA-Z0-9]*)(?=\/|$)/g)];
    if (
      !suffixes.length ||
      suffixes.some(([, name]) => parameterNames.has(name))
    )
      continue;
    // 仅接受 OpenAPI 的 {param} 占位；未声明的占位或 :param 形态都明确拒绝处理。
    const braceParameters = [...path.matchAll(/\{([^}]+)\}/g)];
    if (braceParameters.some(([, name]) => !parameterNames.has(name))) continue;
    if (parameters.some((parameter) => !path.includes(`{${parameter.name}}`)))
      continue;
    let correctPath = path;
    let generatedPath = path.replace(
      /:([^/]*)|{([^}]*)}/gi,
      (_, colonName, braceName) => `\${${colonName || braceName}}`,
    );
    parameters.forEach((parameter, index) => {
      correctPath = correctPath.replace(
        `{${parameter.name}}`,
        `\${param${index}}`,
      );
      generatedPath = generatedPath.replace(
        `\${${parameter.name}}`,
        `\${param${index}}`,
      );
    });
    customVerbPaths.set(generatedPath, correctPath);
  }
}
const servicesDirectory = `${webRoot}src/services/roncin`;
for (const file of await readdir(servicesDirectory)) {
  if (!file.endsWith('Service.ts')) continue;
  const filePath = `${servicesDirectory}/${file}`;
  const content = await readFile(filePath, 'utf8');
  let repaired = content;
  for (const [generatedPath, correctPath] of customVerbPaths) {
    repaired = repaired.replaceAll(generatedPath, correctPath);
  }
  if (repaired !== content) await writeFile(filePath, repaired, 'utf8');
}
