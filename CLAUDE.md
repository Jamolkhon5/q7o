# CLAUDE.md - Q7O Real-Time Communication Backend

## PROJECT OVERVIEW

**Project Type:** Real-time Communication API Backend
**Purpose:** WebRTC-powered video/audio calling platform with contact management and meeting functionality
**Architecture:** Microservice-oriented REST API with WebSocket support
**Primary Language:** Go 1.23.4
**Framework:** Fiber v2 (HTTP framework)
**Real-time:** LiveKit (WebRTC infrastructure) + WebSocket signaling

### Core Features
- User authentication & authorization (JWT-based)
- Real-time video/audio calling with LiveKit integration
- Contact system with friend requests and call restrictions
- Meeting rooms with time-based expiration
- Email verification and notifications
- WebSocket-based call signaling

## ARCHITECTURE

### Layer Structure
```
q7o/
├── cmd/api/                    # Application entry point
├── config/                     # Configuration management
├── internal/                   # Private application code
│   ├── auth/                   # Authentication & authorization
│   ├── call/                   # Call management & signaling
│   ├── contact/                # Contact system & friend requests
│   ├── email/                  # Email service integration
│   ├── meeting/                # Meeting room management
│   ├── user/                   # User profile management
│   └── common/                 # Shared utilities & database
├── pkg/                        # Public packages
│   ├── errors/                 # Error handling utilities
│   └── logger/                 # Logging utilities
├── migrations/                 # Database schema migrations
└── web/                        # Static web assets (if any)
```

### Service Layer Pattern
Each domain follows consistent layering:
- **Handler** - HTTP request handling, validation, response formatting
- **Service** - Business logic, orchestration between repositories
- **Repository** - Data access layer, database operations
- **Models** - Data structures and DTOs

### Dependencies Architecture
- **Database:** PostgreSQL with UUID primary keys
- **Cache:** Redis for sessions, WebSocket connections, and temporary data
- **Real-time:** LiveKit server for WebRTC media handling
- **WebSocket:** Custom hub for call signaling and notifications

## DEVELOPMENT STANDARDS

### Code Style
- **Package Structure:** Domain-driven with clear separation of concerns
- **Naming Convention:** CamelCase for exported items, camelCase for internal
- **Error Handling:** Explicit error returns, no panic in business logic
- **Context Usage:** Always pass context.Context for cancellation support
- **Validation:** go-playground/validator for request validation

### Response Format
All API responses follow consistent structure via `internal/common/response`:
```go
// Success response
return response.Success(c, data)

// Error responses
return response.BadRequest(c, "message")
return response.ValidationError(c, err)
return response.Conflict(c, "message")
return response.InternalError(c, err)
```

### Database Conventions
- UUID for all primary keys
- Snake_case for column names
- Timestamps: `created_at`, `updated_at`, `deleted_at`
- Foreign key pattern: `{table}_id`
- Migration files: `{number}_{description}.up.sql/.down.sql`

## STACK & DEPENDENCIES

### Core Framework & HTTP
- **github.com/gofiber/fiber/v2** v2.52.9 - HTTP framework (Express.js-like)
- **github.com/gofiber/websocket/v2** v2.2.1 - WebSocket support for Fiber

### Database & Storage
- **github.com/lib/pq** v1.10.9 - PostgreSQL driver
- **github.com/redis/go-redis/v9** v9.8.0 - Redis client for caching/sessions
- **github.com/golang-migrate/migrate/v4** v4.18.3 - Database migrations

### Authentication & Security
- **github.com/golang-jwt/jwt/v5** v5.3.0 - JWT token handling
- **golang.org/x/crypto** - Password hashing (bcrypt)

### Real-time Communication
- **github.com/livekit/protocol** v1.39.3 - LiveKit WebRTC integration
- WebSocket signaling hub for call management

### Validation & Utilities
- **github.com/go-playground/validator/v10** v10.27.0 - Request validation
- **github.com/google/uuid** v1.6.0 - UUID generation
- **github.com/joho/godotenv** v1.5.1 - Environment configuration
- **gopkg.in/gomail.v2** - Email sending

### Development Tools
- **github.com/mozillazg/go-unidecode** v0.2.0 - Username normalization

## DEVELOPMENT WORKFLOW

### Quick Start Commands
```bash
# Development setup (creates network, starts LiveKit, app, DB, Redis)
make up

# Individual services
make docker-up        # Start app, PostgreSQL, Redis
make livekit-up      # Start LiveKit server separately
make run             # Run locally without Docker

# Monitoring
make logs            # Application logs
make livekit-logs    # LiveKit server logs

# Cleanup
make down            # Stop all services
make clean           # Full cleanup including volumes
```

### Development Process
1. **Database Changes:** Create migration files in `migrations/`
2. **New Features:** Follow domain structure in `internal/{domain}/`
3. **Testing:** `make test` - runs full test suite
4. **Building:** `make build` - creates binary in `bin/q7o`

### Environment Configuration
Copy `.env.example` to `.env` and configure:
- Database credentials and connection
- Redis connection settings
- JWT secrets (minimum 32 characters)
- LiveKit API keys and endpoints
- SMTP configuration for emails

## CODING GUIDELINES

### New Feature Development
1. **Domain Structure:** Create new domains in `internal/{domain}/`
2. **Layer Order:** Repository → Service → Handler → Routes
3. **Dependency Injection:** Pass dependencies through constructors
4. **Error Handling:** Use explicit error returns and proper HTTP status codes
5. **Validation:** Use struct tags for request validation
6. **Database:** Use transactions for multi-table operations

### WebSocket Integration
```go
// WebSocket hub pattern for real-time features
wsHub := call.NewWSHub(redis)
go wsHub.Run()

// Handler registration
app.Get("/ws/endpoint", websocket.New(handlerFunc))
```

### Authentication Middleware
```go
// Protected routes
authGroup := api.Group("/endpoint", auth.RequireAuth(cfg.JWT))

// Extract user ID in handlers
userID := c.Locals("user_id").(uuid.UUID)
```

### Service Dependencies
- Services can depend on repositories and other services
- Avoid circular dependencies (use interfaces if needed)
- Pass context through all service methods
- Use constructor injection pattern

## FILE ORGANIZATION

### New Components Placement

#### Controllers/Handlers
- Location: `internal/{domain}/handler.go`
- Dependencies: Service layer, validator
- Responsibility: HTTP request/response handling

#### Business Logic
- Location: `internal/{domain}/service.go`
- Dependencies: Repository, other services
- Responsibility: Business rules, orchestration

#### Data Access
- Location: `internal/{domain}/repository.go`
- Dependencies: Database connection
- Responsibility: CRUD operations, queries

#### Data Models
- Location: `internal/{domain}/models.go` or `internal/{domain}/types.go`
- Contains: Request/Response DTOs, database models

#### Database Changes
- Location: `migrations/{number}_{description}.{up|down}.sql`
- Naming: Sequential numbering, descriptive names
- Must include both up and down migrations

#### Shared Utilities
- Common database utilities: `internal/common/`
- Error handling: `pkg/errors/`
- Logging utilities: `pkg/logger/`

### Route Organization
Routes are organized by domain in `cmd/api/main.go`:
```go
// Domain grouping
authGroup := api.Group("/auth")
userGroup := api.Group("/users", auth.RequireAuth(cfg.JWT))
callGroup := api.Group("/calls", auth.RequireAuth(cfg.JWT))
```

### Configuration Management
- Environment variables: `config/config.go`
- Default values provided for development
- Production values via environment variables
- Struct-based configuration with nested types

This project implements a professional WebRTC communication platform with clean architecture, proper error handling, and real-time capabilities suitable for production deployment.