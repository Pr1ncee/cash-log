package tbcbank

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

var tbcBankDataColumnNameMapping = map[datatable.TransactionDataTableColumn]string{
	datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME: "Date",
	datatable.TRANSACTION_DATA_TABLE_AMOUNT:           "Paid Out",
	datatable.TRANSACTION_DATA_TABLE_DESCRIPTION:      "Description",
	datatable.TRANSACTION_DATA_TABLE_PAYEE:            "Additional Information",
	datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY: "Currency",
}

// tbcBankTransactionDataXlsxFileImporter defines the structure of TBC Bank xlsx importer for transaction data
type tbcBankTransactionDataXlsxFileImporter struct {
	converter.DataTableTransactionDataImporter
}

// Initialize a TBC Bank transaction data xlsx file importer singleton instance
var (
	TBCBankTransactionDataXlsxFileImporter = &tbcBankTransactionDataXlsxFileImporter{}
)

// ParseImportedData returns the imported data by parsing the TBC Bank transaction xlsx data
func (c *tbcBankTransactionDataXlsxFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	dataTable, err := excel.CreateNewExcelOOXMLFileBasicDataTable(data, true)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	transactionRowParser := createTBCBankTransactionDataRowParser()
	columnNameMapping := buildTBCBankDataColumnNameMapping(dataTable.HeaderColumnNames())
	transactionDataTable := datatable.CreateNewTransactionDataTableFromBasicDataTableWithRowParser(dataTable, columnNameMapping, transactionRowParser)
	dataTableImporter := converter.CreateNewSimpleImporter(tbcBankTransactionTypeNameMapping)

	return dataTableImporter.ParseImportedData(ctx, user, transactionDataTable, defaultTimezone, additionalOptions, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)
}

func buildTBCBankDataColumnNameMapping(headerColumnNames []string) map[datatable.TransactionDataTableColumn]string {
	if len(headerColumnNames) < 1 {
		return tbcBankDataColumnNameMapping
	}

	headerNameMap := make(map[string]string, len(headerColumnNames))

	for _, headerName := range headerColumnNames {
		normalized := normalizeTBCBankHeaderName(headerName)

		if normalized == "" {
			continue
		}

		headerNameMap[normalized] = headerName
	}

	if matched := matchTBCBankHeaderName(headerNameMap, []string{"date"}); matched != "" {
		log.Infof(nil, "[buildTBCBankDataColumnNameMapping] matched header \"%s\" for column 'TRANSACTION_TIME'", matched)
		tbcBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME] = matched
	}

	if matched := matchTBCBankHeaderName(headerNameMap, []string{"paidout", "debit", "paid"}); matched != "" {
		log.Infof(nil, "[buildTBCBankDataColumnNameMapping] matched header \"%s\" for column 'AMOUNT'", matched)
		tbcBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_AMOUNT] = matched
	}

	if matched := matchTBCBankHeaderName(headerNameMap, []string{"description", "details"}); matched != "" {
		log.Infof(nil, "[buildTBCBankDataColumnNameMapping] matched header \"%s\" for column 'DESCRIPTION'", matched)
		tbcBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION] = matched
	}

	if matched := matchTBCBankHeaderName(headerNameMap, []string{"additionalinformation", "additionalinfo", "addinfo"}); matched != "" {
		log.Infof(nil, "[buildTBCBankDataColumnNameMapping] matched header \"%s\" for column 'PAYEE'", matched)
		tbcBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_PAYEE] = matched
	}

	if matched := matchTBCBankHeaderName(headerNameMap, []string{"currency", "ccy"}); matched != "" {
		log.Infof(nil, "[buildTBCBankDataColumnNameMapping] matched header \"%s\" for column 'ACCOUNT_CURRENCY'", matched)
		tbcBankDataColumnNameMapping[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY] = matched
	}

	return tbcBankDataColumnNameMapping
}

func normalizeTBCBankHeaderName(name string) string {
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

func matchTBCBankHeaderName(headerNameMap map[string]string, candidates []string) string {
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
