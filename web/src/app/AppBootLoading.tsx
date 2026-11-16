import { QuarterRing } from '@/components/ui';

/**
 * 首屏启动加载画面：与 public/scripts/loading.js 的静态占位保持同一外观，
 * 覆盖「认证初始状态获取」与「路由首屏懒加载」两个阶段，避免启动期间
 * 出现空白闪烁和第二个样式不同的加载界面。两处外观须同步修改。
 */
export function AppBootLoading() {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        backgroundColor: '#f5f7fa',
        fontFamily:
          "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
      }}
    >
      <div style={{ padding: 20, color: '#1677ff' }}>
        <QuarterRing size={36} strokeWidth="3px" aria-label="正在加载资源" />
      </div>
      <div
        style={{
          marginTop: 8,
          fontSize: '1.1rem',
          fontWeight: 500,
          color: '#0f172a',
        }}
      >
        正在加载资源
      </div>
      <div style={{ marginTop: 12, fontSize: '0.875rem', color: '#64748b' }}>
        初次加载资源可能需要较多时间 请耐心等待
      </div>
    </div>
  );
}
