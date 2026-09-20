import type { MessageInstance } from 'antd/es/message/interface';
import { exchangeRateServiceDownloadExchangeRateImportTemplate } from '@/services/roncin/exchangeRateService';
import { getErrorMessage } from '@/utils/errorMessage';

/**
 * 下载汇率导入模板并触发浏览器保存，成功/失败提示统一在此维护。
 * loading 状态由调用组件持有；message 传 App.useApp() 实例以保持上下文语义。
 */
export async function downloadExchangeRateImportTemplate(
  message: MessageInstance,
): Promise<void> {
  try {
    const res = await exchangeRateServiceDownloadExchangeRateImportTemplate();
    const base64Data = res.content;
    if (!base64Data) {
      message.error('下载模板失败：文件内容为空');
      return;
    }
    const byteCharacters = atob(base64Data);
    const byteNumbers = new Array(byteCharacters.length);
    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i);
    }
    const byteArray = new Uint8Array(byteNumbers);
    const blob = new Blob([byteArray], {
      type:
        res.contentType ||
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = res.fileName || '汇率导入模板.xlsx';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    message.success('导入模板下载成功');
  } catch (e) {
    message.error(getErrorMessage(e, '下载导入模板失败'));
  }
}
