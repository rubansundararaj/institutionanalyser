package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// EarningsBigMoneyHandler handles earnings calendar with big money flow analysis
type EarningsBigMoneyHandler struct {
	PolygonAPIKey     string
	PolygonBaseURL    string
	TradeAnalysisURL  string
}

// NewEarningsBigMoneyHandler creates a new earnings big money handler
func NewEarningsBigMoneyHandler() *EarningsBigMoneyHandler {
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("POLYGON_API_KEY")
	}

	baseURL := os.Getenv("POLYGON_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.polygon.io"
	}

	tradeAnalysisURL := os.Getenv("TRADE_ANALYSIS_API_URL")
	if tradeAnalysisURL == "" {
		tradeAnalysisURL = "http://localhost:8082"
	}

	return &EarningsBigMoneyHandler{
		PolygonAPIKey:    apiKey,
		PolygonBaseURL:   baseURL,
		TradeAnalysisURL: tradeAnalysisURL,
	}
}

// EarningsBigMoneyResponse represents the aggregated response
type EarningsBigMoneyResponse struct {
	Date           string                      `json:"date"`
	TotalTickers   int                         `json:"total_tickers"`
	Results        []EarningsBigMoneyResult    `json:"results"`
	Summary        EarningsBigMoneySummary     `json:"summary"`
}

// EarningsBigMoneyResult represents a single ticker's earnings + big money analysis
type EarningsBigMoneyResult struct {
	Ticker              string  `json:"ticker"`
	Date                string  `json:"date"`
	Time                string  `json:"time,omitempty"`
	EstimatedEPS        *float64 `json:"estimated_eps,omitempty"`
	ActualEPS           *float64 `json:"actual_eps,omitempty"`
	Importance          int     `json:"importance"`
	BigMoneyDirection   string  `json:"big_money_direction"` // "BUYING_PRESSURE", "SELLING_PRESSURE", "NEUTRAL", "ERROR", "NO_DATA"
	NetBigMoneyFlow     *float64 `json:"net_big_money_flow,omitempty"`
	LargeTradesCount    *int    `json:"large_trades_count,omitempty"`
	BuyerInitiatedVol   *float64 `json:"buyer_initiated_volume,omitempty"`
	SellerInitiatedVol  *float64 `json:"seller_initiated_volume,omitempty"`
	AnalysisDate        *string  `json:"analysis_date,omitempty"`
	Error               *string  `json:"error,omitempty"`
}

// EarningsBigMoneySummary provides aggregated statistics
type EarningsBigMoneySummary struct {
	BullishCount    int `json:"bullish_count"`    // BUYING_PRESSURE
	BearishCount    int `json:"bearish_count"`    // SELLING_PRESSURE
	NeutralCount    int `json:"neutral_count"`    // NEUTRAL
	ErrorCount      int `json:"error_count"`      // ERROR or NO_DATA
	TotalAnalyzed   int `json:"total_analyzed"`
}

// TradeAnalysisResponse represents the response from tradeanalysis API
type TradeAnalysisResponse struct {
	Ticker              string         `json:"ticker"`
	StartTime           time.Time      `json:"start_time"`
	EndTime             time.Time      `json:"end_time"`
	AnalysisDate        time.Time      `json:"analysis_date"`
	LargeTradeThreshold float64        `json:"large_trade_threshold"`
	Result              TradeAnalysisResult `json:"result"`
}

// TradeAnalysisResult holds the results from tradeanalysis API
type TradeAnalysisResult struct {
	TotalTrades           int          `json:"total_trades"`
	AvgTradeSize          float64      `json:"avg_trade_size"`
	LargeTradesCount      int          `json:"large_trades_count"`
	NetBigMoneyFlow       float64      `json:"net_big_money_flow"`
	BuyerInitiatedVolume  float64      `json:"buyer_initiated_volume"`
	SellerInitiatedVolume float64      `json:"seller_initiated_volume"`
	Direction             string       `json:"direction"` // "BUYING_PRESSURE", "SELLING_PRESSURE", "NEUTRAL"
}

// GetEarningsWithBigMoney analyzes earnings calendar and big money flow for each ticker
// Query parameters:
//   - date: Date in YYYY-MM-DD format (required) - earnings date
//   - analysis_date: Date to analyze big money flow (default: one trading day before earnings date)
//   - large_trade_threshold: Threshold multiplier for large trades (default: 10.0)
//   - limit: Maximum number of earnings results per date (default: 100, max: 50000)
func (h *EarningsBigMoneyHandler) GetEarningsWithBigMoney(c *gin.Context) {
	if h.PolygonAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Polygon API key not configured. Please set POLYGON_API_KEY environment variable.",
		})
		return
	}

	// Parse query parameters
	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "date query parameter is required (format: YYYY-MM-DD)",
		})
		return
	}

	// Validate date format
	earningsDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid date format. Use YYYY-MM-DD",
		})
		return
	}

	// Get analysis_date (default: one trading day before earnings date)
	analysisDateStr := c.DefaultQuery("analysis_date", "")
	var analysisDate time.Time
	if analysisDateStr != "" {
		analysisDate, err = time.Parse("2006-01-02", analysisDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid analysis_date format. Use YYYY-MM-DD",
			})
			return
		}
	} else {
		// Default: one trading day before earnings date
		analysisDate = earningsDate.AddDate(0, 0, -1)
		// If earnings is on Monday, go back to Friday
		if analysisDate.Weekday() == time.Sunday {
			analysisDate = analysisDate.AddDate(0, 0, -2)
		} else if analysisDate.Weekday() == time.Saturday {
			analysisDate = analysisDate.AddDate(0, 0, -1)
		}
	}

	// Get large_trade_threshold
	largeThreshold := 10.0
	thresholdStr := c.DefaultQuery("large_trade_threshold", "10.0")
	if thresholdStr != "" {
		threshold, err := strconv.ParseFloat(thresholdStr, 64)
		if err == nil && threshold > 0 {
			largeThreshold = threshold
		}
	}

	// Get limit
	limitStr := c.DefaultQuery("limit", "100")
	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 50000 {
				limit = 50000
			}
		}
	}

	// Fetch earnings calendar for the date
	earningsHandler := NewEarningsHandler()
	earnings, err := earningsHandler.fetchEarningsFromPolygon(dateStr, "", nil, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch earnings calendar",
			"details": err.Error(),
		})
		return
	}

	if len(earnings) == 0 {
		c.JSON(http.StatusOK, EarningsBigMoneyResponse{
			Date:         dateStr,
			TotalTickers: 0,
			Results:      []EarningsBigMoneyResult{},
			Summary: EarningsBigMoneySummary{},
		})
		return
	}

	// Analyze big money flow for each ticker concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]EarningsBigMoneyResult, 0, len(earnings))

	// Limit concurrent API calls to avoid overwhelming services
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

	for _, earning := range earnings {
		wg.Add(1)
		go func(e EarningsResult) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := h.analyzeTickerBigMoney(e, analysisDate, largeThreshold)
			
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(earning)
	}

	wg.Wait()

	// Calculate summary
	summary := EarningsBigMoneySummary{
		TotalAnalyzed: len(results),
	}
	for _, r := range results {
		switch r.BigMoneyDirection {
		case "BUYING_PRESSURE":
			summary.BullishCount++
		case "SELLING_PRESSURE":
			summary.BearishCount++
		case "NEUTRAL":
			summary.NeutralCount++
		default:
			summary.ErrorCount++
		}
	}

	response := EarningsBigMoneyResponse{
		Date:         dateStr,
		TotalTickers: len(results),
		Results:      results,
		Summary:      summary,
	}

	c.JSON(http.StatusOK, response)
}

// analyzeTickerBigMoney analyzes big money flow for a single ticker
func (h *EarningsBigMoneyHandler) analyzeTickerBigMoney(earning EarningsResult, analysisDate time.Time, largeThreshold float64) EarningsBigMoneyResult {
	result := EarningsBigMoneyResult{
		Ticker:     earning.Ticker,
		Date:       earning.Date,
		Time:       earning.Time,
		EstimatedEPS: earning.EstimatedEPS,
		ActualEPS:  earning.ActualEPS,
		Importance: earning.Importance,
	}

	// Call tradeanalysis API
	analysisDateStr := analysisDate.Format("2006-01-02")
	url := fmt.Sprintf("%s/api/v1/trade-analysis/%s?start_date=%s&large_trade_threshold=%.2f",
		h.TradeAnalysisURL, earning.Ticker, analysisDateStr, largeThreshold)

	resp, err := http.Get(url)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to call tradeanalysis API: %v", err)
		result.BigMoneyDirection = "ERROR"
		result.Error = &errorMsg
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := fmt.Sprintf("Tradeanalysis API returned status %d: %s", resp.StatusCode, string(bodyBytes))
		result.BigMoneyDirection = "ERROR"
		result.Error = &errorMsg
		return result
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to read tradeanalysis response: %v", err)
		result.BigMoneyDirection = "ERROR"
		result.Error = &errorMsg
		return result
	}

	var tradeAnalysis TradeAnalysisResponse
	if err := json.Unmarshal(body, &tradeAnalysis); err != nil {
		errorMsg := fmt.Sprintf("Failed to parse tradeanalysis response: %v", err)
		result.BigMoneyDirection = "ERROR"
		result.Error = &errorMsg
		return result
	}

	// Populate result
	result.BigMoneyDirection = tradeAnalysis.Result.Direction
	result.NetBigMoneyFlow = &tradeAnalysis.Result.NetBigMoneyFlow
	result.LargeTradesCount = &tradeAnalysis.Result.LargeTradesCount
	result.BuyerInitiatedVol = &tradeAnalysis.Result.BuyerInitiatedVolume
	result.SellerInitiatedVol = &tradeAnalysis.Result.SellerInitiatedVolume
	
	analysisDateFormatted := tradeAnalysis.AnalysisDate.Format("2006-01-02")
	result.AnalysisDate = &analysisDateFormatted

	// Handle case where no trades were found
	if tradeAnalysis.Result.TotalTrades == 0 {
		result.BigMoneyDirection = "NO_DATA"
	}

	return result
}

