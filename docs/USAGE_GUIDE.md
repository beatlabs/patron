# Patron Usage Guide

This guide provides detailed usage examples and best practices for building microservices with Patron.

## Table of Contents

- [Getting Started](#getting-started)
- [HTTP Services](#http-services)
- [gRPC Services](#grpc-services)
- [Message Processing](#message-processing)
- [Database Integration](#database-integration)
- [Observability](#observability)
- [Reliability Patterns](#reliability-patterns)
- [Production Deployment](#production-deployment)

## Getting Started

### Basic Service Structure

Every Patron service follows this basic structure:

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/beatlabs/patron"
)

func main() {
    // 1. Create service
    service, err := patron.New("my-service", "1.0.0")
    if err != nil {
        slog.Error("Failed to create service", "error", err)
        return
    }
    
    // 2. Create components
    components := []patron.Component{
        // Add your components here
    }
    
    // 3. Run service
    err = service.Run(context.Background(), components...)
    if err != nil {
        slog.Error("Service failed", "error", err)
    }
}
```

### Service Configuration

Configure your service with options:

```go
service, err := patron.New("my-service", "1.0.0",
    // Enable JSON logging for production
    patron.WithJSONLogger(),
    
    // Add custom log fields
    patron.WithLogFields(
        slog.String("environment", os.Getenv("ENV")),
        slog.String("region", os.Getenv("AWS_REGION")),
    ),
    
    // Handle configuration reload on SIGHUP
    patron.WithSIGHUP(func() {
        slog.Info("Reloading configuration...")
        // Reload your configuration here
    }),
)
```

## HTTP Services

### Simple REST API

```go
func createHTTPComponent() (patron.Component, error) {
    // Define handlers
    healthHandler := func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    }
    
    usersHandler := func(w http.ResponseWriter, r *http.Request) {
        userID := r.PathValue("id")
        // Business logic here
        user := getUserByID(r.Context(), userID)
        json.NewEncoder(w).Encode(user)
    }
    
    // Create routes
    var routes patronhttp.Routes
    routes.Append(patronhttp.NewRoute("GET /health", healthHandler))
    routes.Append(patronhttp.NewRoute("GET /users/{id}", usersHandler))
    
    rr, err := routes.Result()
    if err != nil {
        return nil, err
    }
    
    // Create router
    rt, err := router.New(router.WithRoutes(rr...))
    if err != nil {
        return nil, err
    }
    
    return patronhttp.New(rt)
}
```

### Advanced HTTP with Middleware

```go
import (
    "github.com/beatlabs/patron/component/http/middleware"
)

func createAdvancedHTTPComponent() (patron.Component, error) {
    // Custom middleware
    authMiddleware := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if !isValidToken(token) {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
    
    // Protected handler
    protectedHandler := func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromToken(r.Header.Get("Authorization"))
        json.NewEncoder(w).Encode(user)
    }
    
    // Create routes with middleware
    var routes patronhttp.Routes
    routes.Append(patronhttp.NewRoute("GET /profile", protectedHandler,
        patronhttp.WithMiddleware(authMiddleware)))
    
    rr, err := routes.Result()
    if err != nil {
        return nil, err
    }
    
    // Create router with global middleware
    rt, err := router.New(
        router.WithRoutes(rr...),
        router.WithMiddleware(middleware.NewLogging()),
        router.WithMiddleware(middleware.NewRecovery()),
        router.WithMiddleware(middleware.NewCORS()),
    )
    if err != nil {
        return nil, err
    }
    
    return patronhttp.New(rt)
}
```

### Request/Response Handling

```go
func createAPIComponent() (patron.Component, error) {
    // POST handler with JSON body
    createUserHandler := func(w http.ResponseWriter, r *http.Request) {
        var req CreateUserRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }
        
        // Validate request
        if err := validateCreateUserRequest(req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Create user
        user, err := createUser(r.Context(), req)
        if err != nil {
            log.FromContext(r.Context()).Error("Failed to create user", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(user)
    }
    
    // GET handler with query parameters
    listUsersHandler := func(w http.ResponseWriter, r *http.Request) {
        limit := 10
        if l := r.URL.Query().Get("limit"); l != "" {
            if parsed, err := strconv.Atoi(l); err == nil {
                limit = parsed
            }
        }
        
        offset := 0
        if o := r.URL.Query().Get("offset"); o != "" {
            if parsed, err := strconv.Atoi(o); err == nil {
                offset = parsed
            }
        }
        
        users, err := listUsers(r.Context(), limit, offset)
        if err != nil {
            log.FromContext(r.Context()).Error("Failed to list users", "error", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "users":  users,
            "limit":  limit,
            "offset": offset,
        })
    }
    
    var routes patronhttp.Routes
    routes.Append(patronhttp.NewRoute("POST /users", createUserHandler))
    routes.Append(patronhttp.NewRoute("GET /users", listUsersHandler))
    
    rr, err := routes.Result()
    if err != nil {
        return nil, err
    }
    
    rt, err := router.New(router.WithRoutes(rr...))
    if err != nil {
        return nil, err
    }
    
    return patronhttp.New(rt)
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func validateCreateUserRequest(req CreateUserRequest) error {
    if req.Name == "" {
        return errors.New("name is required")
    }
    if req.Email == "" {
        return errors.New("email is required")
    }
    return nil
}
```

## gRPC Services

### Basic gRPC Service

```go
import (
    "github.com/beatlabs/patron/component/grpc"
    pb "your-project/proto/user"
)

func createGRPCComponent() (patron.Component, error) {
    // Create gRPC component
    cmp, err := grpc.New(9090,
        grpc.WithReflection(), // Enable for development
    )
    if err != nil {
        return nil, err
    }
    
    // Register service implementation
    pb.RegisterUserServiceServer(cmp.Server(), &userServiceImpl{})
    
    return cmp, nil
}

type userServiceImpl struct {
    pb.UnimplementedUserServiceServer
}

func (s *userServiceImpl) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    logger := log.FromContext(ctx)
    logger.Info("GetUser called", "user_id", req.Id)
    
    // Business logic
    user, err := getUserByID(ctx, req.Id)
    if err != nil {
        logger.Error("Failed to get user", "error", err)
        return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
    }
    
    if user == nil {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    
    return &pb.User{
        Id:    user.ID,
        Name:  user.Name,
        Email: user.Email,
    }, nil
}

func (s *userServiceImpl) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
    logger := log.FromContext(ctx)
    logger.Info("CreateUser called", "name", req.Name)
    
    // Validate request
    if req.Name == "" {
        return nil, status.Errorf(codes.InvalidArgument, "name is required")
    }
    
    // Create user
    user, err := createUser(ctx, CreateUserRequest{
        Name:  req.Name,
        Email: req.Email,
    })
    if err != nil {
        logger.Error("Failed to create user", "error", err)
        return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
    }
    
    return &pb.User{
        Id:    user.ID,
        Name:  user.Name,
        Email: user.Email,
    }, nil
}
```
