import { spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const colors = {
  reset: '\x1b[0m',
  cyan: '\x1b[36m',
  magenta: '\x1b[35m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  bold: '\x1b[1m',
};

function runTask(name, command, args, prefixColor) {
  const startTime = Date.now();
  const env = { ...process.env };
  if (fs.existsSync('/dev/shm')) {
    const gocache = '/dev/shm/roncin-gocache';
    const gotmp = '/dev/shm/roncin-gotmp';
    try {
      if (!fs.existsSync(gocache)) fs.mkdirSync(gocache, { recursive: true });
      if (!fs.existsSync(gotmp)) fs.mkdirSync(gotmp, { recursive: true });
      env.GOCACHE = gocache;
      env.GOTMPDIR = gotmp;
    } catch {
      // Ignore if cannot create in /dev/shm, fallback to default
    }
  }
  return new Promise((resolve, reject) => {
    const proc = spawn(command, args, {
      cwd: repoRoot,
      env,
      stdio: ['inherit', 'pipe', 'pipe'],
    });

    const prefix = `${prefixColor}[${name}]${colors.reset} `;

    let stdoutBuffer = '';
    proc.stdout.on('data', (chunk) => {
      stdoutBuffer += chunk.toString();
      const lines = stdoutBuffer.split('\n');
      stdoutBuffer = lines.pop() ?? '';
      for (const line of lines) {
        process.stdout.write(`${prefix}${line}\n`);
      }
    });

    let stderrBuffer = '';
    proc.stderr.on('data', (chunk) => {
      stderrBuffer += chunk.toString();
      const lines = stderrBuffer.split('\n');
      stderrBuffer = lines.pop() ?? '';
      for (const line of lines) {
        process.stderr.write(`${prefix}${line}\n`);
      }
    });

    proc.on('error', (err) => {
      reject({ name, error: err, duration: (Date.now() - startTime) / 1000 });
    });

    proc.on('close', (code) => {
      if (stdoutBuffer) process.stdout.write(`${prefix}${stdoutBuffer}\n`);
      if (stderrBuffer) process.stderr.write(`${prefix}${stderrBuffer}\n`);
      const duration = (Date.now() - startTime) / 1000;
      if (code === 0) {
        resolve({ name, duration });
      } else {
        reject({ name, code, duration });
      }
    });
  });
}

async function main() {
  const overallStart = Date.now();
  console.log(`${colors.bold}${colors.cyan}==> 启动 64 核全量并发门禁检查 (Web + Server 并行)...${colors.reset}\n`);

  const tasks = [
    runTask('WEB', 'pnpm', ['run', 'check:web'], colors.cyan),
    runTask('SERVER', 'pnpm', ['run', 'check:server'], colors.magenta),
  ];

  try {
    const results = await Promise.all(tasks);
    const overallDuration = (Date.now() - overallStart) / 1000;
    const sumDuration = results.reduce((acc, r) => acc + r.duration, 0);
    const saved = sumDuration - overallDuration;

    console.log(`\n${colors.bold}${colors.green}✔ 全部门禁检查通过！${colors.reset}`);
    for (const r of results) {
      console.log(`  - ${r.name}: ${r.duration.toFixed(1)}s`);
    }
    console.log(`  - 实际总耗时: ${colors.bold}${overallDuration.toFixed(1)}s${colors.reset}（串行耗时约 ${sumDuration.toFixed(1)}s，节省了 ${colors.yellow}${saved.toFixed(1)}s${colors.reset}）`);
  } catch (err) {
    const overallDuration = (Date.now() - overallStart) / 1000;
    console.error(`\n${colors.bold}${colors.red}✖ 任务 [${err.name}] 失败（退出码: ${err.code ?? 'error'}），总耗时: ${overallDuration.toFixed(1)}s${colors.reset}`);
    process.exit(err.code || 1);
  }
}

main();
