/**
 * loading 占位
 * 解决首次加载时白屏的问题（采用 QuarterRing 四分之一圆弧加载器）
 */
(() => {
  const _root = document.querySelector('#root');
  if (_root && _root.innerHTML === '') {
    _root.innerHTML = `
      <style>
        html,
        body,
        #root {
          height: 100%;
          margin: 0;
          padding: 0;
        }
        #root {
          background-color: #f5f7fa;
        }

        .loading-title {
          font-size: 1.1rem;
          font-weight: 500;
          color: #0f172a;
          margin-top: 8px;
        }

        .loading-sub-title {
          margin-top: 12px;
          font-size: 0.875rem;
          color: #64748b;
        }

        .page-loading-warp {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 20px;
        }

        @keyframes loading-ui-quarter-ring-rotation {
          0% {
            transform: rotate(0deg);
          }
          100% {
            transform: rotate(360deg);
          }
        }

        .quarter-ring-loader {
          display: inline-block;
          width: 36px;
          height: 36px;
          border-radius: 50%;
          box-sizing: border-box;
          border: 3px solid transparent;
          border-top-color: #1677ff;
          animation: loading-ui-quarter-ring-rotation 1s linear infinite;
        }
      </style>

      <div style="
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        min-height: 362px;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
      ">
        <div class="page-loading-warp">
          <span class="quarter-ring-loader" role="status" aria-label="loading"></span>
        </div>
        <div class="loading-title">
          正在加载资源
        </div>
        <div class="loading-sub-title">
          初次加载资源可能需要较多时间 请耐心等待
        </div>
      </div>
    `;
  }
})();
