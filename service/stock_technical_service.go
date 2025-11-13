package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/iter"
	"github.com/polygon-io/client-go/rest/models"
)

type StockTechnicalService struct {
	apiKey string
	ticker string
}

func NewStockTechnicalService(ticker string) *StockTechnicalService {
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("POLYGON_API_KEY") // Fallback, but will error if not set
	}
	return &StockTechnicalService{apiKey: apiKey, ticker: ticker}
}

type TechnicalResponse struct {
	Status  string `json:"status"`
	Results struct {
		Values []struct {
			Value     float64 `json:"value"`
			Timestamp int64   `json:"timestamp"`
		} `json:"values"`
	} `json:"results"`
}

type MACDValue struct {
	Value     float64 `json:"value"`
	Signal    float64 `json:"signal"`
	Histogram float64 `json:"histogram"`
	Timestamp int64   `json:"timestamp"`
}

type MACDResponse struct {
	Status  string `json:"status"`
	Results struct {
		Values []MACDValue `json:"values"`
	} `json:"results"`
}

func (s *StockTechnicalService) FetchTechnicalSummary() (string, error) {

	// Fetch indicators for different time ranges
	// SMA and EMA
	sma20Resp, err := s.FetchSMA(20) // Short-term

	if err != nil {
		return "", fmt.Errorf("failed to fetch SMA: %w", err)
	}

	sma50Resp, werr := s.FetchSMA(50) // Medium-term
	if werr != nil {
		return "", fmt.Errorf("failed to fetch SMA: %w", werr)
	}
	sma200Resp, err := s.FetchSMA(200) // Long-term
	ema20Resp, err := s.FetchEMA(20)
	ema50Resp, err := s.FetchEMA(50)
	ema200Resp, err := s.FetchEMA(200)
	if err != nil {
		return "", fmt.Errorf("failed to fetch EMA: %w", err)
	}

	// RSI
	rsi5Resp, _ := s.FetchRSI(5)   // Short-term
	rsi14Resp, _ := s.FetchRSI(14) // Medium-term
	rsi50Resp, _ := s.FetchRSI(50) // Long-term

	// MACD
	macdShortResp, _ := s.FetchMACD(6, 13, 5)   // Short-term
	macdMediumResp, _ := s.FetchMACD(12, 26, 9) // Medium-term
	macdLongResp, _ := s.FetchMACD(26, 52, 9)   // Long-term

	// Initialize latest values
	latestSMA20, latestSMA50, latestSMA200 := "N/A", "N/A", "N/A"
	latestEMA20, latestEMA50, latestEMA200 := "N/A", "N/A", "N/A"
	latestRSI5, latestRSI14, latestRSI50 := "N/A", "N/A", "N/A"
	var latestMACDShort, latestMACDMedium, latestMACDLong MACDValue

	// Extract latest values for SMA
	if sma20Resp != nil && sma20Resp.Status == "OK" && len(sma20Resp.Results.Values) > 0 {
		latestSMA20 = fmt.Sprintf("%.2f", sma20Resp.Results.Values[0].Value)
	}
	if sma50Resp != nil && sma50Resp.Status == "OK" && len(sma50Resp.Results.Values) > 0 {
		latestSMA50 = fmt.Sprintf("%.2f", sma50Resp.Results.Values[0].Value)
	}
	if sma200Resp != nil && sma200Resp.Status == "OK" && len(sma200Resp.Results.Values) > 0 {
		latestSMA200 = fmt.Sprintf("%.2f", sma200Resp.Results.Values[0].Value)
	}

	// Extract latest values for EMA
	if ema20Resp != nil && ema20Resp.Status == "OK" && len(ema20Resp.Results.Values) > 0 {
		latestEMA20 = fmt.Sprintf("%.2f", ema20Resp.Results.Values[0].Value)
	}
	if ema50Resp != nil && ema50Resp.Status == "OK" && len(ema50Resp.Results.Values) > 0 {
		latestEMA50 = fmt.Sprintf("%.2f", ema50Resp.Results.Values[0].Value)
	}
	if ema200Resp != nil && ema200Resp.Status == "OK" && len(ema200Resp.Results.Values) > 0 {
		latestEMA200 = fmt.Sprintf("%.2f", ema200Resp.Results.Values[0].Value)
	}

	// Extract latest values for RSI
	if rsi5Resp != nil && rsi5Resp.Status == "OK" && len(rsi5Resp.Results.Values) > 0 {
		latestRSI5 = fmt.Sprintf("%.2f", rsi5Resp.Results.Values[0].Value)
	}
	if rsi14Resp != nil && rsi14Resp.Status == "OK" && len(rsi14Resp.Results.Values) > 0 {
		latestRSI14 = fmt.Sprintf("%.2f", rsi14Resp.Results.Values[0].Value)
	}
	if rsi50Resp != nil && rsi50Resp.Status == "OK" && len(rsi50Resp.Results.Values) > 0 {
		latestRSI50 = fmt.Sprintf("%.2f", rsi50Resp.Results.Values[0].Value)
	}

	// Extract latest values for MACD
	if macdShortResp != nil && macdShortResp.Status == "OK" && len(macdShortResp.Results.Values) > 0 {
		latestMACDShort = macdShortResp.Results.Values[0]
	}
	if macdMediumResp != nil && macdMediumResp.Status == "OK" && len(macdMediumResp.Results.Values) > 0 {
		latestMACDMedium = macdMediumResp.Results.Values[0]
	}
	if macdLongResp != nil && macdLongResp.Status == "OK" && len(macdLongResp.Results.Values) > 0 {
		latestMACDLong = macdLongResp.Results.Values[0]
	}

	// Calculate trends
	sma20Trend := getTrend(sma20Resp)
	sma50Trend := getTrend(sma50Resp)
	sma200Trend := getTrend(sma200Resp)
	ema20Trend := getTrend(ema20Resp)
	ema50Trend := getTrend(ema50Resp)
	ema200Trend := getTrend(ema200Resp)
	rsi5Trend := getTrend(rsi5Resp)
	rsi14Trend := getTrend(rsi14Resp)
	rsi50Trend := getTrend(rsi50Resp)
	macdShortTrend := getMACDTrend(macdShortResp)
	macdMediumTrend := getMACDTrend(macdMediumResp)
	macdLongTrend := getMACDTrend(macdLongResp)

	// Determine RSI status for each timeframe
	rsi5Status := "neutral"
	if rsi5, _ := fmt.Sscanf(latestRSI5, "%f"); rsi5 > 70 {
		rsi5Status = "overbought"
	} else if rsi5 < 30 {
		rsi5Status = "oversold"
	}
	rsi14Status := "neutral"
	if rsi14, _ := fmt.Sscanf(latestRSI14, "%f"); rsi14 > 70 {
		rsi14Status = "overbought"
	} else if rsi14 < 30 {
		rsi14Status = "oversold"
	}
	rsi50Status := "neutral"
	if rsi50, _ := fmt.Sscanf(latestRSI50, "%f"); rsi50 > 70 {
		rsi50Status = "overbought"
	} else if rsi50 < 30 {
		rsi50Status = "oversold"
	}

	// Generate the summary
	summary := fmt.Sprintf(`
Here is a summary of the current technical indicator data across multiple timeframes:

### Simple Moving Average (SMA)
• 20-day SMA: %s (latest: %s)
• 50-day SMA: %s (latest: %s)
• 200-day SMA: %s (latest: %s)

### Exponential Moving Average (EMA)
• 20-day EMA: %s (latest: %s)
• 50-day EMA: %s (latest: %s)
• 200-day EMA: %s (latest: %s)

### Relative Strength Index (RSI)
• 5-day RSI: %s, currently at %s (%s)
• 14-day RSI: %s, currently at %s (%s)
• 50-day RSI: %s, currently at %s (%s)

### MACD (Moving Average Convergence Divergence)
• Short-term (6/13/5) MACD: Line: %.2f, Signal: %.2f, Histogram: %.2f (%s)
• Medium-term (12/26/9) MACD: Line: %.2f, Signal: %.2f, Histogram: %.2f (%s)
• Long-term (26/52/9) MACD: Line: %.2f, Signal: %.2f, Histogram: %.2f (%s)
`,
		sma20Trend, latestSMA20, sma50Trend, latestSMA50, sma200Trend, latestSMA200,
		ema20Trend, latestEMA20, ema50Trend, latestEMA50, ema200Trend, latestEMA200,
		rsi5Trend, latestRSI5, rsi5Status, rsi14Trend, latestRSI14, rsi14Status, rsi50Trend, latestRSI50, rsi50Status,
		latestMACDShort.Value, latestMACDShort.Signal, latestMACDShort.Histogram, macdShortTrend,
		latestMACDMedium.Value, latestMACDMedium.Signal, latestMACDMedium.Histogram, macdMediumTrend,
		latestMACDLong.Value, latestMACDLong.Signal, latestMACDLong.Histogram, macdLongTrend)

	return summary, nil
}

func (s *StockTechnicalService) FetchSMA(window int) (*TechnicalResponse, error) {
	return s.fetchTechnical("sma", map[string]string{"window": fmt.Sprintf("%d", window)})
}

func (s *StockTechnicalService) FetchEMA(window int) (*TechnicalResponse, error) {
	return s.fetchTechnical("ema", map[string]string{"window": fmt.Sprintf("%d", window)})
}

func (s *StockTechnicalService) FetchRSI(window int) (*TechnicalResponse, error) {
	return s.fetchTechnical("rsi", map[string]string{"window": fmt.Sprintf("%d", window)})
}

func (s *StockTechnicalService) FetchMACD(shortWindow, longWindow, signalWindow int) (*MACDResponse, error) {
	params := map[string]string{
		"short_window":  fmt.Sprintf("%d", shortWindow),
		"long_window":   fmt.Sprintf("%d", longWindow),
		"signal_window": fmt.Sprintf("%d", signalWindow),
	}
	url := fmt.Sprintf("https://api.polygon.io/v1/indicators/macd/%s", s.ticker)
	return s.fetchMACD(url, params)
}

func (s *StockTechnicalService) GetTickerDetailsFromPolygon() (*models.GetTickerDetailsResponse, error) {

	c := polygon.New(s.apiKey)

	params := models.GetTickerDetailsParams{
		Ticker: s.ticker,
	}

	res, err := c.GetTickerDetails(context.Background(), &params)
	if err != nil {

		return nil, err
	}

	return res, nil
}

func (s *StockTechnicalService) GetTickeSnapshotPolygon() (*models.GetTickerSnapshotResponse, error) {
	c := polygon.New(s.apiKey)

	params := models.GetTickerSnapshotParams{
		Ticker:     s.ticker,
		Locale:     "us",
		MarketType: "stocks",
	}

	res, err := c.GetTickerSnapshot(context.Background(), &params)
	if err != nil {
		return nil, err
	}

	return res, nil

}

func (s *StockTechnicalService) GetSimilarTickers() (*models.GetTickerRelatedCompaniesResponse, error) {
	c := polygon.New(s.apiKey)

	params := models.GetTickerRelatedCompaniesParams{
		Ticker: s.ticker,
	}

	res, err := c.GetTickerRelatedCompanies(context.Background(), &params)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *StockTechnicalService) GetPolygonAggregate(timeSpan, startDate, endDate string, multiplier int) (*iter.Iter[models.Agg], error) {

	c := polygon.New(s.apiKey)

	from, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, err
	}
	to, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, err
	}

	params := models.ListAggsParams{
		Ticker:     s.ticker,
		Multiplier: multiplier,
		Timespan:   models.Timespan(timeSpan),
		From:       models.Millis(from),
		To:         models.Millis(to),
	}.
		WithAdjusted(true).
		WithOrder(models.Order("asc")).
		WithLimit(120)

	iter := c.ListAggs(context.Background(), params)

	return iter, nil

}

func (s *StockTechnicalService) GetPolygonNewsForTicker() (string, *iter.Iter[models.TickerNews]) {
	c := polygon.New(s.apiKey)

	params := models.ListTickerNewsParams{
		TickerEQ: &s.ticker,
		Sort:     (*models.Sort)(ptr("published_utc")),
		Order:    (*models.Order)(ptr("asc")),
		// Limit intentionally left unset to allow full stream, but we break manually
	}

	iter := c.ListTickerNews(context.Background(), &params)

	var sb strings.Builder
	count := 0
	maxItems := 10

	for iter.Next() {
		item := iter.Item()
		sb.WriteString(fmt.Sprintf("Title: %s\nDescription: %s\n\n", item.Title, item.Description))

		count++
		if count >= maxItems {
			break
		}
	}

	return sb.String(), iter
}

func ptr(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}

func (s *StockTechnicalService) fetchTechnical(indicator string, extraParams map[string]string) (*TechnicalResponse, error) {
	baseURL := fmt.Sprintf("https://api.polygon.io/v1/indicators/%s/%s", indicator, s.ticker)
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("timespan", "day")
	q.Set("adjusted", "true")
	q.Set("series_type", "close")
	q.Set("order", "desc")
	q.Set("apiKey", s.apiKey)
	for k, v := range extraParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var data TechnicalResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *StockTechnicalService) fetchMACD(apiURL string, params map[string]string) (*MACDResponse, error) {
	u, _ := url.Parse(apiURL)
	q := u.Query()
	q.Set("timespan", "day")
	q.Set("adjusted", "true")
	q.Set("series_type", "close")
	q.Set("order", "desc")
	q.Set("apiKey", s.apiKey)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var data MACDResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

// getTrend calculates the trend direction from a TechnicalResponse
func getTrend(resp *TechnicalResponse) string {
	if resp == nil || len(resp.Results.Values) < 2 {
		return "unknown"
	}

	values := resp.Results.Values

	// Assuming values[0] is the oldest and values[len-1] is the most recent
	start := values[0].Value
	end := values[len(values)-1].Value

	switch {
	case end > start:
		return "rising"
	case end < start:
		return "falling"
	default:
		return "flat"
	}
}

// getMACDTrend calculates the trend direction from a MACDResponse
func getMACDTrend(resp *MACDResponse) string {
	if resp == nil || len(resp.Results.Values) < 2 {
		return "unknown"
	}

	values := resp.Results.Values

	// Assuming values[0] is oldest, and values[len-1] is most recent
	start := values[0].Histogram
	end := values[len(values)-1].Histogram

	switch {
	case end > start:
		return "rising"
	case end < start:
		return "falling"
	default:
		return "flat"
	}
}
