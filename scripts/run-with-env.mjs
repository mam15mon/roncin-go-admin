import { spawn } from 'node:child_process';

const [command, ...args] = process.argv.slice(2);

if (!command) {
  throw new Error('command is required');
}

const child = spawn(command, args, {
  stdio: 'inherit',
  env: process.env,
  shell: process.platform === 'win32',
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exitCode = code ?? 1;
});
