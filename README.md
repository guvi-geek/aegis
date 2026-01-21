# AEGIS - Automated Engine for Grading Integrity & Similarity

A sophisticated plagiarism detection system that processes code submissions, performs multi-algorithm similarity analysis, and generates comprehensive plagiarism reports.

## Features

- **Redis Stream Consumer**: Processes submissions asynchronously using consumer groups
- **Multi-Algorithm Detection**: 
  - Winnowing fingerprint similarity
  - Greedy String Tiling (GST) for token similarity
  - AST Merkle hashing for structural similarity
  - CFG feature vector comparison for control flow similarity
- **Progressive Short-Circuit Pipeline**: Optimizes computation by skipping expensive algorithms when early results indicate low similarity
- **Worker Pool**: CPU-based worker pool for parallel processing
- **REST API**: Gin-based HTTP server with JWT authentication and rate limiting
- **Comprehensive Edge Case Handling**: Handles single candidates, missing data, computation failures, etc.

## Prerequisites

- Go 1.24 or higher
- MongoDB
- Redis
- Astra preprocessing service (external API)

## Configuration

The application is configured via environment variables:

### Database
- `MONGO_URI`: MongoDB connection string (default: `mongodb://localhost:27017`)
- `MONGO_DB_NAME`: MongoDB database name (default: `aegis`)

### Redis
- `REDIS_HOST`: Redis address (default: `localhost:6379`)
- `REDIS_PASSWORD`: Redis password (default: empty)
- `REDIS_STREAM_KEY`: Redis stream key (default: `aegis:submissions`)
- `REDIS_CONSUMER_GROUP`: Consumer group name (default: `aegis-consumers`)
- `REDIS_DEAD_LETTER_KEY`: Death queue key (default: `aegis:dead-letter`)

### Astra Service
- `ASTRA_BASE_URL`: Base URL for Astra preprocessing API (required)
- `ASTRA_API_KEY`: API key for Astra service (required)

### Authentication
- `JWT_SECRET`: Secret key for JWT validation (required)
- `JWT_ISSUER`: JWT issuer (default: `aegis`)

### Rate Limiting
- `RATE_LIMIT_RPS`: Requests per second per API key (default: `10.0`)

### Concurrency
- `MAX_CONCURRENT_COMPUTE`: Max concurrent computations (default: `NumCPU`)
- `BATCH_SIZE`: Batch size for pair processing (default: `100`)
- `COMPUTATION_TIMEOUT_MINUTES`: Computation timeout in minutes (default: `30`)

### Test Risk Thresholds
- `TEST_RISK_SAFE`: Safe threshold (default: `0.40`)
- `TEST_RISK_MODERATE`: Moderate threshold (default: `0.60`)
- `TEST_RISK_HIGH`: High threshold (default: `0.80`)
- `TEST_RISK_CRITICAL`: Critical threshold (default: `0.80`)

### Logging
- `LOG_LEVEL`: Log level (default: `info`)
- `SERVER_PORT`: HTTP server port (default: `8080`)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd aegis
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables (see Configuration section)

4. Create MongoDB indexes (see `internal/repository/indexes.md`)

5. Run the application:
```bash
> go run ./cmd

or

> air
```

## API Endpoints

### Health Check
```
GET /health
```
No authentication required.

### Compute Plagiarism
```
POST /api/v1/compute
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "driveId": "string"
}
```

Returns `202 Accepted` immediately and processes computation asynchronously.

## Architecture

The system consists of three main components:

1. **Redis Stream Consumer**: Continuously processes submissions from Redis stream, calls Astra API for preprocessing, and stores artifacts in MongoDB
2. **Gin HTTP Server**: Exposes REST API for triggering plagiarism computations
3. **Plagiarism Computation Engine**: Multi-algorithm similarity detection with worker pools and batch processing

## MongoDB Collections

- `plagiarism_artifacts`: Stores preprocessed code artifacts
- `results`: Stores candidate-wise plagiarism results
- `plagiarism_reports`: Stores overall test plagiarism reports

## Error Handling

- Exponential backoff retry (4 attempts: 1s, 2s, 4s, 8s)
- Death queue for failed messages after 5 retries
- Comprehensive error wrapping and logging
- Graceful shutdown handling

## Observability

OpenTelemetry and Prometheus instrumentation are set up but commented out. See `internal/observability/` for implementation details.
