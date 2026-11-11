import { describe, expect, it } from 'vitest';
import { ignorePasswordManagers } from './ignorePasswordManagers';

describe('密码管理器忽略标记', () => {
  it('覆盖已有和动态加入的输入框，包括浮层与密码字段', async () => {
    const root = document.createElement('div');
    root.innerHTML = '<input autocomplete="new-password"><textarea></textarea>';
    const stop = ignorePasswordManagers(root);
    const popup = document.createElement('div');
    popup.innerHTML = '<input type="password" autocomplete="current-password">';
    root.append(popup);
    await Promise.resolve();

    for (const input of root.querySelectorAll('input, textarea')) {
      expect(input).toHaveAttribute('autocomplete', 'off');
      expect(input).toHaveAttribute('data-1p-ignore', 'true');
      expect(input).toHaveAttribute('data-lpignore', 'true');
      expect(input).toHaveAttribute('data-bwignore', 'true');
      expect(input).toHaveAttribute('data-form-type', 'other');
    }
    const input = root.querySelector('input');
    input?.setAttribute('autocomplete', 'new-password');
    await Promise.resolve();
    expect(input).toHaveAttribute('autocomplete', 'off');
    stop();
  });
});
