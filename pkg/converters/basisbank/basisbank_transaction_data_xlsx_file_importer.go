package basisbank

import (
	"strings"
	"time"
	"unicode"

	"github.com/mayswind/ezbookkeeping/pkg/converters/converter"
	"github.com/mayswind/ezbookkeeping/pkg/converters/datatable"
	"github.com/mayswind/ezbookkeeping/pkg/converters/excel"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

var basisBankDataColumnNameMapping = map[datatable.TransactionDataTableColumn]string{
	datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME: "Date",
	datatable.TRANSACTION_DATA_TABLE_AMOUNT:           "Turnover (deb)",
	datatable.TRANSACTION_DATA_TABLE_DESCRIPTION:      "Description",
	datatable.TRANSACTION_DATA_TABLE_PAYEE:            "Additional Info",
}

// basisBankTransactionDataXlsxFileImporter defines the structure of BasisBank xlsx importer for transaction data
type basisBankTransactionDataXlsxFileImporter struct {
	converter.DataTableTransactionDataImporter
}

// Initialize a BasisBank transaction data xlsx file importer singleton instance
var (
	BasisBankTransactionDataXlsxFileImporter = &basisBankTransactionDataXlsxFileImporter{}
)

// ParseImportedData returns the imported data by parsing the BasisBank transaction xlsx data
func (c *basisBankTransactionDataXlsxFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	dataTable, err := excel.CreateNewExcelOOXMLFileBasicDataTable(data, true)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	transactionRowParser := createBasisBankTransactionDataRowParser()
	columnNameMapping := buildBasisBankDataColumnNameMapping(dataTable.HeaderColumnNames())
	transactionDataTable := datatable.CreateNewTransactionDataTableFromBasicDataTableWithRowParser(dataTable, columnNameMapping, transactionRowParser)
	dataTableImporter := converter.CreateNewSimpleImporter(basisBankTransactionTypeNameMapping)

	return dataTableImporter.ParseImportedData(ctx, user, transactionDataTable, defaultTimezone, additionalOptions, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)
}

func buildBasisBankDataColumnNameMapping(headerColumnNames []string) map[datatable.TransactionDataTableColumn]string {
	if len(headerColumnNames) < 1 {
		return basisBankDataColumnNameMapping
	}

	headerNameMap := make(map[string]string, len(headerColumnNames))

	for _, headerName := range headerColumnNames {
		normalized := normalizeBasisBankHeaderName(headerName)

		if normalized == "" {
			continue
		}

		headerNameMap[normalized] = headerName
	}

	if matched := matchHeaderName(headerNameMap, []string{"date"}); matched != "" {
		log.Infof(nil, "[buildBasisBankDataColumnNameMapping] matched header \"%s\" for column \"%s\"", matched, datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME)
		basisBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME] = matched
	}

	if matched := matchHeaderName(headerNameMap, []string{"turnoverdeb", "turnoverdebit"}); matched != "" {
		log.Infof(nil, "[buildBasisBankDataColumnNameMapping] matched header \"%s\" for column \"%s\"", matched, datatable.TRANSACTION_DATA_TABLE_AMOUNT)
		basisBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_AMOUNT] = matched
	}

	if matched := matchHeaderName(headerNameMap, []string{"description", "details"}); matched != "" {
		log.Infof(nil, "[buildBasisBankDataColumnNameMapping] matched header \"%s\" for column \"%s\"", matched, datatable.TRANSACTION_DATA_TABLE_DESCRIPTION)
		basisBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION] = matched
	}

	if matched := matchHeaderName(headerNameMap, []string{"addinfo", "additionalinfo", "extrainfo"}); matched != "" {
		log.Infof(nil, "[buildBasisBankDataColumnNameMapping] matched header \"%s\" for column \"%s\"", matched, datatable.TRANSACTION_DATA_TABLE_PAYEE)
		basisBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_PAYEE] = matched
	}

	return basisBankDataColumnNameMapping
}

func normalizeBasisBankHeaderName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))

	if name == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(name))

	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func matchHeaderName(headerNameMap map[string]string, candidates []string) string {
	for _, candidate := range candidates {
		if headerName, exists := headerNameMap[candidate]; exists {
			return headerName
		}
	}

	for headerKey, headerName := range headerNameMap {
		for _, candidate := range candidates {
			if strings.Contains(headerKey, candidate) {
				return headerName
			}
		}
	}

	return ""
}
