import assert from 'node:assert/strict';
import test from 'node:test';
import { extractFunctions } from './duplicate-functions.mjs';

const first = (source) => extractFunctions(source, 'sample.ts')[0];
test('忽略格式、注释及声明名，保留类型与字面量', () => {
  assert.equal(
    first('function a(x: number) { return x + 1; }').exact,
    first('function b(x: number) { /* 注释 */ return x+1; }').exact,
  );
  for (const source of [
    'function a(x: number) { return x - 1; }',
    'function a(x: number) { return x + 2; }',
    'function a(x: string) { return x + 1; }',
  ])
    assert.notEqual(
      first(source).exact,
      first('function a(x: number) { return x + 1; }').exact,
    );
});
test('参数、局部、块作用域、闭包与自递归按绑定改名', () => {
  const a = first(
    'function a(x: number) { const y = x + 1; { let x = 2; use(x); } return () => a(y); }',
  );
  const b = first(
    'function b(input: number) { const next = input + 1; { let inner = 2; use(inner); } return () => b(next); }',
  );
  assert.notEqual(a.exact, b.exact);
  assert.equal(a.renamed, b.renamed);
  assert.ok(a.renamed);
});
test('自由变量、属性名、简写对象键、标签及捕获目标均保持差异', () => {
  const pairs = [
    ['function a(x) { return api(x); }', 'function b(y) { return other(y); }'],
    ['function a(x) { return x.name; }', 'function b(y) { return y.code; }'],
    ['function a(x) { return {x}; }', 'function b(y) { return {y}; }'],
    ['function a(x) { return () => x; }', 'function b(y) { return () => x; }'],
    [
      'function a(x) { { let y = 1; return () => x; } }',
      'function b(x) { { let y = 1; return () => y; } }',
    ],
    [
      'function a(x) { loop: while(x) break loop; }',
      'function b(y) { again: while(y) break again; }',
    ],
  ];
  for (const [a, b] of pairs)
    assert.notEqual(first(a).renamed, first(b).renamed);
});
test('复杂绑定明确仅参与 exact；解析错误不吞掉', () => {
  for (const source of [
    'function a({x}) { return x; }',
    'function a(x=1) { return x; }',
    'function a(...x) { return x; }',
    'function a(x) { var y=x; return y; }',
    'function a(x) { return eval(x); }',
  ]) {
    assert.equal(first(source).renamed, null);
    assert.ok(first(source).renameUnsupported);
  }
  assert.throws(() => first('function broken( {'));
});
test('覆盖箭头函数、对象方法、类方法和 JSX', () => {
  const records = extractFunctions(
    'const a = () => 1; const o = { method() { return 2; } }; class C { run() { return 3; } }; const View=()=> <div/>;',
    'sample.tsx',
  );
  assert.equal(records.length, 4);
  assert.equal(records.at(-1).renamed, null);
});
test('计算方法名保留表达式，局部标识不能与自由变量占位串碰撞', () => {
  assert.notEqual(
    first('const a={ [foo()]() {return 1;} };').exact,
    first('const b={ [bar()]() {return 1;} };').exact,
  );
  assert.notEqual(
    first('function a(x){return x;}').renamed,
    first('function b(y){return $local0;}').renamed,
  );
});
test('自递归与外部同名调用区分，switch 保守降为 exact', () => {
  const records = extractFunctions(
    'function f(){return f()} function g(){return f()}',
    'sample.ts',
  );
  assert.notEqual(records[0].exact, records[1].exact);
  assert.notEqual(records[0].renamed, records[1].renamed);
  assert.equal(
    first('function a(x){switch(x){default:let x=1;return x}}').renamed,
    null,
  );
});
test('计算方法键不能捕获方法参数', () => {
  const record = first('const a={ [x](x){return x} };');
  assert.equal(record.renamed, null);
  assert.match(record.renameUnsupported, /计算方法名/);
});
test('默认参数中的自引用也与外部同名绑定区分', () => {
  const records = extractFunctions(
    'function f(x=f){return x;} function g(x=f){return x;}',
    'sample.ts',
  );
  assert.notEqual(records[0].exact, records[1].exact);
});

test('TS 运行时包装内按绑定改名，区分参数和外部变量', () => {
  for (const wrap of [
    (x) => `${x} as number`,
    (x) => `${x}!`,
    (x) => `${x} satisfies number`,
    (x) => `<number>${x}`,
    (x) => `${x}<number>`,
    (x) => `(${x}! as number) satisfies number`,
  ]) {
    const record = (param, value) =>
      extractFunctions(
        `function run(${param}) { return ${wrap(value)}; }`,
        'a.ts',
      )[0];
    assert.equal(record('x', 'x').renamed, record('y', 'y').renamed);
    assert.notEqual(record('x', 'x').renamed, record('y', 'x').renamed);
    assert.equal(record('x', 'x').renameUnsupported, null);
  }
});

test('TS 包装中的 eval 与复杂绑定仍降为 exact-only，类型差异保留', () => {
  for (const expression of [
    '(eval as Function)(code)',
    'eval!(code)',
    '(eval satisfies Function)(code)',
    '(eval(code) as unknown)',
    '((eval! as Function) satisfies Function)(code)',
  ]) {
    const record = extractFunctions(
      `function run(code) { return ${expression}; }`,
      'a.ts',
    )[0];
    assert.equal(record.renamed, null);
    assert.match(record.renameUnsupported, /eval/);
  }
  const extract = (body) =>
    extractFunctions(`function run(x) { ${body} }`, 'a.ts')[0];
  assert.notEqual(
    extract('return x as number;').renamed,
    extract('return x as string;').renamed,
  );
  assert.equal(
    extract('return (({ value }) => value) as Function;').renamed,
    null,
  );
});
