package basisbank

import (
	"regexp"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/converters/datatable"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const (
	basisBankCurrency               = "GEL"
	basisBankTransactionTypeIncome  = "Income"
	basisBankTransactionTypeExpense = "Expense"
)

var basisBankTransactionTypeNameMapping = map[string]models.TransactionType{
	basisBankTransactionTypeIncome:  models.TRANSACTION_TYPE_INCOME,
	basisBankTransactionTypeExpense: models.TRANSACTION_TYPE_EXPENSE,
}

var (
	basisBankDateOnlyPattern          = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	basisBankDateTimeNoSecondPattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}$`)
	basisBankDateOnlySlashPattern     = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}$`)
	basisBankDateTimeNoSecondSlash    = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}$`)
	basisBankDescriptionMerchantMatch = regexp.MustCompile(`\bGE\b\s+(?P<shop>.+?)\s*\(`)
)

// basisBankTransactionDataRowParser defines the structure of BasisBank row parser
type basisBankTransactionDataRowParser struct{}

// GetAddedColumns returns the added columns after converting the data row
func (p *basisBankTransactionDataRowParser) GetAddedColumns() []datatable.TransactionDataTableColumn {
	return []datatable.TransactionDataTableColumn{
		datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TYPE,
		datatable.TRANSACTION_DATA_TABLE_ACCOUNT_NAME,
		datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY,
		datatable.TRANSACTION_DATA_TABLE_CATEGORY,
		datatable.TRANSACTION_DATA_TABLE_SUB_CATEGORY,
		datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_NAME,
	}
}

// Parse returns the converted transaction data row
func (p *basisBankTransactionDataRowParser) Parse(data map[datatable.TransactionDataTableColumn]string) (rowData map[datatable.TransactionDataTableColumn]string, rowDataValid bool, err error) {
	dateText := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME])

	if dateText == "" {
		return nil, false, nil
	}

	if basisBankDateOnlyPattern.MatchString(dateText) {
		dateText = dateText + " 00:00:00"
	} else if basisBankDateTimeNoSecondPattern.MatchString(dateText) {
		dateText = dateText + ":00"
	} else if basisBankDateOnlySlashPattern.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-") + " 00:00:00"
	} else if basisBankDateTimeNoSecondSlash.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-") + ":00"
	}

	data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME] = dateText

	debitText := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_AMOUNT])

	if debitText == "" {
		return nil, false, nil
	}

	transactionType := basisBankTransactionTypeExpense
	amountText := debitText

	amountText = strings.ReplaceAll(amountText, ",", "")
	amountText = strings.TrimSpace(amountText)

	if amountText == "" {
		return nil, false, nil
	}

	amountValue, parseErr := utils.ParseAmount(amountText)

	if parseErr != nil {
		return nil, false, errs.ErrAmountInvalid
	}

	if amountValue < 0 {
		amountValue = -amountValue
	}

	if amountValue == 0.00 {
		return nil, false, nil
	}

	data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TYPE] = transactionType
	data[datatable.TRANSACTION_DATA_TABLE_AMOUNT] = utils.FormatAmount(amountValue)
	data[datatable.TRANSACTION_DATA_TABLE_RELATED_AMOUNT] = ""

	description := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION])
	addInfo := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_PAYEE])

	log.Infof(nil, "[basisBankTransactionDataRowParser.Parse] aaaaddInfo: \"%s\"", addInfo)

	if strings.Contains(description, "Private Transfer") {
		privateTransferName := extractPrivateTransferName(addInfo)

		if privateTransferName != "" {
			description = privateTransferName
		}
	} else {
		merchantName := extractMerchantName(description)

		if merchantName != "" {
			description = merchantName
		}
	}

	data[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION] = description
	data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_NAME] = ""
	data[datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_NAME] = ""
	data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY] = basisBankCurrency
	data[datatable.TRANSACTION_DATA_TABLE_CATEGORY] = ""
	data[datatable.TRANSACTION_DATA_TABLE_SUB_CATEGORY] = ""

	return data, true, nil
}

func createBasisBankTransactionDataRowParser() datatable.TransactionDataRowParser {
	return &basisBankTransactionDataRowParser{}
}

func extractMerchantName(description string) string {
	matches := basisBankDescriptionMerchantMatch.FindStringSubmatch(description)

	if len(matches) > 0 {
		return matches[basisBankDescriptionMerchantMatch.SubexpIndex("shop")]
	}

	return ""
}

func extractPrivateTransferName(addInfo string) string {
	if addInfo == "" {
		return ""
	}

	parts := strings.Split(addInfo, ",")

	if len(parts) < 1 {
		return ""
	}

	return strings.TrimSpace(parts[len(parts)-1])
}
