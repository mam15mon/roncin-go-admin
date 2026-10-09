import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  measureStickyTopOffset,
  scrollToFirstFormError,
  scrollToFirstTableError,
  scrollToSectionWithStickyOffset,
} from './formErrorUtils';

// 使用替代尺寸验证真实滚动坐标，避免默认数字恰好相同掩盖配置未接入。
vi.mock('@root/config/layout', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@root/config/layout')>();
  return {
    ...actual,
    layoutDimensions: {
      ...actual.layoutDimensions,
      formScrollOffset: 180,
      formScrollGap: 20,
      tableErrorOffset: 130,
      tableErrorScrollGap: 24,
    },
  };
});

function mockRect(element: HTMLElement, top: number, height = 20) {
  vi.spyOn(element, 'getBoundingClientRect').mockReturnValue({
    x: 0,
    y: top,
    top,
    bottom: top + height,
    left: 0,
    right: 100,
    width: 100,
    height,
    toJSON: () => ({}),
  });
}

afterEach(() => {
  document.body.replaceChildren();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('布局配置驱动表单滚动', () => {
  it('无页头时读取配置默认偏移，同时保留调用方完整偏移', () => {
    expect(measureStickyTopOffset()).toBe(180);
    expect(measureStickyTopOffset(205)).toBe(205);

    const section = document.createElement('section');
    mockRect(section, 500);
    const scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});

    scrollToSectionWithStickyOffset(section);
    expect(scrollTo).toHaveBeenLastCalledWith({ top: 320, behavior: 'smooth' });

    scrollToSectionWithStickyOffset(section, 205);
    expect(scrollTo).toHaveBeenLastCalledWith({ top: 295, behavior: 'smooth' });
  });

  it('有页头时使用实测底边加配置间隙，优先于默认或显式偏移', () => {
    const shell = document.createElement('header');
    shell.className = 'roncin-page-header-shell';
    mockRect(shell, 100, 110.4);
    document.body.appendChild(shell);

    expect(measureStickyTopOffset()).toBe(231);
    expect(measureStickyTopOffset(205)).toBe(231);

    const section = document.createElement('section');
    mockRect(section, 500);
    const scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
    scrollToSectionWithStickyOffset(section, 205);
    expect(scrollTo).toHaveBeenCalledWith({ top: 269, behavior: 'smooth' });
  });

  it('表单错误居中滚动读取替代默认，并保留显式 headerOffset', () => {
    vi.useFakeTimers();
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(800);
    const container = document.createElement('form');
    container.innerHTML =
      '<div class="ant-form-item-has-error"><input /></div>';
    document.body.appendChild(container);
    const error = container.firstElementChild as HTMLElement;
    mockRect(error, 600);
    const scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});

    scrollToFirstFormError({ container });
    vi.advanceTimersByTime(60);
    expect(scrollTo).toHaveBeenLastCalledWith({ top: 150, behavior: 'smooth' });

    scrollToFirstFormError({ container, headerOffset: 240 });
    vi.advanceTimersByTime(60);
    expect(scrollTo).toHaveBeenLastCalledWith({ top: 130, behavior: 'smooth' });
  });

  it('表格错误使用独立阈值与落点间隙，显式 headerOffset 覆盖阈值', () => {
    vi.useFakeTimers();
    const container = document.createElement('div');
    container.innerHTML =
      '<div class="ant-form-item-has-error"><input /></div>';
    document.body.appendChild(container);
    const error = container.firstElementChild as HTMLElement;
    mockRect(error, 110);
    error.scrollIntoView = vi.fn();
    const scrollBy = vi.spyOn(window, 'scrollBy').mockImplementation(() => {});

    scrollToFirstTableError({ container });
    expect(scrollBy).toHaveBeenLastCalledWith({ top: -44, behavior: 'smooth' });

    scrollToFirstTableError({ container, headerOffset: 150 });
    expect(scrollBy).toHaveBeenLastCalledWith({ top: -64, behavior: 'smooth' });

    scrollBy.mockClear();
    scrollToFirstTableError({ container, headerOffset: 100 });
    expect(scrollBy).not.toHaveBeenCalled();
  });
});
