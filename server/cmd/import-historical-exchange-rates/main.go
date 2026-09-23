package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

var currencyCols = []struct {
	Col  string
	Code string
}{
	{"C", "USD"},
	{"D", "GBP"},
	{"E", "EUR"},
	{"F", "CHF"},
	{"G", "AUD"},
	{"H", "SGD"},
	{"I", "HKD"},
	{"J", "THB"},
}

func main() {
	filePath := flag.String("file", "/tmp/dinotty/汇率更新表-26.3.2-汇率，每周一更新12：00之前更新(1)(1).xlsx", "历史汇率 Excel 文件路径")
	apply := flag.Bool("apply", false, "是否真正写入数据库（默认 false，仅执行解析校验与预览）")
	companyID := flag.String("company-id", "", "目标分公司 UUID，必填")
	flag.Parse()

	if err := run(*filePath, *apply, *companyID); err != nil {
		fmt.Fprintf(os.Stderr, "执行失败: %v\n", err)
		os.Exit(1)
	}
}

func run(filePath string, apply bool, companyIDText string) error {
	companyID, err := uuid.Parse(strings.TrimSpace(companyIDText))
	if err != nil || companyID == uuid.Nil {
		return fmt.Errorf("必须通过 -company-id 指定合法目标分公司 UUID")
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	file, err := excelize.OpenFile(absPath)
	if err != nil {
		return fmt.Errorf("打开 Excel 文件失败: %w", err)
	}
	defer file.Close()

	sheetName := "汇率更新表"
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("读取工作表 %s 失败: %w", sheetName, err)
	}

	shanghaiLoc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		shanghaiLoc = time.FixedZone("CST", 8*3600)
	}
	// Excel 序列基准日 1899-12-30
	baseDate := time.Date(1899, 12, 30, 0, 0, 0, 0, shanghaiLoc)

	batches := make([]*weekBatch, 0)
	colToIndex := func(col string) int {
		return int(col[0] - 'A')
	}

	for rIdx := 0; rIdx < len(rows); rIdx++ {
		row0 := rows[rIdx]
		if len(row0) == 0 {
			continue
		}
		cellA := strings.TrimSpace(row0[0])
		var rawDate time.Time
		if t, tErr := time.ParseInLocation("2006-01-02", cellA, shanghaiLoc); tErr == nil {
			rawDate = t
		} else if serial, sErr := strconv.Atoi(cellA); sErr == nil && serial > 40000 {
			rawDate = baseDate.AddDate(0, 0, serial)
		} else {
			continue
		}
		if rIdx+3 >= len(rows) {
			break
		}
		row3 := rows[rIdx+3]

		monday, sunday := biz.ExchangeRateWeekWindow(rawDate)
		mondayStr := monday.Format(time.RFC3339)
		sundayStr := sunday.Format(time.RFC3339)

		settings := make([]*biz.ExchangeRateSetting, 0)
		for _, item := range currencyCols {
			cIdx := colToIndex(item.Col)
			if cIdx >= len(row0) || cIdx >= len(row3) {
				continue
			}
			arStr := strings.TrimSpace(row0[cIdx])
			apStr := strings.TrimSpace(row3[cIdx])
			if arStr == "" || apStr == "" {
				continue
			}
			arDec, arErr := decimal.NewFromString(arStr)
			apDec, apErr := decimal.NewFromString(apStr)
			if arErr != nil || apErr != nil || !arDec.IsPositive() || !apDec.IsPositive() {
				continue
			}
			rateDec := arDec

			settings = append(settings, &biz.ExchangeRateSetting{
				ID:             uuid.Must(uuid.NewV7()),
				OrganizationID: &companyID,
				FromCurrency:   item.Code,
				ToCurrency:     "CNY",
				EffectiveFrom:  mondayStr,
				EffectiveTo:    &sundayStr,
				ARRate:         &arDec,
				APRate:         &apDec,
				Rate:           rateDec,
				Source:         "IMPORT",
			})
		}

		if len(settings) > 0 {
			batches = append(batches, &weekBatch{
				Monday:   monday,
				Sunday:   sunday,
				Settings: settings,
			})
		}
	}

	fmt.Printf("成功解析 Excel 历史汇率档案：%s\n", absPath)
	fmt.Printf("共识别出 %d 个周一开盘周区间，汇率记录总计 %d 条：\n", len(batches), countSettings(batches))
	for idx, b := range batches {
		currencies := make([]string, 0, len(b.Settings))
		for _, s := range b.Settings {
			currencies = append(currencies, s.FromCurrency)
		}
		fmt.Printf("  第 %2d 周 [%s ~ %s] 币种(%d个): %s\n",
			idx+1,
			b.Monday.Format("2006-01-02"),
			b.Sunday.Format("2006-01-02"),
			len(b.Settings),
			strings.Join(currencies, ", "),
		)
	}

	if !apply {
		fmt.Println("\n【预览模式】未指定 -apply 参数，未向数据库写入任何数据。如需正式入库，请执行：")
		fmt.Printf("pnpm run sync:exchange-rates-history -company-id %s\n", companyID)
		return nil
	}

	databaseSource := strings.TrimSpace(os.Getenv("DATABASE_SOURCE"))
	if databaseSource == "" {
		return fmt.Errorf("DATABASE_SOURCE 环境变量不能为空")
	}
	storage, cleanup, err := data.NewData(&conf.Data{Database: &conf.Data_Database{Driver: "postgres", Source: databaseSource}}, slog.Default())
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer cleanup()

	repo := data.NewExchangeRateRepo(storage)
	ctx := context.Background()
	rateContext, err := repo.ResolveContext(ctx, companyID)
	if err != nil || rateContext.OwnerOrganizationID != companyID || rateContext.BaseCurrency != "CNY" {
		return fmt.Errorf("目标组织必须是启用且本币为 CNY 的分公司: %w", biz.ErrExchangeRateOrganizationInvalid)
	}

	audit := &biz.AuditEvent{
		OrganizationID: &companyID,
		Action:         "finance.exchange_rate.import_history",
		Result:         "success",
		ResourceType:   "exchange_rate_setting",
		Details:        map[string]string{"source": "excel_legacy_history", "weeks": strconv.Itoa(len(batches))},
	}

	savedTotal := 0
	for _, b := range batches {
		saved, err := repo.UpsertWeeklyBatch(ctx, "IMPORT", b.Settings, audit)
		if err != nil {
			return fmt.Errorf("周 %s 汇率入库失败: %w", b.Monday.Format("2006-01-02"), err)
		}
		savedTotal += len(saved)
	}

	fmt.Printf("\n【入库成功】已为分公司 %s 幂等写入/更新 %d 条历史汇率行。\n", companyID, savedTotal)
	return nil
}

type weekBatch struct {
	Monday   time.Time
	Sunday   time.Time
	Settings []*biz.ExchangeRateSetting
}

func countSettings(batches []*weekBatch) int {
	total := 0
	for _, b := range batches {
		total += len(b.Settings)
	}
	return total
}
