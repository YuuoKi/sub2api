package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	// excelize is BSD-3-Clause licensed (see github.com/xuri/excelize/v2 LICENSE).
	"github.com/xuri/excelize/v2"
)

func TestProviderBillingParseXLSX_RejectsFormulaHiddenLinksAndZipBomb(t *testing.T) {
	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "xlsx",
	}

	formulaRaw := mustBuildBillingXLSX(t, map[string]string{
		"A1": "external_line_id", "B1": "upstream_task_id", "C1": "model", "D1": "sku",
		"E1": "usage_quantity", "F1": "usage_unit", "G1": "net_amount", "H1": "tax_amount",
		"I1": "gross_amount", "J1": "currency", "K1": "occurred_at",
		"A2": "x1", "E2": "1", "F2": "u", "G2": "=1+1", "H2": "0", "I2": "2", "J2": "USD", "K2": "2026-07-01T00:00:00Z",
	}, false)
	_, err := ParseProviderBillingFile(header, "formula.xlsx", formulaRaw)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingFormulaCell)

	linkedRaw := mustBuildBillingXLSX(t, map[string]string{
		"A1": "external_line_id", "B1": "upstream_task_id", "C1": "model", "D1": "sku",
		"E1": "usage_quantity", "F1": "usage_unit", "G1": "net_amount", "H1": "tax_amount",
		"I1": "gross_amount", "J1": "currency", "K1": "occurred_at",
		"A2": "x1", "E2": "1", "F2": "u", "G2": "1.00", "H2": "0", "I2": "1.00", "J2": "USD", "K2": "2026-07-01T00:00:00Z",
	}, true)
	_, err = ParseProviderBillingFile(header, "linked.xlsx", linkedRaw)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingExternalLink)

	bomb := buildZipBombXLSX(t)
	_, err = ParseProviderBillingFile(header, "bomb.xlsx", bomb)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingZipBomb)

	tooManySheets := mustBuildMultiSheetXLSX(t, providerBillingMaxSheets+1)
	_, err = ParseProviderBillingFile(header, "sheets.xlsx", tooManySheets)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingTooManySheets)
}

func TestProviderBillingParseXLSX_HappyPath(t *testing.T) {
	header := ProviderBillingImportHeader{
		Provider:           "gemini",
		ProviderAccountID:  "acct-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "xlsx",
		InvoiceNumber:      "INV-1",
	}
	raw := mustBuildBillingXLSX(t, map[string]string{
		"A1": "external_line_id", "B1": "upstream_task_id", "C1": "model", "D1": "sku",
		"E1": "usage_quantity", "F1": "usage_unit", "G1": "net_amount", "H1": "tax_amount",
		"I1": "gross_amount", "J1": "currency", "K1": "occurred_at",
		"A2": "xlsx-1", "B2": "op-1", "C2": "gemini-x", "D2": "image",
		"E2": "1", "F2": "image", "G2": "0.04", "H2": "0", "I2": "0.04", "J2": "USD", "K2": "2026-07-01T00:00:00Z",
	}, false)

	parsed, err := ParseProviderBillingFile(header, "ok.xlsx", raw)
	require.NoError(t, err)
	require.Len(t, parsed.Lines, 1)
	require.Equal(t, "xlsx-1", parsed.Lines[0].ExternalLineID)
	require.Equal(t, "0.04", parsed.Lines[0].NetAmount.String())
}

func mustBuildBillingXLSX(t *testing.T, cells map[string]string, withExternalLink bool) []byte {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for addr, value := range cells {
		if strings.HasPrefix(strings.TrimSpace(value), "=") {
			require.NoError(t, f.SetCellFormula(sheet, addr, strings.TrimPrefix(strings.TrimSpace(value), "=")))
			continue
		}
		require.NoError(t, f.SetCellValue(sheet, addr, value))
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	if !withExternalLink {
		return buf.Bytes()
	}
	return injectExternalLinkIntoXLSX(t, buf.Bytes(), "http://evil.example/payload")
}

func mustBuildMultiSheetXLSX(t *testing.T, sheets int) []byte {
	t.Helper()
	f := excelize.NewFile()
	for i := 1; i < sheets; i++ {
		name := fmt.Sprintf("ExtraSheet%d", i)
		_, err := f.NewSheet(name)
		require.NoError(t, err)
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	return buf.Bytes()
}

func injectExternalLinkIntoXLSX(t *testing.T, raw []byte, target string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	require.NoError(t, err)

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, file := range reader.File {
		w, err := writer.Create(file.Name)
		require.NoError(t, err)
		rc, err := file.Open()
		require.NoError(t, err)
		_, err = io.Copy(w, rc)
		_ = rc.Close()
		require.NoError(t, err)
	}
	w, err := writer.Create("xl/externalLinks/externalLink1.xml")
	require.NoError(t, err)
	_, err = w.Write([]byte(`<?xml version="1.0"?><externalLink xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><externalBook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:id="rId1"/></externalLink>`))
	require.NoError(t, err)
	w, err = writer.Create("xl/externalLinks/_rels/externalLink1.xml.rels")
	require.NoError(t, err)
	_, err = w.Write([]byte(`<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/externalLinkPath" Target="` + target + `" TargetMode="External"/></Relationships>`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buf.Bytes()
}

func buildZipBombXLSX(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	header := &zip.FileHeader{
		Name:   "xl/worksheets/sheet1.xml",
		Method: zip.Deflate,
	}
	fw, err := w.CreateHeader(header)
	require.NoError(t, err)
	payload := bytes.Repeat([]byte("0"), providerBillingMaxUncompressedBytes+1024)
	_, err = fw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.Less(t, buf.Len(), providerBillingMaxFileBytes)
	return buf.Bytes()
}
