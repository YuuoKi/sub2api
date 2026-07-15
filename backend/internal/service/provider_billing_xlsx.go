package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"

	// excelize is licensed under BSD-3-Clause:
	// https://github.com/xuri/excelize/v2/blob/master/LICENSE
	"github.com/xuri/excelize/v2"
)

func parseProviderBillingXLSX(header ProviderBillingImportHeader, raw []byte) ([]ProviderBillingNormalizedLine, error) {
	if err := inspectXLSXSecurity(raw); err != nil {
		return nil, err
	}

	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, ErrProviderBillingInvalidFile
	}
	if len(sheets) > providerBillingMaxSheets {
		return nil, ErrProviderBillingTooManySheets
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
	}
	if len(rows) < 2 {
		return nil, ErrProviderBillingInvalidFile
	}

	// Reject formula cells via typed cell inspection on used range.
	if err := rejectXLSXFormulas(f, sheets[0]); err != nil {
		return nil, err
	}

	for _, row := range rows {
		for _, cell := range row {
			if len(cell) > providerBillingMaxCellBytes {
				return nil, ErrProviderBillingCellTooLarge
			}
		}
	}

	col := mapCSVHeader(rows[0])
	required := []string{
		"external_line_id", "upstream_task_id", "model", "sku", "usage_quantity",
		"usage_unit", "net_amount", "tax_amount", "gross_amount", "currency", "occurred_at",
	}
	for _, key := range required {
		if _, ok := col[key]; !ok {
			return nil, fmt.Errorf("%w: missing column %s", ErrProviderBillingInvalidFile, key)
		}
	}

	lines := make([]ProviderBillingNormalizedLine, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if isCSVRowEmpty(row) {
			continue
		}
		line, err := normalizeProviderBillingRow(header, func(name string) string {
			idx := col[name]
			if idx < 0 || idx >= len(row) {
				return ""
			}
			return row[idx]
		})
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	if len(lines) > providerBillingMaxRows {
		return nil, ErrProviderBillingTooManyRows
	}
	return lines, nil
}

func inspectXLSXSecurity(raw []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
	}

	var uncompressed int64
	for _, file := range reader.File {
		name := strings.ToLower(file.Name)
		if strings.Contains(name, "externallink") || strings.Contains(name, "externalinks") {
			return ErrProviderBillingExternalLink
		}
		uncompressed += int64(file.UncompressedSize64)
		if uncompressed > providerBillingMaxUncompressedBytes {
			return ErrProviderBillingZipBomb
		}
		// Bound actual inflate cost as well.
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
		}
		written, copyErr := io.Copy(io.Discard, io.LimitReader(rc, providerBillingMaxUncompressedBytes+1))
		_ = rc.Close()
		if copyErr != nil {
			return fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, copyErr)
		}
		if written > providerBillingMaxUncompressedBytes {
			return ErrProviderBillingZipBomb
		}
	}
	return nil
}

func rejectXLSXFormulas(f *excelize.File, sheet string) error {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
	}
	for rIdx, row := range rows {
		for cIdx := range row {
			cell, err := excelize.CoordinatesToCellName(cIdx+1, rIdx+1)
			if err != nil {
				continue
			}
			formula, err := f.GetCellFormula(sheet, cell)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
			}
			if strings.TrimSpace(formula) != "" {
				return ErrProviderBillingFormulaCell
			}
			value, err := f.GetCellValue(sheet, cell)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
			}
			if strings.HasPrefix(strings.TrimSpace(value), "=") {
				return ErrProviderBillingFormulaCell
			}
		}
	}
	return nil
}
