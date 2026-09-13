package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/cmd/internal/syncrunner"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

const (
	openFlightsSource     = "OPENFLIGHTS"
	defaultOpenFlightsURL = "https://raw.githubusercontent.com/jpatokal/openflights/master/data/airlines.dat"
	httpTimeout           = 30 * time.Second
)

var (
	iataPattern = regexp.MustCompile(`^[A-Z0-9]{2}$`)
	icaoPattern = regexp.MustCompile(`^[A-Z0-9]{3}$`)
)

type airlineParseSummary struct {
	RawRows         int
	ValidAirlines   int
	EnrichedChinese int
	EnrichedAWB     int
	CargoOnly       int
	SkippedInactive int
	SkippedInvalid  int
	SourceHash      string
}

func main() {
	syncrunner.Run(run)
}

func run(ctx context.Context) error {
	options := parseOptions()
	raw, sourceLabel, err := loadAirlineSource(ctx, options.Source)
	if err != nil {
		return fmt.Errorf("读取 OpenFlights 航司数据失败: %w", err)
	}
	hash := sha256.Sum256(raw)
	sourceHash := hex.EncodeToString(hash[:])

	rows, summary, err := parseAirlines(raw, sourceHash)
	if err != nil {
		return fmt.Errorf("解析 OpenFlights 航司数据失败: %w", err)
	}
	if options.Release == "" {
		options.Release = "OpenFlights"
	}

	store, cleanup, err := syncrunner.OpenStore()
	if err != nil {
		return fmt.Errorf("初始化航司同步存储失败: %w", err)
	}
	defer cleanup()

	conflicts, err := store.CheckAirlines(ctx, options.OrganizationCode, openFlightsSource, rows)
	if err != nil {
		return fmt.Errorf("检查航司同步冲突失败: %w", err)
	}

	printSummary(sourceLabel, options, summary, conflicts)
	if len(conflicts) > 0 {
		return fmt.Errorf("航司数据存在数据库冲突: %d 条", len(conflicts))
	}

	if !options.Apply {
		fmt.Println("当前为预览模式；确认统计后使用 -apply 写入数据库")
		return nil
	}

	result, err := store.ApplyAirlines(ctx, options.OrganizationCode, openFlightsSource, options.Release, summary.SourceHash, rows)
	if err != nil {
		return fmt.Errorf("写入航司数据失败: %w", err)
	}
	fmt.Printf("航司同步完成：新增 %d，更新 %d，停用 %d\n", result.Created, result.Updated, result.Disabled)
	return nil
}

func parseOptions() syncrunner.Options {
	apply := flag.Bool("apply", false, "将航司数据写入数据库")
	source := flag.String("source", "", "OpenFlights airlines.dat 路径或下载 URL")
	release := flag.String("release", "", "数据版本；默认 OpenFlights")
	organizationCode := flag.String("org-code", strings.TrimSpace(os.Getenv("BOOTSTRAP_ORGANIZATION_CODE")), "目标组织代码")
	flag.Parse()

	path := resolveSourcePath(strings.TrimSpace(*source))
	if path == "" {
		defaultPaths := []string{
			filepath.Join("seeds", "airlines.dat"),
			filepath.Join("server", "seeds", "airlines.dat"),
			filepath.Join("..", "seeds", "airlines.dat"),
			filepath.Join("..", ".cache", "master-data", "airlines.dat"),
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}
	code := strings.TrimSpace(*organizationCode)
	if code == "" {
		code = "HQ"
	}
	return syncrunner.Options{Apply: *apply, Source: path, Release: strings.TrimSpace(*release), OrganizationCode: code}
}

func resolveSourcePath(source string) string {
	if source == "" || filepath.IsAbs(source) || strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return source
	}
	initialDirectory := strings.TrimSpace(os.Getenv("INIT_CWD"))
	if initialDirectory == "" {
		return filepath.Clean(source)
	}
	return filepath.Clean(filepath.Join(initialDirectory, source))
}

func loadAirlineSource(ctx context.Context, source string) ([]byte, string, error) {
	if source != "" && !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		raw, err := os.ReadFile(source)
		return raw, source, err
	}
	url := source
	if url == "" {
		url = defaultOpenFlightsURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "roncin-go-admin-airline-sync/1.0")
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("下载 OpenFlights 返回 HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return raw, url, nil
}

type candidateAirline struct {
	data.AirlineSyncRecord
	preferredScore int
}

func parseAirlines(raw []byte, sourceHash string) ([]data.AirlineSyncRecord, airlineParseSummary, error) {
	summary := airlineParseSummary{SourceHash: sourceHash}
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	byIATA := make(map[string]candidateAirline)

	// 先注入已知重点航司（确保货运航司如 Cargolux, FedEx, UPS 等不受 OpenFlights Active=N 过滤影响）
	for iata, meta := range knownAirlines {
		nameZH := meta.NameZH
		awbPrefix := meta.AWBPrefix
		nameEN := meta.NameEN
		countryCode := meta.CountryCode
		var icaoPtr *string
		if meta.ICAOCode != "" {
			icao := meta.ICAOCode
			icaoPtr = &icao
		}
		byIATA[iata] = candidateAirline{
			AirlineSyncRecord: data.AirlineSyncRecord{
				IATACode:    iata,
				ICAOCode:    icaoPtr,
				AWBPrefix:   &awbPrefix,
				NameZH:      &nameZH,
				NameEN:      nameEN,
				CountryCode: countryCode,
				CargoOnly:   meta.CargoOnly,
				Enabled:     true,
			},
			preferredScore: 1000,
		}
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			summary.SkippedInvalid++
			continue
		}
		summary.RawRows++
		if len(record) < 8 {
			summary.SkippedInvalid++
			continue
		}

		// 字段：0:ID, 1:Name, 2:Alias, 3:IATA, 4:ICAO, 5:Callsign, 6:Country, 7:Active
		name := strings.TrimSpace(record[1])
		iata := strings.ToUpper(strings.TrimSpace(record[3]))
		icao := strings.ToUpper(strings.TrimSpace(record[4]))
		country := strings.TrimSpace(record[6])
		active := strings.ToUpper(strings.TrimSpace(record[7]))

		if active != "Y" {
			summary.SkippedInactive++
			continue
		}

		if !iataPattern.MatchString(iata) || iata == "--" {
			summary.SkippedInvalid++
			continue
		}

		countryCode := mapOpenFlightsCountry(country)
		if countryCode == "" {
			summary.SkippedInvalid++
			continue
		}

		var icaoPtr *string
		if icaoPattern.MatchString(icao) {
			icaoPtr = &icao
		}

		meta, isKnown := enrichAirlineInfo(iata)
		score := 0
		var nameZH *string
		var awbPrefix *string
		cargoOnly := false

		if isKnown {
			score = 500
			nameZH = &meta.NameZH
			awbPrefix = &meta.AWBPrefix
			cargoOnly = meta.CargoOnly
			if meta.NameEN != "" {
				name = meta.NameEN
			}
			if meta.CountryCode != "" {
				countryCode = meta.CountryCode
			}
			if meta.ICAOCode != "" {
				icaoVal := meta.ICAOCode
				icaoPtr = &icaoVal
			}
		}

		if icaoPtr != nil {
			score += 10
		}
		if !strings.Contains(strings.ToLower(name), "domestic") {
			score += 5
		}

		candidate := candidateAirline{
			AirlineSyncRecord: data.AirlineSyncRecord{
				IATACode:    iata,
				ICAOCode:    icaoPtr,
				AWBPrefix:   awbPrefix,
				NameZH:      nameZH,
				NameEN:      name,
				CountryCode: countryCode,
				CargoOnly:   cargoOnly,
				Enabled:     true,
			},
			preferredScore: score,
		}

		if existing, exists := byIATA[iata]; !exists || candidate.preferredScore > existing.preferredScore {
			byIATA[iata] = candidate
		}
	}

	// 解决 ICAO 重复冲突
	icaoOwner := make(map[string]string)
	// 先让评分高的航司认领 ICAO
	sortedIATAs := make([]string, 0, len(byIATA))
	for iata := range byIATA {
		sortedIATAs = append(sortedIATAs, iata)
	}
	sort.Slice(sortedIATAs, func(i, j int) bool {
		if byIATA[sortedIATAs[i]].preferredScore == byIATA[sortedIATAs[j]].preferredScore {
			return sortedIATAs[i] < sortedIATAs[j]
		}
		return byIATA[sortedIATAs[i]].preferredScore > byIATA[sortedIATAs[j]].preferredScore
	})

	for _, iata := range sortedIATAs {
		cand := byIATA[iata]
		if cand.ICAOCode != nil {
			icao := *cand.ICAOCode
			if owner, exists := icaoOwner[icao]; exists && owner != iata {
				// 该 ICAO 已被其它更优的航司认领，清除本航司的 ICAO 以免触发唯一索引冲突
				cand.ICAOCode = nil
				byIATA[iata] = cand
			} else {
				icaoOwner[icao] = iata
			}
		}
	}

	result := make([]data.AirlineSyncRecord, 0, len(byIATA))
	for _, iata := range sortedIATAs {
		rec := byIATA[iata].AirlineSyncRecord
		result = append(result, rec)
		summary.ValidAirlines++
		if rec.NameZH != nil {
			summary.EnrichedChinese++
		}
		if rec.AWBPrefix != nil {
			summary.EnrichedAWB++
		}
		if rec.CargoOnly {
			summary.CargoOnly++
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i].IATACode < result[j].IATACode })
	return result, summary, nil
}

func printSummary(sourceLabel string, options syncrunner.Options, summary airlineParseSummary, conflicts []data.IndustryReferenceSyncConflict) {
	fmt.Printf("航司数据源：%s\n", sourceLabel)
	fmt.Printf("组织：%s，版本：%s，SHA-256：%s\n", options.OrganizationCode, options.Release, summary.SourceHash)
	fmt.Printf("原始行 %d，有效活跃航司 %d（中文对照 %d，运单前缀 %d，全货机 %d），非活跃跳过 %d，无效跳过 %d\n",
		summary.RawRows, summary.ValidAirlines, summary.EnrichedChinese, summary.EnrichedAWB, summary.CargoOnly, summary.SkippedInactive, summary.SkippedInvalid)
	if len(conflicts) > 0 {
		fmt.Printf("发现 %d 条冲突：\n", len(conflicts))
		limit := len(conflicts)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("- [%s] %s\n", conflicts[i].Code, conflicts[i].Message)
		}
		if len(conflicts) > limit {
			fmt.Printf("- ... 其余 %d 条冲突省略\n", len(conflicts)-limit)
		}
	}
}
