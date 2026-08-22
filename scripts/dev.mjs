import { spawn, spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { join } from 'node:path';

if (process.platform !== 'win32') {
  throw new Error('开发启动脚本仅支持 Windows 本机 PostgreSQL');
}

const userProfile = process.env.USERPROFILE;
if (!userProfile) {
  throw new Error('USERPROFILE 环境变量不存在');
}

const postgresDataDir = join(userProfile, 'scoop', 'persist', 'postgresql', 'data');
if (!existsSync(postgresDataDir)) {
  throw new Error(`PostgreSQL 数据目录不存在: ${postgresDataDir}`);
}

const commandShell = process.env.ComSpec;
if (!commandShell) {
  throw new Error('ComSpec 环境变量不存在');
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
  return runChecked(commandShell, ['/d', '/s', '/c', `pnpm run ${script}`]);
}

function postgresIsRunning() {
  return (
    spawnSync('pg_ctl', ['status', '-D', postgresDataDir], {
      stdio: 'ignore',
    }).status === 0
  );
}

function postgresIsReady() {
  return (
    spawnSync('pg_isready', ['-h', '127.0.0.1', '-p', '5432', '-t', '5'], {
      stdio: 'inherit',
    }).status === 0
  );
}

async function prepareDatabase() {
  if (!postgresIsRunning()) {
    console.log(`[dev] 启动 Windows PostgreSQL: ${postgresDataDir}`);
    await runChecked('pg_ctl', [
      'start',
      '-D',
      postgresDataDir,
      '-w',
      '-t',
      '60',
    ]);
  }

  if (!postgresIsReady()) {
    throw new Error('PostgreSQL 进程已存在，但 127.0.0.1:5432 未就绪');
  }

  console.log('[dev] 执行数据库迁移');
  await runPnpmScript('migrate:server');
}

const children = new Map();
let stopping = false;
let exitCode = 0;

function terminateProcessTree(child) {
  if (!child.pid || child.exitCode !== null) return;
  spawnSync('taskkill.exe', ['/pid', String(child.pid), '/t', '/f'], {
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
    commandShell,
    ['/d', '/s', '/c', `pnpm run ${script}`],
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
  await prepareDatabase();
  console.log('[dev] 启动后端热重载和前端开发服务');
  startService('后端', 'dev:server');
  startService('前端', 'dev:web');
} catch (error) {
  console.error('[dev] 启动失败:', error);
  process.exitCode = 1;
}
