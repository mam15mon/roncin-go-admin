package biz

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrIndustryReferenceNotFound  = errors.NotFound("INDUSTRY_REFERENCE_NOT_FOUND", "行业主数据不存在")
	ErrIndustryReferenceCodeExist = errors.Conflict("INDUSTRY_REFERENCE_CODE_EXISTS", "行业标准码已存在")
)

type IndustryReferenceListOptions struct {
	Page     int
	PageSize int
	Keyword  string
	Enabled  *bool
}

type Port struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UNLocode       string
	NameZH         string
	NameEN         string
	CountryCode    string
	TransportModes []string
	Source         string
	SourceVersion  *string
	SourceHash     *string
	SortOrder      int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PortList = PagedList[*Port]

type Airport struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	IATACode       string
	ICAOCode       *string
	NameZH         string
	NameEN         string
	CityNameZH     string
	CityNameEN     *string
	CountryCode    string
	Source         string
	SourceVersion  *string
	SourceHash     *string
	SortOrder      int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AirportList = PagedList[*Airport]

type Airline struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	IATACode       string
	ICAOCode       *string
	AWBPrefix      string
	NameZH         string
	NameEN         string
	CountryCode    string
	CargoOnly      bool
	Source         string
	SortOrder      int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AirlineList = PagedList[*Airline]

type ShippingLine struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	SCACCode          string
	NameZH            string
	NameEN            string
	CountryCode       string
	TrackingURL       *string
	Alliance          *string
	ContainerPrefixes []string
	Source            string
	SortOrder         int
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ShippingLineList = PagedList[*ShippingLine]

type IndustryReferenceRepo interface {
	ListPorts(context.Context, uuid.UUID, IndustryReferenceListOptions) (*PortList, error)
	CreatePort(context.Context, uuid.UUID, *Port, *AuditEvent) (*Port, error)
	UpdatePort(context.Context, uuid.UUID, uuid.UUID, *Port, *AuditEvent) (*Port, error)
	ListAirports(context.Context, uuid.UUID, IndustryReferenceListOptions) (*AirportList, error)
	CreateAirport(context.Context, uuid.UUID, *Airport, *AuditEvent) (*Airport, error)
	UpdateAirport(context.Context, uuid.UUID, uuid.UUID, *Airport, *AuditEvent) (*Airport, error)
	ListAirlines(context.Context, uuid.UUID, IndustryReferenceListOptions) (*AirlineList, error)
	CreateAirline(context.Context, uuid.UUID, *Airline, *AuditEvent) (*Airline, error)
	UpdateAirline(context.Context, uuid.UUID, uuid.UUID, *Airline, *AuditEvent) (*Airline, error)
	ListShippingLines(context.Context, uuid.UUID, IndustryReferenceListOptions) (*ShippingLineList, error)
	CreateShippingLine(context.Context, uuid.UUID, *ShippingLine, *AuditEvent) (*ShippingLine, error)
	UpdateShippingLine(context.Context, uuid.UUID, uuid.UUID, *ShippingLine, *AuditEvent) (*ShippingLine, error)
}

type IndustryReferenceUsecase struct {
	repo IndustryReferenceRepo
}

func NewIndustryReferenceUsecase(repo IndustryReferenceRepo) *IndustryReferenceUsecase {
	return &IndustryReferenceUsecase{repo: repo}
}

func (uc *IndustryReferenceUsecase) ListPorts(ctx context.Context, organizationID uuid.UUID, options IndustryReferenceListOptions) (*PortList, error) {
	if err := normalizeIndustryReferenceListOptions(organizationID, &options); err != nil {
		return nil, err
	}
	return uc.repo.ListPorts(ctx, organizationID, options)
}

func (uc *IndustryReferenceUsecase) CreatePort(ctx context.Context, organizationID, actorID uuid.UUID, input *Port) (*Port, error) {
	normalized, err := normalizePort(input, true)
	if err != nil || organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	created, err := uc.repo.CreatePort(ctx, organizationID, normalized, newIndustryReferenceAudit(organizationID, actorID, "port.create", uuid.Nil, normalized.UNLocode))
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *IndustryReferenceUsecase) UpdatePort(ctx context.Context, organizationID, actorID, id uuid.UUID, input *Port) (*Port, error) {
	normalized, err := normalizePort(input, false)
	if err != nil || organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	updated, err := uc.repo.UpdatePort(ctx, organizationID, id, normalized, newIndustryReferenceAudit(organizationID, actorID, "port.update", id, normalized.UNLocode))
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (uc *IndustryReferenceUsecase) ListAirports(ctx context.Context, organizationID uuid.UUID, options IndustryReferenceListOptions) (*AirportList, error) {
	if err := normalizeIndustryReferenceListOptions(organizationID, &options); err != nil {
		return nil, err
	}
	return uc.repo.ListAirports(ctx, organizationID, options)
}

func (uc *IndustryReferenceUsecase) CreateAirport(ctx context.Context, organizationID, actorID uuid.UUID, input *Airport) (*Airport, error) {
	normalized, err := normalizeAirport(input, true)
	if err != nil || organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	created, err := uc.repo.CreateAirport(ctx, organizationID, normalized, newIndustryReferenceAudit(organizationID, actorID, "airport.create", uuid.Nil, normalized.IATACode))
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *IndustryReferenceUsecase) UpdateAirport(ctx context.Context, organizationID, actorID, id uuid.UUID, input *Airport) (*Airport, error) {
	normalized, err := normalizeAirport(input, false)
	if err != nil || organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	updated, err := uc.repo.UpdateAirport(ctx, organizationID, id, normalized, newIndustryReferenceAudit(organizationID, actorID, "airport.update", id, normalized.IATACode))
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (uc *IndustryReferenceUsecase) ListAirlines(ctx context.Context, organizationID uuid.UUID, options IndustryReferenceListOptions) (*AirlineList, error) {
	if err := normalizeIndustryReferenceListOptions(organizationID, &options); err != nil {
		return nil, err
	}
	return uc.repo.ListAirlines(ctx, organizationID, options)
}

func (uc *IndustryReferenceUsecase) CreateAirline(ctx context.Context, organizationID, actorID uuid.UUID, input *Airline) (*Airline, error) {
	normalized, err := normalizeAirline(input, true)
	if err != nil || organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	created, err := uc.repo.CreateAirline(ctx, organizationID, normalized, newIndustryReferenceAudit(organizationID, actorID, "airline.create", uuid.Nil, normalized.IATACode))
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *IndustryReferenceUsecase) UpdateAirline(ctx context.Context, organizationID, actorID, id uuid.UUID, input *Airline) (*Airline, error) {
	normalized, err := normalizeAirline(input, false)
	if err != nil || organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	updated, err := uc.repo.UpdateAirline(ctx, organizationID, id, normalized, newIndustryReferenceAudit(organizationID, actorID, "airline.update", id, normalized.IATACode))
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (uc *IndustryReferenceUsecase) ListShippingLines(ctx context.Context, organizationID uuid.UUID, options IndustryReferenceListOptions) (*ShippingLineList, error) {
	if err := normalizeIndustryReferenceListOptions(organizationID, &options); err != nil {
		return nil, err
	}
	return uc.repo.ListShippingLines(ctx, organizationID, options)
}

func (uc *IndustryReferenceUsecase) CreateShippingLine(ctx context.Context, organizationID, actorID uuid.UUID, input *ShippingLine) (*ShippingLine, error) {
	normalized, err := normalizeShippingLine(input, true)
	if err != nil || organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	created, err := uc.repo.CreateShippingLine(ctx, organizationID, normalized, newIndustryReferenceAudit(organizationID, actorID, "shipping_line.create", uuid.Nil, normalized.SCACCode))
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *IndustryReferenceUsecase) UpdateShippingLine(ctx context.Context, organizationID, actorID, id uuid.UUID, input *ShippingLine) (*ShippingLine, error) {
	normalized, err := normalizeShippingLine(input, false)
	if err != nil || organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	updated, err := uc.repo.UpdateShippingLine(ctx, organizationID, id, normalized, newIndustryReferenceAudit(organizationID, actorID, "shipping_line.update", id, normalized.SCACCode))
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func newIndustryReferenceAudit(organizationID, actorID uuid.UUID, action string, id uuid.UUID, standardCode string) *AuditEvent {
	details := map[string]string{"standard_code": standardCode}
	if id != uuid.Nil {
		details["industry_reference.id"] = id.String()
	}
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", Details: details}
}

var (
	unLocodePattern        = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{3}$`)
	iataAirportPattern     = regexp.MustCompile(`^[A-Z]{3}$`)
	icaoAirportPattern     = regexp.MustCompile(`^[A-Z0-9]{4}$`)
	iataAirlinePattern     = regexp.MustCompile(`^[A-Z0-9]{2}$`)
	icaoAirlinePattern     = regexp.MustCompile(`^[A-Z0-9]{3}$`)
	awbPrefixCodePattern   = regexp.MustCompile(`^\d{3}$`)
	scacCodePattern        = regexp.MustCompile(`^[A-Z]{2,4}$`)
	containerPrefixPattern = regexp.MustCompile(`^[A-Z]{3}[UJZ]$`)
	countryCodePattern     = regexp.MustCompile(`^[A-Z]{2}$`)
)

func normalizeIndustryReferenceListOptions(organizationID uuid.UUID, options *IndustryReferenceListOptions) error {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return ErrMasterDataInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return nil
}

func normalizePort(input *Port, creating bool) (*Port, error) {
	if input == nil {
		return nil, ErrMasterDataInvalidArgument
	}
	output := *input
	output.UNLocode = strings.ToUpper(strings.TrimSpace(output.UNLocode))
	output.CountryCode = strings.ToUpper(strings.TrimSpace(output.CountryCode))
	if err := normalizeIndustryNames(&output.NameZH, &output.NameEN, &output.Source, output.SortOrder); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(output.TransportModes))
	output.TransportModes = make([]string, 0, len(input.TransportModes))
	for _, mode := range input.TransportModes {
		mode = strings.ToUpper(strings.TrimSpace(mode))
		if mode != "SEA" && mode != "RAIL" && mode != "ROAD" {
			return nil, ErrMasterDataInvalidArgument
		}
		if _, exists := seen[mode]; !exists {
			seen[mode] = struct{}{}
			output.TransportModes = append(output.TransportModes, mode)
		}
	}
	if _, hasSea := seen["SEA"]; !hasSea || !countryCodePattern.MatchString(output.CountryCode) || creating && !unLocodePattern.MatchString(output.UNLocode) || creating && !strings.HasPrefix(output.UNLocode, output.CountryCode) {
		return nil, ErrMasterDataInvalidArgument
	}
	return &output, nil
}

func normalizeAirport(input *Airport, creating bool) (*Airport, error) {
	if input == nil {
		return nil, ErrMasterDataInvalidArgument
	}
	output := *input
	output.IATACode = strings.ToUpper(strings.TrimSpace(output.IATACode))
	output.ICAOCode = normalizedUpperOptionalString(output.ICAOCode)
	output.CityNameZH = strings.TrimSpace(output.CityNameZH)
	output.CityNameEN = normalizedOptionalString(output.CityNameEN)
	output.CountryCode = strings.ToUpper(strings.TrimSpace(output.CountryCode))
	if err := normalizeIndustryNames(&output.NameZH, &output.NameEN, &output.Source, output.SortOrder); err != nil {
		return nil, err
	}
	if creating && !iataAirportPattern.MatchString(output.IATACode) || output.ICAOCode != nil && !icaoAirportPattern.MatchString(*output.ICAOCode) || output.CityNameZH == "" || utf8.RuneCountInString(output.CityNameZH) > 100 || optionalStringTooLong(output.CityNameEN, 100) || !countryCodePattern.MatchString(output.CountryCode) {
		return nil, ErrMasterDataInvalidArgument
	}
	return &output, nil
}

func normalizeAirline(input *Airline, creating bool) (*Airline, error) {
	if input == nil {
		return nil, ErrMasterDataInvalidArgument
	}
	output := *input
	output.IATACode = strings.ToUpper(strings.TrimSpace(output.IATACode))
	output.ICAOCode = normalizedUpperOptionalString(output.ICAOCode)
	output.AWBPrefix = strings.TrimSpace(output.AWBPrefix)
	output.CountryCode = strings.ToUpper(strings.TrimSpace(output.CountryCode))
	if err := normalizeIndustryNames(&output.NameZH, &output.NameEN, &output.Source, output.SortOrder); err != nil {
		return nil, err
	}
	if creating && !iataAirlinePattern.MatchString(output.IATACode) || output.ICAOCode != nil && !icaoAirlinePattern.MatchString(*output.ICAOCode) || !awbPrefixCodePattern.MatchString(output.AWBPrefix) || !countryCodePattern.MatchString(output.CountryCode) {
		return nil, ErrMasterDataInvalidArgument
	}
	return &output, nil
}

func normalizeShippingLine(input *ShippingLine, creating bool) (*ShippingLine, error) {
	if input == nil {
		return nil, ErrMasterDataInvalidArgument
	}
	output := *input
	output.SCACCode = strings.ToUpper(strings.TrimSpace(output.SCACCode))
	output.CountryCode = strings.ToUpper(strings.TrimSpace(output.CountryCode))
	output.TrackingURL = normalizedOptionalString(output.TrackingURL)
	output.Alliance = normalizedOptionalString(output.Alliance)
	if err := normalizeIndustryNames(&output.NameZH, &output.NameEN, &output.Source, output.SortOrder); err != nil {
		return nil, err
	}
	if creating && !scacCodePattern.MatchString(output.SCACCode) || !countryCodePattern.MatchString(output.CountryCode) || optionalStringTooLong(output.TrackingURL, 500) || optionalStringTooLong(output.Alliance, 100) {
		return nil, ErrMasterDataInvalidArgument
	}
	if output.TrackingURL != nil {
		parsed, err := url.ParseRequestURI(*output.TrackingURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, ErrMasterDataInvalidArgument
		}
	}
	seen := make(map[string]struct{}, len(output.ContainerPrefixes))
	output.ContainerPrefixes = make([]string, 0, len(input.ContainerPrefixes))
	for _, prefix := range input.ContainerPrefixes {
		prefix = strings.ToUpper(strings.TrimSpace(prefix))
		if !containerPrefixPattern.MatchString(prefix) {
			return nil, ErrMasterDataInvalidArgument
		}
		if _, exists := seen[prefix]; !exists {
			seen[prefix] = struct{}{}
			output.ContainerPrefixes = append(output.ContainerPrefixes, prefix)
		}
	}
	return &output, nil
}

func normalizeIndustryNames(nameZH, nameEN, source *string, sortOrder int) error {
	*nameZH = strings.TrimSpace(*nameZH)
	*nameEN = strings.TrimSpace(*nameEN)
	*source = strings.TrimSpace(*source)
	if *source == "" {
		*source = "manual"
	}
	if *nameZH == "" || *nameEN == "" || utf8.RuneCountInString(*nameZH) > 200 || utf8.RuneCountInString(*nameEN) > 200 || utf8.RuneCountInString(*source) > 100 || sortOrder < 0 {
		return ErrMasterDataInvalidArgument
	}
	return nil
}

var _ IndustryReferenceRepo = (IndustryReferenceRepo)(nil)
