package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/shippinglinecontainerprefix"
)

type industryReferenceRepo struct{ data *Data }

func NewIndustryReferenceRepo(data *Data) biz.IndustryReferenceRepo {
	return &industryReferenceRepo{data: data}
}

func (r *industryReferenceRepo) headquartersOrganizationID(ctx context.Context, organizationID uuid.UUID) (uuid.UUID, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return resolveHeadquartersOrganizationID(ctx, client.Organization, organizationID)
}

// writeIndustryLocalCodePrecedence 写入总部共享行的让位条件：本组织已存在同业务
// 代码的行时，总部行不再参与候选（本组织行优先，大小写不敏感比对）。去重下推到 SQL，
// 保证分页计数、排序与去重结果一致；表名与列名来自代码内固定清单，不接收外部输入。
func writeIndustryLocalCodePrecedence(builder *sql.Builder, selector *sql.Selector, table, codeColumn string, organizationID uuid.UUID) {
	builder.WriteString("NOT EXISTS (SELECT 1 FROM ")
	builder.WriteString(table)
	builder.WriteString(" AS local_precedence WHERE local_precedence.organization_id = ")
	builder.Arg(organizationID)
	builder.WriteString(" AND UPPER(local_precedence.")
	builder.WriteString(codeColumn)
	builder.WriteString(") = UPPER(")
	builder.Ident(selector.C(codeColumn))
	builder.WriteString("))")
}

// portOrganizationFilter 返回港口“本组织 + 总部共享”的组织过滤谓词；
// 调用组织即总部时退化为仅本组织，避免无意义的自让位子查询。
func portOrganizationFilter(organizationID, headquartersID uuid.UUID) predicate.Port {
	if organizationID == headquartersID {
		return port.OrganizationIDEQ(organizationID)
	}
	return port.Or(
		port.OrganizationIDEQ(organizationID),
		port.And(
			port.OrganizationIDEQ(headquartersID),
			func(selector *sql.Selector) {
				selector.Where(sql.P(func(builder *sql.Builder) {
					writeIndustryLocalCodePrecedence(builder, selector, "ports", port.FieldUnLocode, organizationID)
				}))
			},
		),
	)
}

// airportOrganizationFilter 返回机场“本组织 + 总部共享”的组织过滤谓词（按 IATA 代码去重）。
func airportOrganizationFilter(organizationID, headquartersID uuid.UUID) predicate.Airport {
	if organizationID == headquartersID {
		return airport.OrganizationIDEQ(organizationID)
	}
	return airport.Or(
		airport.OrganizationIDEQ(organizationID),
		airport.And(
			airport.OrganizationIDEQ(headquartersID),
			func(selector *sql.Selector) {
				selector.Where(sql.P(func(builder *sql.Builder) {
					writeIndustryLocalCodePrecedence(builder, selector, "airports", airport.FieldIataCode, organizationID)
				}))
			},
		),
	)
}

// airlineOrganizationFilter 返回航司“本组织 + 总部共享”的组织过滤谓词（按 IATA 代码去重）。
func airlineOrganizationFilter(organizationID, headquartersID uuid.UUID) predicate.Airline {
	if organizationID == headquartersID {
		return airline.OrganizationIDEQ(organizationID)
	}
	return airline.Or(
		airline.OrganizationIDEQ(organizationID),
		airline.And(
			airline.OrganizationIDEQ(headquartersID),
			func(selector *sql.Selector) {
				selector.Where(sql.P(func(builder *sql.Builder) {
					writeIndustryLocalCodePrecedence(builder, selector, "airlines", airline.FieldIataCode, organizationID)
				}))
			},
		),
	)
}

// shippingLineOrganizationFilter 返回船公司“本组织 + 总部共享”的组织过滤谓词（按 SCAC 代码去重）。
func shippingLineOrganizationFilter(organizationID, headquartersID uuid.UUID) predicate.ShippingLine {
	if organizationID == headquartersID {
		return shippingline.OrganizationIDEQ(organizationID)
	}
	return shippingline.Or(
		shippingline.OrganizationIDEQ(organizationID),
		shippingline.And(
			shippingline.OrganizationIDEQ(headquartersID),
			func(selector *sql.Selector) {
				selector.Where(sql.P(func(builder *sql.Builder) {
					writeIndustryLocalCodePrecedence(builder, selector, "shipping_lines", shippingline.FieldScacCode, organizationID)
				}))
			},
		),
	)
}

func (r *industryReferenceRepo) ListPorts(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.PortList, error) {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.Port.Query().Where(portOrganizationFilter(organizationID, headquartersID))
	if options.Keyword != "" {
		query.Where(port.Or(port.UnLocodeContainsFold(options.Keyword), port.NameZhContainsFold(options.Keyword), port.NameEnContainsFold(options.Keyword), port.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(port.EnabledEQ(*options.Enabled))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.Port, error) {
		return query.Order(port.BySortOrder(), port.ByUnLocode()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(portToBiz))
}

func (r *industryReferenceRepo) CreatePort(ctx context.Context, organizationID uuid.UUID, input *biz.Port, audit *biz.AuditEvent) (*biz.Port, error) {
	var created *ent.Port
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var err error
		created, err = tx.Port.Create().SetOrganizationID(organizationID).SetUnLocode(input.UNLocode).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetTransportModes(input.TransportModes).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		audit.Details["industry_reference.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return portToBiz(created), nil
}

func (r *industryReferenceRepo) UpdatePort(ctx context.Context, organizationID, id uuid.UUID, input *biz.Port, audit *biz.AuditEvent) (*biz.Port, error) {
	var updated *ent.Port
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.Port.Query().Where(port.IDEQ(id), port.OrganizationIDEQ(organizationID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrIndustryReferenceNotFound, nil)
		}
		updated, err = existing.Update().SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetTransportModes(input.TransportModes).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		audit.Details["standard_code"] = updated.UnLocode
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return portToBiz(updated), nil
}

func (r *industryReferenceRepo) ListAirports(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.AirportList, error) {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.Airport.Query().Where(airportOrganizationFilter(organizationID, headquartersID))
	if options.Keyword != "" {
		query.Where(airport.Or(airport.IataCodeContainsFold(options.Keyword), airport.IcaoCodeContainsFold(options.Keyword), airport.NameZhContainsFold(options.Keyword), airport.NameEnContainsFold(options.Keyword), airport.CityNameZhContainsFold(options.Keyword), airport.CityNameEnContainsFold(options.Keyword), airport.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(airport.EnabledEQ(*options.Enabled))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.Airport, error) {
		return query.Order(airport.BySortOrder(), airport.ByIataCode()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(airportToBiz))
}

func (r *industryReferenceRepo) CreateAirport(ctx context.Context, organizationID uuid.UUID, input *biz.Airport, audit *biz.AuditEvent) (*biz.Airport, error) {
	var created *ent.Airport
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var err error
		created, err = tx.Airport.Create().SetOrganizationID(organizationID).SetIataCode(input.IATACode).SetNillableIcaoCode(input.ICAOCode).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCityNameZh(input.CityNameZH).SetNillableCityNameEn(input.CityNameEN).SetCountryCode(input.CountryCode).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		audit.Details["industry_reference.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return airportToBiz(created), nil
}

func (r *industryReferenceRepo) UpdateAirport(ctx context.Context, organizationID, id uuid.UUID, input *biz.Airport, audit *biz.AuditEvent) (*biz.Airport, error) {
	var updated *ent.Airport
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.Airport.Query().Where(airport.IDEQ(id), airport.OrganizationIDEQ(organizationID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrIndustryReferenceNotFound, nil)
		}
		update := existing.Update().SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCityNameZh(input.CityNameZH).SetCountryCode(input.CountryCode).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled)
		if input.ICAOCode == nil {
			update.ClearIcaoCode()
		} else {
			update.SetIcaoCode(*input.ICAOCode)
		}
		if input.CityNameEN == nil {
			update.ClearCityNameEn()
		} else {
			update.SetCityNameEn(*input.CityNameEN)
		}
		updated, err = update.Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		audit.Details["standard_code"] = updated.IataCode
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return airportToBiz(updated), nil
}

func (r *industryReferenceRepo) ListAirlines(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.AirlineList, error) {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.Airline.Query().Where(airlineOrganizationFilter(organizationID, headquartersID))
	if options.Keyword != "" {
		query.Where(airline.Or(airline.IataCodeContainsFold(options.Keyword), airline.IcaoCodeContainsFold(options.Keyword), airline.AwbPrefixContainsFold(options.Keyword), airline.NameZhContainsFold(options.Keyword), airline.NameEnContainsFold(options.Keyword), airline.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(airline.EnabledEQ(*options.Enabled))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.Airline, error) {
		return query.Order(airline.BySortOrder(), airline.ByIataCode()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(airlineToBiz))
}

func (r *industryReferenceRepo) CreateAirline(ctx context.Context, organizationID uuid.UUID, input *biz.Airline, audit *biz.AuditEvent) (*biz.Airline, error) {
	var created *ent.Airline
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var err error
		create := tx.Airline.Create().
			SetOrganizationID(organizationID).
			SetIataCode(input.IATACode).
			SetNillableIcaoCode(input.ICAOCode).
			SetNillableAwbPrefix(optionalString(input.AWBPrefix)).
			SetNillableNameZh(optionalString(input.NameZH)).
			SetNameEn(input.NameEN).
			SetCountryCode(input.CountryCode).
			SetCargoOnly(input.CargoOnly).
			SetSource(input.Source).
			SetNillableSourceVersion(input.SourceVersion).
			SetNillableSourceHash(input.SourceHash).
			SetSortOrder(input.SortOrder).
			SetEnabled(true)
		created, err = create.Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		audit.Details["industry_reference.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return airlineToBiz(created), nil
}

func (r *industryReferenceRepo) UpdateAirline(ctx context.Context, organizationID, id uuid.UUID, input *biz.Airline, audit *biz.AuditEvent) (*biz.Airline, error) {
	var updated *ent.Airline
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.Airline.Query().Where(airline.IDEQ(id), airline.OrganizationIDEQ(organizationID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrIndustryReferenceNotFound, nil)
		}
		update := existing.Update().
			SetNameEn(input.NameEN).
			SetCountryCode(input.CountryCode).
			SetCargoOnly(input.CargoOnly).
			SetSource(input.Source).
			SetSortOrder(input.SortOrder).
			SetEnabled(input.Enabled)
		if input.AWBPrefix == "" {
			update.ClearAwbPrefix()
		} else {
			update.SetAwbPrefix(input.AWBPrefix)
		}
		if input.NameZH == "" {
			update.ClearNameZh()
		} else {
			update.SetNameZh(input.NameZH)
		}
		if input.ICAOCode == nil {
			update.ClearIcaoCode()
		} else {
			update.SetIcaoCode(*input.ICAOCode)
		}
		updated, err = update.Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		audit.Details["standard_code"] = updated.IataCode
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return airlineToBiz(updated), nil
}

func (r *industryReferenceRepo) ListShippingLines(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.ShippingLineList, error) {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.ShippingLine.Query().Where(shippingLineOrganizationFilter(organizationID, headquartersID)).WithContainerPrefixes(func(query *ent.ShippingLineContainerPrefixQuery) { query.Order(shippinglinecontainerprefix.ByPrefix()) })
	if options.Keyword != "" {
		query.Where(shippingline.Or(shippingline.ScacCodeContainsFold(options.Keyword), shippingline.NameZhContainsFold(options.Keyword), shippingline.NameEnContainsFold(options.Keyword), shippingline.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(shippingline.EnabledEQ(*options.Enabled))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.ShippingLine, error) {
		return query.Order(shippingline.BySortOrder(), shippingline.ByScacCode()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(shippingLineToBiz))
}

func (r *industryReferenceRepo) CreateShippingLine(ctx context.Context, organizationID uuid.UUID, input *biz.ShippingLine, audit *biz.AuditEvent) (*biz.ShippingLine, error) {
	var item *ent.ShippingLine
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		created, err := tx.ShippingLine.Create().SetOrganizationID(organizationID).SetScacCode(input.SCACCode).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetNillableTrackingURL(input.TrackingURL).SetNillableAlliance(input.Alliance).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		if err := replaceShippingLinePrefixes(ctx, tx, organizationID, created.ID, input.ContainerPrefixes); err != nil {
			return err
		}
		item, err = tx.ShippingLine.Query().Where(shippingline.IDEQ(created.ID), shippingline.OrganizationIDEQ(organizationID)).WithContainerPrefixes(func(query *ent.ShippingLineContainerPrefixQuery) { query.Order(shippinglinecontainerprefix.ByPrefix()) }).Only(ctx)
		if err != nil {
			return err
		}
		audit.Details["industry_reference.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return shippingLineToBiz(item), nil
}

func (r *industryReferenceRepo) UpdateShippingLine(ctx context.Context, organizationID, id uuid.UUID, input *biz.ShippingLine, audit *biz.AuditEvent) (*biz.ShippingLine, error) {
	var item *ent.ShippingLine
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.ShippingLine.Query().Where(shippingline.IDEQ(id), shippingline.OrganizationIDEQ(organizationID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrIndustryReferenceNotFound, nil)
		}
		update := existing.Update().SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled)
		if input.TrackingURL == nil {
			update.ClearTrackingURL()
		} else {
			update.SetTrackingURL(*input.TrackingURL)
		}
		if input.Alliance == nil {
			update.ClearAlliance()
		} else {
			update.SetAlliance(*input.Alliance)
		}
		if _, err := update.Save(ctx); err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
		if _, err := tx.ShippingLineContainerPrefix.Delete().Where(shippinglinecontainerprefix.ShippingLineIDEQ(id)).Exec(ctx); err != nil {
			return err
		}
		if err := replaceShippingLinePrefixes(ctx, tx, organizationID, id, input.ContainerPrefixes); err != nil {
			return err
		}
		item, err = tx.ShippingLine.Query().Where(shippingline.IDEQ(id), shippingline.OrganizationIDEQ(organizationID)).WithContainerPrefixes(func(query *ent.ShippingLineContainerPrefixQuery) { query.Order(shippinglinecontainerprefix.ByPrefix()) }).Only(ctx)
		if err != nil {
			return err
		}
		audit.Details["standard_code"] = item.ScacCode
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return shippingLineToBiz(item), nil
}

func replaceShippingLinePrefixes(ctx context.Context, tx *ent.Tx, organizationID, shippingLineID uuid.UUID, prefixes []string) error {
	for _, prefix := range prefixes {
		if _, err := tx.ShippingLineContainerPrefix.Create().SetOrganizationID(organizationID).SetShippingLineID(shippingLineID).SetPrefix(prefix).Save(ctx); err != nil {
			return mapEntError(err, nil, biz.ErrIndustryReferenceCodeExist)
		}
	}
	return nil
}

func portToBiz(item *ent.Port) *biz.Port {
	return &biz.Port{ID: item.ID, OrganizationID: item.OrganizationID, UNLocode: item.UnLocode, NameZH: stringValue(item.NameZh), NameEN: item.NameEn, CountryCode: item.CountryCode, TransportModes: append([]string(nil), item.TransportModes...), Source: item.Source, SourceVersion: item.SourceVersion, SourceHash: item.SourceHash, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func airportToBiz(item *ent.Airport) *biz.Airport {
	return &biz.Airport{ID: item.ID, OrganizationID: item.OrganizationID, IATACode: item.IataCode, ICAOCode: item.IcaoCode, NameZH: stringValue(item.NameZh), NameEN: item.NameEn, CityNameZH: stringValue(item.CityNameZh), CityNameEN: item.CityNameEn, CountryCode: item.CountryCode, Source: item.Source, SourceVersion: item.SourceVersion, SourceHash: item.SourceHash, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func airlineToBiz(item *ent.Airline) *biz.Airline {
	return &biz.Airline{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		IATACode:       item.IataCode,
		ICAOCode:       item.IcaoCode,
		AWBPrefix:      stringValue(item.AwbPrefix),
		NameZH:         stringValue(item.NameZh),
		NameEN:         item.NameEn,
		CountryCode:    item.CountryCode,
		CargoOnly:      item.CargoOnly,
		Source:         item.Source,
		SourceVersion:  item.SourceVersion,
		SourceHash:     item.SourceHash,
		SortOrder:      item.SortOrder,
		Enabled:        item.Enabled,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func shippingLineToBiz(item *ent.ShippingLine) *biz.ShippingLine {
	result := &biz.ShippingLine{ID: item.ID, OrganizationID: item.OrganizationID, SCACCode: item.ScacCode, NameZH: item.NameZh, NameEN: item.NameEn, CountryCode: item.CountryCode, TrackingURL: item.TrackingURL, Alliance: item.Alliance, Source: item.Source, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	for _, prefix := range item.Edges.ContainerPrefixes {
		result.ContainerPrefixes = append(result.ContainerPrefixes, prefix.Prefix)
	}
	return result
}

var _ biz.IndustryReferenceRepo = (*industryReferenceRepo)(nil)
