import { spawn, spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

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

const developmentServerExecutable = fileURLToPath(
  new URL('../server/tmp/roncin-server.exe', import.meta.url),
);

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

function postgresIsReady(stdio = 'inherit') {
  return (
    spawnSync('pg_isready', ['-h', '127.0.0.1', '-p', '5432', '-t', '1'], {
      stdio,
    }).status === 0
  );
}

async function waitForPostgres() {
  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    if (postgresIsReady('ignore')) {
      return true;
    }
    await new Promise((resolve) => setTimeout(resolve, 1_000));
  }
  return false;
}

function findExistingDevelopmentServerTrees() {
  const script = `
$targetPath = [System.IO.Path]::GetFullPath($env:RONCIN_DEV_SERVER_EXECUTABLE)
$targetProcessIds = @()

foreach ($serverProcess in @(Get-CimInstance Win32_Process -Filter "Name = 'roncin-server.exe'")) {
  if ([string]::IsNullOrWhiteSpace($serverProcess.ExecutablePath)) {
    continue
  }

  $serverPath = [System.IO.Path]::GetFullPath($serverProcess.ExecutablePath)
  if (-not [System.StringComparer]::OrdinalIgnoreCase.Equals($serverPath, $targetPath)) {
    continue
  }

  $targetProcess = $serverProcess
  $parentProcessId = [uint32]$serverProcess.ParentProcessId
  while ($parentProcessId -ne 0) {
    $parentProcess = Get-CimInstance Win32_Process -Filter "ProcessId = $parentProcessId"
    if ($null -eq $parentProcess) {
      break
    }
    if ($parentProcess.Name -ieq 'air.exe') {
      $targetProcess = $parentProcess
      break
    }
    $parentProcessId = [uint32]$parentProcess.ParentProcessId
  }

  $targetProcessIds += [int]$targetProcess.ProcessId
}

$targetProcessIds | Sort-Object -Unique
`;
  const result = spawnSync(
    'powershell.exe',
    ['-NoProfile', '-NonInteractive', '-Command', script],
    {
      encoding: 'utf8',
      env: {
        ...process.env,
        RONCIN_DEV_SERVER_EXECUTABLE: developmentServerExecutable,
      },
    },
  );
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`查询已有后端开发进程失败: ${result.stderr.trim()}`);
  }
  return result.stdout
    .split(/\r?\n/)
    .map((value) => Number.parseInt(value.trim(), 10))
    .filter(Number.isInteger);
}

function stopExistingDevelopmentServers() {
  for (const processId of findExistingDevelopmentServerTrees()) {
    console.log(`[dev] 关闭本仓库已有后端开发进程树: PID ${processId}`);
    const result = spawnSync(
      'taskkill.exe',
      ['/pid', String(processId), '/t', '/f'],
      { stdio: 'ignore' },
    );
    if (result.error) {
      throw result.error;
    }
    if (result.status !== 0) {
      throw new Error(`关闭已有后端开发进程树失败: PID ${processId}`);
    }
  }
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

  console.log('[dev] 等待 PostgreSQL 就绪');
  if (!(await waitForPostgres())) {
    console.log('[dev] PostgreSQL 连续 10 秒无响应，执行快速重启');
    await runChecked('pg_ctl', [
      'restart',
      '-D',
      postgresDataDir,
      '-m',
      'fast',
      '-w',
      '-t',
      '60',
    ]);
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
  stopExistingDevelopmentServers();
  await prepareDatabase();
  console.log('[dev] 启动后端热重载和前端开发服务');
  startService('后端', 'dev:server');
  startService('前端', 'dev:web');
} catch (error) {
  console.error('[dev] 启动失败:', error);
  process.exitCode = 1;
}
