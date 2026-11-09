// 主数据响应映射复用门禁：只检查已核对 Proto presence 的实体，不猜测全站字段语义。
import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from '@babel/parser';

const DTO_TYPES = new Set([
  'MasterDataItem',
  'Port',
  'Airport',
  'Airline',
  'ShippingLine',
]);
const SHARED_MAPPERS = new Set([
  'mapPersistedMasterDataFields',
  'mapPersistedMasterDataItem',
]);
const COMMON_FIELDS = new Set(['enabled', 'sortOrder']);

function walk(node, visit, parent) {
  if (!node || typeof node !== 'object') return;
  if (Array.isArray(node)) {
    for (const child of node) walk(child, visit, parent);
    return;
  }
  if (typeof node.type === 'string' && visit(node, parent) === false) return;
  for (const [key, child] of Object.entries(node)) {
    if (key !== 'loc' && key !== 'type') walk(child, visit, node);
  }
}

function dtoType(parameter) {
  const type = parameter.typeAnnotation?.typeAnnotation;
  return (
    type?.type === 'TSTypeReference' &&
    type.typeName?.type === 'TSQualifiedName' &&
    type.typeName.left.name === 'API' &&
    DTO_TYPES.has(type.typeName.right.name)
  );
}

export function checkMapperSource(source, file) {
  const ast = parse(source, {
    sourceType: 'module',
    plugins: ['typescript', 'jsx'],
  });
  const trustedCalls = new Set();
  for (const statement of ast.program.body) {
    if (
      statement.type !== 'ImportDeclaration' ||
      ![
        './masterDataMapper',
        '@/pages/master-data/components/masterDataMapper',
      ].includes(statement.source.value)
    )
      continue;
    for (const specifier of statement.specifiers) {
      if (
        specifier.type === 'ImportSpecifier' &&
        SHARED_MAPPERS.has(specifier.imported.name)
      ) {
        trustedCalls.add(specifier.local.name);
      }
    }
  }
  const findings = [];
  const report = (node, reason) =>
    findings.push({ file, line: node.loc.start.line, reason });
  walk(ast.program, (node, parent) => {
    if (
      ![
        'ArrowFunctionExpression',
        'FunctionExpression',
        'FunctionDeclaration',
      ].includes(node.type)
    )
      return;
    // 仅约束 map* DTO 适配器，不把读取同一 DTO 的业务/展示函数当成转换入口。
    const name =
      node.id?.name ??
      (parent?.type === 'VariableDeclarator' ? parent.id.name : undefined);
    if (!/^map[A-Z]/.test(name ?? '')) return;
    const parameter = node.params[0];
    if (!parameter || !dtoType(parameter)) return;
    let consumesShared = false;
    walk(node.body, (child) => {
      // 内层函数有独立形参作用域，不能把同名局部变量认作外层 DTO。
      if (
        [
          'ArrowFunctionExpression',
          'FunctionExpression',
          'FunctionDeclaration',
        ].includes(child.type)
      )
        return false;
      if (
        child.type === 'CallExpression' &&
        trustedCalls.has(child.callee.name) &&
        child.arguments[0]?.type === 'Identifier' &&
        child.arguments[0].name === parameter.name
      )
        consumesShared = true;
      if (
        ['MemberExpression', 'OptionalMemberExpression'].includes(child.type) &&
        child.object.type === 'Identifier' &&
        child.object.name === parameter.name &&
        COMMON_FIELDS.has(
          child.computed ? child.property.value : child.property.name,
        )
      ) {
        report(
          child,
          'enabled/sortOrder 必须由公共主数据映射处理，禁止逐页读取或重写默认值/缺失校验',
        );
      }
      if (
        child.type === 'VariableDeclarator' &&
        child.id.type === 'ObjectPattern' &&
        child.init?.type === 'Identifier' &&
        child.init.name === parameter.name &&
        child.id.properties.some((property) =>
          COMMON_FIELDS.has(property.key?.name ?? property.key?.value),
        )
      ) {
        report(child, '禁止解构后重新实现 enabled/sortOrder 映射');
      }
    });
    if (!consumesShared)
      report(node, '主数据 DTO 映射必须调用公共 masterDataMapper 入口');
  });
  return findings;
}

export function scanMasterDataMappers(componentsRoot) {
  return readdirSync(componentsRoot)
    .filter(
      (file) =>
        /\.[jt]sx?$/.test(file) &&
        !/\.(test|spec)\./.test(file) &&
        file !== 'masterDataMapper.ts',
    )
    .flatMap((file) =>
      checkMapperSource(readFileSync(join(componentsRoot, file), 'utf8'), file),
    );
}

if (
  process.argv[1] &&
  resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  const root = resolve(
    dirname(fileURLToPath(import.meta.url)),
    '../src/pages/master-data/components',
  );
  const findings = scanMasterDataMappers(root);
  for (const finding of findings)
    console.error(`${finding.file}:${finding.line} ${finding.reason}`);
  if (findings.length) process.exitCode = 1;
  else console.log('主数据公共映射复用检查通过');
}
