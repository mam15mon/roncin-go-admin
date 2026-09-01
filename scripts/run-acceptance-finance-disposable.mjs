import { spawn, spawnSync } from 'node:child_process';
import crypto from 'node:crypto';
import net from 'node:net';

function requireEnvironment(name) {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(
      `缺少必需环境变量：${name}。\n` +
        '一次性验收编排必须显式提供所有必需配置与凭据，严禁使用默认值或回退到 .env.local/开发数据库。',
    );
  }
  return value;
}

const adminDatabaseSource = requireEnvironment('RONCIN_ACCEPTANCE_ADMIN_DATABASE_SOURCE');
const BOOTSTRAP_ADMIN_USERNAME = requireEnvironment('BOOTSTRAP_ADMIN_USERNAME');
const BOOTSTRAP_ADMIN_PASSWORD = requireEnvironment('BOOTSTRAP_ADMIN_PASSWORD');
const BOOTSTRAP_ADMIN_DISPLAY_NAME = requireEnvironment('BOOTSTRAP_ADMIN_DISPLAY_NAME');
const BOOTSTRAP_ORGANIZATION_CODE = requireEnvironment('BOOTSTRAP_ORGANIZATION_CODE');
const BOOTSTRAP_ORGANIZATION_NAME = requireEnvironment('BOOTSTRAP_ORGANIZATION_NAME');

let adminUrl;
try {
  adminUrl = new URL(adminDatabaseSource);
} catch {
  console.error('RONCIN_ACCEPTANCE_ADMIN_DATABASE_SOURCE 必须是合法的 PostgreSQL URL 连接串。');
  process.exit(1);
}

if (!['postgres:', 'postgresql:'].includes(adminUrl.protocol)) {
  console.error('RONCIN_ACCEPTANCE_ADMIN_DATABASE_SOURCE 协议必须为 postgres: 或 postgresql:');
  process.exit(1);
}

const adminHost = adminUrl.hostname || '127.0.0.1';
const adminPort = adminUrl.port || '5432';

const GO_SERVER_PORT = 8010;
const WEB_SERVER_PORT = 8001;
const GO_SERVER_BASE_URL = `http://127.0.0.1:${GO_SERVER_PORT}`;
const WEB_SERVER_BASE_URL = `http://127.0.0.1:${WEB_SERVER_PORT}`;

const PREFIX = 'roncin_acc_fin_';

function isPortAvailable(port, host = '127.0.0.1') {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once('error', () => {
      resolve(false);
    });
    server.once('listening', () => {
      server.close(() => resolve(true));
    });
    server.listen(port, host);
  });
}

const activeChildProcesses = new Set();

function isProcessExited(child) {
  return !child || child.exitCode !== null || child.signalCode !== null;
}

function isTargetAlive(child) {
  if (!child?.pid) return false;
  const isUnix = process.platform !== 'win32';
  if (isUnix) {
    try {
      process.kill(-child.pid, 0);
      return true;
    } catch (err) {
      if (err.code === 'ESRCH') {
        try {
          process.kill(child.pid, 0);
          return true;
        } catch (err2) {
          if (err2.code === 'ESRCH') return false;
          throw err2;
        }
      }
      throw err;
    }
  } else {
    try {
      process.kill(child.pid, 0);
      return true;
    } catch (err) {
      if (err.code === 'ESRCH') return false;
      throw err;
    }
  }
}

function registerChild(child) {
  if (!child) return;
  activeChildProcesses.add(child);
  child.once('exit', () => {
    try {
      if (!isTargetAlive(child)) {
        activeChildProcesses.delete(child);
      }
    } catch {
      // 存活检查异常时保守保留句柄，不得误删
    }
  });
  child.once('error', () => {
    try {
      if (!isTargetAlive(child)) {
        activeChildProcesses.delete(child);
      }
    } catch {
      // 存活检查异常时保守保留
    }
  });
}

function spawnTrackedProcess(command, args, options = {}) {
  const isUnix = process.platform !== 'win32';
  const { capture, ...spawnOptions } = options;
  const child = spawn(command, args, {
    stdio: 'inherit',
    detached: isUnix,
    ...spawnOptions,
  });
  registerChild(child);
  return child;
}

function runCommand(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const isUnix = process.platform !== 'win32';
    const { capture, onSpawn, ...spawnOptions } = options;
    const child = spawn(command, args, {
      stdio: capture ? ['ignore', 'pipe', 'pipe'] : 'inherit',
      detached: isUnix,
      ...spawnOptions,
    });
    registerChild(child);
    if (typeof onSpawn === 'function') {
      try {
        onSpawn(child);
      } catch {
        // 忽略通知异常
      }
    }

    let stdout = '';
    let stderr = '';
    if (capture) {
      child.stdout?.on('data', (d) => {
        stdout += d.toString();
      });
      child.stderr?.on('data', (d) => {
        stderr += d.toString();
      });
    }

    child.once('error', async (err) => {
      if (settled) return;
      settled = true;
      let stopErr = null;
      try {
        if (child && isTargetAlive(child)) {
          await stopProcess(child);
        } else if (child) {
          activeChildProcesses.delete(child);
        }
      } catch (sErr) {
        stopErr = sErr;
      }
      if (stopErr) {
        if (typeof AggregateError !== 'undefined') {
          reject(
            new AggregateError(
              [err, stopErr],
              `命令启动出错且清理进程失败:\n1. ${err.message}\n2. ${stopErr.message}`,
            ),
          );
        } else {
          err.cleanupError = stopErr;
          reject(err);
        }
      } else {
        reject(err);
      }
    });

    child.once('exit', async (code, signal) => {
      if (settled) return;
      settled = true;

      let groupStillAlive = false;
      try {
        groupStillAlive = isTargetAlive(child);
      } catch {
        groupStillAlive = true;
      }

      if (groupStillAlive) {
        // leader 进程已退出但进程组内仍有孤儿孙进程残留，拒绝 resolve 成功并强制清理组
        let stopErr = null;
        try {
          await stopProcess(child);
        } catch (sErr) {
          stopErr = sErr;
        }

        const sanitizedStdout = sanitizeSensitiveOutput(stdout);
        const sanitizedStderr = sanitizeSensitiveOutput(stderr);

        if (code === 0 && !signal) {
          const leaderErr = new Error(
            `命令执行异常：leader 进程已正常退出 (code=0)，但进程组仍有孤儿进程存活: command=${command}`,
          );
          if (stopErr) {
            if (typeof AggregateError !== 'undefined') {
              reject(
                new AggregateError(
                  [leaderErr, stopErr],
                  `命令 leader 退出且清理残留进程组失败:\n1. ${leaderErr.message}\n2. ${stopErr.message}`,
                ),
              );
            } else {
              leaderErr.cleanupError = stopErr;
              reject(leaderErr);
            }
          } else {
            reject(leaderErr);
          }
          return;
        }

        // 非零退出同时伴随孤儿进程残留：必须同时保留原命令失败（含 capture 内容）与孤儿进程组/清理错误
        const cmdErr = new Error(
          signal
            ? `${command} 被信号 ${signal} 终止`
            : `${command} 退出，代码 ${code ?? 1}\n${sanitizedStderr || sanitizedStdout}`,
        );
        const orphanErr = new Error(
          `进程组异常：命令以非零状态退出 (${signal ? 'signal=' + signal : 'code=' + code})，且进程组仍有孤儿进程残留: command=${command}`,
        );
        const errors = [cmdErr, orphanErr, stopErr].filter(Boolean);
        if (typeof AggregateError !== 'undefined' && errors.length > 1) {
          reject(
            new AggregateError(
              errors,
              `命令执行失败且进程组残留异常:\n${errors.map((e) => e.message).join('\n')}`,
            ),
          );
        } else {
          cmdErr.orphanError = orphanErr;
          if (stopErr) cmdErr.cleanupError = stopErr;
          reject(cmdErr);
        }
        return;
      }

      if (code === 0) {
        resolve({ stdout, stderr });
        return;
      }
      const sanitizedStdout = sanitizeSensitiveOutput(stdout);
      const sanitizedStderr = sanitizeSensitiveOutput(stderr);
      reject(
        new Error(
          signal
            ? `${command} 被信号 ${signal} 终止`
            : `${command} 退出，代码 ${code ?? 1}\n${sanitizedStderr || sanitizedStdout}`,
        ),
      );
    });
  });
}

function sanitizeSensitiveOutput(str) {
  if (!str) return '';
  return String(str).replace(/PASSWORD\s+'[^']*'/gi, "PASSWORD '[REDACTED]'");
}

function execSQL(sqlQuery, label = 'SQL Command') {
  const result = spawnSync(
    'psql',
    [adminDatabaseSource, '-v', 'ON_ERROR_STOP=1', '-c', sqlQuery],
    { encoding: 'utf8' },
  );
  if (result.status !== 0) {
    const sanitizedQuery = sanitizeSensitiveOutput(sqlQuery);
    const sanitizedStderr = sanitizeSensitiveOutput(result.stderr);
    const sanitizedStdout = sanitizeSensitiveOutput(result.stdout);
    throw new Error(
      `执行 SQL 失败 [${label}]:\nQuery: ${sanitizedQuery}\nError: ${sanitizedStderr || sanitizedStdout}`,
    );
  }
  return result.stdout.trim();
}

function querySQLSingleValue(sqlQuery, label = 'SQL Query') {
  const result = spawnSync(
    'psql',
    [adminDatabaseSource, '-v', 'ON_ERROR_STOP=1', '-t', '-A', '-c', sqlQuery],
    { encoding: 'utf8' },
  );
  if (result.status !== 0) {
    const sanitizedQuery = sanitizeSensitiveOutput(sqlQuery);
    const sanitizedStderr = sanitizeSensitiveOutput(result.stderr);
    const sanitizedStdout = sanitizeSensitiveOutput(result.stdout);
    throw new Error(
      `查询 SQL 失败 [${label}]:\nQuery: ${sanitizedQuery}\nError: ${sanitizedStderr || sanitizedStdout}`,
    );
  }
  return result.stdout.trim();
}

function querySQLCount01(sqlQuery, label = 'SQL Count') {
  const value = querySQLSingleValue(sqlQuery, label);
  if (value !== '0' && value !== '1') {
    throw new Error(
      `[${label}] SQL 计数值异常，必须为 '0' 或 '1'，实际为: '${value}'\nQuery: ${sanitizeSensitiveOutput(sqlQuery)}`,
    );
  }
  return value;
}

async function stopProcess(child, timeoutMs = 8000) {
  if (!child?.pid) {
    if (child) activeChildProcesses.delete(child);
    return;
  }
  const isUnix = process.platform !== 'win32';

  // 1. 若目标已消亡，从追踪集合删除并直接返回
  let alive = false;
  try {
    alive = isTargetAlive(child);
  } catch (err) {
    throw new Error(`[stopProcess] 检查进程存活状态失败 (pid=${child.pid}): ${err.message}`);
  }
  if (!alive) {
    activeChildProcesses.delete(child);
    return;
  }

  // 2. 发送 SIGTERM
  try {
    if (isUnix) {
      process.kill(-child.pid, 'SIGTERM');
    } else {
      process.kill(child.pid, 'SIGTERM');
    }
  } catch (err) {
    if (err.code !== 'ESRCH') {
      throw new Error(`[stopProcess] 发送 SIGTERM 失败 (pid=${child.pid}): ${err.message}`);
    }
  }

  // 3. 等待 SIGTERM 退出 (有界轮询)
  const termWaitTimeout = Math.min(timeoutMs, 5000);
  const start = Date.now();
  while (Date.now() - start < termWaitTimeout) {
    try {
      if (!isTargetAlive(child)) {
        activeChildProcesses.delete(child);
        return;
      }
    } catch (err) {
      throw new Error(`[stopProcess] 轮询进程状态失败 (pid=${child.pid}): ${err.message}`);
    }
    await new Promise((r) => setTimeout(r, 100));
  }

  // 4. 若仍存活，升级为 SIGKILL
  try {
    if (isUnix) {
      process.kill(-child.pid, 'SIGKILL');
    } else {
      process.kill(child.pid, 'SIGKILL');
    }
  } catch (err) {
    if (err.code !== 'ESRCH') {
      throw new Error(`[stopProcess] 发送 SIGKILL 失败 (pid=${child.pid}): ${err.message}`);
    }
  }

  // 5. 等待 SIGKILL 退出 (有界轮询 3000ms)
  const killStart = Date.now();
  while (Date.now() - killStart < 3000) {
    try {
      if (!isTargetAlive(child)) {
        activeChildProcesses.delete(child);
        return;
      }
    } catch (err) {
      throw new Error(`[stopProcess] 轮询 SIGKILL 状态失败 (pid=${child.pid}): ${err.message}`);
    }
    await new Promise((r) => setTimeout(r, 100));
  }

  // 6. 最终验证
  if (isTargetAlive(child)) {
    throw new Error(`[stopProcess] 进程组终止失败：发送 SIGKILL 后进程组依然存活 (pid=${child.pid})`);
  }
  activeChildProcesses.delete(child);
}

async function waitForPortFree(port, host = '127.0.0.1', timeoutMs = 10000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const isFree = await new Promise((resolve) => {
      const server = net.createServer();
      server.once('error', () => resolve(false));
      server.once('listening', () => {
        server.close(() => resolve(true));
      });
      server.listen(port, host);
    });
    if (isFree) return;
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error(`端口 ${port} 未能在 ${timeoutMs}ms 内释放`);
}

async function waitForHttpReady(url, timeoutMs = 45000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const response = await fetch(url, { signal: AbortSignal.timeout(2000) });
      if (response.ok) {
        return;
      }
    } catch {
      // 忽略连接异常，等待重试
    }
    await new Promise((r) => setTimeout(r, 400));
  }
  throw new Error(`服务未能就绪（超时 ${timeoutMs}ms）：${url}`);
}

const lifetimeCreatedDatabases = new Set();
const pendingDatabases = new Map(); // dbName -> { roleName }

const lifetimeCreatedRoles = new Set();
const pendingRoles = new Set();     // roleName

function createExplicitDisposableDatabaseAndRole(dbName, roleName) {
  if (!dbName.startsWith(PREFIX) || !roleName.startsWith(PREFIX)) {
    throw new Error(`拒绝创建非任务前缀的数据库或角色: db=${dbName}, role=${roleName}`);
  }
  const rolePassword = crypto.randomBytes(16).toString('hex');

  console.log(`[disposable] 创建一次性角色: ${roleName}`);
  execSQL(`CREATE ROLE ${roleName} WITH LOGIN PASSWORD '${rolePassword}';`, `CREATE ROLE ${roleName}`);
  lifetimeCreatedRoles.add(roleName);
  pendingRoles.add(roleName);

  console.log(`[disposable] 创建一次性数据库: ${dbName} (OWNER ${roleName})`);
  execSQL(`CREATE DATABASE ${dbName} OWNER ${roleName};`, `CREATE DATABASE ${dbName}`);
  lifetimeCreatedDatabases.add(dbName);
  pendingDatabases.set(dbName, { roleName });

  execSQL(`GRANT ALL PRIVILEGES ON DATABASE ${dbName} TO ${roleName};`, `GRANT ALL PRIVILEGES`);

  const connectionSource = `postgresql://${roleName}:${rolePassword}@${adminHost}:${adminPort}/${dbName}?sslmode=disable`;
  return { dbName, roleName, connectionSource };
}

function createDisposableDatabaseAndRole(tag) {
  const uniqueSuffix = `${tag}_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
  return createExplicitDisposableDatabaseAndRole(`${PREFIX}${uniqueSuffix}`, `${PREFIX}${uniqueSuffix}`);
}

function destroyDisposableDatabase(dbName, expectedOwner) {
  if (!dbName.startsWith(PREFIX)) {
    throw new Error(`拒绝清理非任务前缀数据库：db=${dbName}`);
  }
  if (!lifetimeCreatedDatabases.has(dbName)) {
    throw new Error(`拒绝清理非本任务创建的未知数据库：db=${dbName}`);
  }
  if (!pendingDatabases.has(dbName)) {
    return;
  }

  // 1. 查询数据库存在且 owner 归属与本次记录一致
  const owner = querySQLSingleValue(
    `SELECT pg_user.usename FROM pg_database JOIN pg_user ON pg_database.datdba = pg_user.usesysid WHERE pg_database.datname = '${dbName}';`,
  );
  if (!owner) {
    throw new Error(`清理前校验失败：数据库 ${dbName} 不存在`);
  }
  if (expectedOwner && owner !== expectedOwner) {
    throw new Error(`清理前校验失败：数据库 ${dbName} 所有者为 ${owner}，与预期 ${expectedOwner} 不符`);
  }

  console.log(`[disposable] 销毁一次性数据库: ${dbName}`);
  execSQL(
    `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${dbName}' AND pid <> pg_backend_pid();`,
  );
  execSQL(`DROP DATABASE ${dbName};`);

  // 2. 确认删除后不存在
  const dbRemaining = querySQLCount01(
    `SELECT COUNT(*) FROM pg_database WHERE datname = '${dbName}';`,
    'Check db remaining after drop',
  );
  if (dbRemaining !== '0') {
    throw new Error(`数据库清理验证失败：数据库残留=${dbRemaining} (db=${dbName})`);
  }

  pendingDatabases.delete(dbName);
  console.log(`[disposable] 数据库清理验证成功：${dbName}`);
}

function destroyDisposableRole(roleName) {
  if (!roleName.startsWith(PREFIX)) {
    throw new Error(`拒绝清理非任务前缀角色：role=${roleName}`);
  }
  if (!lifetimeCreatedRoles.has(roleName)) {
    throw new Error(`拒绝清理非本任务创建的未知角色：role=${roleName}`);
  }
  if (!pendingRoles.has(roleName)) {
    return;
  }

  // 1. 查询角色存在
  const roleCount = querySQLCount01(
    `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
    'Check role count before drop',
  );
  if (roleCount !== '1') {
    throw new Error(`清理前校验失败：角色 ${roleName} 不存在或不唯一 (count=${roleCount})`);
  }

  console.log(`[disposable] 销毁一次性角色: ${roleName}`);
  execSQL(`DROP ROLE ${roleName};`);

  // 2. 确认删除后不存在
  const roleRemaining = querySQLCount01(
    `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
    'Check role remaining after drop',
  );
  if (roleRemaining !== '0') {
    throw new Error(`角色清理验证失败：角色残留=${roleRemaining} (role=${roleName})`);
  }

  pendingRoles.delete(roleName);
  console.log(`[disposable] 角色清理验证成功：${roleName}`);
}

let cleanupQueue = Promise.resolve();

function runSerializedCleanup(cleanupFn) {
  const next = cleanupQueue.then(cleanupFn, cleanupFn);
  cleanupQueue = next.catch(() => {});
  return next;
}

async function cleanupStageResources({ processes = [], ports = [], resource = null }) {
  return runSerializedCleanup(async () => {
    const cleanupErrors = [];
    for (const p of processes) {
      if (p) {
        try {
          await stopProcess(p);
        } catch (err) {
          cleanupErrors.push(err);
        }
      }
    }
    if (ports.length > 0) {
      try {
        await Promise.all(ports.map((port) => waitForPortFree(port)));
      } catch (err) {
        cleanupErrors.push(err);
      }
    }
    if (resource) {
      if (resource.dbName && pendingDatabases.has(resource.dbName)) {
        try {
          destroyDisposableDatabase(resource.dbName, resource.roleName);
        } catch (err) {
          cleanupErrors.push(err);
        }
      }
      if (resource.roleName && pendingRoles.has(resource.roleName)) {
        try {
          destroyDisposableRole(resource.roleName);
        } catch (err) {
          cleanupErrors.push(err);
        }
      }
    }
    if (cleanupErrors.length === 1) {
      throw cleanupErrors[0];
    } else if (cleanupErrors.length > 1) {
      throw new AggregateError(
        cleanupErrors,
        `[disposable] 阶段资源清理发生多个错误:\n${cleanupErrors.map((e) => e.message).join('\n')}`,
      );
    }
  });
}

function combineExecutionAndCleanupErrors(execError, cleanupError, contextLabel) {
  if (execError && cleanupError) {
    if (typeof AggregateError !== 'undefined') {
      return new AggregateError(
        [execError, cleanupError],
        `[${contextLabel}] 业务执行与阶段资源清理均发生错误:\n1. ${execError.message}\n2. ${cleanupError.message}`,
      );
    }
    execError.cleanupError = cleanupError;
    return execError;
  }
  return cleanupError || execError || null;
}

async function performGlobalCleanup(reason) {
  return runSerializedCleanup(async () => {
    console.log(`\n[disposable] 执行安全全局清理 (触发原因: ${reason})...`);
    const cleanupErrors = [];

    // 1. 停止本轮追踪的所有活跃子进程
    for (const child of [...activeChildProcesses]) {
      try {
        await stopProcess(child);
      } catch (err) {
        console.error(`[disposable] 停止子进程失败: ${err.message}`);
        cleanupErrors.push(err);
      }
    }

    // 2. 等待端口释放
    try {
      await Promise.all([
        waitForPortFree(GO_SERVER_PORT),
        waitForPortFree(9010),
        waitForPortFree(WEB_SERVER_PORT),
      ]);
    } catch (err) {
      console.error(`[disposable] 端口释放等待异常: ${err.message}`);
      cleanupErrors.push(err);
    }

    // 3. 分别清理 pendingDatabases 与 pendingRoles（涵盖 only-role 状态）
    for (const [dbName, { roleName }] of [...pendingDatabases.entries()]) {
      try {
        destroyDisposableDatabase(dbName, roleName);
      } catch (err) {
        console.error(`[disposable] 兜底数据库清理失败: ${err.message}`);
        cleanupErrors.push(err);
      }
    }

    for (const roleName of [...pendingRoles]) {
      try {
        destroyDisposableRole(roleName);
      } catch (err) {
        console.error(`[disposable] 兜底角色清理失败: ${err.message}`);
        cleanupErrors.push(err);
      }
    }

    if (cleanupErrors.length === 1) {
      console.error(`[disposable] 全局清理完成但存在错误: ${cleanupErrors[0].message}`);
      throw cleanupErrors[0];
    } else if (cleanupErrors.length > 1) {
      const agg = new AggregateError(
        cleanupErrors,
        `[disposable] 全局清理发生多个错误:\n${cleanupErrors.map((e) => e.message).join('\n')}`,
      );
      console.error(`[disposable] 全局清理完成但存在多个错误:\n${cleanupErrors.map((e) => e.message).join('\n')}`);
      throw agg;
    } else {
      console.log(`[disposable] 全局清理完成，资源已全部回收`);
    }
  });
}

for (const sig of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
  process.on(sig, async () => {
    console.warn(`\n[disposable] 收到系统信号 ${sig}，启动安全中断处理...`);
    try {
      await performGlobalCleanup(`信号 ${sig}`);
      process.exit(128 + (sig === 'SIGINT' ? 2 : sig === 'SIGTERM' ? 15 : 1));
    } catch (err) {
      console.error(`[disposable] 信号中断清理失败:`, err);
      process.exit(1);
    }
  });
}

async function runStageA() {
  console.log('\n========================================');
  console.log('  Stage 1: 临时库 A（CNY 基线验收）');
  console.log('========================================\n');

  let resource = null;
  let serverProcess = null;
  let webProcess = null;
  let execError = null;
  let cleanupError = null;

  try {
    resource = createDisposableDatabaseAndRole('cny');
    const { connectionSource } = resource;

    console.log('[Stage 1] 执行数据库迁移');
    await runCommand('go', ['-C', 'server', 'run', './cmd/migrate', '-dir', 'migrations'], {
      env: { ...process.env, DATABASE_SOURCE: connectionSource },
    });

    console.log('[Stage 1] 运行真实 PostgreSQL 集成测试（禁止 SKIP 冒充 PASS）');
    const testResult = await runCommand(
      'go',
      ['-C', 'server', 'test', '-v', './internal/data', '-run', 'Postgres$', '-count=1'],
      {
        env: {
          ...process.env,
          RONCIN_INTEGRATION_DATABASE_SOURCE: connectionSource,
        },
        capture: true,
      },
    );

    const testOutput = testResult.stdout;
    // 匹配真实顶层测试用例（行首以 --- PASS: / --- SKIP: / --- FAIL: 开头）
    const passMatches = testOutput.match(/^--- PASS:\s+\w+/gm) || [];
    const skipMatches = testOutput.match(/^--- SKIP:\s+\w+/gm) || [];
    const failMatches = testOutput.match(/^--- FAIL:\s+\w+/gm) || [];

    console.log(`[Stage 1] PostgreSQL 集成测试结果: PASS=${passMatches.length}, SKIP=${skipMatches.length}, FAIL=${failMatches.length}`);
    if (failMatches.length > 0 || passMatches.length === 0 || skipMatches.length > 0) {
      throw new Error(
        `PostgreSQL 集成测试未通过、存在 SKIP 或无执行用例: PASS=${passMatches.length}, SKIP=${skipMatches.length}, FAIL=${failMatches.length}`,
      );
    }

    console.log('[Stage 1] 初始化系统管理员');
    await runCommand('go', ['-C', 'server', 'run', './cmd/bootstrap-admin'], {
      env: {
        ...process.env,
        DATABASE_SOURCE: connectionSource,
        BOOTSTRAP_ADMIN_USERNAME,
        BOOTSTRAP_ADMIN_PASSWORD,
        BOOTSTRAP_ADMIN_DISPLAY_NAME,
        BOOTSTRAP_ORGANIZATION_CODE,
        BOOTSTRAP_ORGANIZATION_NAME,
      },
    });

    console.log('[Stage 1] 启动 Go 后端测试服务 (:8010)');
    serverProcess = spawnTrackedProcess(
      'go',
      ['-C', 'server', 'run', './cmd/server', '-conf', 'configs/config.acceptance.yaml'],
      {
        env: {
          ...process.env,
          DATABASE_SOURCE: connectionSource,
        },
      },
    );

    await waitForHttpReady(`${GO_SERVER_BASE_URL}/health/ready`);
    console.log('[Stage 1] Go 后端测试服务就绪');

    console.log('[Stage 1] 启动 Web 测试服务 (:8001)');
    webProcess = spawnTrackedProcess(
      'pnpm',
      ['--dir', 'web', 'exec', 'cross-env', `PORT=${WEB_SERVER_PORT}`, 'UMI_ENV=test', 'MOCK=none', `RONCIN_API_PROXY_TARGET=${GO_SERVER_BASE_URL}`, 'max', 'dev'],
      {
        env: {
          ...process.env,
          PORT: String(WEB_SERVER_PORT),
          UMI_ENV: 'test',
          MOCK: 'none',
          RONCIN_API_PROXY_TARGET: GO_SERVER_BASE_URL,
        },
      },
    );

    await waitForHttpReady(`${WEB_SERVER_BASE_URL}`);
    console.log('[Stage 1] Web 测试服务就绪');

    console.log('[Stage 1] 执行 acceptance:finance 全量验收（应收 + 应付 + Playwright）');
    await runCommand('pnpm', ['run', 'acceptance:finance'], {
      env: {
        ...process.env,
        RONCIN_ACCEPTANCE_BASE_URL: GO_SERVER_BASE_URL,
        RONCIN_WEB_BASE_URL: WEB_SERVER_BASE_URL,
        BOOTSTRAP_ADMIN_USERNAME,
        BOOTSTRAP_ADMIN_PASSWORD,
      },
    });
    console.log('[Stage 1] CNY 基线验收全部通过');
  } catch (err) {
    execError = err;
  } finally {
    try {
      await cleanupStageResources({
        processes: [webProcess, serverProcess],
        ports: [GO_SERVER_PORT, 9010, WEB_SERVER_PORT],
        resource,
      });
    } catch (cErr) {
      cleanupError = cErr;
    }

    const combinedErr = combineExecutionAndCleanupErrors(execError, cleanupError, 'Stage 1');
    if (combinedErr) throw combinedErr;
  }
}

async function runStageB() {
  console.log('\n========================================');
  console.log('  Stage 2: 临时库 B（USD 外币全链路验收）');
  console.log('========================================\n');

  let resource = null;
  let serverProcess = null;
  let execError = null;
  let cleanupError = null;

  try {
    resource = createDisposableDatabaseAndRole('usd');
    const { connectionSource } = resource;

    console.log('[Stage 2] 执行数据库迁移');
    await runCommand('go', ['-C', 'server', 'run', './cmd/migrate', '-dir', 'migrations'], {
      env: { ...process.env, DATABASE_SOURCE: connectionSource },
    });

    console.log('[Stage 2] 初始化系统管理员');
    await runCommand('go', ['-C', 'server', 'run', './cmd/bootstrap-admin'], {
      env: {
        ...process.env,
        DATABASE_SOURCE: connectionSource,
        BOOTSTRAP_ADMIN_USERNAME,
        BOOTSTRAP_ADMIN_PASSWORD,
        BOOTSTRAP_ADMIN_DISPLAY_NAME,
        BOOTSTRAP_ORGANIZATION_CODE,
        BOOTSTRAP_ORGANIZATION_NAME,
      },
    });

    console.log('[Stage 2] 启动 Go 后端测试服务 (:8010)');
    serverProcess = spawnTrackedProcess(
      'go',
      ['-C', 'server', 'run', './cmd/server', '-conf', 'configs/config.acceptance.yaml'],
      {
        env: {
          ...process.env,
          DATABASE_SOURCE: connectionSource,
        },
      },
    );

    await waitForHttpReady(`${GO_SERVER_BASE_URL}/health/ready`);
    console.log('[Stage 2] Go 后端测试服务就绪');

    console.log('[Stage 2] 执行 acceptance:finance:foreign-currency 外币连续链路验收');
    await runCommand('node', ['scripts/acceptance-finance-foreign-currency.mjs'], {
      env: {
        ...process.env,
        RONCIN_ACCEPTANCE_BASE_URL: GO_SERVER_BASE_URL,
        BOOTSTRAP_ADMIN_USERNAME,
        BOOTSTRAP_ADMIN_PASSWORD,
      },
    });
    console.log('[Stage 2] 外币财务全链路验收全部通过');
  } catch (err) {
    execError = err;
  } finally {
    try {
      await cleanupStageResources({
        processes: [serverProcess],
        ports: [GO_SERVER_PORT, 9010],
        resource,
      });
    } catch (cErr) {
      cleanupError = cErr;
    }

    const combinedErr = combineExecutionAndCleanupErrors(execError, cleanupError, 'Stage 2');
    if (combinedErr) throw combinedErr;
  }
}

class SelfTestInjectedError extends Error {
  constructor(code, message) {
    super(message);
    this.name = 'SelfTestInjectedError';
    this.code = code;
  }
}

async function runSelfTestSignalChild() {
  const dbIndex = process.argv.indexOf('--db-name');
  const roleIndex = process.argv.indexOf('--role-name');
  if (dbIndex === -1 || roleIndex === -1 || !process.argv[dbIndex + 1] || !process.argv[roleIndex + 1]) {
    throw new Error('[child] 必须显式传入 --db-name 与 --role-name 参数');
  }
  const dbName = process.argv[dbIndex + 1];
  const roleName = process.argv[roleIndex + 1];

  if (!dbName.startsWith(PREFIX) || !roleName.startsWith(PREFIX)) {
    throw new Error(`[child] 拒绝使用非任务前缀资源: db=${dbName}, role=${roleName}`);
  }

  // 验证父进程已创建的资源存在且归属正确 (adopt 契约)
  const dbCount = querySQLCount01(
    `SELECT COUNT(*) FROM pg_database WHERE datname = '${dbName}';`,
    'Child check db exists for adopt',
  );
  const roleCount = querySQLCount01(
    `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
    'Child check role exists for adopt',
  );
  if (dbCount !== '1' || roleCount !== '1') {
    throw new Error(`[child] 待接管资源在 PostgreSQL 中不存在: dbCount=${dbCount}, roleCount=${roleCount}`);
  }

  const owner = querySQLSingleValue(
    `SELECT pg_user.usename FROM pg_database JOIN pg_user ON pg_database.datdba = pg_user.usesysid WHERE pg_database.datname = '${dbName}';`,
    'Child check db owner for adopt',
  );
  if (owner !== roleName) {
    throw new Error(`[child] 待接管数据库 ${dbName} 的所有者为 ${owner}，与预期 ${roleName} 不符`);
  }

  // 加入子进程自身的生命周期跟踪集合，以便信号触发时由子进程自身信号处理器回收
  lifetimeCreatedRoles.add(roleName);
  pendingRoles.add(roleName);
  lifetimeCreatedDatabases.add(dbName);
  pendingDatabases.set(dbName, { roleName });

  console.log('=== [child] 启动受控信号自测子进程 (已接管资源) ===');
  console.log(`[RESOURCE_ADOPTED] db=${dbName} role=${roleName}`);

  if (process.argv.includes('--delay-ready')) {
    // 资源已成功接管，但在输出 CHILD_READY 之前延迟
    await new Promise((r) => setTimeout(r, 4000));
  }

  console.log(`[CHILD_READY] db=${dbName} role=${roleName}`);

  // 保持事件循环活跃，等待接收系统信号
  const keepAlive = setInterval(() => {}, 60000);
  await new Promise(() => {});
  clearInterval(keepAlive);
}

async function executeSignalSubprocessTest({
  mode = 'happy_path',
  readyTimeoutMs = 10000,
  onResourceCreatedVerified = null,
}) {
  const uniqueSuffix = `selftest_sig_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
  const dbName = `${PREFIX}${uniqueSuffix}`;
  const roleName = `${PREFIX}${uniqueSuffix}`;

  // 1. Preflight 安全校验：待创建名称在数据库中必须不存在 (COUNT=0)
  const preDb = querySQLCount01(
    `SELECT COUNT(*) FROM pg_database WHERE datname = '${dbName}';`,
    'Preflight test db',
  );
  const preRole = querySQLCount01(
    `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
    'Preflight test role',
  );
  if (preDb !== '0' || preRole !== '0') {
    throw new Error(
      `[self-test] Preflight 校验失败：待测资源名称已存在 db=${dbName} (${preDb}), role=${roleName} (${preRole})`,
    );
  }

  let child = null;
  let execError = null;
  try {
    // 2. 由父进程受控创建测试资源 (即使后续失败，也有 pending 跟踪集合保护)
    console.log(`[self-test] 父进程创建测试资源: db=${dbName}, role=${roleName}`);
    createExplicitDisposableDatabaseAndRole(dbName, roleName);

    // 3. 确证资源真实存在于 PostgreSQL 中
    const dbCountAfterCreate = querySQLCount01(
      `SELECT COUNT(*) FROM pg_database WHERE datname = '${dbName}';`,
      'Check test db created in postgres',
    );
    const roleCountAfterCreate = querySQLCount01(
      `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
      'Check test role created in postgres',
    );
    if (dbCountAfterCreate !== '1' || roleCountAfterCreate !== '1') {
      throw new Error(
        `[self-test] 父进程资源创建后未在 PostgreSQL 中检测到实体: dbCount=${dbCountAfterCreate}, roleCount=${roleCountAfterCreate}`,
      );
    }

    if (typeof onResourceCreatedVerified === 'function') {
      onResourceCreatedVerified({ dbName, roleName });
    }

    // 4. 启动子进程进行 adopt 接管
    const isUnix = process.platform !== 'win32';
    const childArgs = [
      'scripts/run-acceptance-finance-disposable.mjs',
      '--self-test-signal-child',
      '--db-name',
      dbName,
      '--role-name',
      roleName,
    ];
    if (mode === 'delay_ready') {
      childArgs.push('--delay-ready');
    }

    child = spawn(process.execPath, childArgs, {
      env: process.env,
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: isUnix,
    });
    registerChild(child);

    let childOutput = '';
    const onStdoutData = (data) => {
      childOutput += data.toString();
    };
    const onStderrData = (data) => {
      childOutput += data.toString();
    };
    child.stdout?.on('data', onStdoutData);
    child.stderr?.on('data', onStderrData);

    const waitForMarker = (regex, timeoutMs, timeoutCode = null) => {
      const match = childOutput.match(regex);
      if (match) {
        return Promise.resolve(match);
      }
      return new Promise((resolve, reject) => {
        let timer = null;
        const cleanup = () => {
          if (timer) clearTimeout(timer);
          child.stdout?.removeListener('data', onData);
          child.removeListener('error', onError);
          child.removeListener('exit', onEarlyExit);
        };
        const onData = () => {
          const m = childOutput.match(regex);
          if (m) {
            cleanup();
            resolve(m);
          }
        };
        const onError = (err) => {
          cleanup();
          reject(err);
        };
        const onEarlyExit = (code, signal) => {
          cleanup();
          reject(new Error(`子进程过早退出 (code=${code}, signal=${signal}):\n${childOutput}`));
        };

        timer = setTimeout(() => {
          cleanup();
          if (timeoutCode) {
            reject(
              new SelfTestInjectedError(
                timeoutCode,
                `[self-test] 等待标记 ${regex} 超时 (${timeoutMs}ms):\n${childOutput}`,
              ),
            );
          } else {
            reject(new Error(`[self-test] 等待标记 ${regex} 超时 (${timeoutMs}ms):\n${childOutput}`));
          }
        }, timeoutMs);

        child.stdout?.on('data', onData);
        child.once('error', onError);
        child.once('exit', onEarlyExit);
      });
    };

    // 5. 等待子进程接管资源握手 [RESOURCE_ADOPTED] (最长 8000ms)
    const adoptedMatch = await waitForMarker(/\[RESOURCE_ADOPTED\]\s+db=(\S+)\s+role=(\S+)/, 8000);
    if (adoptedMatch[1] !== dbName || adoptedMatch[2] !== roleName) {
      throw new Error(
        `[self-test] 子进程报告接管的资源名称与父进程不匹配: expected=(${dbName}, ${roleName}), actual=(${adoptedMatch[1]}, ${adoptedMatch[2]})`,
      );
    }
    console.log(`[self-test] 子进程已成功接管资源: db=${dbName}, role=${roleName}`);

    // 6. 等待子进程就绪 [CHILD_READY] (带超时)
    const readyMatch = await waitForMarker(
      /\[CHILD_READY\]\s+db=(\S+)\s+role=(\S+)/,
      readyTimeoutMs,
      'READY_TIMEOUT',
    );
    if (readyMatch[1] !== dbName || readyMatch[2] !== roleName) {
      throw new Error(
        `[self-test] 子进程报告就绪的资源名称与父进程不匹配: expected=(${dbName}, ${roleName}), actual=(${readyMatch[1]}, ${readyMatch[2]})`,
      );
    }

    console.log(`[self-test] 子进程已就绪: db=${dbName}, role=${roleName}`);

    // 分支 1: 故障注入 - 模拟父流程异常
    if (mode === 'simulate_parent_failure') {
      throw new SelfTestInjectedError('SIMULATED_PARENT_FAILURE', '故障注入: 模拟父流程在子进程就绪后发生严重异常');
    }

    // 分支 2: 正常 SIGTERM 中断测试 (Happy path)
    if (mode === 'happy_path') {
      const exitPromise = new Promise((resolve, reject) => {
        let timeoutTimer = null;
        const cleanup = () => {
          if (timeoutTimer) clearTimeout(timeoutTimer);
          child.removeListener('error', onError);
          child.removeListener('exit', onExit);
        };
        const onError = (err) => {
          cleanup();
          reject(err);
        };
        const onExit = (code, signal) => {
          cleanup();
          resolve({ code, signal });
        };
        timeoutTimer = setTimeout(() => {
          cleanup();
          reject(new Error('[self-test] 等待子进程响应 SIGTERM 退出超时 (10000ms)'));
        }, 10000);

        child.once('error', onError);
        child.once('exit', onExit);
      });

      console.log('[self-test] 向子进程发送 SIGTERM 信号...');
      if (isUnix) {
        try {
          process.kill(-child.pid, 'SIGTERM');
        } catch {
          process.kill(child.pid, 'SIGTERM');
        }
      } else {
        process.kill(child.pid, 'SIGTERM');
      }

      const exitInfo = await exitPromise;
      console.log(`[self-test] 子进程退出响应: code=${exitInfo.code}, signal=${exitInfo.signal}`);

      if (exitInfo.code === 0) {
        throw new Error('[self-test] 收到信号的子进程不应以 0 成功退出');
      }
    }
  } catch (err) {
    execError = err;
  } finally {
    let stopError = null;
    let dbCleanupError = null;
    let roleCleanupError = null;

    // 1. 确保子进程完全停止（先停 child，再清理资源）
    try {
      if (child && isTargetAlive(child)) {
        await stopProcess(child);
      }
    } catch (err) {
      stopError = err;
    }

    // 2. 清理残留数据库 (核验归属并确保最终 COUNT=0)
    try {
      if (pendingDatabases.has(dbName)) {
        const dbRemaining = querySQLCount01(
          `SELECT COUNT(*) FROM pg_database WHERE datname = '${dbName}';`,
          'Check test db remaining in finally',
        );
        if (dbRemaining === '1') {
          const owner = querySQLSingleValue(
            `SELECT pg_user.usename FROM pg_database JOIN pg_user ON pg_database.datdba = pg_user.usesysid WHERE pg_database.datname = '${dbName}';`,
            'Check db owner before finally cleanup',
          );
          if (owner !== roleName) {
            throw new Error(`[self-test] 残留数据库 ${dbName} 所有者为 ${owner}，与预期 ${roleName} 不符`);
          }
          console.log(`[self-test] 父进程清理残留测试数据库: ${dbName}`);
          execSQL(
            `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${dbName}' AND pid <> pg_backend_pid();`,
            'Terminate test db connections',
          );
          execSQL(`DROP DATABASE ${dbName};`, `DROP DATABASE ${dbName}`);
          const dbAfter = querySQLCount01(
            `SELECT COUNT(*) FROM pg_database WHERE datname = '${dbName}';`,
            'Check test db count after drop',
          );
          if (dbAfter !== '0') {
            throw new Error(`[self-test] 数据库清理验证失败: db=${dbName}`);
          }
        }
        pendingDatabases.delete(dbName);
      }
    } catch (err) {
      dbCleanupError = err;
    }

    // 3. 清理残留角色 (确保最终 COUNT=0)
    try {
      if (pendingRoles.has(roleName)) {
        const roleRemaining = querySQLCount01(
          `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
          'Check test role remaining in finally',
        );
        if (roleRemaining === '1') {
          console.log(`[self-test] 父进程清理残留测试角色: ${roleName}`);
          execSQL(`DROP ROLE ${roleName};`, `DROP ROLE ${roleName}`);
          const roleAfter = querySQLCount01(
            `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
            'Check test role count after drop',
          );
          if (roleAfter !== '0') {
            throw new Error(`[self-test] 角色清理验证失败: role=${roleName}`);
          }
        }
        pendingRoles.delete(roleName);
      }
    } catch (err) {
      roleCleanupError = err;
    }

    const cleanupErrors = [stopError, dbCleanupError, roleCleanupError].filter(Boolean);
    let combinedCleanupError = null;
    if (cleanupErrors.length === 1) {
      combinedCleanupError = cleanupErrors[0];
    } else if (cleanupErrors.length > 1) {
      combinedCleanupError = new AggregateError(
        cleanupErrors,
        `[self-test] 清理阶段发生多个错误:\n${cleanupErrors.map((e) => e.message).join('\n')}`,
      );
    }

    if (execError && combinedCleanupError) {
      if (typeof AggregateError !== 'undefined') {
        throw new AggregateError(
          [execError, combinedCleanupError],
          `[self-test] 执行错误与清理错误同时发生:\n1. ${execError.message}\n2. ${combinedCleanupError.message}`,
        );
      } else {
        execError.cleanupError = combinedCleanupError;
        throw execError;
      }
    } else if (combinedCleanupError) {
      throw combinedCleanupError;
    } else if (execError) {
      throw execError;
    }
  }
}

async function executeOnlyRoleTest({ mode = 'happy_path', onRoleCreatedVerified = null }) {
  const uniqueTag = `selftest_only_role_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
  const roleName = `${PREFIX}${uniqueTag}`;
  const rolePassword = crypto.randomBytes(16).toString('hex');

  // 1. Preflight 安全校验：待创建角色必须不存在
  const preflightCount = querySQLCount01(
    `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
    'Preflight only-role',
  );
  if (preflightCount !== '0') {
    throw new Error(`[self-test] Preflight 校验失败：角色 ${roleName} 已存在`);
  }

  let roleCreated = false;
  let execError = null;
  try {
    console.log(`[self-test] 创建测试角色 (模式: ${mode}): ${roleName}`);
    execSQL(`CREATE ROLE ${roleName} WITH LOGIN PASSWORD '${rolePassword}';`, `CREATE ROLE ${roleName}`);
    roleCreated = true;
    lifetimeCreatedRoles.add(roleName);
    pendingRoles.add(roleName);

    const initialRoleCount = querySQLCount01(
      `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
      'Check only-role initial',
    );
    if (initialRoleCount !== '1') {
      throw new Error('self-test only-role 角色创建验证失败');
    }
    if (typeof onRoleCreatedVerified === 'function') {
      onRoleCreatedVerified({ roleName });
    }

    if (mode === 'simulate_failure') {
      throw new SelfTestInjectedError(
        'SIMULATED_ONLY_ROLE_FAILURE',
        '故障注入: 模拟 only-role 创建后发生父流程异常',
      );
    }

    if (mode === 'happy_path') {
      await performGlobalCleanup('self-test only-role validation');
      const afterRoleCount = querySQLCount01(
        `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
        'Check only-role after cleanup',
      );
      if (afterRoleCount !== '0') {
        throw new Error('self-test only-role 角色清理验证失败：角色残留');
      }
      if (pendingRoles.has(roleName)) {
        throw new Error('pendingRoles 集合应已移除该角色');
      }
    }
  } catch (err) {
    execError = err;
  } finally {
    let cleanupError = null;
    try {
      if (roleCreated && pendingRoles.has(roleName)) {
        const remaining = querySQLCount01(
          `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
          'Check only-role in finally',
        );
        if (remaining === '1') {
          console.log(`[self-test] finally 兜底清理测试角色: ${roleName}`);
          execSQL(`DROP ROLE ${roleName};`, `DROP ROLE ${roleName}`);
          const countAfter = querySQLCount01(
            `SELECT COUNT(*) FROM pg_roles WHERE rolname = '${roleName}';`,
            'Check only-role count after finally drop',
          );
          if (countAfter !== '0') {
            throw new Error(`[self-test] only-role 兜底清理验证失败: role=${roleName}`);
          }
        }
        pendingRoles.delete(roleName);
      }
    } catch (cleanErr) {
      cleanupError = cleanErr;
    }

    if (execError && cleanupError) {
      if (typeof AggregateError !== 'undefined') {
        throw new AggregateError(
          [execError, cleanupError],
          `[self-test] only-role 执行与清理均发生错误:\n1. ${execError.message}\n2. ${cleanupError.message}`,
        );
      } else {
        execError.cleanupError = cleanupError;
        throw execError;
      }
    } else if (cleanupError) {
      throw cleanupError;
    } else if (execError) {
      throw execError;
    }
  }
}

async function executeProcessGroupFaultInjectionTest() {
  const isUnix = process.platform !== 'win32';
  console.log('\n--- [Test 5] 故障注入: 验证 leader 进程退出但孤儿孙进程残留时的直接进程组清理与追踪 ---');

  const childScript = `
    const { spawn } = require('child_process');
    const sub = spawn(process.execPath, ['-e', 'setInterval(() => {}, 60000);'], {
      stdio: 'ignore',
      detached: false,
    });
    process.exit(0);
  `;

  let leader = null;
  let execError = null;
  try {
    leader = spawn(process.execPath, ['-e', childScript], {
      stdio: 'ignore',
      detached: isUnix,
    });
    registerChild(leader);

    // 等待 leader 进程退出 (最长 5000ms)
    await new Promise((resolve, reject) => {
      let timer = setTimeout(() => {
        reject(new Error('[self-test] 等待 leader 退出超时'));
      }, 5000);
      leader.once('exit', () => {
        clearTimeout(timer);
        resolve();
      });
      leader.once('error', (err) => {
        clearTimeout(timer);
        reject(err);
      });
    });

    if (isUnix) {
      const isGroupAlive = isTargetAlive(leader);
      if (!isGroupAlive) {
        throw new Error('[self-test] 进程组故障注入前置失败：孤儿孙进程未能在进程组中存活');
      }
      if (!activeChildProcesses.has(leader)) {
        throw new Error('[self-test] 进程组追踪验证失败：leader 退出后活跃追踪集合不应丢弃存活的进程组');
      }
      console.log('[self-test-evidence] Test 5 确证 leader 已退出但进程组仍然存活且保持追踪');

      // 调用 stopProcess 清理整个进程组
      await stopProcess(leader);

      if (isTargetAlive(leader)) {
        throw new Error('[self-test] stopProcess 未能彻底终止残留的孤儿孙进程组');
      }
      if (activeChildProcesses.has(leader)) {
        throw new Error('[self-test] stopProcess 彻底终止后未从 activeChildProcesses 移除');
      }
      console.log('[self-test] 进程组故障注入自测通过！孤儿孙进程已被彻底终止，追踪集合已清理');
    } else {
      await stopProcess(leader);
      console.log('[self-test] 非 Unix 环境进程测试跳过组存活断言');
    }
  } catch (err) {
    execError = err;
  } finally {
    let cleanupError = null;
    try {
      if (leader && isTargetAlive(leader)) {
        console.log('[self-test] finally 兜底清理残留测试进程组');
        await stopProcess(leader);
      }
      if (leader) {
        activeChildProcesses.delete(leader);
      }
    } catch (cleanErr) {
      cleanupError = cleanErr;
    }

    if (execError && cleanupError) {
      if (typeof AggregateError !== 'undefined') {
        throw new AggregateError(
          [execError, cleanupError],
          `[self-test] Test 5 执行与清理均发生错误:\n1. ${execError.message}\n2. ${cleanupError.message}`,
        );
      } else {
        execError.cleanupError = cleanupError;
        throw execError;
      }
    } else if (cleanupError) {
      throw cleanupError;
    } else if (execError) {
      throw execError;
    }
  }
}

async function executeRunCommandProcessGroupSafetyTest() {
  const isUnix = process.platform !== 'win32';
  console.log('\n--- [Test 6A] 故障注入: 验证 runCommand 在 leader 正常退出 (0) 但进程组残留时的拒绝与自动清组 ---');

  if (!isUnix) {
    console.log('[self-test] 非 Unix 环境跳过 runCommand 进程组残留检测');
    return;
  }

  // 6A: exit 0 + orphan
  const childScript0 = `
    const { spawn } = require('child_process');
    const sub = spawn(process.execPath, ['-e', 'setInterval(() => {}, 60000);'], {
      stdio: 'ignore',
      detached: false,
    });
    process.exit(0);
  `;

  let spawnedChild0 = null;
  let execError0 = null;
  try {
    let cmdError0 = null;
    try {
      await runCommand(process.execPath, ['-e', childScript0], {
        onSpawn: (c) => {
          spawnedChild0 = c;
        },
      });
    } catch (err) {
      cmdError0 = err;
    }

    if (!cmdError0) {
      throw new Error('[self-test] runCommand 在 leader exit 0 但进程组残留时不应 resolve 成功');
    }

    const errMsg0 = cmdError0.message || '';
    if (!errMsg0.includes('leader 进程已正常退出') && !errMsg0.includes('leader 进程已退出')) {
      throw new Error(`[self-test] runCommand exit 0 拒绝原因不符合预期: ${errMsg0}`);
    }
    if (!errMsg0.includes('进程组仍有孤儿进程存活') && !errMsg0.includes('进程组仍有孤儿进程残留')) {
      throw new Error(`[self-test] runCommand exit 0 孤儿信息不符合预期: ${errMsg0}`);
    }

    if (spawnedChild0 && isTargetAlive(spawnedChild0)) {
      throw new Error('[self-test] runCommand 拒绝后未能彻底终止残留的孤儿进程组');
    }
    if (spawnedChild0 && activeChildProcesses.has(spawnedChild0)) {
      throw new Error('[self-test] runCommand 终止孤儿组后未从 activeChildProcesses 移除');
    }
    console.log('[self-test] runCommand exit 0 进程组安全防护自测通过！确证拒绝假 PASS 并自动清组');
  } catch (err) {
    execError0 = err;
  } finally {
    let cleanupError = null;
    try {
      if (spawnedChild0 && isTargetAlive(spawnedChild0)) {
        console.log('[self-test] finally 兜底清理 runCommand 测试进程组 (0)');
        await stopProcess(spawnedChild0);
      }
      if (spawnedChild0) {
        activeChildProcesses.delete(spawnedChild0);
      }
    } catch (cleanErr) {
      cleanupError = cleanErr;
    }

    if (execError0 && cleanupError) {
      if (typeof AggregateError !== 'undefined') {
        throw new AggregateError(
          [execError0, cleanupError],
          `[self-test] Test 6A 执行与清理均发生错误:\n1. ${execError0.message}\n2. ${cleanupError.message}`,
        );
      } else {
        execError0.cleanupError = cleanupError;
        throw execError0;
      }
    } else if (cleanupError) {
      throw cleanupError;
    } else if (execError0) {
      throw execError0;
    }
  }

  // 6B: non-zero exit + captured output + orphan
  console.log('\n--- [Test 6B] 故障注入: 验证 runCommand 在命令非零失败且进程组残留时同时保留命令错误与孤儿组错误 ---');
  const childScriptNonZero = `
    const { spawn } = require('child_process');
    const sub = spawn(process.execPath, ['-e', 'setInterval(() => {}, 60000);'], {
      stdio: 'ignore',
      detached: false,
    });
    console.error('SIMULATED_FAIL_OUTPUT_FROM_CHILD');
    process.exit(42);
  `;

  let spawnedChildNonZero = null;
  let execErrorNonZero = null;
  try {
    let cmdErrorNonZero = null;
    try {
      await runCommand(process.execPath, ['-e', childScriptNonZero], {
        capture: true,
        onSpawn: (c) => {
          spawnedChildNonZero = c;
        },
      });
    } catch (err) {
      cmdErrorNonZero = err;
    }

    if (!cmdErrorNonZero) {
      throw new Error('[self-test] runCommand 在非零退出且进程组残留时不应 resolve 成功');
    }

    const fullErrMsg =
      (cmdErrorNonZero.message || '') +
      (cmdErrorNonZero.errors ? '\n' + cmdErrorNonZero.errors.map((e) => e.message).join('\n') : '');
    if (!fullErrMsg.includes('代码 42') && !fullErrMsg.includes('code=42')) {
      throw new Error(`[self-test] runCommand 非零错误未保留退出码 42: ${fullErrMsg}`);
    }
    if (!fullErrMsg.includes('SIMULATED_FAIL_OUTPUT_FROM_CHILD')) {
      throw new Error(`[self-test] runCommand 非零错误未保留 captured stderr 输出: ${fullErrMsg}`);
    }
    if (!fullErrMsg.includes('进程组仍有孤儿进程残留') && !fullErrMsg.includes('进程组仍有孤儿进程存活')) {
      throw new Error(`[self-test] runCommand 非零错误未保留孤儿进程组异常信息: ${fullErrMsg}`);
    }

    if (spawnedChildNonZero && isTargetAlive(spawnedChildNonZero)) {
      throw new Error('[self-test] runCommand 拒绝后未能彻底终止残留的孤儿进程组');
    }
    if (spawnedChildNonZero && activeChildProcesses.has(spawnedChildNonZero)) {
      throw new Error('[self-test] runCommand 终止孤儿组后未从 activeChildProcesses 移除');
    }
    console.log(
      '[self-test] runCommand non-zero 进程组双错误保留自测通过！确证同时保留命令错误与孤儿组信息并自动清组',
    );
  } catch (err) {
    execErrorNonZero = err;
  } finally {
    let cleanupError = null;
    try {
      if (spawnedChildNonZero && isTargetAlive(spawnedChildNonZero)) {
        console.log('[self-test] finally 兜底清理 runCommand 测试进程组 (non-zero)');
        await stopProcess(spawnedChildNonZero);
      }
      if (spawnedChildNonZero) {
        activeChildProcesses.delete(spawnedChildNonZero);
      }
    } catch (cleanErr) {
      cleanupError = cleanErr;
    }

    if (execErrorNonZero && cleanupError) {
      if (typeof AggregateError !== 'undefined') {
        throw new AggregateError(
          [execErrorNonZero, cleanupError],
          `[self-test] Test 6B 执行与清理均发生错误:\n1. ${execErrorNonZero.message}\n2. ${cleanupError.message}`,
        );
      } else {
        execErrorNonZero.cleanupError = cleanupError;
        throw execErrorNonZero;
      }
    } else if (cleanupError) {
      throw cleanupError;
    } else if (execErrorNonZero) {
      throw execErrorNonZero;
    }
  }
}

async function executeStageDualFailureSafetyTest() {
  console.log('\n--- [Test 7] 故障注入: 验证 stage 阶段业务执行错误与清理错误双失败时的完整保留 ---');

  // 1. 验证双错误合并
  const err1 = new Error('模拟业务执行失败');
  const err2 = new Error('模拟清理阶段失败');
  const combined = combineExecutionAndCleanupErrors(err1, err2, 'TestStage');
  if (!(combined instanceof Error)) {
    throw new Error('[self-test] combineExecutionAndCleanupErrors 未返回 Error 对象');
  }
  const combinedMsg =
    (combined.message || '') + (combined.errors ? '\n' + combined.errors.map((e) => e.message).join('\n') : '');
  if (!combinedMsg.includes('模拟业务执行失败') || !combinedMsg.includes('模拟清理阶段失败')) {
    throw new Error(`[self-test] combineExecutionAndCleanupErrors 未能同时保留双错误: ${combinedMsg}`);
  }

  // 2. 验证单执行错误
  const singleExec = combineExecutionAndCleanupErrors(err1, null, 'TestStage');
  if (singleExec !== err1) {
    throw new Error('[self-test] combineExecutionAndCleanupErrors 单执行错误应原样返回');
  }

  // 3. 验证单清理错误
  const singleClean = combineExecutionAndCleanupErrors(null, err2, 'TestStage');
  if (singleClean !== err2) {
    throw new Error('[self-test] combineExecutionAndCleanupErrors 单清理错误应原样返回');
  }

  // 4. 验证无错误
  const noErr = combineExecutionAndCleanupErrors(null, null, 'TestStage');
  if (noErr !== null) {
    throw new Error('[self-test] combineExecutionAndCleanupErrors 无错误应返回 null');
  }

  // 5. 验证模拟 stage 工作流 try/catch/finally 结构
  let stageWorkflowThrew = false;
  try {
    let mockExecErr = null;
    let mockCleanErr = null;
    try {
      throw new Error('模拟 Stage 内部业务断言失败');
    } catch (err) {
      mockExecErr = err;
    } finally {
      try {
        throw new Error('模拟 Stage 内部清理资源失败');
      } catch (cErr) {
        mockCleanErr = cErr;
      }
      const combinedStageErr = combineExecutionAndCleanupErrors(mockExecErr, mockCleanErr, 'MockStage');
      if (combinedStageErr) throw combinedStageErr;
    }
  } catch (stageErr) {
    stageWorkflowThrew = true;
    const stageErrMsg =
      (stageErr.message || '') + (stageErr.errors ? '\n' + stageErr.errors.map((e) => e.message).join('\n') : '');
    if (
      !stageErrMsg.includes('模拟 Stage 内部业务断言失败') ||
      !stageErrMsg.includes('模拟 Stage 内部清理资源失败')
    ) {
      throw new Error(`[self-test] 阶段双失败工作流未能同时保留双错误: ${stageErrMsg}`);
    }
  }

  if (!stageWorkflowThrew) {
    throw new Error('[self-test] 阶段双失败工作流应抛出聚合错误');
  }

  console.log('[self-test] stage 阶段双错误保留与合并自测通过！确证执行错误与清理错误均完整保留');
}

async function runSelfTestLifecycle() {
  console.log('=== 开始执行资源生命周期与信号中断清理自测 ===');

  // Test 1A: Only-Role 状态正常清理自测 (Happy Path)
  console.log('\n--- [Test 1A] 验证 only-role 状态正常清理路径 (Happy Path) ---');
  await executeOnlyRoleTest({ mode: 'happy_path' });
  console.log('[self-test] only-role 正常清理自测通过！');

  // Test 1B: Only-Role 故障注入自测
  console.log('\n--- [Test 1B] 故障注入: 验证 only-role 创建后发生异常时 finally 兜底清理 ---');
  let test1bError = null;
  let test1bVerified = false;
  try {
    await executeOnlyRoleTest({
      mode: 'simulate_failure',
      onRoleCreatedVerified: ({ roleName }) => {
        test1bVerified = true;
        console.log(`[self-test-evidence] Test 1B 故障注入已通过直接 SQL 证明角色真实创建: role=${roleName}`);
      },
    });
  } catch (err) {
    test1bError = err;
  }
  if (!(test1bError instanceof SelfTestInjectedError) || test1bError.code !== 'SIMULATED_ONLY_ROLE_FAILURE') {
    throw (
      test1bError ||
      new Error('[self-test] only-role 故障注入未能捕获到预期的 SelfTestInjectedError(SIMULATED_ONLY_ROLE_FAILURE)')
    );
  }
  if (!test1bVerified) {
    throw new Error('[self-test] Test 1B 故障注入前未确认角色真实创建');
  }
  console.log('[self-test] only-role 故障注入自测通过！确认角色真实创建且已由 finally 完全回收');

  // Test 2: 真实子进程信号中断 (SIGTERM) Happy Path
  console.log('\n--- [Test 2] 验证真实子进程 SIGTERM 信号清理路径 (Happy Path) ---');
  await executeSignalSubprocessTest({ mode: 'happy_path' });
  console.log('[self-test] SIGTERM 正常信号中断清理自测通过！');

  // Test 3: 故障注入 1 - 子进程就绪后父流程模拟异常，验证 finally 兜底清理与进程终止
  console.log('\n--- [Test 3] 故障注入: 验证父流程发生异常时子进程终止与资源回收 ---');
  let test3Error = null;
  try {
    await executeSignalSubprocessTest({ mode: 'simulate_parent_failure' });
  } catch (err) {
    test3Error = err;
  }
  if (!(test3Error instanceof SelfTestInjectedError) || test3Error.code !== 'SIMULATED_PARENT_FAILURE') {
    throw (
      test3Error ||
      new Error('[self-test] 故障注入测试未能捕获到预期的 SelfTestInjectedError(SIMULATED_PARENT_FAILURE)')
    );
  }
  console.log('[self-test] 故障注入 (父流程异常) 验证通过！finally 已成功停止子进程并回收全部资源');

  // Test 4: 故障注入 2 - 就绪超时模拟，验证 child 真实创建资源后超时被 finally 兜底清理
  console.log('\n--- [Test 4] 故障注入: 验证子进程就绪超时时子进程终止与资源回收 ---');
  let test4Error = null;
  let test4ResourceCreatedVerified = false;
  try {
    await executeSignalSubprocessTest({
      mode: 'delay_ready',
      readyTimeoutMs: 200,
      onResourceCreatedVerified: ({ dbName, roleName }) => {
        test4ResourceCreatedVerified = true;
        console.log(
          `[self-test-evidence] Test 4 故障注入已通过直接 SQL 证明父进程资源真实创建: db=${dbName}, role=${roleName}`,
        );
      },
    });
  } catch (err) {
    test4Error = err;
  }
  if (!(test4Error instanceof SelfTestInjectedError) || test4Error.code !== 'READY_TIMEOUT') {
    throw (
      test4Error ||
      new Error('[self-test] 就绪超时故障注入未能捕获到预期的 SelfTestInjectedError(READY_TIMEOUT)')
    );
  }
  if (!test4ResourceCreatedVerified) {
    throw new Error('[self-test] Test 4 故障注入前未确认资源真实创建');
  }
  console.log('[self-test] 故障注入 (就绪超时) 验证通过！确认资源真实创建、child 已接管且已由 finally 完全回收');

  // Test 5: 进程组孤儿孙进程存活与直接清理故障注入自测
  await executeProcessGroupFaultInjectionTest();

  // Test 6: runCommand 在 leader 正常/非零退出但进程组残留时的拒绝与双错误保留自测
  await executeRunCommandProcessGroupSafetyTest();

  // Test 7: stage 阶段业务执行与清理双失败时的错误保留自测
  await executeStageDualFailureSafetyTest();

  // 最终活跃跟踪集合归零断言
  if (activeChildProcesses.size !== 0) {
    throw new Error(`[self-test] 全部自测结束后活跃进程跟踪集合不为 0: size=${activeChildProcesses.size}`);
  }
  console.log(`[self-test] 活跃追踪进程数检查通过: activeChildProcesses.size = 0`);

  console.log('\n======================================================');
  console.log('  全部生命周期、信号中断与故障注入清理自测顺利通过！');
  console.log('======================================================\n');
}

async function main() {
  if (process.argv.includes('--self-test-signal-child')) {
    await runSelfTestSignalChild();
    return;
  }
  if (process.argv.includes('--self-test-lifecycle')) {
    await runSelfTestLifecycle();
    return;
  }

  console.log('=== 开始一次性 PostgreSQL 双环境财务验收编排 ===');

  // 1. 端口检查
  const [port8010Available, port9010Available, port8001Available] = await Promise.all([
    isPortAvailable(GO_SERVER_PORT),
    isPortAvailable(9010),
    isPortAvailable(WEB_SERVER_PORT),
  ]);

  if (!port8010Available || !port9010Available || !port8001Available) {
    throw new Error(
      `端口检查失败：` +
        (!port8010Available ? `[${GO_SERVER_PORT} 已被占用] ` : '') +
        (!port9010Available ? `[9010 已被占用] ` : '') +
        (!port8001Available ? `[${WEB_SERVER_PORT} 已被占用] ` : '') +
        `。编排器直接终止，禁止终止未知进程或变更端口。`,
    );
  }
  console.log(`[disposable] 端口检查通过：${GO_SERVER_PORT}, 9010 与 ${WEB_SERVER_PORT} 均空闲可用`);

  let finalError = null;
  try {
    await runStageA();
    await runStageB();
  } catch (err) {
    finalError = err;
  } finally {
    try {
      await performGlobalCleanup('编排执行结束');
    } catch (cleanupErr) {
      if (finalError) {
        if (typeof AggregateError !== 'undefined') {
          finalError = new AggregateError(
            [finalError, cleanupErr],
            `[disposable] 验收编排执行与全局清理均发生错误:\n1. ${finalError.message}\n2. ${cleanupErr.message}`,
          );
        } else {
          finalError.cleanupError = cleanupErr;
        }
      } else {
        finalError = cleanupErr;
      }
    }
  }

  if (finalError) {
    console.error('\n[disposable] 验收编排执行失败：', finalError);
    process.exit(1);
  }

  console.log('\n======================================================');
  console.log('  一次性 PostgreSQL 双环境财务全链路验收全部成功！');
  console.log('======================================================\n');
}

main();
