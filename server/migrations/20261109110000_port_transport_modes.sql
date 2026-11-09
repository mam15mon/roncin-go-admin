-- 一次性修正已存在的非标准运输代码；运行时仍只接受 SEA / RAIL / ROAD。
-- 保留港口标识、业务引用、名称及维护来源，不合并不同 UN/LOCODE 的记录。
UPDATE ports
SET transport_modes = '["SEA"]'::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE transport_modes = '["maritime"]'::jsonb;
