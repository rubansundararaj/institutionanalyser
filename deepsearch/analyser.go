package deepsearch

import (
	"errors"
	"fmt"
	"log"
	"math"

	"os"
	"sort"
	"strings"
	"time"

	models "institutionanalyser/models"
	"institutionanalyser/service"

	"github.com/lib/pq"
	"github.com/polygon-io/client-go/rest/iter"
	polygonmodels "github.com/polygon-io/client-go/rest/models"
	chart "github.com/wcharczuk/go-chart/v2"
	"gorm.io/gorm"
)

type EnhancedBar struct {
	Timestamp         time.Time
	Open              float64
	Close             float64
	High              float64
	Low               float64
	Volume            float64
	Transactions      float64
	CumulativeVWAP    float64
	VolumeZScore      float64
	IsDoji            bool
	BearishEngulfing  bool
	BullishEngulfing  bool
	InstitutionalFlow bool
	ATR               float64
	VWAP              float64
}

type DeepSearchService struct {
	//polygonSvc    *service.StockTechnicalService
	startDuration string
	endDuration   string
	timeSpan      string
	multiplier    int
	ticker        string
	userId        string
	db            *gorm.DB
}

func NewDeepSearchService(startDuration, endDuration, timeSpan string, multiplier int, ticker string, userId string, db *gorm.DB) *DeepSearchService {
	return &DeepSearchService{
		startDuration: startDuration,
		endDuration:   endDuration,
		timeSpan:      timeSpan,
		multiplier:    multiplier,
		ticker:        ticker,
		userId:        userId,
		db:            db,
	}
}

func (s *DeepSearchService) StartDuration() string {
	return s.startDuration
}
func (s *DeepSearchService) EndDuration() string {
	return s.endDuration
}
func (s *DeepSearchService) TimeSpan() string {
	return s.timeSpan
}
func (s *DeepSearchService) Multiplier() int {
	return s.multiplier
}
func (s *DeepSearchService) Ticker() string {
	return s.ticker
}

func (s *DeepSearchService) UserId() string {
	return s.userId
}

func (s *DeepSearchService) AnalyseWithTechnicals() error {
	// Minute-by-minute data
	svc := service.NewStockTechnicalService(s.ticker)
	bars, err := svc.GetPolygonAggregate(s.timeSpan, s.startDuration, s.endDuration, s.multiplier)

	if err != nil {
		return err
	}

	enhancedBars := enhanceData(bars)

	if len(enhancedBars) == 0 {
		return errors.New("no enhanced bars")
	}

	signals := generateSignals(enhancedBars)

	// Store signals in the database if there are any
	if len(signals) > 0 && len(enhancedBars) > 0 {
		s.storeSignalsInDatabase(enhancedBars, signals, s.ticker)
	}

	// Daily technicals

	sma, _ := svc.FetchSMA(20)
	rsi, _ := svc.FetchRSI(14)
	macd, _ := svc.FetchMACD(12, 26, 9)

	// Latest values
	latestBar := enhancedBars[len(enhancedBars)-1]
	latestSMA := sma.Results.Values[0].Value
	latestRSI := rsi.Results.Values[0].Value
	latestMACD := macd.Results.Values[0]

	fmt.Printf("\nLatest %s Analysis:\n", s.ticker)
	fmt.Printf("Price: %.2f | VWAP: %.2f | ATR: %.2f\n", latestBar.Close, latestBar.CumulativeVWAP, latestBar.ATR)
	fmt.Printf("SMA(20): %.2f | RSI: %.2f | MACD: %.2f (Signal: %.2f)\n", latestSMA, latestRSI, latestMACD.Value, latestMACD.Signal)

	// Decision logic
	if latestBar.Close < latestBar.CumulativeVWAP && latestRSI < 30 && latestMACD.Value > latestMACD.Signal {
		fmt.Println("Decision: BUY - Cheap price, oversold, bullish momentum.")
	} else if latestBar.Close > latestBar.CumulativeVWAP && latestRSI > 70 && latestMACD.Value < latestMACD.Signal {
		fmt.Println("Decision: SELL - Expensive price, overbought, bearish momentum.")
	} else if len(enhancedBars) > 1 && latestBar.ATR > enhancedBars[len(enhancedBars)-2].ATR*1.5 {
		fmt.Println("Decision: HOLD/STRADDLE - Volatility spiking, no clear trend.")
	} else {
		fmt.Println("Decision: HOLD - No strong signals.")
	}

	printSignals(signals)
	// winRate := evaluateSignals(enhancedBars, signals)
	// fmt.Printf("Signal Win Rate: %.2f%%\n", winRate*100)

	return nil
}

func (s *DeepSearchService) AnalyseMain() error {
	// Fetch data from Polygon
	svc := service.NewStockTechnicalService(s.ticker)

	bars, err := svc.GetPolygonAggregate(s.timeSpan, s.startDuration, s.endDuration, s.multiplier)
	if err != nil {
		log.Fatal(err)
	}

	// Enhance data with technical indicators
	enhancedBars := enhanceData(bars)

	if len(enhancedBars) == 0 {
		return errors.New("no enhanced bars")
	}

	// Generate trading signals
	signals := generateSignals(enhancedBars)

	// Store signals in the database if there are any
	if len(signals) > 0 && len(enhancedBars) > 0 {
		err := s.storeSignalsInDatabase(enhancedBars, signals, s.ticker)

		if err != nil {
			return err
		}

	} else {
		return errors.New("no signals or enhanced bars")
	}

	// Print and visualize results
	printSignals(signals)

	return nil
}

func enhanceData(bars *iter.Iter[polygonmodels.Agg]) []EnhancedBar {
	var enhanced []EnhancedBar
	var (
		cumulativeVolume float64
		cumulativeVWAP   float64
		volumes          []float64
		ranges           []float64
		volumePerTrade   []float64
	)

	for bars.Next() {
		agg := bars.Item()
		millis := time.Time(agg.Timestamp).UnixMilli() // Convert Millis to int64
		timestamp := time.UnixMilli(millis)
		// Convert Agg to EnhancedBar
		bar := EnhancedBar{
			Timestamp:    timestamp,
			Open:         agg.Open,
			Close:        agg.Close,
			High:         agg.High,
			Low:          agg.Low,
			Volume:       agg.Volume,
			Transactions: float64(agg.Transactions),
			VWAP:         agg.VWAP,
		}

		// Calculate cumulative VWAP
		cumulativeVolume += bar.Volume
		cumulativeVWAP += bar.Volume * bar.VWAP
		if cumulativeVolume > 0 {
			bar.CumulativeVWAP = cumulativeVWAP / cumulativeVolume
		}

		// Calculate volatility metrics
		barRange := bar.High - bar.Low
		ranges = append(ranges, barRange)
		bar.ATR = calculateATR(ranges, 14)

		// Volume analysis
		volumes = append(volumes, bar.Volume)
		bar.VolumeZScore = volumeZScore(volumes, 14)

		// Candlestick patterns
		body := math.Abs(bar.Close - bar.Open)
		bar.IsDoji = (body/barRange < 0.1) && barRange > 0

		if len(enhanced) > 0 {
			prevBar := enhanced[len(enhanced)-1]
			bar.BearishEngulfing = bar.Close < bar.Open &&
				bar.Open > prevBar.Close &&
				bar.Close < prevBar.Open

			bar.BullishEngulfing = bar.Close > bar.Open &&
				bar.Open < prevBar.Close &&
				bar.Close > prevBar.Open
		}

		// Institutional flow
		if bar.Transactions > 0 {
			vpt := bar.Volume / bar.Transactions
			volumePerTrade = append(volumePerTrade, vpt)
			bar.InstitutionalFlow = vpt > quantile(volumePerTrade, 0.9)
		}

		enhanced = append(enhanced, bar)
	}

	return enhanced
}

func generateSignals(bars []EnhancedBar) []string {
	var signals []string
	for i, bar := range bars {
		if i < 3 {
			continue // Skip first few bars to ensure enough data for indicators
		}

		// Doji pattern
		if bar.IsDoji {
			signals = append(signals, fmt.Sprintf("%s STRADDLE: Doji Pattern - Indecision Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.Close))
		}

		// Engulfing patterns
		if bar.BearishEngulfing {
			signals = append(signals, fmt.Sprintf("%s PUT: Bearish Engulfing - Reversal Likely Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.Close))
		}
		if bar.BullishEngulfing {
			signals = append(signals, fmt.Sprintf("%s CALL: Bullish Engulfing - Reversal Likely Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.Close))
		}

		// Volume-based signals
		if bar.VolumeZScore > 2 && bar.Close < bar.Open {
			signals = append(signals, fmt.Sprintf("%s PUT: Volume Spike + Price Drop (%.2f) - Institutional Selling Likely Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.Volume, bar.Close))
		}
		if bar.VolumeZScore > 2 && bar.Close > bar.Open {
			signals = append(signals, fmt.Sprintf("%s CALL: Volume Spike + Institutional Flow (%.2f) - Institutional Buying Likely Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.Volume, bar.Close))
		}
		if i > 0 && bar.ATR > bars[i-1].ATR*1.5 {
			signals = append(signals, fmt.Sprintf("%s STRADDLE: Volatility Expansion (ATR %.2f) - Institutional Activity Likely Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.ATR, bar.Close))
		}

		// New directional flow check
		if bar.InstitutionalFlow && bar.Close > bar.Open && bar.VolumeZScore > 1 {
			signals = append(signals, fmt.Sprintf("%s UP: Institutional Buying Detected (Volume %.0f) - Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.Volume, bar.Close))
		} else if bar.InstitutionalFlow && bar.Close < bar.Open && bar.VolumeZScore > 1 {
			signals = append(signals, fmt.Sprintf("%s DOWN: Institutional Selling Detected (Volume %.0f) - Closing price (%.2f)",
				bar.Timestamp.Format("15:04"), bar.Volume, bar.Close))
		}
	}

	return signals
}

func getFinalDecisionFromSignals(signals []string) string {
	counts := map[string]int{
		"BUY":      0,
		"SELL":     0,
		"STRADDLE": 0,
		"HOLD":     0,
	}

	for _, signal := range signals {
		s := strings.ToUpper(signal)
		switch {
		case strings.Contains(s, "CALL") || strings.Contains(s, "UP") || strings.Contains(s, "BUY"):
			counts["BUY"]++
		case strings.Contains(s, "PUT") || strings.Contains(s, "DOWN") || strings.Contains(s, "SELL"):
			counts["SELL"]++
		case strings.Contains(s, "STRADDLE"):
			counts["STRADDLE"]++
		default:
			counts["HOLD"]++
		}
	}

	// Find the decision with the highest count
	final := "HOLD"
	maxCount := counts["HOLD"]
	for k, v := range counts {
		if v > maxCount {
			final = k
			maxCount = v
		}
	}

	return final
}

// storeSignalsInDatabase stores the technical signals in the PostgreSQL database
func (s *DeepSearchService) storeSignalsInDatabase(bars []EnhancedBar, signals []string, ticker string) error {
	if len(bars) == 0 || len(signals) == 0 {
		return errors.New("no bars or signals")
	}

	// Get the first and last bar to determine the time range
	firstBar := bars[0]
	lastBar := bars[len(bars)-1]

	finalDecision := getFinalDecisionFromSignals(signals)

	// Create a new TechnicalSignal record
	technicalSignal := models.TechnicalSignal{
		StartDate:    firstBar.Timestamp,
		EndDate:      lastBar.Timestamp,
		Interval:     "minute", // Assuming we're working with minute data
		WindowSize:   len(bars),
		Ticker:       ticker,
		AnalysisType: "technical",
		Signals:      pq.StringArray(signals),

		PolyStartDuration: s.StartDuration(),
		PolyEndDuration:   s.EndDuration(),
		PolyTimeSpan:      s.TimeSpan(),
		PolyMultiplier:    s.Multiplier(),
		FinalDecision:     finalDecision,
		UserId:            s.UserId(),
	}

	fmt.Println("--------------------------------")
	fmt.Println("Final Decision: ", finalDecision)
	fmt.Println("Technical Signal: ", technicalSignal)
	fmt.Println("--------------------------------")

	// Store in the database
	result := s.db.Create(&technicalSignal)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// evaluateSignals calculates the win rate of CALL and PUT signals based on the next bar's price movement
func evaluateSignals(bars []EnhancedBar, signals []string) float64 {
	if len(bars) < 2 || len(signals) == 0 {
		return 0.0
	}

	var wins, total int
	signalIndex := 0

	// Iterate through bars, aligning signals with their corresponding bar
	for i := 0; i < len(bars)-1 && signalIndex < len(signals); i++ {
		bar := bars[i]
		nextBar := bars[i+1]
		// Check if this bar has a signal
		if signalIndex < len(signals) && bar.Timestamp.Format("15:04") == strings.Split(signals[signalIndex], " ")[0] {
			signal := signals[signalIndex]
			if strings.Contains(signal, "CALL") {
				total++
				if nextBar.Close > bar.Close {
					wins++
				}
			} else if strings.Contains(signal, "PUT") {
				total++
				if nextBar.Close < bar.Close {
					wins++
				}
			}
			// Note: STRADDLE isn't directional, so we skip it for win rate
			signalIndex++
		}
	}

	if total == 0 {
		return 0.0
	}
	return float64(wins) / float64(total)
}

// Helper functions
func calculateATR(ranges []float64, period int) float64 {
	if len(ranges) < period {
		return 0.0
	}
	sum := 0.0
	for _, r := range ranges[len(ranges)-period:] {
		sum += r
	}
	return sum / float64(period)
}

func volumeZScore(volumes []float64, lookback int) float64 {
	if len(volumes) < lookback || lookback == 0 {
		return 0.0
	}

	start := len(volumes) - lookback
	if start < 0 {
		start = 0
	}
	window := volumes[start:]

	mean := 0.0
	for _, v := range window {
		mean += v
	}
	mean /= float64(len(window))

	stdDev := 0.0
	for _, v := range window {
		stdDev += math.Pow(v-mean, 2)
	}
	stdDev = math.Sqrt(stdDev / float64(len(window)))

	if stdDev == 0 {
		return 0.0
	}
	return (volumes[len(volumes)-1] - mean) / stdDev
}

func quantile(data []float64, q float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	sort.Float64s(data)
	index := q * float64(len(data)-1)
	return data[int(index)]
}

func printSignals(signals []string) {
	fmt.Println("\nTRADING SIGNALS:")
	for _, signal := range signals {
		fmt.Println("→", signal)
	}
}

func plotChart(bars []EnhancedBar) {
	var timeSeries []time.Time
	var prices, vwap []float64

	for _, bar := range bars {
		timeSeries = append(timeSeries, bar.Timestamp)
		prices = append(prices, bar.Close)
		vwap = append(vwap, bar.CumulativeVWAP)
	}

	graph := chart.Chart{
		Title: "SPY Intraday Analysis",
		XAxis: chart.XAxis{
			Name:           "Time",
			ValueFormatter: chart.TimeHourValueFormatter,
		},
		YAxis: chart.YAxis{
			Name: "Price",
		},
		Series: []chart.Series{
			chart.TimeSeries{
				Name:    "Price",
				XValues: timeSeries,
				YValues: prices,
			},
			chart.TimeSeries{
				Name:    "VWAP",
				XValues: timeSeries,
				YValues: vwap,
				Style: chart.Style{
					StrokeColor:     chart.ColorBlue,
					StrokeDashArray: []float64{5.0, 5.0},
				},
			},
		},
	}

	f, _ := os.Create("intraday_chart.png")
	defer f.Close()
	graph.Render(chart.PNG, f)
	fmt.Println("\nChart saved as intraday_chart.png")
}
