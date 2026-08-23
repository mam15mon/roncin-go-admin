package data

import (
	"context"
	"fmt"
	"sort"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

type AirportSyncRecord struct {
	IATACode    string
	ICAOCode    *string
	NameEN      string
	CityNameEN  *string
	CountryCode string
	Enabled     bool
}

type IndustryReferenceSyncConflict struct {
	Code    string
	Message string
}

type IndustryReferenceSyncResult struct {
	Created  int
	Updated  int
	Disabled int
}

type IndustryReferenceSyncStore struct{ data *Data }

func NewIndustryReferenceSyncStore(data *Data) *IndustryReferenceSyncStore {
	return &IndustryReferenceSyncStore{data: data}
}

func (s *IndustryReferenceSyncStore) CheckAirports(ctx context.Context, organizationCode, source string, rows []AirportSyncRecord) ([]IndustryReferenceSyncConflict, error) {
	organizationID, err := s.organizationID(ctx, organizationCode)
	if err != nil {
		return nil, err
	}
	items, err := s.data.db.Airport.Query().Where(airport.OrganizationIDEQ(organizationID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有机场失败: %w", err)
	}
	return airportSyncConflicts(items, source, rows), nil
}

func (s *IndustryReferenceSyncStore) ApplyAirports(ctx context.Context, organizationCode, source, sourceVersion, sourceHash string, rows []AirportSyncRecord) (IndustryReferenceSyncResult, error) {
	organizationID, err := s.organizationID(ctx, organizationCode)
	if err != nil {
		return IndustryReferenceSyncResult{}, err
	}
	tx, err := s.data.db.Tx(ctx)
	if err != nil {
		return IndustryReferenceSyncResult{}, fmt.Errorf("开启机场同步事务失败: %w", err)
	}
	defer tx.Rollback()

	items, err := tx.Airport.Query().Where(airport.OrganizationIDEQ(organizationID)).All(ctx)
	if err != nil {
		return IndustryReferenceSyncResult{}, fmt.Errorf("查询现有机场失败: %w", err)
	}
	conflicts := airportSyncConflicts(items, source, rows)
	if len(conflicts) > 0 {
		return IndustryReferenceSyncResult{}, fmt.Errorf("机场同步存在 %d 条数据库冲突", len(conflicts))
	}

	if _, err := tx.Airport.Update().Where(airport.OrganizationIDEQ(organizationID), airport.SourceEQ(source)).SetEnabled(false).ClearIcaoCode().Save(ctx); err != nil {
		return IndustryReferenceSyncResult{}, fmt.Errorf("停用旧机场数据失败: %w", err)
	}
	existingByCode := make(map[string]*ent.Airport, len(items))
	for _, item := range items {
		existingByCode[item.IataCode] = item
	}
	result := IndustryReferenceSyncResult{}
	for _, row := range rows {
		if existing := existingByCode[row.IATACode]; existing != nil {
			update := tx.Airport.UpdateOneID(existing.ID).
				SetNameEn(row.NameEN).
				SetNillableCityNameEn(row.CityNameEN).
				SetCountryCode(row.CountryCode).
				SetSourceVersion(sourceVersion).
				SetSourceHash(sourceHash).
				SetEnabled(row.Enabled)
			if row.ICAOCode == nil {
				update.ClearIcaoCode()
			} else {
				update.SetIcaoCode(*row.ICAOCode)
			}
			if _, err := update.Save(ctx); err != nil {
				return IndustryReferenceSyncResult{}, fmt.Errorf("更新机场 %s 失败: %w", row.IATACode, err)
			}
			result.Updated++
			continue
		}
		create := tx.Airport.Create().
			SetOrganizationID(organizationID).
			SetIataCode(row.IATACode).
			SetNillableIcaoCode(row.ICAOCode).
			SetNameEn(row.NameEN).
			SetNillableCityNameEn(row.CityNameEN).
			SetCountryCode(row.CountryCode).
			SetSource(source).
			SetSourceVersion(sourceVersion).
			SetSourceHash(sourceHash).
			SetEnabled(row.Enabled)
		if _, err := create.Save(ctx); err != nil {
			return IndustryReferenceSyncResult{}, fmt.Errorf("新增机场 %s 失败: %w", row.IATACode, err)
		}
		result.Created++
	}
	result.Disabled, err = tx.Airport.Query().Where(airport.OrganizationIDEQ(organizationID), airport.SourceEQ(source), airport.EnabledEQ(false)).Count(ctx)
	if err != nil {
		return IndustryReferenceSyncResult{}, fmt.Errorf("统计停用机场失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return IndustryReferenceSyncResult{}, fmt.Errorf("提交机场同步事务失败: %w", err)
	}
	return result, nil
}

func (s *IndustryReferenceSyncStore) organizationID(ctx context.Context, code string) ([16]byte, error) {
	item, err := s.data.db.Organization.Query().Where(organization.CodeEQ(code), organization.EnabledEQ(true)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return [16]byte{}, fmt.Errorf("未找到启用的目标组织 %q", code)
		}
		return [16]byte{}, fmt.Errorf("查询目标组织失败: %w", err)
	}
	return item.ID, nil
}

func airportSyncConflicts(items []*ent.Airport, source string, rows []AirportSyncRecord) []IndustryReferenceSyncConflict {
	byCode := make(map[string]*ent.Airport, len(items))
	byICAO := make(map[string]*ent.Airport, len(items))
	for _, item := range items {
		byCode[item.IataCode] = item
		if item.IcaoCode != nil {
			byICAO[*item.IcaoCode] = item
		}
	}
	conflicts := make([]IndustryReferenceSyncConflict, 0)
	for _, row := range rows {
		if existing := byCode[row.IATACode]; existing != nil && existing.Source != source {
			conflicts = append(conflicts, IndustryReferenceSyncConflict{Code: row.IATACode, Message: fmt.Sprintf("IATA 已由来源 %s 占用", existing.Source)})
		}
		if row.ICAOCode == nil {
			continue
		}
		if existing := byICAO[*row.ICAOCode]; existing != nil && existing.IataCode != row.IATACode && existing.Source != source {
			conflicts = append(conflicts, IndustryReferenceSyncConflict{Code: row.IATACode, Message: fmt.Sprintf("ICAO %s 已被机场 %s 的来源 %s 占用", *row.ICAOCode, existing.IataCode, existing.Source)})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Code == conflicts[j].Code {
			return conflicts[i].Message < conflicts[j].Message
		}
		return conflicts[i].Code < conflicts[j].Code
	})
	return conflicts
}
