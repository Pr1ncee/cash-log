package services

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

const (
	pricePrecision          = 100000000 // 10^8, same as Python microservice used
	goldAPIBaseURL          = "https://www.goldapi.io/api"
	goldAPIRateLimitSeconds = 86400 // 24 hours
	historicalDataSource    = "historical_csv"
	goldAPIDataSource       = "goldapi.io"
	csvBatchSize            = 200
)

// PreciousMetalsService represents the precious metals service
type PreciousMetalsService struct {
	ServiceUsingDB
	ServiceUsingConfig
}

// Initialize a precious metals service singleton instance
var (
	PreciousMetals = &PreciousMetalsService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingConfig: ServiceUsingConfig{
			container: settings.Container,
		},
	}
)

// ImportHistoricalDataIfNeeded loads historical price data from CSV into the database on first startup
func (s *PreciousMetalsService) ImportHistoricalDataIfNeeded(c core.Context) error {
	config := s.CurrentConfig()

	if !config.EnablePreciousMetals {
		return nil
	}

	db := s.UserDataDB(0)

	// Check if data already exists
	count, err := db.NewSession(c).Where("data_source = ?", historicalDataSource).Count(new(models.PreciousMetalPrice))
	if err != nil {
		return err
	}

	if count > 0 {
		log.BootInfof(c, "[precious_metals.ImportHistoricalDataIfNeeded] historical data already imported (%d records), skipping", count)
		return nil
	}

	csvPath := config.PreciousMetalsHistoricalDataPath
	if !filepath.IsAbs(csvPath) {
		csvPath = filepath.Join(config.WorkingPath, csvPath)
	}

	file, err := os.Open(csvPath)
	if err != nil {
		log.BootErrorf(c, "[precious_metals.ImportHistoricalDataIfNeeded] failed to open CSV file %s: %s", csvPath, err.Error())
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return err
	}

	var batch []*models.PreciousMetalPrice
	totalInserted := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.BootErrorf(c, "[precious_metals.ImportHistoricalDataIfNeeded] error reading CSV row: %s", err.Error())
			continue
		}

		// CSV format: timestamp,price,date,metal,currency,unit
		if len(record) < 6 {
			continue
		}

		timestamp, err := strconv.ParseInt(record[0], 10, 64)
		if err != nil {
			continue
		}

		// CSV timestamps are in milliseconds; convert to seconds
		timestamp = timestamp / 1000

		price, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}

		priceInt := int64(math.Round(price * pricePrecision))

		batch = append(batch, &models.PreciousMetalPrice{
			Metal:      record[3],
			Currency:   record[4],
			Unit:       record[5],
			Price:      priceInt,
			Timestamp:  timestamp,
			DataSource: historicalDataSource,
		})

		if len(batch) >= csvBatchSize {
			err = db.DoTransaction(c, func(sess *xorm.Session) error {
				_, insertErr := sess.Insert(&batch)
				return insertErr
			})
			if err != nil {
				log.BootErrorf(c, "[precious_metals.ImportHistoricalDataIfNeeded] failed to insert batch: %s", err.Error())
				return err
			}
			totalInserted += len(batch)
			batch = batch[:0]
		}
	}

	// Insert remaining records
	if len(batch) > 0 {
		err = db.DoTransaction(c, func(sess *xorm.Session) error {
			_, insertErr := sess.Insert(&batch)
			return insertErr
		})
		if err != nil {
			log.BootErrorf(c, "[precious_metals.ImportHistoricalDataIfNeeded] failed to insert final batch: %s", err.Error())
			return err
		}
		totalInserted += len(batch)
	}

	log.BootInfof(c, "[precious_metals.ImportHistoricalDataIfNeeded] successfully imported %d historical price records", totalInserted)
	return nil
}

// GetPriceHistory returns historical price data for a precious metal from the database
func (s *PreciousMetalsService) GetPriceHistory(c core.Context, metal, currency, unit, timeline string) (*models.PreciousMetalPriceResponse, *errs.Error) {
	config := s.CurrentConfig()

	if !config.EnablePreciousMetals {
		return nil, errs.ErrPreciousMetalsNotEnabled
	}

	db := s.UserDataDB(0)

	// Calculate the start timestamp based on timeline
	startTimestamp := getStartTimestamp(timeline)

	var prices []*models.PreciousMetalPrice
	err := db.NewSession(c).
		Where("metal = ? AND currency = ? AND unit = ? AND timestamp >= ?", metal, currency, unit, startTimestamp).
		OrderBy("timestamp ASC").
		Find(&prices)

	if err != nil {
		log.Errorf(c, "[precious_metals.GetPriceHistory] failed to query prices: %s", err.Error())
		return nil, errs.ErrOperationFailed
	}

	if len(prices) == 0 {
		return nil, errs.ErrPreciousMetalsDataNotFound
	}

	// Get current price (latest from DB or API)
	currentPrice := s.getLatestPrice(c, metal, currency)

	// Convert to response
	historicalData := make([]*models.PreciousMetalPriceDataPoint, len(prices))
	var totalPrice float64

	for i, p := range prices {
		priceFloat := float64(p.Price) / pricePrecision
		historicalData[i] = &models.PreciousMetalPriceDataPoint{
			Timestamp: p.Timestamp,
			Price:     priceFloat,
			Date:      time.Unix(p.Timestamp, 0).UTC().Format("2006-01-02"),
		}
		totalPrice += priceFloat
	}

	// Calculate statistics
	stats := calculateStatistics(historicalData)

	response := &models.PreciousMetalPriceResponse{
		Metal:          metal,
		Currency:       currency,
		Unit:           unit,
		Timeline:       timeline,
		CurrentPrice:   currentPrice,
		HistoricalData: historicalData,
		Statistics:     stats,
		DataSource:     "historical_csv + goldapi.io",
	}

	return response, nil
}

// GetCurrentPrice returns the latest price for a precious metal
func (s *PreciousMetalsService) GetCurrentPrice(c core.Context, metal, currency, unit string) (*models.PreciousMetalPriceResponse, *errs.Error) {
	config := s.CurrentConfig()

	if !config.EnablePreciousMetals {
		return nil, errs.ErrPreciousMetalsNotEnabled
	}

	currentPrice := s.getLatestPrice(c, metal, currency)

	return &models.PreciousMetalPriceResponse{
		Metal:        metal,
		Currency:     currency,
		Unit:         unit,
		CurrentPrice: currentPrice,
		DataSource:   goldAPIDataSource,
	}, nil
}

// RefreshPrice fetches the latest price from GoldAPI and stores it, respecting rate limits
func (s *PreciousMetalsService) RefreshPrice(c core.Context, metal, currency string) (*models.PreciousMetalPriceResponse, *errs.Error) {
	config := s.CurrentConfig()

	if !config.EnablePreciousMetals {
		return nil, errs.ErrPreciousMetalsNotEnabled
	}

	if config.PreciousMetalsGoldAPIKey == "" {
		return nil, errs.ErrPreciousMetalsAPIKeyMissing
	}

	db := s.UserDataDB(0)

	// Check rate limit: only allow one API call per 24 hours
	cutoff := time.Now().Unix() - goldAPIRateLimitSeconds
	var lastFetch models.PreciousMetalAPIFetchLog
	has, err := db.NewSession(c).
		Where("metal = ? AND currency = ? AND fetch_unix_time > ?", metal, currency, cutoff).
		OrderBy("fetch_unix_time DESC").
		Limit(1).
		Get(&lastFetch)

	if err != nil {
		log.Errorf(c, "[precious_metals.RefreshPrice] failed to check rate limit: %s", err.Error())
		return nil, errs.ErrOperationFailed
	}

	if has {
		log.Warnf(c, "[precious_metals.RefreshPrice] API rate limited, last fetch at %d", lastFetch.FetchUnixTime)
		// Return the cached price from the last fetch
		return &models.PreciousMetalPriceResponse{
			Metal:        metal,
			Currency:     currency,
			Unit:         "grams",
			CurrentPrice: float64(lastFetch.PriceGram24k) / pricePrecision,
			DataSource:   goldAPIDataSource,
		}, errs.ErrPreciousMetalsAPIRateLimited
	}

	// Call GoldAPI
	apiResp, apiErr := s.callGoldAPI(c, metal, currency, config.PreciousMetalsGoldAPIKey)
	if apiErr != nil {
		return nil, apiErr
	}

	now := time.Now().Unix()
	priceGram24kInt := int64(math.Round(apiResp.PriceGram24k * pricePrecision))

	// Save fetch log
	fetchLog := &models.PreciousMetalAPIFetchLog{
		Metal:         metal,
		Currency:      currency,
		FetchUnixTime: now,
		PriceGram24k:  priceGram24kInt,
	}
	_, err = db.NewSession(c).Insert(fetchLog)
	if err != nil {
		log.Errorf(c, "[precious_metals.RefreshPrice] failed to save fetch log: %s", err.Error())
	}

	// Save the price as a new data point
	priceRecord := &models.PreciousMetalPrice{
		Metal:      metal,
		Currency:   currency,
		Unit:       "grams",
		Price:      priceGram24kInt,
		Timestamp:  now,
		DataSource: goldAPIDataSource,
	}
	_, err = db.NewSession(c).Insert(priceRecord)
	if err != nil {
		log.Errorf(c, "[precious_metals.RefreshPrice] failed to save price record: %s", err.Error())
	}

	return &models.PreciousMetalPriceResponse{
		Metal:        metal,
		Currency:     currency,
		Unit:         "grams",
		CurrentPrice: apiResp.PriceGram24k,
		DataSource:   goldAPIDataSource,
	}, nil
}

// callGoldAPI makes an HTTP request to goldapi.io
func (s *PreciousMetalsService) callGoldAPI(c core.Context, metal, currency, apiKey string) (*models.GoldAPIResponse, *errs.Error) {
	// Map metal names to GoldAPI symbols
	symbol := metalToGoldAPISymbol(metal)
	if symbol == "" {
		log.Errorf(c, "[precious_metals.callGoldAPI] unsupported metal: %s", metal)
		return nil, errs.ErrPreciousMetalsDataNotFound
	}

	url := fmt.Sprintf("%s/%s/%s", goldAPIBaseURL, symbol, currency)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf(c, "[precious_metals.callGoldAPI] failed to create request: %s", err.Error())
		return nil, errs.ErrPreciousMetalsAPIRequestFailed
	}

	req.Header.Set("x-access-token", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf(c, "[precious_metals.callGoldAPI] request failed: %s", err.Error())
		return nil, errs.ErrPreciousMetalsAPIRequestFailed
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf(c, "[precious_metals.callGoldAPI] API returned status %d", resp.StatusCode)
		return nil, errs.ErrPreciousMetalsAPIRequestFailed
	}

	var apiResp models.GoldAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		log.Errorf(c, "[precious_metals.callGoldAPI] failed to decode response: %s", err.Error())
		return nil, errs.ErrPreciousMetalsAPIRequestFailed
	}

	log.Infof(c, "[precious_metals.callGoldAPI] fetched price for %s/%s: gram24k=%.4f", metal, currency, apiResp.PriceGram24k)
	return &apiResp, nil
}

// getLatestPrice returns the most recent price from the database
func (s *PreciousMetalsService) getLatestPrice(c core.Context, metal, currency string) float64 {
	db := s.UserDataDB(0)

	var price models.PreciousMetalPrice
	has, err := db.NewSession(c).
		Where("metal = ? AND currency = ?", metal, currency).
		OrderBy("timestamp DESC").
		Limit(1).
		Get(&price)

	if err != nil || !has {
		return 0
	}

	return float64(price.Price) / pricePrecision
}

// metalToGoldAPISymbol maps metal names to GoldAPI symbols
func metalToGoldAPISymbol(metal string) string {
	switch metal {
	case "gold":
		return "XAU"
	case "silver":
		return "XAG"
	case "platinum":
		return "XPT"
	case "palladium":
		return "XPD"
	default:
		return ""
	}
}

// getStartTimestamp calculates the start timestamp for a given timeline
func getStartTimestamp(timeline string) int64 {
	now := time.Now()

	switch timeline {
	case "today", "live":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	case "week":
		return now.AddDate(0, 0, -7).Unix()
	case "thismonth":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
	case "lastmonth":
		lastMonth := now.AddDate(0, -1, 0)
		return time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
	case "month":
		return now.AddDate(0, -1, 0).Unix()
	case "3month":
		return now.AddDate(0, -3, 0).Unix()
	case "6month":
		return now.AddDate(0, -6, 0).Unix()
	case "thisyear":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Unix()
	case "lastyear":
		return time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location()).Unix()
	case "year":
		return now.AddDate(-1, 0, 0).Unix()
	case "3year":
		return now.AddDate(-3, 0, 0).Unix()
	case "5year":
		return now.AddDate(-5, 0, 0).Unix()
	case "10year":
		return now.AddDate(-10, 0, 0).Unix()
	case "alltime":
		return 0
	default:
		return now.AddDate(-1, 0, 0).Unix()
	}
}

// calculateStatistics computes statistics for a price series
func calculateStatistics(data []*models.PreciousMetalPriceDataPoint) *models.PreciousMetalStatistics {
	if len(data) == 0 {
		return &models.PreciousMetalStatistics{}
	}

	stats := &models.PreciousMetalStatistics{
		High:  data[0].Price,
		Low:   data[0].Price,
		Open:  data[0].Price,
		Close: data[len(data)-1].Price,
	}

	var total float64
	for _, d := range data {
		if d.Price > stats.High {
			stats.High = d.Price
		}
		if d.Price < stats.Low {
			stats.Low = d.Price
		}
		total += d.Price
	}

	stats.Average = total / float64(len(data))
	stats.Change = stats.Close - stats.Open

	if stats.Open != 0 {
		stats.ChangePct = (stats.Change / stats.Open) * 100
	}

	return stats
}
