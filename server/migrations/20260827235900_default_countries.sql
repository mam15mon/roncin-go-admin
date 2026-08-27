INSERT INTO "master_data_items" (
    "id", "created_at", "updated_at", "kind", "code", "name", "name_en",
    "attributes", "source", "sort_order", "enabled", "search_keywords", "organization_id"
)
SELECT
    gen_random_uuid(), NOW(), NOW(), 'country', defaults.code, defaults.name, defaults.name_en,
    jsonb_build_object('continent', defaults.continent, 'currency_code', defaults.currency_code),
    'system', defaults.sort_order, true,
    LOWER(defaults.code || ' ' || defaults.name || ' ' || defaults.name_en),
    organizations.id
FROM "organizations"
CROSS JOIN (
    VALUES
        -- 亚洲 (Asia)
        ('CN', '中国', 'China', 'Asia', 'CNY', 10),
        ('HK', '中国香港', 'Hong Kong', 'Asia', 'HKD', 20),
        ('MO', '中国澳门', 'Macao', 'Asia', 'MOP', 30),
        ('TW', '中国台湾', 'Taiwan', 'Asia', 'TWD', 40),
        ('JP', '日本', 'Japan', 'Asia', 'JPY', 50),
        ('KR', '韩国', 'South Korea', 'Asia', 'KRW', 60),
        ('SG', '新加坡', 'Singapore', 'Asia', 'SGD', 70),
        ('MY', '马来西亚', 'Malaysia', 'Asia', 'MYR', 80),
        ('TH', '泰国', 'Thailand', 'Asia', 'THB', 90),
        ('VN', '越南', 'Vietnam', 'Asia', 'VND', 100),
        ('ID', '印度尼西亚', 'Indonesia', 'Asia', 'IDR', 110),
        ('PH', '菲律宾', 'Philippines', 'Asia', 'PHP', 120),
        ('IN', '印度', 'India', 'Asia', 'INR', 130),
        ('PK', '巴基斯坦', 'Pakistan', 'Asia', 'PKR', 140),
        ('BD', '孟加拉国', 'Bangladesh', 'Asia', 'BDT', 150),
        ('AE', '阿联酋', 'United Arab Emirates', 'Asia', 'AED', 160),
        ('SA', '沙特阿拉伯', 'Saudi Arabia', 'Asia', 'SAR', 170),
        ('QA', '卡塔尔', 'Qatar', 'Asia', 'QAR', 180),
        ('TR', '土耳其', 'Turkey', 'Asia', 'TRY', 190),
        ('IL', '以色列', 'Israel', 'Asia', 'ILS', 200),
        ('KZ', '哈萨克斯坦', 'Kazakhstan', 'Asia', 'KZT', 210),
        ('UZ', '乌兹别克斯坦', 'Uzbekistan', 'Asia', 'UZS', 220),

        -- 欧洲 (Europe)
        ('DE', '德国', 'Germany', 'Europe', 'EUR', 230),
        ('GB', '英国', 'United Kingdom', 'Europe', 'GBP', 240),
        ('FR', '法国', 'France', 'Europe', 'EUR', 250),
        ('IT', '意大利', 'Italy', 'Europe', 'EUR', 260),
        ('NL', '荷兰', 'Netherlands', 'Europe', 'EUR', 270),
        ('BE', '比利时', 'Belgium', 'Europe', 'EUR', 280),
        ('ES', '西班牙', 'Spain', 'Europe', 'EUR', 290),
        ('PL', '波兰', 'Poland', 'Europe', 'PLN', 300),
        ('RU', '俄罗斯', 'Russia', 'Europe', 'RUB', 310),
        ('CH', '瑞士', 'Switzerland', 'Europe', 'CHF', 320),
        ('SE', '瑞典', 'Sweden', 'Europe', 'SEK', 330),
        ('NO', '挪威', 'Norway', 'Europe', 'NOK', 340),
        ('DK', '丹麦', 'Denmark', 'Europe', 'DKK', 350),
        ('FI', '芬兰', 'Finland', 'Europe', 'EUR', 360),
        ('AT', '奥地利', 'Austria', 'Europe', 'EUR', 370),
        ('CZ', '捷克', 'Czech Republic', 'Europe', 'CZK', 380),
        ('HU', '匈牙利', 'Hungary', 'Europe', 'HUF', 390),
        ('GR', '希腊', 'Greece', 'Europe', 'EUR', 400),
        ('PT', '葡萄牙', 'Portugal', 'Europe', 'EUR', 410),
        ('IE', '爱尔兰', 'Ireland', 'Europe', 'EUR', 420),

        -- 北美洲 (North America)
        ('US', '美国', 'United States', 'North America', 'USD', 430),
        ('CA', '加拿大', 'Canada', 'North America', 'CAD', 440),
        ('MX', '墨西哥', 'Mexico', 'North America', 'MXN', 450),
        ('PA', '巴拿马', 'Panama', 'North America', 'USD', 460),

        -- 南美洲 (South America)
        ('BR', '巴西', 'Brazil', 'South America', 'BRL', 470),
        ('CL', '智利', 'Chile', 'South America', 'CLP', 480),
        ('AR', '阿根廷', 'Argentina', 'South America', 'ARS', 490),
        ('PE', '秘鲁', 'Peru', 'South America', 'PEN', 500),
        ('CO', '哥伦比亚', 'Colombia', 'South America', 'COP', 510),

        -- 大洋洲 (Oceania)
        ('AU', '澳大利亚', 'Australia', 'Oceania', 'AUD', 520),
        ('NZ', '新西兰', 'New Zealand', 'Oceania', 'NZD', 530),

        -- 非洲 (Africa)
        ('EG', '埃及', 'Egypt', 'Africa', 'EGP', 540),
        ('ZA', '南非', 'South Africa', 'Africa', 'ZAR', 550),
        ('NG', '尼日利亚', 'Nigeria', 'Africa', 'NGN', 560),
        ('KE', '肯尼亚', 'Kenya', 'Africa', 'KES', 570),
        ('MA', '摩洛哥', 'Morocco', 'Africa', 'MAD', 580),
        ('GH', '加纳', 'Ghana', 'Africa', 'GHS', 590)
) AS defaults(code, name, name_en, continent, currency_code, sort_order)
ON CONFLICT ("organization_id", "kind", "code") DO NOTHING;
