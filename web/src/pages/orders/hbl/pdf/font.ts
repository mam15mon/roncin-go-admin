import { Font } from '@react-pdf/renderer';

export const HBL_PDF_FONT_FAMILY = 'Roncin HBL Sans SC';

let registered = false;

export function registerHblPdfFont() {
  if (registered) return;
  Font.register({
    family: HBL_PDF_FONT_FAMILY,
    src: '/fonts/roncin-hbl/RoncinHBLSansSC-Regular.ttf',
  });
  registered = true;
}
