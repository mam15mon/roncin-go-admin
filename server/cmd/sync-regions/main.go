package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

const (
	mcaRegionsURL   = "https://dmfw.mca.gov.cn/xzqh/getList"
	mcaSource       = "MCA_DMFW"
	defaultTable    = "china_geo_names"
	requestInterval = 250 * time.Millisecond
	requestTimeout  = 20 * time.Second
	maxAttempts     = 6
)

var regionCodePattern = regexp.MustCompile(`^\d{12}$`)

type flexibleInt int

func (value *flexibleInt) UnmarshalJSON(data []byte) error {
	text := strings.Trim(string(data), `"`)
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return fmt.Errorf("行政区划层级 %q 不是整数: %w", text, err)
	}
	*value = flexibleInt(parsed)
	return nil
}

type regionNode struct {
	Code     string       `json:"code"`
	Name     string       `json:"name"`
	Level    flexibleInt  `json:"level"`
	Type     string       `json:"type"`
	Children []regionNode `json:"children"`
}

type mcaResponse struct {
	Data    *regionNode `json:"data"`
	Message string      `json:"message"`
}

type regionRow = data.AdministrativeRegionSyncRecord

type syncOptions struct {
	Apply         bool
	TableName     string
	SourceVersion string
}

func main() {
	options := parseOptions()
	ctx := context.Background()
	client := &http.Client{Timeout: requestTimeout}

	rows, err := fetchRegions(ctx, client, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "同步行政区划失败: %v\n", err)
		os.Exit(1)
	}
	provinces, cities, districts := countLevels(rows)
	fmt.Printf("已抓取行政区划：省级 %d，市级 %d，区县级 %d，共 %d 条\n", provinces, cities, districts, len(rows))
	if !options.Apply {
		fmt.Println("当前为预览模式；确认数据后使用 -apply 写入数据库")
		return
	}
	result, err := applyRegions(ctx, rows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "写入行政区划失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("行政区划写入完成：新增 %d，更新 %d，停用 %d，数据版本：%s\n", result.Created, result.Updated, result.Disabled, options.SourceVersion)
}

func parseOptions() syncOptions {
	apply := flag.Bool("apply", false, "将抓取结果写入数据库")
	tableName := flag.String("table-name", defaultTable, "民政部行政区划数据表名称")
	sourceVersion := flag.String("source-version", "", "数据版本标识，默认使用表名称")
	flag.Parse()

	name := strings.TrimSpace(*tableName)
	if name == "" {
		fmt.Fprintln(os.Stderr, "table-name 不能为空")
		os.Exit(2)
	}
	version := strings.TrimSpace(*sourceVersion)
	if version == "" {
		version = name
	}
	return syncOptions{Apply: *apply, TableName: name, SourceVersion: version}
}

func fetchRegions(ctx context.Context, client *http.Client, options syncOptions) ([]regionRow, error) {
	root, err := fetchTree(ctx, client, options.TableName, "", nil)
	if err != nil {
		return nil, err
	}
	rowsByCode := make(map[string]regionRow)
	for _, province := range root.Children {
		if int(province.Level) != 1 {
			continue
		}
		addRegion(rowsByCode, province, nil, options.SourceVersion)
		if !regionCodePattern.MatchString(strings.TrimSpace(province.Code)) {
			continue
		}
		maxLevel := 2
		tree, fetchErr := fetchTree(ctx, client, options.TableName, province.Code, &maxLevel)
		if fetchErr != nil {
			return nil, fmt.Errorf("抓取省级节点 %s 失败: %w", province.Code, fetchErr)
		}
		provinceCode := strings.TrimSpace(province.Code)
		for _, city := range tree.Children {
			if int(city.Level) != 2 {
				continue
			}
			addRegion(rowsByCode, city, &provinceCode, options.SourceVersion)
			cityCode := strings.TrimSpace(city.Code)
			if !regionCodePattern.MatchString(cityCode) {
				continue
			}
			for _, district := range city.Children {
				if int(district.Level) != 3 {
					continue
				}
				addRegion(rowsByCode, district, &cityCode, options.SourceVersion)
			}
		}
		time.Sleep(requestInterval)
	}
	if len(rowsByCode) == 0 {
		return nil, errors.New("民政部接口未返回有效的 12 位行政区划编码")
	}
	rows := make([]regionRow, 0, len(rowsByCode))
	for _, row := range rowsByCode {
		rows = append(rows, row)
	}
	return rows, nil
}

func addRegion(rows map[string]regionRow, node regionNode, parentCode *string, sourceVersion string) {
	code := strings.TrimSpace(node.Code)
	name := strings.TrimSpace(node.Name)
	level := int(node.Level)
	if !regionCodePattern.MatchString(code) || name == "" || level < 1 || level > 3 {
		return
	}
	regionType := strings.TrimSpace(node.Type)
	var regionTypePointer *string
	if regionType != "" {
		regionTypePointer = &regionType
	}
	rows[code] = regionRow{
		Code: code, Name: name, Level: level, ParentCode: parentCode,
		RegionType: regionTypePointer, SourceVersion: sourceVersion,
	}
}

func fetchTree(ctx context.Context, client *http.Client, tableName, code string, maxLevel *int) (*regionNode, error) {
	endpoint, err := url.Parse(mcaRegionsURL)
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("code", code)
	query.Set("type", "2")
	query.Set("name", "")
	query.Set("isCheck", "0")
	query.Set("tableName", tableName)
	if maxLevel != nil {
		query.Set("maxLevel", strconv.Itoa(*maxLevel))
	}
	endpoint.RawQuery = query.Encode()

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, retry, requestErr := requestTree(ctx, client, endpoint.String())
		if requestErr == nil {
			return result, nil
		}
		lastErr = requestErr
		if !retry || attempt == maxAttempts {
			break
		}
		time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	}
	return nil, lastErr
}

func requestTree(ctx context.Context, client *http.Client, endpoint string) (*regionNode, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126 Safari/537.36")
	response, err := client.Do(request)
	if err != nil {
		return nil, false, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return nil, false, err
	}
	if response.StatusCode != http.StatusOK {
		retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusServiceUnavailable
		return nil, retry, fmt.Errorf("民政部接口返回 HTTP %d", response.StatusCode)
	}
	payload := mcaResponse{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("解析民政部响应失败: %w", err)
	}
	if payload.Data == nil {
		retry := strings.Contains(payload.Message, "调用过于频繁")
		if payload.Message == "" {
			payload.Message = "响应缺少 data 节点"
		}
		return nil, retry, errors.New(payload.Message)
	}
	return payload.Data, false, nil
}

func applyRegions(ctx context.Context, rows []regionRow) (data.IndustryReferenceSyncResult, error) {
	source := strings.TrimSpace(os.Getenv("DATABASE_SOURCE"))
	if source == "" {
		return data.IndustryReferenceSyncResult{}, errors.New("DATABASE_SOURCE 不能为空")
	}
	storage, cleanup, err := data.NewData(&conf.Data{Database: &conf.Data_Database{Driver: "postgres", Source: source}}, slog.Default())
	if err != nil {
		return data.IndustryReferenceSyncResult{}, err
	}
	defer cleanup()
	return data.NewIndustryReferenceSyncStore(storage).ApplyAdministrativeRegions(ctx, mcaSource, rows)
}

func countLevels(rows []regionRow) (int, int, int) {
	var provinces, cities, districts int
	for _, row := range rows {
		switch row.Level {
		case 1:
			provinces++
		case 2:
			cities++
		case 3:
			districts++
		}
	}
	return provinces, cities, districts
}
