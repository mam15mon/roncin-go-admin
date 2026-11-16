import { Font } from '@react-pdf/renderer';

export const ORDER_PDF_FONT_FAMILY = 'Roncin HBL Sans SC';

let registered = false;

export function registerOrderPdfFont() {
  if (registered) return;
  Font.register({
    family: ORDER_PDF_FONT_FAMILY,
    src: '/fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf',
  });
  registered = true;
}
