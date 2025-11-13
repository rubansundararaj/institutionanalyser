# Institution Analyser API

A REST API server for querying unusual option activity data from the OTM Analyser database.

## Features

- RESTful API endpoints for querying unusual option activities
- Advanced filtering and sorting capabilities
- Pagination support
- Statistics and analytics endpoints
- CORS enabled for cross-origin requests
- Connection pooling for optimal database performance

## Prerequisites

- Go 1.23 or higher
- PostgreSQL database (shared with OTM Analyser project)
- Access to the `unusual_option_activities` table

## Installation

1. **Clone or navigate to the project directory:**
   ```bash
   cd /Users/ruban/Documents/algotrading/options/institutionanalyser
   ```

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Create a `.env` file** (copy from `.env.example`):
   ```bash
   cp .env.example .env
   ```

4. **Configure your `.env` file:**
   ```env
   DATABASE_URL=postgres://username:password@localhost:5432/otmanalyser?sslmode=disable
   PORT=8080
   GIN_MODE=release
   ```

## Running the Server

```bash
go run main.go
```

Or build and run:

```bash
go build -o institutionanalyser .
./institutionanalyser
```

The server will start on port 8080 (or the port specified in your `.env` file).

## API Endpoints

### Health Check
```
GET /health
```
Returns server health status.

### Get All Activities
```
GET /api/v1/activities
```

**Query Parameters:**
- `ticker` - Filter by ticker symbol (e.g., `AAPL`)
- `contract_type` - Filter by contract type (`call` or `put`)
- `min_volume` - Minimum volume threshold (e.g., `100000`)
- `min_otm` - Minimum OTM percentage (e.g., `10.0`)
- `limit` - Maximum number of results (default: 100, max: 1000)
- `offset` - Pagination offset (default: 0)
- `sort` - Sort field: `volume`, `otm_percentage`, `captured_at` (default: `volume`)
- `order` - Sort order: `asc` or `desc` (default: `desc`)

**Example:**
```bash
curl "http://localhost:8080/api/v1/activities?ticker=AAPL&min_volume=100000&limit=50"
```

### Get Activity by ID
```
GET /api/v1/activities/:id
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/activities/1"
```

### Get Activities by Ticker
```
GET /api/v1/activities/ticker/:ticker
```

**Query Parameters:**
- `limit` - Maximum number of results (default: 100, max: 1000)

**Example:**
```bash
curl "http://localhost:8080/api/v1/activities/ticker/AAPL?limit=50"
```

### Get High Volume Activities
```
GET /api/v1/activities/high-volume
```

**Query Parameters:**
- `min_volume` - Minimum volume threshold (default: 100000)
- `limit` - Maximum number of results (default: 100, max: 1000)

**Example:**
```bash
curl "http://localhost:8080/api/v1/activities/high-volume?min_volume=500000&limit=100"
```

### Get Activities by Date Range
```
GET /api/v1/activities/date-range
```

**Query Parameters:**
- `start_date` - Start date in YYYY-MM-DD format (required)
- `end_date` - End date in YYYY-MM-DD format (required)
- `limit` - Maximum number of results (default: 100, max: 1000)

**Example:**
```bash
curl "http://localhost:8080/api/v1/activities/date-range?start_date=2025-01-01&end_date=2025-01-31"
```

### Get Statistics
```
GET /api/v1/activities/stats
```

Returns aggregate statistics about unusual option activities.

**Example:**
```bash
curl "http://localhost:8080/api/v1/activities/stats"
```

**Response:**
```json
{
  "data": {
    "total_activities": 12345,
    "total_volume": 50000000,
    "avg_volume": 4050.5,
    "max_volume": 1000000,
    "unique_tickers": 500,
    "call_count": 6500,
    "put_count": 5845,
    "avg_otm_percentage": 15.2,
    "latest_capture_time": "2025-01-15T10:30:00Z"
  }
}
```

## Response Format

All endpoints return JSON responses in the following format:

**Success Response:**
```json
{
  "data": [...],
  "pagination": {
    "total": 1000,
    "limit": 100,
    "offset": 0,
    "count": 100
  }
}
```

**Error Response:**
```json
{
  "error": "Error message",
  "details": "Detailed error information"
}
```

## Database Connection Pooling

The API uses connection pooling for optimal database performance. Configure pool settings in your `.env` file:

```env
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=25
DB_CONN_MAX_LIFETIME_MINUTES=5
DB_CONN_MAX_IDLE_TIME_MINUTES=10
```

## CORS

CORS is enabled by default to allow cross-origin requests. All origins are allowed. Modify the CORS middleware in `main.go` if you need to restrict access.

## Development

### Running in Development Mode

Set `GIN_MODE=debug` in your `.env` file for detailed logging and error messages.

### Building

```bash
go build -o institutionanalyser .
```

## Dependencies

- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [GORM](https://gorm.io/) - ORM for database operations
- [godotenv](https://github.com/joho/godotenv) - Environment variable management

## License

MIT

