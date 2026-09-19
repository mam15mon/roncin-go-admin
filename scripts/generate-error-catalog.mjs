import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { basename, dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

// 依据 server/internal/biz/**/*.go 提取 errors.* 业务错误定义，
// 重新生成 .trellis/spec/domain/error-catalog.md。
// --check 模式校验文档与代码一致（供 CI 防漂移），用法：
//   node scripts/generate-error-catalog.mjs          生成目录
//   node scripts/generate-error-catalog.mjs --check  校验目录未过期

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const bizRoot = join(repoRoot, 'server/internal/biz');
const outputPath = join(repoRoot, '.trellis/spec/domain/error-catalog.md');

// 构造函数 → HTTP 状态类别（语义列）。PreconditionFailed 当前未使用，预留以覆盖后续新增。
const CONSTRUCTORS = new Map([
  ['NotFound', '404 Not Found'],
  ['BadRequest', '400 Bad Request'],
  ['Conflict', '409 Conflict'],
  ['Forbidden', '403 Forbidden'],
  ['Unauthorized', '401 Unauthorized'],
  ['InternalServer', '500 Internal Server Error'],
  ['ServiceUnavailable', '503 Service Unavailable'],
  ['PreconditionFailed', '412 Precondition Failed'],
]);

// 文件名前缀 → 业务域（先命中者优先），未命中归入「平台」。
const DOMAIN_RULES = [
  ['提成', ['finance_commission']],
  ['财务', ['finance_', 'fee_catalog', 'fee_ledger_preference', 'exchange_rate', 'settlement']],
  ['往来单位', ['partner']],
  ['权限', ['auth', 'admin', 'organization', 'dingtalk']],
  ['订单', ['order', 'orderconfig', 'sea_cargo_allocation', 'sea_order_change']],
  ['单证', ['sea_document', 'sea_master_bill']],
];
const DEFAULT_DOMAIN = '平台';
const DOMAIN_ORDER = ['订单', '单证', '财务', '提成', '往来单位', '权限', '平台'];

function domainOf(filename) {
  const base = filename.replace(/\.go$/, '');
  for (const [domain, prefixes] of DOMAIN_RULES) {
    if (prefixes.some((prefix) => base.startsWith(prefix))) return domain;
  }
  return DEFAULT_DOMAIN;
}

function listGoFiles(directory) {
  return readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) => {
      const path = join(directory, entry.name);
      return entry.isDirectory()
        ? listGoFiles(path)
        : entry.name.endsWith('.go') && !entry.name.endsWith('_test.go')
          ? [path]
          : [];
    })
    .sort();
}

// ---- Go 源码扫描（跳过注释与字符串，正确处理多行定义） ----

function isIdentChar(ch) {
  return /[A-Za-z0-9_]/.test(ch);
}

function readQuoted(text, start, quote) {
  let i = start + 1;
  while (i < text.length) {
    if (quote === '`') {
      if (text[i] === '`') return i + 1;
      i++;
      continue;
    }
    if (text[i] === '\\') {
      i += 2;
      continue;
    }
    if (text[i] === quote) return i + 1;
    i++;
  }
  return i;
}

function findMatchingParen(text, open) {
  let depth = 0;
  let i = open;
  while (i < text.length) {
    const ch = text[i];
    if (ch === '"' || ch === '`' || ch === "'") {
      i = readQuoted(text, i, ch);
      continue;
    }
    if (ch === '/' && text[i + 1] === '/') {
      while (i < text.length && text[i] !== '\n') i++;
      continue;
    }
    if (ch === '/' && text[i + 1] === '*') {
      i += 2;
      while (i < text.length && !(text[i] === '*' && text[i + 1] === '/')) i++;
      i += 2;
      continue;
    }
    if (ch === '(') depth++;
    if (ch === ')') {
      depth--;
      if (depth === 0) return i;
    }
    i++;
  }
  return text.length;
}

// 提取文件中全部 errors.<构造函数>(...) 调用的参数文本。
function extractErrorCalls(source) {
  const calls = [];
  const n = source.length;
  let i = 0;
  while (i < n) {
    const ch = source[i];
    if (ch === '/' && source[i + 1] === '/') {
      while (i < n && source[i] !== '\n') i++;
      continue;
    }
    if (ch === '/' && source[i + 1] === '*') {
      i += 2;
      while (i < n && !(source[i] === '*' && source[i + 1] === '/')) i++;
      i += 2;
      continue;
    }
    if (ch === '"' || ch === '`' || ch === "'") {
      i = readQuoted(source, i, ch);
      continue;
    }
    if (/[A-Za-z_]/.test(ch)) {
      let j = i + 1;
      while (j < n && isIdentChar(source[j])) j++;
      const ident = source.slice(i, j);
      if (ident === 'errors' && source[j] === '.') {
        let m = j + 1;
        while (m < n && isIdentChar(source[m])) m++;
        const fn = source.slice(j + 1, m);
        if (CONSTRUCTORS.has(fn)) {
          let p = m;
          while (p < n && /\s/.test(source[p])) p++;
          if (source[p] === '(') {
            const close = findMatchingParen(source, p);
            calls.push({ fn, argsText: source.slice(p + 1, close) });
            i = close;
            continue;
          }
        }
      }
      i = j;
      continue;
    }
    i++;
  }
  return calls;
}

// 按顶层逗号切分参数（字符串与嵌套括号内的逗号不切）。
function splitArgs(text) {
  const args = [];
  let depth = 0;
  let current = '';
  let i = 0;
  while (i < text.length) {
    const ch = text[i];
    if (ch === '"' || ch === '`' || ch === "'") {
      const end = readQuoted(text, i, ch);
      current += text.slice(i, end);
      i = end;
      continue;
    }
    if (ch === '(' || ch === '[' || ch === '{') {
      depth++;
      current += ch;
      i++;
      continue;
    }
    if (ch === ')' || ch === ']' || ch === '}') {
      depth--;
      current += ch;
      i++;
      continue;
    }
    if (ch === ',' && depth === 0) {
      args.push(current.trim());
      current = '';
      i++;
      continue;
    }
    current += ch;
    i++;
  }
  if (current.trim()) args.push(current.trim());
  return args;
}

// ---- 参数解析 ----

const LITERAL_CODE = /^"([A-Z0-9_]+)"$/;
const REASON_FROM_PROTO = /^reasonFromProto\(\s*[\w.]*?ErrorReason_(ERROR_REASON_[A-Z0-9_]+)\s*\)$/;
const STRING_LITERAL = /^"((?:[^"\\]|\\.)*)"$/;

function unescapeGoString(inner) {
  return inner
    .replace(/\\n/g, '<br>')
    .replace(/\\t/g, ' ')
    .replace(/\\"/g, '"')
    .replace(/\\\\/g, '\\');
}

// 消息列：纯字面量直接采用；fmt.Sprintf 取格式串；字符串拼接把非字面量片段缩成 %s；
// 完全动态（变量等）记为「（动态消息）」。
function messageOf(expr) {
  const trimmed = expr.trim();
  const literal = trimmed.match(STRING_LITERAL);
  if (literal) return unescapeGoString(literal[1]);
  const sprintf = trimmed.match(/^fmt\.Sprintf\(\s*"((?:[^"\\]|\\.)*)"/);
  if (sprintf) return unescapeGoString(sprintf[1]);
  let out = '';
  let hasLiteral = false;
  let i = 0;
  while (i < trimmed.length) {
    if (trimmed[i] === '"') {
      const end = readQuoted(trimmed, i, '"');
      out += unescapeGoString(trimmed.slice(i + 1, end - 1));
      hasLiteral = true;
      i = end;
      continue;
    }
    let j = i;
    while (j < trimmed.length && trimmed[j] !== '"') j++;
    out += '%s';
    i = j;
  }
  return hasLiteral ? out : '（动态消息）';
}

function parseErrorCall(call, filename) {
  const args = splitArgs(call.argsText);
  if (args.length === 0) return null;
  let code;
  let protoReason;
  const direct = args[0].match(LITERAL_CODE);
  if (direct) {
    code = direct[1];
  } else {
    const viaProto = args[0].match(REASON_FROM_PROTO);
    if (!viaProto) return null;
    protoReason = viaProto[1];
    code = protoReason.replace(/^ERROR_REASON_/, '');
  }
  return {
    code,
    httpKind: CONSTRUCTORS.get(call.fn),
    message: args.length > 1 ? messageOf(args[1]) : '（动态消息）',
    protoReason,
    file: filename,
  };
}

// ---- 聚合与渲染 ----

function pushUnique(list, value) {
  if (!list.includes(value)) list.push(value);
}

const domains = new Map(DOMAIN_ORDER.map((domain) => [domain, new Map()]));
const totalCodes = new Set();

for (const path of listGoFiles(bizRoot)) {
  const filename = basename(path);
  const source = readFileSync(path, 'utf8');
  for (const call of extractErrorCalls(source)) {
    const entry = parseErrorCall(call, filename);
    if (!entry) continue;
    const domain = domainOf(filename);
    const codes = domains.get(domain);
    if (!codes.has(entry.code)) {
      codes.set(entry.code, { httpKinds: [], messages: [], files: [], protoReasons: [] });
    }
    const aggregate = codes.get(entry.code);
    pushUnique(aggregate.httpKinds, entry.httpKind);
    pushUnique(aggregate.messages, entry.message);
    pushUnique(aggregate.files, entry.file);
    if (entry.protoReason) pushUnique(aggregate.protoReasons, entry.protoReason);
    totalCodes.add(entry.code);
  }
}

function mdCell(text) {
  return text.replace(/\|/g, '\\|');
}

function domainSection(domain) {
  const codes = domains.get(domain);
  if (codes.size === 0) return '';
  const header = `## ${domain}\n\n| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |\n| --- | --- | --- | --- | --- |\n`;
  const rows = [...codes.keys()]
    .sort()
    .map((code) => {
      const aggregate = codes.get(code);
      return `| ${code} | ${mdCell(aggregate.httpKinds.join(' / '))} | ${mdCell(aggregate.messages.join('<br>'))} | ${mdCell(aggregate.files.slice().sort().join('、'))} | ${mdCell(aggregate.protoReasons.length > 0 ? aggregate.protoReasons.join('、') : '—')} |`;
    });
  return `${header}${rows.join('\n')}\n`;
}

const domainCounts = DOMAIN_ORDER.map((domain) => [domain, domains.get(domain).size]);
const rowCount = domainCounts.reduce((sum, [, count]) => sum + count, 0);

const content = `# biz 层业务错误码目录

> 自动生成：依据 \`server/internal/biz/**/*.go\` 扫描 \`errors.*\` 错误定义生成，请勿手工修改。
> 再生成命令：\`node scripts/generate-error-catalog.mjs\`；一致性校验：\`node scripts/generate-error-catalog.mjs --check\`。

## 口径

- 只收录 biz 层以 \`errors.NotFound / BadRequest / Conflict / Forbidden / Unauthorized / InternalServer / ServiceUnavailable\`（及预留的 \`PreconditionFailed\`）定义的领域错误；传输层参数校验、数据层驱动错误不在此表。
- 「语义」列为错误构造函数对应的 HTTP 状态类别；同一错误码以不同构造函数定义时列出全部类别。
- 「中文消息」列为定义处的静态消息；同一错误码多处定义时逐行列出全部消息。含 \`%s\` 的为动态模板（\`fmt.Sprintf\` 或字符串拼接），完全动态（变量传参）时记为「（动态消息）」。
- 「关联 proto ErrorReason」列仅在错误码经 \`reasonFromProto(...)\` 关联 \`server/api/**/error_reason.proto\` 枚举时填写。
- 分组按定义文件所属业务域（同一错误码出现在多个业务域时在各域分别列出），组内按错误码排序；「定义位置」为 \`server/internal/biz/\` 下的文件名。
- 当前共 ${totalCodes.size} 个唯一错误码，${rowCount} 条目录记录。

## 汇总

| 业务域 | 错误码数 |
| --- | --- |
${domainCounts.map(([domain, count]) => `| ${domain} | ${count} |`).join('\n')}

${DOMAIN_ORDER.map(domainSection)
  .filter(Boolean)
  .join('\n')}
`;

if (process.argv.includes('--check')) {
  let current = '';
  try {
    current = readFileSync(outputPath, 'utf8');
  } catch {
    current = null;
  }
  if (current !== content) {
    console.error('错误码目录已过期或缺失，请执行 node scripts/generate-error-catalog.mjs');
    process.exit(1);
  }
  console.log(
    `[generate-error-catalog] 已验证 ${totalCodes.size} 个错误码（${domainCounts.map(([domain, count]) => `${domain} ${count}`).join('、')}）`,
  );
  process.exit(0);
}

writeFileSync(outputPath, content);
console.log(
  `[generate-error-catalog] 已生成 ${totalCodes.size} 个错误码（${domainCounts.map(([domain, count]) => `${domain} ${count}`).join('、')}）`,
);
