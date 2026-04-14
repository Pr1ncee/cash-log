package models

// PreciousMetalPrice represents precious metal price data stored in database
type PreciousMetalPrice struct {
	Id              int64  `xorm:"PK autoincr"`
	Metal           string `xorm:"VARCHAR(20) NOT NULL INDEX"`
	Currency        string `xorm:"VARCHAR(3) NOT NULL INDEX"`
	Unit            string `xorm:"VARCHAR(20) NOT NULL"`
	Price           int64  `xorm:"NOT NULL"`
	Timestamp       int64  `xorm:"NOT NULL INDEX"`
	DataSource      string `xorm:"VARCHAR(100)"`
	CreatedUnixTime int64  `xorm:"created"`
}

// PreciousMetalScrapeLog represents a log entry for a precious metal scrape operation
type PreciousMetalScrapeLog struct {
	Id              int64  `xorm:"PK autoincr"`
	Metal           string `xorm:"VARCHAR(20) NOT NULL"`
	Currency        string `xorm:"VARCHAR(3) NOT NULL"`
	Unit            string `xorm:"VARCHAR(20) NOT NULL"`
	Timeline        string `xorm:"VARCHAR(20) NOT NULL"`
	ScrapeDate      string `xorm:"VARCHAR(10) NOT NULL INDEX"`
	DataPointsCount int    `xorm:"NOT NULL DEFAULT 0"`
	Status          string `xorm:"VARCHAR(20) NOT NULL"`
	CreatedUnixTime int64  `xorm:"created"`
}

// PreciousMetalAPIFetchLog tracks API calls to enforce rate limiting (once per 24 hours)
type PreciousMetalAPIFetchLog struct {
	Id              int64  `xorm:"PK autoincr"`
	Metal           string `xorm:"VARCHAR(20) NOT NULL"`
	Currency        string `xorm:"VARCHAR(3) NOT NULL"`
	FetchUnixTime   int64  `xorm:"NOT NULL INDEX"`
	PriceGram24k    int64  `xorm:"NOT NULL"`
	CreatedUnixTime int64  `xorm:"created"`
}

// PreciousMetalPriceRequest represents a request for precious metal prices
type PreciousMetalPriceRequest struct {
	Metal    string `form:"metal" binding:"required,max=20"`
	Currency string `form:"currency" binding:"required,len=3"`
	Unit     string `form:"unit" binding:"required,max=20"`
	Timeline string `form:"timeline" binding:"required,max=20"`
}

// PreciousMetalPortfolioRequest represents a request for the user's precious metal portfolio
type PreciousMetalPortfolioRequest struct {
	Metal string `form:"metal" binding:"omitempty,max=20"`
}

// PreciousMetalRefreshRequest represents a request to trigger data refresh
type PreciousMetalRefreshRequest struct {
	Metal    string `json:"metal" binding:"required,max=20"`
	Currency string `json:"currency" binding:"required,len=3"`
}

// PreciousMetalPriceDataPoint represents a single data point in price history
type PreciousMetalPriceDataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Price     float64 `json:"price"`
	Date      string  `json:"date"`
}

// PreciousMetalStatistics represents statistics for a price series
type PreciousMetalStatistics struct {
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	Average   float64 `json:"average"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"changePct"`
}

// PreciousMetalPriceResponse represents the response for price history
type PreciousMetalPriceResponse struct {
	Metal          string                         `json:"metal"`
	Currency       string                         `json:"currency"`
	Unit           string                         `json:"unit"`
	Timeline       string                         `json:"timeline"`
	CurrentPrice   float64                        `json:"currentPrice"`
	HistoricalData []*PreciousMetalPriceDataPoint `json:"historicalData"`
	Statistics     *PreciousMetalStatistics       `json:"statistics"`
	DataSource     string                         `json:"dataSource"`
}

// PreciousMetalPortfolioItem represents a single metal holding in the portfolio
type PreciousMetalPortfolioItem struct {
	Metal         string  `json:"metal"`
	Currency      string  `json:"currency"`
	TotalQuantity float64 `json:"totalQuantity"`
	Unit          string  `json:"unit"`
	AvgBuyPrice   float64 `json:"avgBuyPrice"`
	CurrentPrice  float64 `json:"currentPrice"`
	TotalCost     float64 `json:"totalCost"`
	CurrentValue  float64 `json:"currentValue"`
	ProfitLoss    float64 `json:"profitLoss"`
	ProfitLossPct float64 `json:"profitLossPct"`
}

// PreciousMetalPortfolioResponse represents the full portfolio response
type PreciousMetalPortfolioResponse struct {
	Items      []*PreciousMetalPortfolioItem `json:"items"`
	TotalValue float64                       `json:"totalValue"`
	TotalCost  float64                       `json:"totalCost"`
}

// GoldAPIResponse represents the response from goldapi.io
type GoldAPIResponse struct {
	Timestamp      int64   `json:"timestamp"`
	Metal          string  `json:"metal"`
	Currency       string  `json:"currency"`
	Exchange       string  `json:"exchange"`
	Symbol         string  `json:"symbol"`
	PrevClosePrice float64 `json:"prev_close_price"`
	OpenPrice      float64 `json:"open_price"`
	LowPrice       float64 `json:"low_price"`
	HighPrice      float64 `json:"high_price"`
	OpenTime       int64   `json:"open_time"`
	Price          float64 `json:"price"`
	Ch             float64 `json:"ch"`
	Chp            float64 `json:"chp"`
	Ask            float64 `json:"ask"`
	Bid            float64 `json:"bid"`
	PriceGram24k   float64 `json:"price_gram_24k"`
	PriceGram22k   float64 `json:"price_gram_22k"`
	PriceGram21k   float64 `json:"price_gram_21k"`
	PriceGram20k   float64 `json:"price_gram_20k"`
	PriceGram18k   float64 `json:"price_gram_18k"`
	PriceGram16k   float64 `json:"price_gram_16k"`
	PriceGram14k   float64 `json:"price_gram_14k"`
	PriceGram10k   float64 `json:"price_gram_10k"`
}
