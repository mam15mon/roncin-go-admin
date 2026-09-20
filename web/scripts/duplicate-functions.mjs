// 函数结构指纹：仅作疑似重复线索，不推断业务等价。
import { parse } from '@babel/parser';

const ignored = new Set([
  'start',
  'end',
  'loc',
  'extra',
  'leadingComments',
  'trailingComments',
  'innerComments',
  'comments',
  'tokens',
  'errors',
]);
const functions = new Set([
  'FunctionDeclaration',
  'FunctionExpression',
  'ArrowFunctionExpression',
  'ObjectMethod',
  'ClassMethod',
  'ClassPrivateMethod',
]);

function children(node, visit) {
  for (const [key, value] of Object.entries(node)) {
    if (ignored.has(key)) continue;
    if (Array.isArray(value)) {
      for (const child of value) if (child?.type) visit(child, key);
    } else if (value?.type) visit(value, key);
  }
}

function canonical(node, rename = new Map(), root = node) {
  if (Array.isArray(node))
    return node.map((item) => canonical(item, rename, root));
  if (!node || typeof node !== 'object') return node;
  const result = {};
  for (const key of Object.keys(node).sort()) {
    if (
      ignored.has(key) ||
      (node === root && (key === 'id' || (key === 'key' && !node.computed)))
    )
      continue;
    if (key === 'name' && rename.has(node)) result.binding = rename.get(node);
    else result[key] = canonical(node[key], rename, root);
  }
  return result;
}

// 遇到暂不支持的绑定形式整函数降为 exact-only，不能猜测标识符用途。
function localBindings(root) {
  let reason = null;
  let nextId = 0;
  const scopes = new Map();
  const declarations = new Map();
  const names = new Map();
  function bind(id, scope) {
    if (id?.type !== 'Identifier') {
      reason ??= '解构、默认值或剩余参数暂不规范化';
      return;
    }
    if (!scope.names.has(id.name))
      scope.names.set(id.name, `$local${nextId++}`);
    declarations.set(id, scope.names.get(id.name));
  }
  function build(node, scope, typePosition = false) {
    if (typePosition || node.type.startsWith('TS')) return;
    if (
      node.type.startsWith('JSX') ||
      ['ClassDeclaration', 'ClassExpression', 'WithStatement'].includes(
        node.type,
      )
    ) {
      reason ??= 'JSX、类或动态作用域暂不规范化';
    }
    if (node.type === 'SwitchStatement')
      reason ??= 'switch 判别式与分支作用域暂不规范化';
    if (node.type === 'CallExpression' && node.callee?.name === 'eval')
      reason ??= '直接 eval 使用动态绑定';
    if (node !== root && node.type === 'FunctionDeclaration')
      reason ??= '嵌套函数声明暂不规范化';
    if (functions.has(node.type)) {
      if (node.computed) reason ??= '计算方法名在外层求值，暂不规范化';
      scope = { parent: scope, names: new Map() };
      if (node.id) bind(node.id, scope);
      for (const param of node.params) bind(param, scope);
    } else if (
      [
        'BlockStatement',
        'ForStatement',
        'ForInStatement',
        'ForOfStatement',
        'CatchClause',
        'SwitchStatement',
      ].includes(node.type)
    ) {
      scope = { parent: scope, names: new Map() };
      if (node.type === 'CatchClause' && node.param) bind(node.param, scope);
    }
    scopes.set(node, scope);
    if (node.type === 'VariableDeclaration') {
      if (node.kind === 'var') reason ??= 'var 提升暂不规范化';
      for (const declaration of node.declarations) bind(declaration.id, scope);
    }
    children(node, (child, key) =>
      build(
        child,
        scope,
        [
          'typeAnnotation',
          'returnType',
          'typeParameters',
          'typeArguments',
        ].includes(key),
      ),
    );
  }
  build(root, { parent: null, names: new Map() });
  function resolve(name, scope) {
    for (let current = scope; current; current = current.parent)
      if (current.names.has(name)) return current.names.get(name);
    return null;
  }
  function references(node, parent, key) {
    if (
      node.type.startsWith('TS') ||
      [
        'typeAnnotation',
        'returnType',
        'typeParameters',
        'typeArguments',
      ].includes(key)
    )
      return;
    if (node.type === 'Identifier') {
      if (declarations.has(node)) names.set(node, declarations.get(node));
      else {
        const property =
          (['MemberExpression', 'OptionalMemberExpression'].includes(
            parent?.type,
          ) &&
            key === 'property' &&
            !parent.computed) ||
          ([
            'ObjectProperty',
            'ObjectMethod',
            'ClassMethod',
            'ClassPrivateMethod',
          ].includes(parent?.type) &&
            key === 'key' &&
            !parent.computed) ||
          (['LabeledStatement', 'BreakStatement', 'ContinueStatement'].includes(
            parent?.type,
          ) &&
            key === 'label');
        if (!property) {
          const name = resolve(node.name, scopes.get(node));
          if (name) names.set(node, name);
        }
      }
    }
    children(node, (child, childKey) => references(child, node, childKey));
  }
  references(root, null, null);
  return { rename: names, reason };
}

// 自引用保留声明名，避免自递归与同名外部调用被判为 exact。
function exactShape(node) {
  const shape = canonical(node);
  if (node.id) {
    let referenced = false;
    function check(child) {
      if (
        child !== node.id &&
        child.type === 'Identifier' &&
        child.name === node.id.name
      )
        referenced = true;
      children(child, check);
    }
    check(node);
    if (referenced) shape.selfDeclaration = node.id.name;
  }
  return shape;
}

export function extractFunctions(source, path) {
  const ast = parse(source, {
    sourceType: 'unambiguous',
    plugins: [
      ...(/\.[cm]?tsx?$/.test(path) ? ['typescript'] : []),
      ...(/\.[jt]sx$/.test(path) ? ['jsx'] : []),
    ],
  });
  const records = [];
  function visit(node, parent) {
    if (functions.has(node.type) && node.body) {
      let nodes = 0;
      function count(child) {
        nodes++;
        children(child, count);
      }
      count(node);
      const binding = localBindings(node);
      const symbol =
        node.id?.name ??
        node.key?.name ??
        node.key?.value ??
        (parent?.type === 'VariableDeclarator' ? parent.id.name : null) ??
        `<匿名@${node.loc.start.line}:${node.loc.start.column + 1}>`;
      records.push({
        language: 'javascript',
        path,
        symbol,
        startLine: node.loc.start.line,
        endLine: node.loc.end.line,
        start: node.start,
        end: node.end,
        nodes,
        exact: JSON.stringify(exactShape(node)),
        renamed: binding.reason
          ? null
          : JSON.stringify(canonical(node, binding.rename)),
        renameUnsupported: binding.reason,
      });
    }
    children(node, (child) => visit(child, node));
  }
  visit(ast, null);
  return records;
}
