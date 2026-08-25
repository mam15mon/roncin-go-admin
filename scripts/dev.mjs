import { spawn, spawnSync } from 'node:child_process';
import { readlinkSync, readdirSync } from 'node:fs';
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

function findExistingDevelopmentServerTrees() {
  const processes = new Map();
  for (const entry of readdirSync('/proc')) {
    if (!/^\d+$/.test(entry)) continue;
    const processId = Number.parseInt(entry, 10);
    try {
      const [stat] = readlinkSync(`/proc/${entry}/exe`).split('\n');
      const status = spawnSync('ps', ['-o', 'ppid=', '-p', entry], {
        encoding: 'utf8',
      });
      if (status.status !== 0) continue;
      processes.set(processId, {
        executable: stat,
        parentProcessId: Number.parseInt(status.stdout.trim(), 10),
      });
    } catch {
      // 进程可能已在枚举期间退出，忽略即可。
    }
  }

  const targetProcessIds = new Set();
  for (const [processId, process] of processes) {
    if (process.executable !== developmentServerExecutable) continue;
    let targetProcessId = processId;
    let parentProcessId = process.parentProcessId;
    while (parentProcessId && processes.has(parentProcessId)) {
      const parent = processes.get(parentProcessId);
      if (parent.executable.endsWith('/air')) {
        targetProcessId = parentProcessId;
        break;
      }
      parentProcessId = parent.parentProcessId;
    }
    targetProcessIds.add(targetProcessId);
  }
  return [...targetProcessIds];
}

function stopExistingDevelopmentServers() {
  for (const processId of findExistingDevelopmentServerTrees()) {
    console.log(`[dev] 关闭本仓库已有后端开发进程树: PID ${processId}`);
    const result = spawnSync('kill', ['-TERM', String(processId)], {
      stdio: 'ignore',
    });
    if (result.error) {
      throw result.error;
    }
    if (result.status !== 0) {
      throw new Error(`关闭已有后端开发进程树失败: PID ${processId}`);
    }
  }
}

async function prepareDatabase() {
  if (!pgIsReadyIsAvailable()) {
    throw new Error(
      '未找到 pg_isready。请在 WSL 中安装 PostgreSQL 客户端后重试。',
    );
  }

  if (!nativePostgresIsReady()) {
    throw new Error(
      `本机 PostgreSQL (${postgresHost}:${postgresPort}) 未就绪。请先在 WSL 中启动 PostgreSQL，并确认 DATABASE_SOURCE 对应的数据库和用户可用。`,
    );
  }

  console.log(`[dev] 本机 PostgreSQL (${postgresHost}:${postgresPort}) 已就绪`);
  console.log('[dev] 执行数据库迁移');
  await runPnpmScript('migrate:server');
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
