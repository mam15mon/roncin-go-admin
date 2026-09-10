import { spawn, spawnSync } from 'node:child_process';
import { readlinkSync, readdirSync, readFileSync } from 'node:fs';
import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'linux') {
  throw new Error('开发启动脚本仅支持 Linux');
}

const databaseSource = process.env.DATABASE_SOURCE;
if (!databaseSource) {
  throw new Error('.env.local 中缺少 DATABASE_SOURCE');
}

const databaseUrl = new URL(databaseSource);
if (!['postgres:', 'postgresql:'].includes(databaseUrl.protocol)) {
  throw new Error('DATABASE_SOURCE 必须是 PostgreSQL 连接地址');
}

const postgresDatabase = decodeURIComponent(databaseUrl.pathname.slice(1));
const postgresUser = decodeURIComponent(databaseUrl.username);
const postgresPassword = decodeURIComponent(databaseUrl.password);
const postgresHost = databaseUrl.hostname || '127.0.0.1';
const postgresPort = databaseUrl.port || '5432';

if (!postgresDatabase || !postgresUser || !postgresPassword) {
  throw new Error('DATABASE_SOURCE 必须包含数据库名、用户名和密码');
}

const developmentServerExecutable = fileURLToPath(
  new URL('../server/tmp/roncin-server', import.meta.url),
);

const repositoryRoot = dirname(dirname(fileURLToPath(import.meta.url)));

function isWithinRepository(target) {
  return target === repositoryRoot || target.startsWith(`${repositoryRoot}/`);
}

function scanProcesses() {
  const processes = new Map();
  for (const entry of readdirSync('/proc')) {
    if (!/^\d+$/.test(entry)) continue;
    const processId = Number.parseInt(entry, 10);
    try {
      const executable = readlinkSync(`/proc/${entry}/exe`);
      const cwd = readlinkSync(`/proc/${entry}/cwd`);
      const stat = readFileSync(`/proc/${entry}/stat`, 'utf-8');
      // stat 的 comm 字段可能含空格和括号，从最后一个 ')' 之后切分剩余字段；
      // 依次为 state、ppid。
      const parentProcessId = Number.parseInt(
        stat.slice(stat.lastIndexOf(')') + 2).split(' ')[1],
        10,
      );
      const command = (readFileSync(`/proc/${entry}/cmdline`, 'utf-8').split('\0')[0] ?? '').trim();
      processes.set(processId, { executable, cwd, command, parentProcessId });
    } catch {
      // 进程可能已在枚举期间退出，或属于内核线程等其他用户，忽略即可。
    }
  }
  return processes;
}

function collectAncestors(processes, processId) {
  const chain = new Set();
  let current = processId;
  while (current && processes.has(current) && !chain.has(current)) {
    chain.add(current);
    current = processes.get(current).parentProcessId;
  }
  return chain;
}

function collectDescendants(processes, rootProcessId) {
  const tree = new Set([rootProcessId]);
  let expanded = true;
  while (expanded) {
    expanded = false;
    for (const [processId, process] of processes) {
      if (tree.has(processId) || !tree.has(process.parentProcessId)) continue;
      tree.add(processId);
      expanded = true;
    }
  }
  return tree;
}

function findAirAncestor(processes, processId) {
  let parentProcessId = processes.get(processId)?.parentProcessId;
  while (parentProcessId && processes.has(parentProcessId)) {
    const parent = processes.get(parentProcessId);
    if (parent.executable.endsWith('/air')) return parentProcessId;
    parentProcessId = parent.parentProcessId;
  }
  return null;
}

function findExistingDevelopmentServerProcesses() {
  const processes = scanProcesses();
  // 本脚本自身及其祖先永远不作为清理目标，避免向上误伤启动它的终端。
  const protectedProcessIds = collectAncestors(processes, process.pid);

  const rootProcessIds = new Set();
  for (const [processId, process] of processes) {
    if (protectedProcessIds.has(processId)) continue;
    let rootProcessId = null;
    if (process.executable === developmentServerExecutable) {
      // 后端 roncin-server：向上定位 air 祖先，从 air 起整棵关闭。
      rootProcessId = findAirAncestor(processes, processId) ?? processId;
    } else if (process.executable.endsWith('/air') && isWithinRepository(process.cwd)) {
      // 孤儿 air：roncin-server 子进程可能已被热重载替换或退出，按 air 进程本身识别。
      rootProcessId = processId;
    } else if (
      (process.command === 'umi' || process.command === 'utoopack-dev-server') &&
      isWithinRepository(process.cwd)
    ) {
      // 前端 dev 进程：umi 会把 argv0 改写为 umi / utoopack-dev-server。
      // 只杀命中的进程及其子树，外层 pnpm/sh 包装进程随子进程退出自然收敛。
      rootProcessId = processId;
    }
    if (rootProcessId !== null && !protectedProcessIds.has(rootProcessId)) {
      rootProcessIds.add(rootProcessId);
    }
  }

  const targetProcessIds = new Set();
  for (const rootProcessId of rootProcessIds) {
    for (const processId of collectDescendants(processes, rootProcessId)) {
      targetProcessIds.add(processId);
    }
  }
  return [...targetProcessIds];
}

function runChecked(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      stdio: 'inherit',
      ...options,
    });
    child.once('error', reject);
    child.once('exit', (code, signal) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(
        new Error(
          signal
            ? `${command} 被信号 ${signal} 终止`
            : `${command} 退出，代码 ${code ?? 1}`,
        ),
      );
    });
  });
}

function runPnpmScript(script) {
  return runChecked('pnpm', ['run', script]);
}

function nativePostgresIsReady() {
  return (
    spawnSync(
      'pg_isready',
      ['-h', postgresHost, '-p', postgresPort, '-U', postgresUser, '-d', postgresDatabase],
      { stdio: 'ignore' },
    ).status === 0
  );
}

function pgIsReadyIsAvailable() {
  return spawnSync('pg_isready', ['--version'], { stdio: 'ignore' }).status === 0;
}

function stopExistingDevelopmentServers() {
  const processIds = findExistingDevelopmentServerProcesses();
  if (processIds.length === 0) return;
  console.log(`[dev] 关闭本仓库已有开发进程树: PID ${processIds.join(', ')}`);
  for (const processId of processIds) {
    const result = spawnSync('kill', ['-TERM', String(processId)], {
      stdio: 'ignore',
    });
    if (result.error) {
      throw result.error;
    }
    // 单个 PID 退出码非零通常意味着枚举后进程已自行退出，继续处理其余 PID。
  }
}

async function prepareDatabase() {
  if (!pgIsReadyIsAvailable()) {
    throw new Error(
      '未找到 pg_isready。请在系统环境中安装 PostgreSQL 客户端后重试。',
    );
  }

  if (!nativePostgresIsReady()) {
    throw new Error(
      `本机 PostgreSQL (${postgresHost}:${postgresPort}) 未就绪。请先启动 PostgreSQL 服务，并确认 DATABASE_SOURCE 对应的数据库和用户可用。`,
    );
  }

  console.log(`[dev] 本机 PostgreSQL (${postgresHost}:${postgresPort}) 已就绪`);
  console.log('[dev] 执行数据库迁移');
  // 开发期迁移文件常在应用到本地库后继续修改，migrate:dev 允许重录校验和自愈，
  // 生产的 migrate:server 保持严格校验。
  await runPnpmScript('migrate:dev');
}

const children = new Map();
let stopping = false;
let exitCode = 0;

function terminateProcessTree(child) {
  if (!child.pid || child.exitCode !== null) return;
  spawnSync('kill', ['-TERM', String(child.pid)], {
    stdio: 'ignore',
  });
}

function stopAll(code) {
  if (stopping) return;
  stopping = true;
  exitCode = code;
  for (const child of children.values()) {
    terminateProcessTree(child);
  }
  if (children.size === 0) {
    process.exit(exitCode);
  }
}

function startService(name, script) {
  const child = spawn(
    'pnpm',
    ['run', script],
    { stdio: 'inherit' },
  );
  children.set(name, child);

  child.once('error', (error) => {
    children.delete(name);
    console.error(`[dev] ${name} 启动失败:`, error);
    stopAll(1);
  });
  child.once('exit', (code, signal) => {
    children.delete(name);
    if (!stopping) {
      console.error(
        signal
          ? `[dev] ${name} 被信号 ${signal} 终止`
          : `[dev] ${name} 退出，代码 ${code ?? 1}`,
      );
      stopAll(code ?? 1);
      return;
    }
    if (children.size === 0) {
      process.exit(exitCode);
    }
  });
}

process.once('SIGINT', () => stopAll(0));
process.once('SIGTERM', () => stopAll(0));

try {
  stopExistingDevelopmentServers();
  await prepareDatabase();
  console.log('[dev] 启动后端热重载和前端开发服务');
  startService('后端', 'dev:server');
  startService('前端', 'dev:web');
} catch (error) {
  console.error('[dev] 启动失败:', error);
  process.exitCode = 1;
}
