package credobank

import (
	"strings"
	"time"
	"unicode"

	"github.com/mayswind/ezbookkeeping/pkg/converters/converter"
	"github.com/mayswind/ezbookkeeping/pkg/converters/datatable"
	"github.com/mayswind/ezbookkeeping/pkg/converters/excel"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

var credoBankDataColumnNameMapping = map[datatable.TransactionDataTableColumn]string{
	datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME: "Date",
	datatable.TRANSACTION_DATA_TABLE_AMOUNT:           "Turnover (DB)",
	datatable.TRANSACTION_DATA_TABLE_DESCRIPTION:      "Description",
	datatable.TRANSACTION_DATA_TABLE_PAYEE:            "Beneficiary Name",
}

// credoBankTransactionDataXlsxFileImporter defines the structure of CredoBank xlsx importer for transaction data
type credoBankTransactionDataXlsxFileImporter struct {
	converter.DataTableTransactionDataImporter
}

// Initialize a CredoBank transaction data xlsx file importer singleton instance
var (
	CredoBankTransactionDataXlsxFileImporter = &credoBankTransactionDataXlsxFileImporter{}
)

// ParseImportedData returns the imported data by parsing the CredoBank transaction xlsx data
func (c *credoBankTransactionDataXlsxFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	dataTable, err := excel.CreateNewExcelOOXMLFileBasicDataTable(data, true)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	transactionRowParser := createCredoBankTransactionDataRowParser()
	columnNameMapping := buildCredoBankDataColumnNameMapping(dataTable.HeaderColumnNames())
	transactionDataTable := datatable.CreateNewTransactionDataTableFromBasicDataTableWithRowParser(dataTable, columnNameMapping, transactionRowParser)
	dataTableImporter := converter.CreateNewSimpleImporter(credoBankTransactionTypeNameMapping)

	return dataTableImporter.ParseImportedData(ctx, user, transactionDataTable, defaultTimezone, additionalOptions, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)
}

func buildCredoBankDataColumnNameMapping(headerColumnNames []string) map[datatable.TransactionDataTableColumn]string {
	if len(headerColumnNames) < 1 {
		return credoBankDataColumnNameMapping
	}

	headerNameMap := make(map[string]string, len(headerColumnNames))

	for _, headerName := range headerColumnNames {
		normalized := normalizeCredoBankHeaderName(headerName)

		if normalized == "" {
			continue
		}

		headerNameMap[normalized] = headerName
	}

	if matched := matchCredoBankHeaderName(headerNameMap, []string{"date"}); matched != "" {
		credoBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME] = matched
	}

	if matched := matchCredoBankHeaderName(headerNameMap, []string{"turnoverdb", "turnoverdebit"}); matched != "" {
		credoBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_AMOUNT] = matched
	}

	if matched := matchCredoBankHeaderName(headerNameMap, []string{"description", "details"}); matched != "" {
		credoBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION] = matched
	}

	if matched := matchCredoBankHeaderName(headerNameMap, []string{"beneficiaryname", "beneficiary"}); matched != "" {
		credoBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_PAYEE] = matched
	}

	return credoBankDataColumnNameMapping
}

func normalizeCredoBankHeaderName(name string) string {
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

func matchCredoBankHeaderName(headerNameMap map[string]string, candidates []string) string {
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
