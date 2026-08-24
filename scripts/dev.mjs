import { spawn, spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'win32') {
  throw new Error('开发启动脚本仅支持 Windows 本机 PostgreSQL');
}

const programFiles = process.env.ProgramFiles;
if (!programFiles) {
  throw new Error('ProgramFiles 环境变量不存在');
}

const dockerExecutable = join(
  programFiles,
  'Docker',
  'Docker',
  'resources',
  'bin',
  'docker.exe',
);
if (!existsSync(dockerExecutable)) {
  throw new Error(`Docker CLI 不存在: ${dockerExecutable}`);
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
if (!postgresDatabase || !postgresUser || !postgresPassword) {
  throw new Error('DATABASE_SOURCE 必须包含数据库名、用户名和密码');
}

const composeEnvironment = {
  ...process.env,
  POSTGRES_DB: postgresDatabase,
  POSTGRES_USER: postgresUser,
  POSTGRES_PASSWORD: postgresPassword,
};

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

function postgresIsReady() {
  return (
    spawnSync(
      dockerExecutable,
      [
        'compose',
        'exec',
        '-T',
        'postgres',
        'pg_isready',
        '-U',
        postgresUser,
        '-d',
        postgresDatabase,
      ],
      {
        env: composeEnvironment,
        stdio: 'ignore',
      },
    ).status === 0
  );
}

async function waitForPostgres(timeout) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (postgresIsReady()) {
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
  console.log('[dev] 启动 Docker Desktop');
  await runChecked(dockerExecutable, ['desktop', 'start', '--timeout', '60']);

  console.log('[dev] 启动 Docker PostgreSQL');
  await runChecked(
    dockerExecutable,
    ['compose', 'up', '-d', 'postgres'],
    { env: composeEnvironment },
  );

  console.log('[dev] 等待 Docker PostgreSQL 就绪');
  if (!(await waitForPostgres(10_000))) {
    console.log('[dev] Docker PostgreSQL 连续 10 秒无响应，重启容器');
    await runChecked(
      dockerExecutable,
      ['compose', 'restart', 'postgres'],
      { env: composeEnvironment },
    );
  }

  if (!(await waitForPostgres(60_000))) {
    throw new Error(
      'Docker PostgreSQL 重启后仍未在 60 秒内就绪，请检查 docker compose logs postgres',
    );
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
