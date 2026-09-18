import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

// GO-2026-6452 的漏洞库版本范围暂未标记修复版本，但下列 Excelize 伪版本
// 已包含上游对内存与临时文件两条路径的完整修复。这里只允许回归测试确认的精确版本；
// 依赖版本变化时必须重新审计，不能继承豁免。
const auditedExceptions = new Map([
  [
    'GO-2026-6452',
    {
      module: 'github.com/xuri/excelize/v2',
      version: 'v2.11.1-0.20260728235842-f98df08a8f6a',
      fixedCommit: 'f98df08a8f6aed8bc3b193115d1abb5b1ea2e433',
    },
  ],
]);

function runGo(args, options = {}) {
  return spawnSync('go', args, {
    cwd: repoRoot,
    encoding: 'utf8',
    maxBuffer: 16 * 1024 * 1024,
    ...options,
  });
}

export function classifyVulnerabilityReport(report, getModuleVersion) {
  if (!report || !Array.isArray(report.statements) ||
      !report.statements.every((statement) => statement && typeof statement.status === 'string')) {
    throw new Error('govulncheck OpenVEX 报告缺少有效的 statements');
  }
  const unresolved = [];
  const accepted = [];
  const versionMismatches = [];
  for (const statement of report.statements) {
    if (statement.status !== 'affected') continue;
    const id = statement.vulnerability?.name;
    const exception = auditedExceptions.get(id);
    if (!exception) {
      unresolved.push(statement);
      continue;
    }
    const version = getModuleVersion(exception.module);
    if (version !== exception.version) {
      unresolved.push(statement);
      versionMismatches.push({ id, exception, version });
      continue;
    }
    accepted.push({ id, ...exception });
  }
  return { unresolved, accepted, versionMismatches };
}

function main() {
  const scan = runGo([
    '-C', 'server', 'tool', 'govulncheck', '-format', 'openvex', './...',
  ]);
  if (scan.error || scan.status !== 0) {
    throw new Error(scan.stderr || scan.error?.message || 'govulncheck 执行失败');
  }
  let report;
  try {
    report = JSON.parse(scan.stdout);
  } catch (error) {
    throw new Error(`无法解析 govulncheck OpenVEX 输出：${error.message}`);
  }
  const { unresolved, accepted, versionMismatches } = classifyVulnerabilityReport(
    report,
    (module) => {
      const result = runGo(['-C', 'server', 'list', '-m', '-f', '{{.Version}}', module]);
      return result.error || result.status !== 0 ? null : result.stdout.trim();
    },
  );
  for (const { id, exception, version } of versionMismatches) {
    process.stderr.write(
      `[check-server-vuln] ${id} 的审计豁免仅适用于 ${exception.module}@${exception.version}，当前版本为 ${version || '无法读取'}\n`,
    );
  }
  for (const exception of accepted) {
    process.stdout.write(
      `[check-server-vuln] 已审计豁免 ${exception.id}：${exception.module}@${exception.version} 已包含上游修复 ${exception.fixedCommit}\n`,
    );
  }
  if (unresolved.length > 0) {
    process.stderr.write('[check-server-vuln] 发现未解决的可达漏洞：\n');
    for (const statement of unresolved) {
      const vulnerability = statement.vulnerability ?? {};
      process.stderr.write(
        `- ${vulnerability.name || vulnerability['@id'] || '未知漏洞'}：${vulnerability.description || '无描述'}\n`,
      );
    }
    process.stderr.write(
      '请运行 `go -C server tool govulncheck ./...` 查看完整调用链。\n',
    );
    process.exitCode = 1;
    return;
  }
  process.stdout.write(
    `[check-server-vuln] 扫描通过：未发现未处理的可达漏洞${accepted.length > 0 ? `，审计豁免 ${accepted.length} 项` : ''}\n`,
  );
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`[check-server-vuln] ${error.message}\n`);
    process.exitCode = 1;
  }
}
