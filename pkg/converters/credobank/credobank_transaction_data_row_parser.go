package credobank

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mayswind/ezbookkeeping/pkg/converters/datatable"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const (
	credoBankCurrency               = "GEL"
	credoBankTransactionTypeExpense = "Expense"
)

var credoBankTransactionTypeNameMapping = map[string]models.TransactionType{
	credoBankTransactionTypeExpense: models.TRANSACTION_TYPE_EXPENSE,
}

var (
	credoBankDateOnlyPattern            = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	credoBankDateTimeNoSecondPattern    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}$`)
	credoBankDateOnlySlashPattern       = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}$`)
	credoBankDateTimeNoSecondSlash      = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}$`)
	credoBankDateTimeWithSecondSlash    = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2}$`)
	credoBankDateOnlyDotPattern         = regexp.MustCompile(`^(\d{2})\.(\d{2})\.(\d{4})$`)
	credoBankDateTimeNoSecondDotPattern = regexp.MustCompile(`^(\d{2})\.(\d{2})\.(\d{4})\s+(\d{2}):(\d{2})$`)
	credoBankDescriptionShopMatch       = regexp.MustCompile(`-\s*(?P<shop>.+?)\s+\d+(?:[.,]\d+)?\s+GEL\b`)
)

// credoBankTransactionDataRowParser defines the structure of CredoBank row parser
type credoBankTransactionDataRowParser struct{}

// GetAddedColumns returns the added columns after converting the data row
func (p *credoBankTransactionDataRowParser) GetAddedColumns() []datatable.TransactionDataTableColumn {
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
func (p *credoBankTransactionDataRowParser) Parse(data map[datatable.TransactionDataTableColumn]string) (rowData map[datatable.TransactionDataTableColumn]string, rowDataValid bool, err error) {
	dateText := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME])

	if dateText == "" {
		return nil, false, nil
	}

	if credoBankDateOnlyPattern.MatchString(dateText) {
		dateText = dateText + " 00:00:00"
	} else if credoBankDateTimeNoSecondPattern.MatchString(dateText) {
		dateText = dateText + ":00"
	} else if credoBankDateOnlySlashPattern.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-") + " 00:00:00"
	} else if credoBankDateTimeNoSecondSlash.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-") + ":00"
	} else if credoBankDateTimeWithSecondSlash.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-")
	} else if credoBankDateOnlyDotPattern.MatchString(dateText) {
		dateText = convertDotDateToDash(dateText) + " 00:00:00"
	} else if credoBankDateTimeNoSecondDotPattern.MatchString(dateText) {
		dateText = convertDotDateTimeToDash(dateText)
	}

	data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME] = dateText

	amountText := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_AMOUNT])
	amountText = strings.ReplaceAll(amountText, ",", "")
	amountText = strings.TrimSpace(amountText)

	if amountText == "" {
		return nil, false, nil
	}

	amountValue, parseErr := utils.ParseAmount(amountText)

	if parseErr != nil {
		return nil, false, errs.ErrAmountInvalid
	}

	if amountValue <= 0 {
		return nil, false, nil
	}

	description := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION])
	beneficName := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_PAYEE])

	if strings.Contains(description, "Personal Transfer") {
		description = beneficName
	} else {
		merchantName := utils.ExtractMerchantName(credoBankDescriptionShopMatch, description)

		if merchantName != "" {
			description = merchantName
		}
	}

	data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TYPE] = credoBankTransactionTypeExpense
	data[datatable.TRANSACTION_DATA_TABLE_AMOUNT] = utils.FormatAmount(amountValue)
	data[datatable.TRANSACTION_DATA_TABLE_RELATED_AMOUNT] = ""
	data[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION] = description
	data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_NAME] = ""
	data[datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_NAME] = ""
	currency := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY])

	if currency == "" {
		currency = credoBankCurrency
	}

	data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY] = strings.ToUpper(currency)
	data[datatable.TRANSACTION_DATA_TABLE_CATEGORY] = ""
	data[datatable.TRANSACTION_DATA_TABLE_SUB_CATEGORY] = ""

	return data, true, nil
}

func createCredoBankTransactionDataRowParser() datatable.TransactionDataRowParser {
	return &credoBankTransactionDataRowParser{}
}

func convertDotDateToDash(dateText string) string {
	matches := credoBankDateOnlyDotPattern.FindStringSubmatch(dateText)

	if len(matches) < 4 {
		return dateText
	}

	day := matches[1]
	month := matches[2]
	year := matches[3]
	return fmt.Sprintf("%s-%s-%s", year, month, day)
}

func convertDotDateTimeToDash(dateText string) string {
	matches := credoBankDateTimeNoSecondDotPattern.FindStringSubmatch(dateText)

	if len(matches) < 6 {
		return dateText
	}

	day := matches[1]
	month := matches[2]
	year := matches[3]
	hour := matches[4]
	minute := matches[5]
	return fmt.Sprintf("%s-%s-%s %s:%s:00", year, month, day, hour, minute)
}
