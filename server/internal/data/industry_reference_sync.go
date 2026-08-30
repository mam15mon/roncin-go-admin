package data

import (
	"context"
	"fmt"
	"sort"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
)

type AirportSyncRecord struct {
	IATACode    string
	ICAOCode    *string
	NameEN      string
	CityNameEN  *string
	CountryCode string
	Enabled     bool
}

type PortSyncRecord struct {
	UNLocode       string
	NameEN         string
	CountryCode    string
	TransportModes []string
	Enabled        bool
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

func (s *IndustryReferenceSyncStore) CheckPorts(ctx context.Context, organizationCode, source string, rows []PortSyncRecord) ([]IndustryReferenceSyncConflict, error) {
	organizationID, err := s.organizationID(ctx, organizationCode)
	if err != nil {
		return nil, err
	}
	items, err := s.data.db.Port.Query().Where(port.OrganizationIDEQ(organizationID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有港口失败: %w", err)
	}
	return portSyncConflicts(items, source, rows), nil
}

func (s *IndustryReferenceSyncStore) ApplyAirports(ctx context.Context, organizationCode, source, sourceVersion, sourceHash string, rows []AirportSyncRecord) (IndustryReferenceSyncResult, error) {
	organizationID, err := s.organizationID(ctx, organizationCode)
	if err != nil {
		return IndustryReferenceSyncResult{}, err
	}
	result := IndustryReferenceSyncResult{}
	err = s.data.WithTx(ctx, func(tx *ent.Tx) error {
		items, queryErr := tx.Airport.Query().Where(airport.OrganizationIDEQ(organizationID)).All(ctx)
		if queryErr != nil {
			return fmt.Errorf("查询现有机场失败: %w", queryErr)
		}
		conflicts := airportSyncConflicts(items, source, rows)
		if len(conflicts) > 0 {
			return fmt.Errorf("机场同步存在 %d 条数据库冲突", len(conflicts))
		}
		if _, updateErr := tx.Airport.Update().Where(airport.OrganizationIDEQ(organizationID), airport.SourceEQ(source)).SetEnabled(false).ClearIcaoCode().Save(ctx); updateErr != nil {
			return fmt.Errorf("停用旧机场数据失败: %w", updateErr)
		}
		existingByCode := make(map[string]*ent.Airport, len(items))
		for _, item := range items {
			existingByCode[item.IataCode] = item
		}
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
				if _, updateErr := update.Save(ctx); updateErr != nil {
					return fmt.Errorf("更新机场 %s 失败: %w", row.IATACode, updateErr)
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
			if _, createErr := create.Save(ctx); createErr != nil {
				return fmt.Errorf("新增机场 %s 失败: %w", row.IATACode, createErr)
			}
			result.Created++
		}
		var countErr error
		result.Disabled, countErr = tx.Airport.Query().Where(airport.OrganizationIDEQ(organizationID), airport.SourceEQ(source), airport.EnabledEQ(false)).Count(ctx)
		if countErr != nil {
			return fmt.Errorf("统计停用机场失败: %w", countErr)
		}
		return nil
	})
	if err != nil {
		return IndustryReferenceSyncResult{}, err
	}
	return result, nil
}

func (s *IndustryReferenceSyncStore) ApplyPorts(ctx context.Context, organizationCode, source, sourceVersion, sourceHash string, rows []PortSyncRecord) (IndustryReferenceSyncResult, error) {
	organizationID, err := s.organizationID(ctx, organizationCode)
	if err != nil {
		return IndustryReferenceSyncResult{}, err
	}
	result := IndustryReferenceSyncResult{}
	err = s.data.WithTx(ctx, func(tx *ent.Tx) error {
		items, queryErr := tx.Port.Query().Where(port.OrganizationIDEQ(organizationID)).All(ctx)
		if queryErr != nil {
			return fmt.Errorf("查询现有港口失败: %w", queryErr)
		}
		conflicts := portSyncConflicts(items, source, rows)
		if len(conflicts) > 0 {
			return fmt.Errorf("港口同步存在 %d 条数据库冲突", len(conflicts))
		}
		if _, updateErr := tx.Port.Update().Where(port.OrganizationIDEQ(organizationID), port.SourceEQ(source)).SetEnabled(false).Save(ctx); updateErr != nil {
			return fmt.Errorf("停用旧港口数据失败: %w", updateErr)
		}
		existingByCode := make(map[string]*ent.Port, len(items))
		for _, item := range items {
			existingByCode[item.UnLocode] = item
		}
		for _, row := range rows {
			if existing := existingByCode[row.UNLocode]; existing != nil {
				if _, updateErr := tx.Port.UpdateOneID(existing.ID).
					SetNameEn(row.NameEN).
					SetCountryCode(row.CountryCode).
					SetTransportModes(row.TransportModes).
					SetSourceVersion(sourceVersion).
					SetSourceHash(sourceHash).
					SetEnabled(row.Enabled).
					Save(ctx); updateErr != nil {
					return fmt.Errorf("更新港口 %s 失败: %w", row.UNLocode, updateErr)
				}
				result.Updated++
				continue
			}
			if _, createErr := tx.Port.Create().
				SetOrganizationID(organizationID).
				SetUnLocode(row.UNLocode).
				SetNameEn(row.NameEN).
				SetCountryCode(row.CountryCode).
				SetTransportModes(row.TransportModes).
				SetSource(source).
				SetSourceVersion(sourceVersion).
				SetSourceHash(sourceHash).
				SetEnabled(row.Enabled).
				Save(ctx); createErr != nil {
				return fmt.Errorf("新增港口 %s 失败: %w", row.UNLocode, createErr)
			}
			result.Created++
		}
		var countErr error
		result.Disabled, countErr = tx.Port.Query().Where(port.OrganizationIDEQ(organizationID), port.SourceEQ(source), port.EnabledEQ(false)).Count(ctx)
		if countErr != nil {
			return fmt.Errorf("统计停用港口失败: %w", countErr)
		}
		return nil
	})
	if err != nil {
		return IndustryReferenceSyncResult{}, err
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

func portSyncConflicts(items []*ent.Port, source string, rows []PortSyncRecord) []IndustryReferenceSyncConflict {
	byCode := make(map[string]*ent.Port, len(items))
	for _, item := range items {
		byCode[item.UnLocode] = item
	}
	conflicts := make([]IndustryReferenceSyncConflict, 0)
	for _, row := range rows {
		if existing := byCode[row.UNLocode]; existing != nil && existing.Source != source {
			conflicts = append(conflicts, IndustryReferenceSyncConflict{Code: row.UNLocode, Message: fmt.Sprintf("UN/LOCODE 已由来源 %s 占用", existing.Source)})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Code < conflicts[j].Code })
	return conflicts
}
