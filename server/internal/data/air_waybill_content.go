package data

import (
	"context"
	"encoding/json"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
)

func airContentJSON(v *biz.AirWaybillContent) json.RawMessage {
	if v == nil {
		v = &biz.AirWaybillContent{}
	}
	b, _ := json.Marshal(v)
	return b
}
func airContentFromJSON(raw json.RawMessage) *biz.AirWaybillContent {
	if len(raw) == 0 {
		return nil
	}
	var v biz.AirWaybillContent
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return &v
}

// 调用方已按 Order集合→MAWB 加锁。历史模式固定，活动成员限制单票直单。
func ensureAirModeCanAttach(ctx context.Context, tx *ent.Tx, m *ent.AirMasterWaybill, mode string) error {
	if mode != "DIRECT" && mode != "HOUSE" {
		return biz.ErrAirWaybillInvalid
	}
	links, e := tx.AirMasterWaybillOrderLink.Query().Where(airlink.MasterWaybillIDEQ(m.ID)).All(ctx)
	if e != nil {
		return e
	}
	for _, l := range links {
		if l.OrganizationID != m.OrganizationID || l.DocumentMode != mode {
			return biz.ErrAirWaybillConflict
		}
		if mode == "DIRECT" && l.Status == "ACTIVE" {
			return biz.ErrAirWaybillConflict
		}
	}
	return nil
}
