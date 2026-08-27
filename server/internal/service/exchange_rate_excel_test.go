package service

import (
	"bytes"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/xuri/excelize/v2"
)

func TestExchangeRateImportTemplateCanBeParsed(t *testing.T) {
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		t.Fatalf("生成汇率导入模板失败: %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("生成结果不是有效 xlsx: %v", err)
	}
	defer file.Close()
	if version, _ := file.GetCellValue(exchangeRateImportHelpSheet, "B1"); version != "1" {
		t.Fatalf("模板版本错误: %q", version)
	}
}

func TestParseExchangeRateImportWorkbook(t *testing.T) {
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		t.Fatalf("生成模板失败: %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("打开模板失败: %v", err)
	}
	values := []any{"账单汇率", "USD", "CNY", "7.20000000", "7.10000000", "2026-08-27 09:30:01", "2026-08-27 18:00:00"}
	for index, value := range values {
		cell, _ := excelize.CoordinatesToCellName(index+1, 2)
		if err = file.SetCellValue(exchangeRateImportSheet, cell, value); err != nil {
			t.Fatalf("写测试数据失败: %v", err)
		}
	}
	buffer, err := file.WriteToBuffer()
	_ = file.Close()
	if err != nil {
		t.Fatalf("输出测试工作簿失败: %v", err)
	}
	input, err := parseExchangeRateImportWorkbook("汇率.xlsx", buffer.Bytes())
	if err != nil {
		t.Fatalf("解析汇率工作簿失败: %v", err)
	}
	if input.TemplateVersion != biz.ExchangeRateImportTemplateVersion || len(input.Rows) != 1 || input.Rows[0].RowNumber != 2 || input.Rows[0].EffectiveFrom != "2026-08-27 09:30:01" {
		t.Fatalf("工作簿解析结果不正确: %#v", input)
	}
}

func TestParseExchangeRateImportWorkbookRejectsFormula(t *testing.T) {
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		t.Fatalf("生成模板失败: %v", err)
	}
	file, _ := excelize.OpenReader(bytes.NewReader(content))
	_ = file.SetCellFormula(exchangeRateImportSheet, "D2", "=1+1")
	_ = file.SetCellValue(exchangeRateImportSheet, "A2", "账单汇率")
	buffer, _ := file.WriteToBuffer()
	_ = file.Close()
	if _, err = parseExchangeRateImportWorkbook("汇率.xlsx", buffer.Bytes()); err != biz.ErrExchangeRateImportFileInvalid {
		t.Fatalf("含公式的汇率文件应被拒绝，实际错误为 %v", err)
	}
}

func TestParseExchangeRateImportWorkbookFormatsExcelDateTimeCells(t *testing.T) {
	content, _ := buildExchangeRateImportTemplate()
	file, _ := excelize.OpenReader(bytes.NewReader(content))
	values := []any{"账单汇率", "USD", "CNY", "7.2", "7.1"}
	for index, value := range values {
		cell, _ := excelize.CoordinatesToCellName(index+1, 2)
		_ = file.SetCellValue(exchangeRateImportSheet, cell, value)
	}
	_ = file.SetCellValue(exchangeRateImportSheet, "F2", time.Date(2026, 8, 27, 9, 30, 1, 0, time.Local))
	_ = file.SetCellValue(exchangeRateImportSheet, "G2", time.Date(2026, 8, 27, 18, 0, 0, 0, time.Local))
	buffer, _ := file.WriteToBuffer()
	_ = file.Close()
	input, err := parseExchangeRateImportWorkbook("汇率.xlsx", buffer.Bytes())
	if err != nil {
		t.Fatalf("Excel 日期时间单元格应能解析: %v", err)
	}
	if input.Rows[0].EffectiveFrom != "2026-08-27 09:30:01" || input.Rows[0].EffectiveTo == nil || *input.Rows[0].EffectiveTo != "2026-08-27 18:00:00" {
		t.Fatalf("Excel 日期时间格式化错误: %#v", input.Rows[0])
	}
}
