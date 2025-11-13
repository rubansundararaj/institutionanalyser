package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// EarningsHandler handles earnings-related API endpoints
type EarningsHandler struct {
	PolygonAPIKey string
	PolygonBaseURL string
}

// NewEarningsHandler creates a new earnings handler
func NewEarningsHandler() *EarningsHandler {
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("POLYGON_API_KEY") // Fallback, but will error if not set
	}
	
	baseURL := os.Getenv("POLYGON_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.polygon.io"
	}
	
	return &EarningsHandler{
		PolygonAPIKey: apiKey,
		PolygonBaseURL: baseURL,
	}
}

// PolygonEarningsResponse represents the response from Polygon API
type PolygonEarningsResponse struct {
	Status    string    `json:"status"`
	RequestID string    `json:"request_id"`
	Count     int       `json:"count"`
	Results   []EarningsResult `json:"results"`
}

// EarningsResult represents a single earnings announcement
type EarningsResult struct {
	Ticker         string  `json:"ticker"`
	Date           string  `json:"date"`
	ActualEPS      *float64 `json:"actual_eps,omitempty"`
	ActualRevenue  *float64 `json:"actual_revenue,omitempty"`
	EstimatedEPS   *float64 `json:"estimated_eps,omitempty"`
	EstimatedRevenue *float64 `json:"estimated_revenue,omitempty"`
	Importance     int     `json:"importance"`
	Time           string  `json:"time,omitempty"`
	Updated        string  `json:"updated,omitempty"`
}

// GetEarnings retrieves earnings announcements within a given time frame
// Query parameters:
//   - start_date: Start date in YYYY-MM-DD format (required)
//   - end_date: End date in YYYY-MM-DD format (required)
//   - ticker: Optional filter by ticker symbol
//   - importance: Optional filter by importance (0-5)
//   - limit: Maximum number of results per date (default: 100, max: 50000)
func (h *EarningsHandler) GetEarnings(c *gin.Context) {
	if h.PolygonAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Polygon API key not configured. Please set POLYGON_API_KEY environment variable.",
		})
		return
	}

	// Parse query parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	ticker := c.Query("ticker")
	importanceStr := c.Query("importance")
	limitStr := c.DefaultQuery("limit", "100")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start_date and end_date query parameters are required (format: YYYY-MM-DD)",
		})
		return
	}

	// Validate date format
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid start_date format. Use YYYY-MM-DD",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid end_date format. Use YYYY-MM-DD",
		})
		return
	}

	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "end_date must be after or equal to start_date",
		})
		return
	}

	// Validate date range (limit to reasonable range to avoid too many API calls)
	daysDiff := int(endDate.Sub(startDate).Hours() / 24)
	if daysDiff > 90 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Date range cannot exceed 90 days",
		})
		return
	}

	// Parse limit
	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 50000 {
				limit = 50000 // Polygon API max limit
			}
		}
	}

	// Parse importance if provided
	var importance *int
	if importanceStr != "" {
		if parsedImportance, err := strconv.Atoi(importanceStr); err == nil {
			if parsedImportance >= 0 && parsedImportance <= 5 {
				importance = &parsedImportance
			}
		}
	}

	// Collect earnings from all dates in the range
	var allEarnings []EarningsResult
	currentDate := startDate
	
	for !currentDate.After(endDate) {
		dateStr := currentDate.Format("2006-01-02")
		
		earnings, err := h.fetchEarningsFromPolygon(dateStr, ticker, importance, limit)
		if err != nil {
			// Log error but continue with other dates
			fmt.Printf("Error fetching earnings for %s: %v\n", dateStr, err)
		} else {
			allEarnings = append(allEarnings, earnings...)
		}
		
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Remove duplicates based on ticker and date combination
	uniqueEarnings := removeDuplicateEarnings(allEarnings)

	c.JSON(http.StatusOK, gin.H{
		"data": uniqueEarnings,
		"count": len(uniqueEarnings),
		"start_date": startDateStr,
		"end_date": endDateStr,
		"date_range_days": daysDiff + 1,
	})
}

// fetchEarningsFromPolygon makes a request to Polygon API for a specific date
func (h *EarningsHandler) fetchEarningsFromPolygon(date, ticker string, importance *int, limit int) ([]EarningsResult, error) {
	// Build URL
	url := fmt.Sprintf("%s/benzinga/v1/earnings?date=%s&limit=%d&apiKey=%s", 
		h.PolygonBaseURL, date, limit, h.PolygonAPIKey)
	
	if ticker != "" {
		url += fmt.Sprintf("&ticker=%s", ticker)
	}
	
	if importance != nil {
		url += fmt.Sprintf("&importance=%d", *importance)
	}

	// Make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Polygon API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Polygon API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var polygonResp PolygonEarningsResponse
	if err := json.Unmarshal(body, &polygonResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if polygonResp.Status != "OK" {
		return nil, fmt.Errorf("Polygon API returned non-OK status: %s", polygonResp.Status)
	}

	return polygonResp.Results, nil
}

// removeDuplicateEarnings removes duplicate earnings entries based on ticker and date
func removeDuplicateEarnings(earnings []EarningsResult) []EarningsResult {
	seen := make(map[string]bool)
	var unique []EarningsResult

	for _, e := range earnings {
		key := fmt.Sprintf("%s-%s", e.Ticker, e.Date)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, e)
		}
	}

	return unique
}

