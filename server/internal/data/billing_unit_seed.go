package data

import (
	"context"
	"fmt"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
)

type billingUnitSeed struct {
	code, name            string
	sortOrder             int
	isContainer           bool
	quantityMustBeInteger bool
}

// defaultBillingUnitSeeds 与正式补缺迁移的目录保持一致，已有同码记录始终保留。
// 箱型沿用既有代码，HC/HQ 不合并；包装类型和通用托盘也分别表达业务含义。
var defaultBillingUnitSeeds = []billingUnitSeed{
	{"CONT", "箱", 10, true, true},
	{"BL", "票", 20, false, true},
	{"CBM", "立方米", 30, false, false},
	{"TON", "吨", 40, false, false},
	{"SET", "套", 50, false, true},
	{"DOC", "份", 60, false, true},
	{"DAY", "天", 70, false, false},
	{"HOUR", "小时", 80, false, false},
	{"KG", "公斤", 90, false, false},
	{"RT", "运费吨", 100, false, false},
	{"20GP", "20GP", 200, true, true},
	{"40GP", "40GP", 210, true, true},
	{"40HQ", "40HQ", 220, true, true},
	{"20HC", "20HC", 230, true, true},
	{"40HC", "40HC", 240, true, true},
	{"45HC", "45HC", 250, true, true},
	{"20RF", "20RF", 260, true, true},
	{"40RF", "40RF", 270, true, true},
	{"20OT", "20OT", 280, true, true},
	{"40OT", "40OT", 290, true, true},
	{"20FR", "20FR", 300, true, true},
	{"40FR", "40FR", 310, true, true},
	{"20TK", "20TK", 320, true, true},
	{"20HT", "20HT", 330, true, true},
	{"20RH", "20RH", 340, true, true},
	{"40RH", "40RH", 350, true, true},
	{"40PF", "40PF", 360, true, true},
	{"12GP", "12GP", 370, true, true},
	{"CHE", "车", 400, false, true},
	{"PCS", "件", 410, false, true},
	{"PKG", "包裹", 420, false, true},
	{"CTN", "纸箱", 430, false, true},
	{"PLT", "托盘", 440, false, true},
	{"TRIP", "趟", 450, false, true},
	{"KM", "公里", 460, false, false},
	{"WEEK", "周", 470, false, false},
	{"MONTH", "月", 480, false, false},
	{"TEU", "标准箱", 490, false, false},
}

// CreateDefaultBillingUnits 在调用方事务内补齐公共计费单位，不覆盖管理员配置。
// 正式环境由有序 SQL 迁移初始化，本入口供开发数据重复同步使用。
func CreateDefaultBillingUnits(ctx context.Context, tx *ent.Tx) (map[string]*ent.BillingUnit, error) {
	builders := make([]*ent.BillingUnitCreate, 0, len(defaultBillingUnitSeeds))
	codes := make([]string, 0, len(defaultBillingUnitSeeds))
	for _, unit := range defaultBillingUnitSeeds {
		codes = append(codes, unit.code)
		builders = append(builders, tx.BillingUnit.Create().
			SetCode(unit.code).SetName(unit.name).SetSortOrder(unit.sortOrder).
			SetIsContainerUnit(unit.isContainer).SetQuantityMustBeInteger(unit.quantityMustBeInteger).
			SetEnabled(true))
	}
	if err := tx.BillingUnit.CreateBulk(builders...).OnConflictColumns(billingunitent.FieldCode).DoNothing().Exec(ctx); err != nil {
		return nil, fmt.Errorf("补齐公共计费单位: %w", err)
	}
	units, err := tx.BillingUnit.Query().Where(billingunitent.CodeIn(codes...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取公共计费单位: %w", err)
	}
	result := make(map[string]*ent.BillingUnit, len(units))
	for _, unit := range units {
		result[unit.Code] = unit
	}
	return result, nil
}
