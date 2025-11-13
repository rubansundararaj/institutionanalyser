package handlers

import (
	"fmt"
	"net/http"

	"time"

	"institutionanalyser/deepsearch"
	"institutionanalyser/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeepSearchHandler struct {
	db *gorm.DB
}

func NewDeepSearchHandler(db *gorm.DB) *DeepSearchHandler {
	return &DeepSearchHandler{db: db}
}

// HandleGetAnalysis returns the latest technical analysis signals for a ticker
func (deepSearchHandler *DeepSearchHandler) HandleGetAnalysis(c *gin.Context) {
	ticker := c.Query("ticker")
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker is required"})
		return
	}

	end_duration := c.Query("end_duration")
	if end_duration == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_duration is required"})
		return
	}

	var signals []models.TechnicalSignal
	result := deepSearchHandler.db.Where("ticker = ? and poly_start_duration = ?", ticker, end_duration).Order("created_at desc").Limit(1).Find(&signals)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"signals": signals})
}

func (deepSearchHandler *DeepSearchHandler) HandleTriggerAnalysis(c *gin.Context) {
	ticker := c.Query("ticker")
	if ticker == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker is required"})
		return
	}

	startDuration := c.Query("start_duration")
	if startDuration == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_duration is required"})
		return
	}

	fmt.Printf("Start Duration: %s\n", startDuration)
	fmt.Printf("Ticker: %s\n", ticker)

	// Parse end_date
	_, err := time.Parse("2006-01-02", startDuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_duration format, use YYYY-MM-DD"})
		return
	}

	// Get user_id from context (set by auth middleware) or query parameter (for system/orchestrator calls)

	// Fallback to query parameter if not in context

	// Add one day for start_date
	//endDate := end.AddDate(0, 0, 1)
	//endDuration := endDate.Format("2006-01-02")

	endDuration := time.Now().Format("2006-01-02")

	// Trigger analysis

	fmt.Printf("Trigger search params: %s - %s\n", startDuration, endDuration)

	//store the deepsearch request in the database
	deepSearchRequest := models.DeepSearchRequest{
		StartDate: startDuration,
		EndDate:   endDuration,
		Ticker:    ticker,
		UserId:    "orchestrator",
	}
	deepSearchHandler.db.Create(&deepSearchRequest)

	svc := deepsearch.NewDeepSearchService(startDuration, endDuration, "minute", 5, ticker, "orchestrator", deepSearchHandler.db)
	err = svc.AnalyseMain()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Analysis triggered successfully"})
}
