package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/xuri/excelize/v2"
)

const (
	exchangeRateImportSheet       = "汇率导入"
	exchangeRateImportHelpSheet   = "填写说明"
	exchangeRateImportMaxFileSize = 5 << 20
)

var exchangeRateImportHeaders = []string{"汇率类型", "原币", "本币", "应收汇率", "应付汇率", "生效开始时间", "生效结束时间"}

func buildExchangeRateImportTemplate() ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()
	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, exchangeRateImportSheet); err != nil {
		return nil, err
	}
	if _, err := file.NewSheet(exchangeRateImportHelpSheet); err != nil {
		return nil, err
	}
	for index, header := range exchangeRateImportHeaders {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		if err := file.SetCellValue(exchangeRateImportSheet, cell, header); err != nil {
			return nil, err
		}
	}
	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"1677FF"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	if err != nil {
		return nil, err
	}
	if err = file.SetCellStyle(exchangeRateImportSheet, "A1", "G1", headerStyle); err != nil {
		return nil, err
	}
	timeFormat := "yyyy-mm-dd hh:mm:ss"
	timeStyle, err := file.NewStyle(&excelize.Style{CustomNumFmt: &timeFormat})
	if err != nil {
		return nil, err
	}
	if err = file.SetCellStyle(exchangeRateImportSheet, "F2", "G501", timeStyle); err != nil {
		return nil, err
	}
	widths := map[string]float64{"A": 20, "B": 12, "C": 12, "D": 16, "E": 16, "F": 24, "G": 24}
	for column, width := range widths {
		if err = file.SetColWidth(exchangeRateImportSheet, column, column, width); err != nil {
			return nil, err
		}
	}
	if err = file.SetPanes(exchangeRateImportSheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return nil, err
	}
	help := [][]any{
		{"模板版本", biz.ExchangeRateImportTemplateVersion},
		{"填写规则", "所有时间精确到秒，格式为 YYYY-MM-DD HH:mm:ss，按 Asia/Shanghai 解释。"},
		{"汇率类型", "汇率（折本币）、账单汇率、开票汇率、结算汇率、核销汇率。"},
		{"生效区间", "左闭右开：[生效开始时间, 生效结束时间)；结束时间留空表示长期有效。"},
		{"导入策略", "严格整批导入：任一行错误、文件内重叠或与现有启用汇率重叠时均不能确认。"},
		{"最大行数", biz.ExchangeRateImportMaxRows},
	}
	for rowIndex, row := range help {
		for columnIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err = file.SetCellValue(exchangeRateImportHelpSheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	if err = file.SetColWidth(exchangeRateImportHelpSheet, "A", "A", 16); err != nil {
		return nil, err
	}
	if err = file.SetColWidth(exchangeRateImportHelpSheet, "B", "B", 100); err != nil {
		return nil, err
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func parseExchangeRateImportWorkbook(fileName string, content []byte) (biz.PreviewExchangeRateImportInput, error) {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" || strings.ToLower(filepath.Ext(fileName)) != ".xlsx" || len(content) == 0 || len(content) > exchangeRateImportMaxFileSize {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	file, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{UnzipSizeLimit: 32 << 20, UnzipXMLSizeLimit: 8 << 20})
	if err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	defer file.Close()
	versionText, err := file.GetCellValue(exchangeRateImportHelpSheet, "B1")
	if err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	version, err := strconv.Atoi(strings.TrimSpace(versionText))
	if err != nil || version != biz.ExchangeRateImportTemplateVersion {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	iterator, err := file.Rows(exchangeRateImportSheet)
	if err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	defer iterator.Close()
	rows := make([]*biz.ExchangeRateImportRow, 0)
	rowNumber := 0
	for iterator.Next() {
		rowNumber++
		columns, rowErr := iterator.Columns()
		if rowErr != nil {
			return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
		}
		if rowNumber == 1 {
			if !validExchangeRateImportHeaders(columns) {
				return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
			}
			continue
		}
		if exchangeRateImportRowBlank(columns) {
			continue
		}
		if len(rows) >= biz.ExchangeRateImportMaxRows {
			return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportTooManyRows
		}
		for columnIndex := 1; columnIndex <= len(exchangeRateImportHeaders); columnIndex++ {
			cell, _ := excelize.CoordinatesToCellName(columnIndex, rowNumber)
			formula, formulaErr := file.GetCellFormula(exchangeRateImportSheet, cell)
			if formulaErr != nil || formula != "" {
				return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
			}
		}
		values := make([]string, len(exchangeRateImportHeaders))
		for index := range values {
			if index < len(columns) {
				values[index] = strings.TrimSpace(columns[index])
			}
		}
		var effectiveTo *string
		if values[6] != "" {
			value := values[6]
			effectiveTo = &value
		}
		rows = append(rows, &biz.ExchangeRateImportRow{RowNumber: rowNumber, RateType: values[0], FromCurrency: values[1], ToCurrency: values[2], ReceivableRate: values[3], PayableRate: values[4], EffectiveFrom: values[5], EffectiveTo: effectiveTo, Errors: []string{}})
	}
	if err = iterator.Error(); err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	checksum := sha256.Sum256(content)
	return biz.PreviewExchangeRateImportInput{FileName: fileName, FileChecksum: hex.EncodeToString(checksum[:]), TemplateVersion: version, Rows: rows}, nil
}

func validExchangeRateImportHeaders(columns []string) bool {
	if len(columns) < len(exchangeRateImportHeaders) {
		return false
	}
	for index, expected := range exchangeRateImportHeaders {
		if strings.TrimSpace(columns[index]) != expected {
			return false
		}
	}
	return true
}

func exchangeRateImportRowBlank(columns []string) bool {
	for index := 0; index < len(columns) && index < len(exchangeRateImportHeaders); index++ {
		if strings.TrimSpace(columns[index]) != "" {
			return false
		}
	}
	return true
}
