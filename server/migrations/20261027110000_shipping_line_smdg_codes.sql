-- 对齐 Ent ShippingLine 的主代码约束：SMDG 三位字母数字码、SCAC 二至四位字母码。
ALTER TABLE "shipping_lines" DROP CONSTRAINT "shipping_lines_scac_code_check";
ALTER TABLE "shipping_lines" ADD CONSTRAINT "shipping_lines_scac_code_check"
    CHECK ("scac_code" ~ '^([A-Z]{2,4}|[A-Z0-9]{3})$');
