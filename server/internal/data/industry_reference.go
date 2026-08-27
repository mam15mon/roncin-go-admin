package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/shippinglinecontainerprefix"
)

type industryReferenceRepo struct{ data *Data }

func NewIndustryReferenceRepo(data *Data) biz.IndustryReferenceRepo {
	return &industryReferenceRepo{data: data}
}

func (r *industryReferenceRepo) ListPorts(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.PortList, error) {
	query := r.data.db.Port.Query().Where(port.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(port.Or(port.UnLocodeContainsFold(options.Keyword), port.NameZhContainsFold(options.Keyword), port.NameEnContainsFold(options.Keyword), port.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(port.EnabledEQ(*options.Enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(port.BySortOrder(), port.ByUnLocode()).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Port, 0, len(items))
	for _, item := range items {
		result = append(result, portToBiz(item))
	}
	return &biz.PortList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *industryReferenceRepo) CreatePort(ctx context.Context, organizationID uuid.UUID, input *biz.Port) (*biz.Port, error) {
	created, err := r.data.db.Port.Create().SetOrganizationID(organizationID).SetUnLocode(input.UNLocode).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetTransportModes(input.TransportModes).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
	if err != nil {
		return nil, mapIndustryReferenceConstraint(err)
	}
	return portToBiz(created), nil
}

func (r *industryReferenceRepo) UpdatePort(ctx context.Context, organizationID, id uuid.UUID, input *biz.Port) (*biz.Port, error) {
	existing, err := r.data.db.Port.Query().Where(port.IDEQ(id), port.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapIndustryReferenceNotFound(err)
	}
	updated, err := existing.Update().SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetTransportModes(input.TransportModes).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled).Save(ctx)
	if err != nil {
		return nil, mapIndustryReferenceConstraint(err)
	}
	return portToBiz(updated), nil
}

func (r *industryReferenceRepo) ListAirports(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.AirportList, error) {
	query := r.data.db.Airport.Query().Where(airport.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(airport.Or(airport.IataCodeContainsFold(options.Keyword), airport.IcaoCodeContainsFold(options.Keyword), airport.NameZhContainsFold(options.Keyword), airport.NameEnContainsFold(options.Keyword), airport.CityNameZhContainsFold(options.Keyword), airport.CityNameEnContainsFold(options.Keyword), airport.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(airport.EnabledEQ(*options.Enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(airport.BySortOrder(), airport.ByIataCode()).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Airport, 0, len(items))
	for _, item := range items {
		result = append(result, airportToBiz(item))
	}
	return &biz.AirportList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *industryReferenceRepo) CreateAirport(ctx context.Context, organizationID uuid.UUID, input *biz.Airport) (*biz.Airport, error) {
	created, err := r.data.db.Airport.Create().SetOrganizationID(organizationID).SetIataCode(input.IATACode).SetNillableIcaoCode(input.ICAOCode).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCityNameZh(input.CityNameZH).SetNillableCityNameEn(input.CityNameEN).SetCountryCode(input.CountryCode).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
	if err != nil {
		return nil, mapIndustryReferenceConstraint(err)
	}
	return airportToBiz(created), nil
}

func (r *industryReferenceRepo) UpdateAirport(ctx context.Context, organizationID, id uuid.UUID, input *biz.Airport) (*biz.Airport, error) {
	existing, err := r.data.db.Airport.Query().Where(airport.IDEQ(id), airport.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapIndustryReferenceNotFound(err)
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
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, mapIndustryReferenceConstraint(err)
	}
	return airportToBiz(updated), nil
}

func (r *industryReferenceRepo) ListAirlines(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.AirlineList, error) {
	query := r.data.db.Airline.Query().Where(airline.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(airline.Or(airline.IataCodeContainsFold(options.Keyword), airline.IcaoCodeContainsFold(options.Keyword), airline.AwbPrefixContainsFold(options.Keyword), airline.NameZhContainsFold(options.Keyword), airline.NameEnContainsFold(options.Keyword), airline.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(airline.EnabledEQ(*options.Enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(airline.BySortOrder(), airline.ByIataCode()).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Airline, 0, len(items))
	for _, item := range items {
		result = append(result, airlineToBiz(item))
	}
	return &biz.AirlineList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *industryReferenceRepo) CreateAirline(ctx context.Context, organizationID uuid.UUID, input *biz.Airline) (*biz.Airline, error) {
	created, err := r.data.db.Airline.Create().SetOrganizationID(organizationID).SetIataCode(input.IATACode).SetNillableIcaoCode(input.ICAOCode).SetAwbPrefix(input.AWBPrefix).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetCargoOnly(input.CargoOnly).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
	if err != nil {
		return nil, mapIndustryReferenceConstraint(err)
	}
	return airlineToBiz(created), nil
}

func (r *industryReferenceRepo) UpdateAirline(ctx context.Context, organizationID, id uuid.UUID, input *biz.Airline) (*biz.Airline, error) {
	existing, err := r.data.db.Airline.Query().Where(airline.IDEQ(id), airline.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapIndustryReferenceNotFound(err)
	}
	update := existing.Update().SetAwbPrefix(input.AWBPrefix).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetCargoOnly(input.CargoOnly).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled)
	if input.ICAOCode == nil {
		update.ClearIcaoCode()
	} else {
		update.SetIcaoCode(*input.ICAOCode)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, mapIndustryReferenceConstraint(err)
	}
	return airlineToBiz(updated), nil
}

func (r *industryReferenceRepo) ListShippingLines(ctx context.Context, organizationID uuid.UUID, options biz.IndustryReferenceListOptions) (*biz.ShippingLineList, error) {
	query := r.data.db.ShippingLine.Query().Where(shippingline.OrganizationIDEQ(organizationID)).WithContainerPrefixes(func(query *ent.ShippingLineContainerPrefixQuery) { query.Order(shippinglinecontainerprefix.ByPrefix()) })
	if options.Keyword != "" {
		query.Where(shippingline.Or(shippingline.ScacCodeContainsFold(options.Keyword), shippingline.NameZhContainsFold(options.Keyword), shippingline.NameEnContainsFold(options.Keyword), shippingline.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(shippingline.EnabledEQ(*options.Enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(shippingline.BySortOrder(), shippingline.ByScacCode()).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.ShippingLine, 0, len(items))
	for _, item := range items {
		result = append(result, shippingLineToBiz(item))
	}
	return &biz.ShippingLineList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *industryReferenceRepo) CreateShippingLine(ctx context.Context, organizationID uuid.UUID, input *biz.ShippingLine) (*biz.ShippingLine, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	created, err := tx.ShippingLine.Create().SetOrganizationID(organizationID).SetScacCode(input.SCACCode).SetNameZh(input.NameZH).SetNameEn(input.NameEN).SetCountryCode(input.CountryCode).SetNillableTrackingURL(input.TrackingURL).SetNillableAlliance(input.Alliance).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapIndustryReferenceConstraint(err)
	}
	if err := replaceShippingLinePrefixes(ctx, tx, organizationID, created.ID, input.ContainerPrefixes); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findShippingLine(ctx, organizationID, created.ID)
}

func (r *industryReferenceRepo) UpdateShippingLine(ctx context.Context, organizationID, id uuid.UUID, input *biz.ShippingLine) (*biz.ShippingLine, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.ShippingLine.Query().Where(shippingline.IDEQ(id), shippingline.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapIndustryReferenceNotFound(err)
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
		_ = tx.Rollback()
		return nil, mapIndustryReferenceConstraint(err)
	}
	if _, err := tx.ShippingLineContainerPrefix.Delete().Where(shippinglinecontainerprefix.ShippingLineIDEQ(id)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := replaceShippingLinePrefixes(ctx, tx, organizationID, id, input.ContainerPrefixes); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findShippingLine(ctx, organizationID, id)
}

func replaceShippingLinePrefixes(ctx context.Context, tx *ent.Tx, organizationID, shippingLineID uuid.UUID, prefixes []string) error {
	for _, prefix := range prefixes {
		if _, err := tx.ShippingLineContainerPrefix.Create().SetOrganizationID(organizationID).SetShippingLineID(shippingLineID).SetPrefix(prefix).Save(ctx); err != nil {
			return mapIndustryReferenceConstraint(err)
		}
	}
	return nil
}

func (r *industryReferenceRepo) findShippingLine(ctx context.Context, organizationID, id uuid.UUID) (*biz.ShippingLine, error) {
	item, err := r.data.db.ShippingLine.Query().Where(shippingline.IDEQ(id), shippingline.OrganizationIDEQ(organizationID)).WithContainerPrefixes(func(query *ent.ShippingLineContainerPrefixQuery) { query.Order(shippinglinecontainerprefix.ByPrefix()) }).Only(ctx)
	if err != nil {
		return nil, mapIndustryReferenceNotFound(err)
	}
	return shippingLineToBiz(item), nil
}

func portToBiz(item *ent.Port) *biz.Port {
	return &biz.Port{ID: item.ID, OrganizationID: item.OrganizationID, UNLocode: item.UnLocode, NameZH: stringValue(item.NameZh), NameEN: item.NameEn, CountryCode: item.CountryCode, TransportModes: append([]string(nil), item.TransportModes...), Source: item.Source, SourceVersion: item.SourceVersion, SourceHash: item.SourceHash, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func airportToBiz(item *ent.Airport) *biz.Airport {
	return &biz.Airport{ID: item.ID, OrganizationID: item.OrganizationID, IATACode: item.IataCode, ICAOCode: item.IcaoCode, NameZH: stringValue(item.NameZh), NameEN: item.NameEn, CityNameZH: stringValue(item.CityNameZh), CityNameEN: item.CityNameEn, CountryCode: item.CountryCode, Source: item.Source, SourceVersion: item.SourceVersion, SourceHash: item.SourceHash, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func airlineToBiz(item *ent.Airline) *biz.Airline {
	return &biz.Airline{ID: item.ID, OrganizationID: item.OrganizationID, IATACode: item.IataCode, ICAOCode: item.IcaoCode, AWBPrefix: item.AwbPrefix, NameZH: item.NameZh, NameEN: item.NameEn, CountryCode: item.CountryCode, CargoOnly: item.CargoOnly, Source: item.Source, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func shippingLineToBiz(item *ent.ShippingLine) *biz.ShippingLine {
	result := &biz.ShippingLine{ID: item.ID, OrganizationID: item.OrganizationID, SCACCode: item.ScacCode, NameZH: item.NameZh, NameEN: item.NameEn, CountryCode: item.CountryCode, TrackingURL: item.TrackingURL, Alliance: item.Alliance, Source: item.Source, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	for _, prefix := range item.Edges.ContainerPrefixes {
		result.ContainerPrefixes = append(result.ContainerPrefixes, prefix.Prefix)
	}
	return result
}

func mapIndustryReferenceNotFound(err error) error {
	if ent.IsNotFound(err) {
		return biz.ErrIndustryReferenceNotFound
	}
	return err
}

func mapIndustryReferenceConstraint(err error) error {
	if ent.IsConstraintError(err) {
		return biz.ErrIndustryReferenceCodeExist
	}
	return err
}

var _ biz.IndustryReferenceRepo = (*industryReferenceRepo)(nil)
