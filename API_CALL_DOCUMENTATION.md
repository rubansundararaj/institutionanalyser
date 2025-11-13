# API Call Documentation: `/api/v1/deepsearch/trigger`

## Endpoint Details

**Method:** `POST`  
**Route:** `/api/v1/deepsearch/trigger`  
**Base URL:** `http://localhost:8080` (or configured PORT from environment)

## Required Query Parameters

1. **`ticker`** (string, required)
   - Stock ticker symbol
   - Example: `AAPL`, `TSLA`, `SPY`

2. **`start_duration`** (string, required)
   - Date in `YYYY-MM-DD` format
   - Example: `2025-01-15`
   - Note: The API automatically calculates `end_duration` as `start_duration + 1 day`

## Example API Calls

### Using cURL
```bash
curl -X POST "http://localhost:8080/api/v1/deepsearch/trigger?ticker=AAPL&start_duration=2025-01-15"
```

### Using JavaScript (fetch)
```javascript
fetch('http://localhost:8080/api/v1/deepsearch/trigger?ticker=AAPL&start_duration=2025-01-15', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  }
})
.then(response => response.json())
.then(data => console.log(data))
.catch(error => console.error('Error:', error));
```

### Using Python (requests)
```python
import requests

url = "http://localhost:8080/api/v1/deepsearch/trigger"
params = {
    "ticker": "AAPL",
    "start_duration": "2025-01-15"
}

response = requests.post(url, params=params)
print(response.json())
```

## Response Format

### Success Response (200 OK)
```json
{
  "message": "Analysis triggered successfully"
}
```

### Error Responses

**400 Bad Request** - Missing required parameter:
```json
{
  "error": "Ticker is required"
}
```
or
```json
{
  "error": "start_duration is required"
}
```

**400 Bad Request** - Invalid date format:
```json
{
  "error": "Invalid start_duration format, use YYYY-MM-DD"
}
```

**500 Internal Server Error** - Analysis failure:
```json
{
  "error": "Error message from analysis service"
}
```

## What the Endpoint Does

1. Validates required query parameters (`ticker` and `start_duration`)
2. Parses and validates the `start_duration` date format
3. Calculates `end_duration` as `start_duration + 1 day`
4. Retrieves `user_id` from the request context (⚠️ **See Important Note below**)
5. Creates a `DeepSearchRequest` record in the database
6. Triggers the deep search analysis service with:
   - Time span: `"minute"`
   - Multiplier: `5`
   - The provided ticker and date range
7. Returns success or error response

## Important Notes

### ⚠️ Authentication Issue

The handler attempts to retrieve `user_id` from the Gin context using:
```go
userId, _ := c.Get("user_id")
```

**However, there is no authentication middleware configured** in the routes setup. This means:
- If `user_id` is not set in the context, `userId` will be `nil`
- The code then attempts to convert it to a string: `userId.(string)`
- **This will cause a panic** if `user_id` is not present

### Recommended Fix

You should either:
1. **Add authentication middleware** to set `user_id` in the context before the handler runs
2. **Make `user_id` optional** or provide a default value
3. **Add validation** to check if `user_id` exists before using it

Example fix for option 3:
```go
userId, exists := c.Get("user_id")
if !exists {
    userId = "anonymous" // or return an error
}
```

## CORS Configuration

The API has CORS enabled with the following configuration:
- **Allowed Origins:** `http://localhost:3000`
- **Allowed Methods:** `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`
- **Allowed Headers:** `Origin`, `Content-Type`, `Accept`, `Authorization`
- **Allow Credentials:** `true`

If calling from a different origin, you may need to update the CORS configuration in `routes/routes.go`.

## Environment Variables Required

- `DATABASE_URL` - PostgreSQL connection string (required)
- `PORT` - Server port (default: `8080`)
- `GIN_MODE` - Gin framework mode (default: `release`)
- `POLYGON_API_KEY` - Required by the underlying analysis service

## Related Endpoints

- `GET /api/v1/deepsearch/analysis` - Retrieve analysis results
  - Query params: `ticker`, `end_duration`

