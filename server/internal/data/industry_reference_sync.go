package data

import (
	"context"
	"fmt"
	"sort"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/administrativeregion"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/shippinglinecontainerprefix"
)

type AirlineSyncRecord struct {
	IATACode    string
	ICAOCode    *string
	AWBPrefix   *string
	NameZH      *string
	NameEN      string
	CountryCode string
	CargoOnly   bool
	Enabled     bool
}

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

type ShippingLineSyncRecord struct {
	SCACCode          string
	NameZH            string
	NameEN            string
	CountryCode       string
	TrackingURL       *string
	Alliance          *string
	ContainerPrefixes []string
	Enabled           bool
}

type AdministrativeRegionSyncRecord struct {
	Code          string
	Name          string
	Level         int
	ParentCode    *string
	RegionType    *string
	SourceVersion string
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

// IndustryReferenceSyncStore 行业参考大数据同步存储。A 型实体（航司/船司）直写全局行，
// B 型实体（港口/机场）直写基线行（organization_id IS NULL），不再解析目标组织。
type IndustryReferenceSyncStore struct{ data *Data }

func NewIndustryReferenceSyncStore(data *Data) *IndustryReferenceSyncStore {
	return &IndustryReferenceSyncStore{data: data}
}

func (s *IndustryReferenceSyncStore) CheckAirlines(ctx context.Context, source string, rows []AirlineSyncRecord) ([]IndustryReferenceSyncConflict, error) {
	items, err := s.data.db.Airline.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有航司失败: %w", err)
	}
	return airlineSyncConflicts(items, source, rows), nil
}

func (s *IndustryReferenceSyncStore) CheckAirports(ctx context.Context, source string, rows []AirportSyncRecord) ([]IndustryReferenceSyncConflict, error) {
	items, err := s.data.db.Airport.Query().Where(airport.OrganizationIDIsNil()).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有机场失败: %w", err)
	}
	return airportSyncConflicts(items, source, rows), nil
}

func (s *IndustryReferenceSyncStore) CheckPorts(ctx context.Context, source string, rows []PortSyncRecord) ([]IndustryReferenceSyncConflict, error) {
	items, err := s.data.db.Port.Query().Where(port.OrganizationIDIsNil()).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有港口失败: %w", err)
	}
	return portSyncConflicts(items, source, rows), nil
}

func (s *IndustryReferenceSyncStore) CheckShippingLines(ctx context.Context, source string, rows []ShippingLineSyncRecord) ([]IndustryReferenceSyncConflict, error) {
	items, err := s.data.db.ShippingLine.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有船公司失败: %w", err)
	}
	return shippingLineSyncConflicts(items, source, rows), nil
}

func (s *IndustryReferenceSyncStore) ApplyAirports(ctx context.Context, source, sourceVersion, sourceHash string, rows []AirportSyncRecord) (IndustryReferenceSyncResult, error) {
	result := IndustryReferenceSyncResult{}
	err := s.data.WithTx(ctx, func(tx *ent.Tx) error {
		items, queryErr := tx.Airport.Query().Where(airport.OrganizationIDIsNil()).All(ctx)
		if queryErr != nil {
			return fmt.Errorf("查询现有机场失败: %w", queryErr)
		}
		conflicts := airportSyncConflicts(items, source, rows)
		if len(conflicts) > 0 {
			return fmt.Errorf("机场同步存在 %d 条数据库冲突", len(conflicts))
		}
		if _, updateErr := tx.Airport.Update().Where(airport.OrganizationIDIsNil(), airport.SourceEQ(source)).SetEnabled(false).Save(ctx); updateErr != nil {
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
			// 行业参考大数据是客观事实，同步直写基线行（NULL）。
			create := tx.Airport.Create().
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
		result.Disabled, countErr = tx.Airport.Query().Where(airport.OrganizationIDIsNil(), airport.SourceEQ(source), airport.EnabledEQ(false)).Count(ctx)
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

func (s *IndustryReferenceSyncStore) ApplyAirlines(ctx context.Context, source, sourceVersion, sourceHash string, rows []AirlineSyncRecord) (IndustryReferenceSyncResult, error) {
	result := IndustryReferenceSyncResult{}
	err := s.data.WithTx(ctx, func(tx *ent.Tx) error {
		items, queryErr := tx.Airline.Query().All(ctx)
		if queryErr != nil {
			return fmt.Errorf("查询现有航司失败: %w", queryErr)
		}
		conflicts := airlineSyncConflicts(items, source, rows)
		if len(conflicts) > 0 {
			return fmt.Errorf("航司同步存在 %d 条数据库冲突", len(conflicts))
		}
		if _, updateErr := tx.Airline.Update().Where(airline.SourceEQ(source)).SetEnabled(false).Save(ctx); updateErr != nil {
			return fmt.Errorf("停用旧航司数据失败: %w", updateErr)
		}
		existingByCode := make(map[string]*ent.Airline, len(items))
		for _, item := range items {
			existingByCode[item.IataCode] = item
		}
		for _, row := range rows {
			if existing := existingByCode[row.IATACode]; existing != nil {
				update := tx.Airline.UpdateOneID(existing.ID).
					SetNameEn(row.NameEN).
					SetCountryCode(row.CountryCode).
					SetCargoOnly(row.CargoOnly).
					SetSourceVersion(sourceVersion).
					SetSourceHash(sourceHash).
					SetEnabled(row.Enabled)
				if row.ICAOCode == nil || *row.ICAOCode == "" {
					update.ClearIcaoCode()
				} else {
					update.SetIcaoCode(*row.ICAOCode)
				}
				if row.NameZH != nil && *row.NameZH != "" {
					update.SetNameZh(*row.NameZH)
				} else if existing.NameZh != nil && *existing.NameZh != "" {
					update.SetNameZh(*existing.NameZh)
				} else {
					update.ClearNameZh()
				}
				if row.AWBPrefix != nil && *row.AWBPrefix != "" {
					update.SetAwbPrefix(*row.AWBPrefix)
				} else {
					update.ClearAwbPrefix()
				}
				if _, updateErr := update.Save(ctx); updateErr != nil {
					return fmt.Errorf("更新航司 %s 失败: %w", row.IATACode, updateErr)
				}
				result.Updated++
				continue
			}
			create := tx.Airline.Create().
				SetIataCode(row.IATACode).
				SetNameEn(row.NameEN).
				SetCountryCode(row.CountryCode).
				SetCargoOnly(row.CargoOnly).
				SetSource(source).
				SetSourceVersion(sourceVersion).
				SetSourceHash(sourceHash).
				SetSortOrder(100).
				SetEnabled(row.Enabled)
			if row.ICAOCode != nil && *row.ICAOCode != "" {
				create.SetIcaoCode(*row.ICAOCode)
			}
			if row.NameZH != nil && *row.NameZH != "" {
				create.SetNameZh(*row.NameZH)
			}
			if row.AWBPrefix != nil && *row.AWBPrefix != "" {
				create.SetAwbPrefix(*row.AWBPrefix)
			}
			if _, createErr := create.Save(ctx); createErr != nil {
				return fmt.Errorf("新增航司 %s 失败: %w", row.IATACode, createErr)
			}
			result.Created++
		}
		var countErr error
		result.Disabled, countErr = tx.Airline.Query().Where(airline.SourceEQ(source), airline.EnabledEQ(false)).Count(ctx)
		if countErr != nil {
			return fmt.Errorf("统计停用航司失败: %w", countErr)
		}
		return nil
	})
	if err != nil {
		return IndustryReferenceSyncResult{}, err
	}
	return result, nil
}

func (s *IndustryReferenceSyncStore) ApplyPorts(ctx context.Context, source, sourceVersion, sourceHash string, rows []PortSyncRecord) (IndustryReferenceSyncResult, error) {
	result := IndustryReferenceSyncResult{}
	err := s.data.WithTx(ctx, func(tx *ent.Tx) error {
		items, queryErr := tx.Port.Query().Where(port.OrganizationIDIsNil()).All(ctx)
		if queryErr != nil {
			return fmt.Errorf("查询现有港口失败: %w", queryErr)
		}
		conflicts := portSyncConflicts(items, source, rows)
		if len(conflicts) > 0 {
			return fmt.Errorf("港口同步存在 %d 条数据库冲突", len(conflicts))
		}
		if _, updateErr := tx.Port.Update().Where(port.OrganizationIDIsNil(), port.SourceEQ(source)).SetEnabled(false).Save(ctx); updateErr != nil {
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
			// 行业参考大数据是客观事实，同步直写基线行（NULL）。
			if _, createErr := tx.Port.Create().
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
		result.Disabled, countErr = tx.Port.Query().Where(port.OrganizationIDIsNil(), port.SourceEQ(source), port.EnabledEQ(false)).Count(ctx)
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

func (s *IndustryReferenceSyncStore) ApplyShippingLines(ctx context.Context, source string, rows []ShippingLineSyncRecord) (IndustryReferenceSyncResult, error) {
	result := IndustryReferenceSyncResult{}
	err := s.data.WithTx(ctx, func(tx *ent.Tx) error {
		items, queryErr := tx.ShippingLine.Query().All(ctx)
		if queryErr != nil {
			return fmt.Errorf("查询现有船公司失败: %w", queryErr)
		}
		conflicts := shippingLineSyncConflicts(items, source, rows)
		if len(conflicts) > 0 {
			return fmt.Errorf("船公司同步存在 %d 条数据库冲突", len(conflicts))
		}
		if _, updateErr := tx.ShippingLine.Update().Where(shippingline.SourceEQ(source)).SetEnabled(false).Save(ctx); updateErr != nil {
			return fmt.Errorf("停用旧船公司数据失败: %w", updateErr)
		}
		existingByCode := make(map[string]*ent.ShippingLine, len(items))
		for _, item := range items {
			existingByCode[item.ScacCode] = item
		}
		for _, row := range rows {
			if existing := existingByCode[row.SCACCode]; existing != nil {
				update := tx.ShippingLine.UpdateOneID(existing.ID).
					SetNameZh(row.NameZH).
					SetNameEn(row.NameEN).
					SetCountryCode(row.CountryCode).
					SetEnabled(row.Enabled)
				if row.TrackingURL != nil && *row.TrackingURL != "" {
					update.SetTrackingURL(*row.TrackingURL)
				} else {
					update.ClearTrackingURL()
				}
				if row.Alliance != nil && *row.Alliance != "" {
					update.SetAlliance(*row.Alliance)
				} else {
					update.ClearAlliance()
				}
				if _, updateErr := update.Save(ctx); updateErr != nil {
					return fmt.Errorf("更新船公司 %s 失败: %w", row.SCACCode, updateErr)
				}
				if _, delErr := tx.ShippingLineContainerPrefix.Delete().Where(shippinglinecontainerprefix.ShippingLineIDEQ(existing.ID)).Exec(ctx); delErr != nil {
					return fmt.Errorf("清理船公司 %s 前缀失败: %w", row.SCACCode, delErr)
				}
				if err := replaceShippingLinePrefixes(ctx, tx, existing.ID, row.ContainerPrefixes); err != nil {
					return fmt.Errorf("更新船公司 %s 前缀失败: %w", row.SCACCode, err)
				}
				result.Updated++
				continue
			}
			create := tx.ShippingLine.Create().
				SetScacCode(row.SCACCode).
				SetNameZh(row.NameZH).
				SetNameEn(row.NameEN).
				SetCountryCode(row.CountryCode).
				SetSource(source).
				SetEnabled(row.Enabled).
				SetSortOrder(100)
			if row.TrackingURL != nil && *row.TrackingURL != "" {
				create.SetTrackingURL(*row.TrackingURL)
			}
			if row.Alliance != nil && *row.Alliance != "" {
				create.SetAlliance(*row.Alliance)
			}
			created, createErr := create.Save(ctx)
			if createErr != nil {
				return fmt.Errorf("新增船公司 %s 失败: %w", row.SCACCode, createErr)
			}
			if err := replaceShippingLinePrefixes(ctx, tx, created.ID, row.ContainerPrefixes); err != nil {
				return fmt.Errorf("新增船公司 %s 前缀失败: %w", row.SCACCode, err)
			}
			result.Created++
		}
		var countErr error
		result.Disabled, countErr = tx.ShippingLine.Query().Where(shippingline.SourceEQ(source), shippingline.EnabledEQ(false)).Count(ctx)
		if countErr != nil {
			return fmt.Errorf("统计停用船公司失败: %w", countErr)
		}
		return nil
	})
	if err != nil {
		return IndustryReferenceSyncResult{}, err
	}
	return result, nil
}

func (s *IndustryReferenceSyncStore) ApplyAdministrativeRegions(ctx context.Context, source string, rows []AdministrativeRegionSyncRecord) (IndustryReferenceSyncResult, error) {
	result := IndustryReferenceSyncResult{}
	err := s.data.WithTx(ctx, func(tx *ent.Tx) error {
		items, queryErr := tx.AdministrativeRegion.Query().All(ctx)
		if queryErr != nil {
			return fmt.Errorf("查询现有行政区划失败: %w", queryErr)
		}
		if _, updateErr := tx.AdministrativeRegion.Update().Where(administrativeregion.SourceEQ(source)).SetEnabled(false).Save(ctx); updateErr != nil {
			return fmt.Errorf("停用旧行政区划失败: %w", updateErr)
		}
		existingByCode := make(map[string]*ent.AdministrativeRegion, len(items))
		for _, item := range items {
			existingByCode[item.Code] = item
		}
		for _, row := range rows {
			if existing := existingByCode[row.Code]; existing != nil {
				update := tx.AdministrativeRegion.UpdateOneID(existing.ID).
					SetName(row.Name).
					SetLevel(row.Level).
					SetSource(source).
					SetSourceVersion(row.SourceVersion).
					SetEnabled(true)
				if row.ParentCode == nil {
					update.ClearParentCode()
				} else {
					update.SetParentCode(*row.ParentCode)
				}
				if row.RegionType == nil {
					update.ClearRegionType()
				} else {
					update.SetRegionType(*row.RegionType)
				}
				if _, updateErr := update.Save(ctx); updateErr != nil {
					return fmt.Errorf("更新行政区划 %s 失败: %w", row.Code, updateErr)
				}
				result.Updated++
				continue
			}
			if _, createErr := tx.AdministrativeRegion.Create().
				SetCode(row.Code).
				SetName(row.Name).
				SetLevel(row.Level).
				SetNillableParentCode(row.ParentCode).
				SetNillableRegionType(row.RegionType).
				SetSource(source).
				SetSourceVersion(row.SourceVersion).
				SetEnabled(true).
				Save(ctx); createErr != nil {
				return fmt.Errorf("新增行政区划 %s 失败: %w", row.Code, createErr)
			}
			result.Created++
		}
		var countErr error
		result.Disabled, countErr = tx.AdministrativeRegion.Query().Where(administrativeregion.SourceEQ(source), administrativeregion.EnabledEQ(false)).Count(ctx)
		if countErr != nil {
			return fmt.Errorf("统计停用行政区划失败: %w", countErr)
		}
		return nil
	})
	if err != nil {
		return IndustryReferenceSyncResult{}, err
	}
	return result, nil
}

func airlineSyncConflicts(items []*ent.Airline, source string, rows []AirlineSyncRecord) []IndustryReferenceSyncConflict {
	byCode := make(map[string]*ent.Airline, len(items))
	byICAO := make(map[string]*ent.Airline, len(items))
	byAWB := make(map[string]*ent.Airline, len(items))
	for _, item := range items {
		byCode[item.IataCode] = item
		if item.IcaoCode != nil {
			byICAO[*item.IcaoCode] = item
		}
		if item.AwbPrefix != nil {
			byAWB[*item.AwbPrefix] = item
		}
	}
	conflicts := make([]IndustryReferenceSyncConflict, 0)
	for _, row := range rows {
		if existing := byCode[row.IATACode]; existing != nil && existing.Source != source {
			conflicts = append(conflicts, IndustryReferenceSyncConflict{Code: row.IATACode, Message: fmt.Sprintf("IATA 已由来源 %s 占用", existing.Source)})
		}
		if row.ICAOCode != nil && *row.ICAOCode != "" {
			if existing := byICAO[*row.ICAOCode]; existing != nil && existing.IataCode != row.IATACode && existing.Source != source {
				conflicts = append(conflicts, IndustryReferenceSyncConflict{Code: row.IATACode, Message: fmt.Sprintf("ICAO %s 已被航司 %s 的来源 %s 占用", *row.ICAOCode, existing.IataCode, existing.Source)})
			}
		}
		if row.AWBPrefix != nil && *row.AWBPrefix != "" {
			if existing := byAWB[*row.AWBPrefix]; existing != nil && existing.IataCode != row.IATACode && existing.Source != source {
				conflicts = append(conflicts, IndustryReferenceSyncConflict{Code: row.IATACode, Message: fmt.Sprintf("AWB 前缀 %s 已被航司 %s 的来源 %s 占用", *row.AWBPrefix, existing.IataCode, existing.Source)})
			}
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

func shippingLineSyncConflicts(items []*ent.ShippingLine, source string, rows []ShippingLineSyncRecord) []IndustryReferenceSyncConflict {
	byCode := make(map[string]*ent.ShippingLine, len(items))
	for _, item := range items {
		byCode[item.ScacCode] = item
	}
	conflicts := make([]IndustryReferenceSyncConflict, 0)
	for _, row := range rows {
		if existing := byCode[row.SCACCode]; existing != nil && existing.Source != source {
			conflicts = append(conflicts, IndustryReferenceSyncConflict{Code: row.SCACCode, Message: fmt.Sprintf("SCAC 已由来源 %s 占用", existing.Source)})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Code < conflicts[j].Code })
	return conflicts
}
