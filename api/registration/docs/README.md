# DePIN Provider Registration API

The Provider Registration API enables compute resource providers to register with the DePIN network, authenticate, and manage their provider profiles.

## Overview

This API provides the following functionality:
- Provider registration and onboarding
- JWT-based authentication and authorization
- Provider profile management
- Token refresh and session management
- Health monitoring

## API Endpoints

### Authentication Flow

1. **Register Provider** (`POST /api/v1/registration/register`)
   - Register a new provider account
   - Receive API key for subsequent authentication
   - Provider status starts as "pending" until approved

2. **Authenticate** (`POST /api/v1/registration/auth`)
   - Exchange provider ID and API key for JWT token
   - Token expires after 24 hours (configurable)

3. **Access Protected Endpoints**
   - Include JWT token in Authorization header: `Bearer <token>`
   - Token automatically validates provider status and permissions

### Core Endpoints

#### Registration
```http
POST /api/v1/registration/register
Content-Type: application/json

{
  "name": "ACME Compute Provider",
  "email": "admin@acme.com",
  "organization": "ACME Corp",
  "public_key": "-----BEGIN PUBLIC KEY-----\n...",
  "endpoints": [
    {
      "type": "api",
      "url": "https://api.acme.com",
      "protocol": "https",
      "secure": true
    }
  ],
  "metadata": {
    "region": "us-west-2",
    "supported_formats": ["ONNX", "TensorRT"]
  },
  "terms": true
}
```

#### Authentication
```http
POST /api/v1/registration/auth
Content-Type: application/json

{
  "provider_id": "123e4567-e89b-12d3-a456-426614174000",
  "api_key": "pk_1234567890abcdef..."
}
```

#### Profile Management
```http
GET /api/v1/registration/profile
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## Authentication

The API uses JWT-based authentication with the following flow:

1. Register provider account and receive API key
2. Authenticate with provider ID and API key to receive JWT token
3. Include JWT token in Authorization header for protected endpoints
4. Refresh token before expiration to maintain session

### Security Features

- **RSA Public Key Validation**: Minimum 2048-bit keys required
- **API Key Hashing**: Keys stored using bcrypt with configurable cost
- **JWT Tokens**: HS256 signed tokens with configurable expiration
- **Rate Limiting**: Configurable request limits per IP address
- **CORS Protection**: Configurable allowed origins and headers
- **Security Headers**: Comprehensive security headers for production

## Validation Rules

### Provider Name
- Required, 3-100 characters
- Alphanumeric, spaces, hyphens, underscores allowed

### Email
- Required, valid RFC 5322 format
- Must be unique across all providers

### Public Key
- Required, RSA public key in PEM format
- Minimum 2048-bit key size for security
- Supports both PKIX and PKCS1 formats

### Endpoints
- At least one endpoint required
- Valid URL format with scheme and host
- Supported protocols: http, https, grpc, grpc+tls, websocket, wss
- Port validation (1-65535)
- Security flag consistency with protocol

## Error Handling

The API returns structured error responses with the following format:

```json
{
  "error": "Bad Request",
  "message": "Invalid request format",
  "code": 400,
  "validation_errors": [
    {
      "field": "email",
      "message": "Invalid email format",
      "code": "INVALID_EMAIL"
    }
  ],
  "timestamp": "2024-08-24T12:00:00Z",
  "request_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

### HTTP Status Codes

- **200 OK**: Successful request
- **201 Created**: Resource created successfully
- **400 Bad Request**: Invalid request data or validation errors
- **401 Unauthorized**: Authentication required or failed
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource already exists
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server-side error

## Rate Limiting

The API implements rate limiting to prevent abuse:

- **Default Limit**: 60 requests per minute per IP address
- **Configurable**: Adjustable through configuration
- **Headers**: Rate limit information included in response headers
- **Backoff**: Clients should implement exponential backoff for 429 responses

## Configuration

The service can be configured through environment variables or YAML files:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  cors:
    allowed_origins: ["*"]

jwt:
  secret: "your-secret-key"
  expiry_hours: 24

rate_limit:
  requests_per_minute: 60
  enabled: true
```

### Environment Variables

All configuration options can be overridden using environment variables with the `PROVIDER_REGISTRY_` prefix:

- `PROVIDER_REGISTRY_SERVER_PORT=8080`
- `PROVIDER_REGISTRY_JWT_SECRET=your-secret`
- `PROVIDER_REGISTRY_RATE_LIMIT_ENABLED=true`

## Development

### Running the Service

```bash
# With default config
./provider-registry

# With custom config
./provider-registry -config config.yaml

# With environment variables
PROVIDER_REGISTRY_LOG_LEVEL=debug ./provider-registry
```

### Health Check

The service provides a health check endpoint for monitoring:

```http
GET /health
GET /api/v1/registration/health
```

Returns:
```json
{
  "status": "healthy",
  "service": "provider-registry",
  "timestamp": "2024-08-24T12:00:00Z",
  "version": "1.0.0"
}
```

## Integration

### With Kubernetes

The service is designed to run in Kubernetes with:
- Service discovery through DNS
- ConfigMaps for configuration
- Secrets for sensitive data (JWT secret, etc.)
- Health checks for liveness/readiness probes

### With Other Services

- **GPU Discovery Service**: Registered providers can be queried for available resources
- **Resource Allocation**: Authentication tokens used for resource requests
- **Monitoring**: Structured logging for observability
- **Database**: Repository interface ready for database integration

## Security Considerations

### Production Deployment

1. **Change Default JWT Secret**: Use a strong, random secret in production
2. **HTTPS Only**: Configure TLS termination at load balancer or ingress
3. **CORS Configuration**: Restrict allowed origins to known domains
4. **Rate Limiting**: Tune limits based on expected traffic patterns
5. **Monitoring**: Implement comprehensive logging and alerting
6. **Key Rotation**: Plan for JWT secret rotation strategy

### Network Security

- Deploy behind API gateway or load balancer
- Use network policies to restrict inter-service communication
- Implement Web Application Firewall (WAF) for additional protection
- Regular security assessments and penetration testing