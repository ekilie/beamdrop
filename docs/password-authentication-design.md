# Beam Drop Password Authentication System Design

## Overview

This document outlines the design for implementing a password authentication system in Beam Drop. The system will allow users to set a password via CLI flag and protect most endpoints while keeping certain public endpoints accessible.

## Current State Analysis

### CLI Interface

- Password flag `-p` is already implemented in `cmd/beam/main.go`
- Password is passed through `config.Flags` struct to the server
- Basic structure exists but no authentication logic is implemented

### Database Layer

- `pkg/db/config.go` already has a `Config` model with `Password` field
- AutoMigrate in `pkg/db/migrate.go` includes `Config` model
- Database infrastructure is ready for password storage

### Server Architecture

- `beam/server/server.go` has basic password flag detection
- Routes are defined in `beam/server/routes.go`
- Middleware hook exists in `ServeHTTP` method

### Frontend

- `PasswordDialog` component already exists in frontend
- UI infrastructure for password input is implemented
- Basic authentication flow UI is in place

## Design Architecture

### 1. Password Management Flow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Startup   │    │   Password      │    │   Database      │
│                 │    │   Processing    │    │   Storage       │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ beamdrop -p="x" │───▶│ Hash Password   │───▶│ Store Hash      │
│                 │    │ with bcrypt     │    │ in Config Table │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 2. Authentication Middleware Flow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Request  │    │   Auth Check    │    │   Route Handler │
│                 │    │                 │    │                 │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ Check Headers   │───▶│ Validate Token/ │───▶│ Process Request │
│ or Session      │    │ Session         │    │ if Authorized   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Reject with   │
                       │   401/403       │
                       └─────────────────┘
```

### 3. Session Management Strategy

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Login Success  │    │   JWT Token     │    │   Client        │
│                 │    │   Generation    │    │   Storage       │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ Password Valid  │───▶│ Create JWT with │───▶│ Store in        │
│                 │    │ Expiration+JTI  │    │ HttpOnly cookie │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Implementation Design

### 1. Database Schema Extensions

#### Config Table Enhancement

```sql
CREATE TABLE server_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    password_hash TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### Session Table (Optional for stateful sessions)

```sql
CREATE TABLE user_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_token TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2. Backend Components

#### A. Password Service (`pkg/auth/password.go`)

```go
type PasswordService interface {
    SetPassword(password string) error
    ValidatePassword(password string) bool
    IsPasswordSet() bool
    GenerateToken() (string, error)
    ValidateToken(token string) bool
}
```

#### B. Auth Middleware (`pkg/auth/middleware.go`)

```go
type AuthMiddleware struct {
    passwordService PasswordService
    publicRoutes    []string
}

func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc
func (m *AuthMiddleware) IsPublicRoute(path string) bool
```

#### C. Auth Handler (`beam/server/handlers/auth.go`)

```go
type AuthHandler struct {
    passwordService PasswordService
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) ValidateSession(w http.ResponseWriter, r *http.Request)
```

### 3. Server Integration

#### Modified Server Structure

```go
type Server struct {
    sharedDir       string
    flags           config.Flags
    mux             *http.ServeMux
    authMiddleware  *auth.AuthMiddleware
    passwordService auth.PasswordService
}
```

#### Route Protection Strategy

```go
// Public routes (no authentication required)
var PublicRoutes = []string{
    "/",           // Landing page
    "/health",     // Health check
    "/ready",      // Readiness check
    "/auth/login", // Login endpoint
    "/static/",    // Static assets
}

// Protected routes (authentication required)
var ProtectedRoutes = []string{
    "/files",
    "/download",
    "/upload",
    "/move",
    "/copy",
    "/mkdir",
    "/rename",
    "/write",
    "/search",
    "/star",
    "/starred",
    "/stats",
    "/ws/stats",
}
```

### 4. Authentication Flow

#### Startup Flow

1. Server starts with `-p="password"` flag
2. Password is hashed using bcrypt
3. Hash is stored in database `Config` table
4. Auth middleware is initialized with password service
5. Routes are wrapped with authentication middleware

#### Login Flow

1. User accesses protected route
2. If no valid session/token, redirect to login
3. User submits password via `/auth/login`
4. Password is validated against stored hash
5. On success, JWT token is generated and returned
6. Token is stored in client (localStorage/sessionStorage)
7. Subsequent requests include token in headers

#### Request Flow

1. Client makes request to protected route
2. Auth middleware checks for Authorization header
3. Token is validated (signature, expiration)
4. If valid, request proceeds to handler
5. If invalid, 401/403 response is returned

### 5. Frontend Integration

#### Authentication State Management

```typescript
interface AuthState {
  isAuthenticated: boolean;
  token: string | null;
  isLoading: boolean;
}
```

#### Login Component Enhancement

- Modify existing `PasswordDialog` to handle actual login API
- Add token storage and management
- Implement automatic token refresh
- Handle authentication errors and redirects

#### HTTP Client Enhancement

- Add Authorization header to all requests
- Handle 401 responses (redirect to login)
- Implement token refresh logic

### 6. Security Considerations

#### Password Security

- Use bcrypt with appropriate work factor (12+)
- Implement password complexity requirements if needed
- Store only hashed passwords, never plaintext

#### Token Security

- Use JWT with strong secret key (random 256-bit, generated per process)
- Implement reasonable expiration times (24 hours)
- Each token includes a unique JTI (JWT ID) for revocation
- Tokens are revoked on logout via in-memory JTI blocklist
- Validate token signature, expiration, and revocation status on each request

#### Session Security

- CSRF protection via double-submit cookie (`beamdrop_csrf` cookie + `X-CSRF-Token` header)
- Tokens stored in `HttpOnly`, `SameSite=Strict` cookies (not localStorage)
- Session timeout via JWT expiration (24 hours)
- Token revocation cleanup runs every 10 minutes

### 7. Configuration Options

#### Extended Flags Structure

```go
type Flags struct {
    SharedDir    string
    NoQR         bool
    Port         int
    Help         bool
    Password     string
    TokenExpiry  time.Duration  // New: JWT token expiration
    SessionStore string         // New: "memory" or "database"
}
```

#### Environment Variables

```bash
BEAMDROP_JWT_SECRET=your-secret-key
BEAMDROP_TOKEN_EXPIRY=24h
BEAMDROP_SESSION_STORE=memory
```

## API Endpoints

### Authentication Endpoints

```
POST /auth/login
    Request: { "password": "string" }
    Response: { "token": "jwt-token", "expires_in": 86400 }

POST /auth/logout
    Headers: Authorization: Bearer jwt-token
    Response: { "message": "Logged out successfully" }

GET /auth/validate
    Headers: Authorization: Bearer jwt-token
    Response: { "valid": true, "expires_at": "2026-01-17T12:00:00Z" }

GET /auth/status
    Response: { "password_required": true|false }
```

### Modified File Endpoints

All existing file endpoints remain the same but now require authentication:

```
Authorization: Bearer <jwt-token>
```

## Error Handling

### HTTP Status Codes

- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: Valid token but insufficient permissions
- `422 Unprocessable Entity`: Invalid password format
- `429 Too Many Requests`: Rate limiting on login attempts

### Error Response Format

```json
{
  "error": "authentication_required",
  "message": "Valid authentication token required",
  "details": {
    "login_url": "/auth/login"
  }
}
```

## Migration Strategy

### Phase 1: Backend Implementation

1. Implement password service and hashing
2. Add authentication middleware
3. Create auth handlers
4. Modify server initialization

### Phase 2: Frontend Integration

1. Enhance PasswordDialog component
2. Implement token management
3. Add authentication state management
4. Update HTTP client with auth headers

### Phase 3: Testing & Security

1. Implement comprehensive tests
2. Security audit and penetration testing
3. Performance testing with auth overhead
4. Documentation updates

### Phase 4: Optional Enhancements

1. Add password change functionality
2. Implement user management (multiple users)
3. Add 2FA support
4. OAuth2 integration

## Testing Strategy

### Unit Tests

- Password hashing and validation
- JWT token generation and validation
- Auth middleware logic
- Route protection verification

### Integration Tests

- Full authentication flow
- Token expiration handling
- Public route access
- Protected route blocking

### Security Tests

- Brute force protection
- Token tampering detection
- Session hijacking prevention
- CSRF protection verification

## Monitoring & Logging

### Auth Events to Log

- Password set/changed events
- Login attempts (success/failure)
- Token generation and validation
- Authentication failures with reasons
- Session timeouts and cleanup

### Metrics to Track

- Authentication success/failure rates
- Token usage patterns
- Session duration statistics
- Failed authentication attempts by IP

## Deployment Considerations

### Production Settings

- Use strong JWT secrets (environment variables)
- Enable HTTPS for token security
- Configure appropriate token expiration
- Set up monitoring and alerting

### Backup & Recovery

- Backup encrypted password hashes
- Document password recovery procedures
- Plan for emergency access scenarios

## Conclusion

This design provides a robust, secure, and user-friendly authentication system for Beam Drop. It builds upon the existing infrastructure while adding comprehensive security features. The phased implementation approach allows for gradual rollout and testing, ensuring stability and security throughout the deployment process.

The design balances security with usability, providing a simple password-based authentication system that can be extended in the future for more advanced authentication methods like OAuth2 or multi-factor authentication.
