package routes

import (
	"institutionanalyser/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600, // 12 hours
	}))

	deepSearchHandler := handlers.NewDeepSearchHandler(db)
	earningsBigMoneyHandler := handlers.NewEarningsBigMoneyHandler()

	router.GET("/api/v1/deepsearch/analysis", deepSearchHandler.HandleGetAnalysis)
	router.POST("/api/v1/deepsearch/trigger", deepSearchHandler.HandleTriggerAnalysis)
	router.GET("/api/v1/earnings/bigmoney", earningsBigMoneyHandler.GetEarningsWithBigMoney)

}
