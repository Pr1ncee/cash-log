package tbcbank

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/converters/datatable"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const (
	tbcBankCurrency               = "GEL"
	tbcBankTransactionTypeExpense = "Expense"
)

var tbcBankTransactionTypeNameMapping = map[string]models.TransactionType{
	tbcBankTransactionTypeExpense: models.TRANSACTION_TYPE_EXPENSE,
}

// excelEpoch is the base date Excel uses for serial date numbers (Dec 30, 1899),
// accounting for Excel's intentional Lotus 1-2-3 compatibility bug that treats 1900 as a leap year.
var tbcBankExcelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

var (
	tbcBankExcelSerialDatePattern            = regexp.MustCompile(`^\d+(\.\d+)?$`)
	tbcBankDateOnlyPattern                   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	tbcBankDateTimeNoSecondPattern           = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}$`)
	tbcBankDateOnlySlashPattern              = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}$`)
	tbcBankDateTimeNoSecondSlash             = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}$`)
	tbcBankDateTimeWithSecondSlash           = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2}$`)
	tbcBankDateOnlyDMYSlashPattern           = regexp.MustCompile(`^(\d{2})/(\d{2})/(\d{4})$`)
	tbcBankDateTimeNoSecondDMYSlashPattern   = regexp.MustCompile(`^(\d{2})/(\d{2})/(\d{4})\s+(\d{2}):(\d{2})$`)
	tbcBankDateTimeWithSecondDMYSlashPattern = regexp.MustCompile(`^(\d{2})/(\d{2})/(\d{4})\s+(\d{2}):(\d{2}):(\d{2})$`)
	tbcBankDateOnlyDotPattern                = regexp.MustCompile(`^(\d{2})\.(\d{2})\.(\d{4})$`)
	tbcBankDateTimeNoSecondDotPattern        = regexp.MustCompile(`^(\d{2})\.(\d{2})\.(\d{4})\s+(\d{2}):(\d{2})$`)
)

// tbcBankTransactionDataRowParser defines the structure of TBC Bank row parser
type tbcBankTransactionDataRowParser struct{}

// GetAddedColumns returns the added columns after converting the data row
func (p *tbcBankTransactionDataRowParser) GetAddedColumns() []datatable.TransactionDataTableColumn {
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
func (p *tbcBankTransactionDataRowParser) Parse(data map[datatable.TransactionDataTableColumn]string) (rowData map[datatable.TransactionDataTableColumn]string, rowDataValid bool, err error) {
	dateText := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME])

	if dateText == "" {
		return nil, false, nil
	}

	if tbcBankExcelSerialDatePattern.MatchString(dateText) {
		dateText = tbcBankConvertExcelSerialDate(dateText)
	} else if tbcBankDateOnlyPattern.MatchString(dateText) {
		dateText = dateText + " 00:00:00"
	} else if tbcBankDateTimeNoSecondPattern.MatchString(dateText) {
		dateText = dateText + ":00"
	} else if tbcBankDateOnlySlashPattern.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-") + " 00:00:00"
	} else if tbcBankDateTimeNoSecondSlash.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-") + ":00"
	} else if tbcBankDateTimeWithSecondSlash.MatchString(dateText) {
		dateText = strings.ReplaceAll(dateText, "/", "-")
	} else if tbcBankDateTimeWithSecondDMYSlashPattern.MatchString(dateText) {
		dateText = tbcBankConvertDMYSlashDateTimeToDash(dateText)
	} else if tbcBankDateTimeNoSecondDMYSlashPattern.MatchString(dateText) {
		dateText = tbcBankConvertDMYSlashDateTimeToDash(dateText)
	} else if tbcBankDateOnlyDMYSlashPattern.MatchString(dateText) {
		dateText = tbcBankConvertDMYSlashDateToDash(dateText) + " 00:00:00"
	} else if tbcBankDateOnlyDotPattern.MatchString(dateText) {
		dateText = tbcBankConvertDotDateToDash(dateText) + " 00:00:00"
	} else if tbcBankDateTimeNoSecondDotPattern.MatchString(dateText) {
		dateText = tbcBankConvertDotDateTimeToDash(dateText)
	}

	data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME] = dateText

	amountText := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_AMOUNT])

	if amountText == "" {
		return nil, false, nil
	}

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
	addInfo := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_PAYEE])

	if description == "" && addInfo != "" {
		description = addInfo
	}

	data[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TYPE] = tbcBankTransactionTypeExpense
	data[datatable.TRANSACTION_DATA_TABLE_AMOUNT] = utils.FormatAmount(amountValue)
	data[datatable.TRANSACTION_DATA_TABLE_RELATED_AMOUNT] = ""
	data[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION] = description
	data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_NAME] = ""
	data[datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_NAME] = ""

	currency := strings.TrimSpace(data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY])

	if currency == "" {
		currency = tbcBankCurrency
	}

	data[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY] = strings.ToUpper(currency)
	data[datatable.TRANSACTION_DATA_TABLE_CATEGORY] = ""
	data[datatable.TRANSACTION_DATA_TABLE_SUB_CATEGORY] = ""

	return data, true, nil
}

func createTBCBankTransactionDataRowParser() datatable.TransactionDataRowParser {
	return &tbcBankTransactionDataRowParser{}
}

// tbcBankConvertExcelSerialDate converts an Excel serial date number (e.g. "46124" or "46124.5")
// to "YYYY-MM-DD HH:MM:SS" format. The integer part is days since Dec 30, 1899; the fractional
// part represents the time within that day.
func tbcBankConvertExcelSerialDate(serialText string) string {
	serial, err := strconv.ParseFloat(serialText, 64)

	if err != nil {
		return serialText
	}

	days := int(serial)
	fracDay := serial - float64(days)
	date := tbcBankExcelEpoch.AddDate(0, 0, days)

	totalSeconds := int(fracDay * 86400)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		date.Year(), int(date.Month()), date.Day(),
		hours, minutes, seconds)
}

func tbcBankConvertDMYSlashDateToDash(dateText string) string {
	matches := tbcBankDateOnlyDMYSlashPattern.FindStringSubmatch(dateText)

	if len(matches) < 4 {
		return dateText
	}

	day := matches[1]
	month := matches[2]
	year := matches[3]
	return fmt.Sprintf("%s-%s-%s", year, month, day)
}

func tbcBankConvertDMYSlashDateTimeToDash(dateText string) string {
	matches := tbcBankDateTimeWithSecondDMYSlashPattern.FindStringSubmatch(dateText)

	if len(matches) >= 7 {
		return fmt.Sprintf("%s-%s-%s %s:%s:%s", matches[3], matches[2], matches[1], matches[4], matches[5], matches[6])
	}

	matches = tbcBankDateTimeNoSecondDMYSlashPattern.FindStringSubmatch(dateText)

	if len(matches) >= 6 {
		return fmt.Sprintf("%s-%s-%s %s:%s:00", matches[3], matches[2], matches[1], matches[4], matches[5])
	}

	return dateText
}

func tbcBankConvertDotDateToDash(dateText string) string {
	matches := tbcBankDateOnlyDotPattern.FindStringSubmatch(dateText)

	if len(matches) < 4 {
		return dateText
	}

	day := matches[1]
	month := matches[2]
	year := matches[3]
	return fmt.Sprintf("%s-%s-%s", year, month, day)
}

func tbcBankConvertDotDateTimeToDash(dateText string) string {
	matches := tbcBankDateTimeNoSecondDotPattern.FindStringSubmatch(dateText)

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
