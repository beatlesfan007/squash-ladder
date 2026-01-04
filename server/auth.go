package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// key for context value
type authContextKey string

const (
	userIDKey    authContextKey = "userID"
	authErrorKey authContextKey = "authError"
)

var (
	jwksSet  keyfunc.Keyfunc
	jwksOnce sync.Once
)

// AuthInterceptor verifies the JWT token from Supabase
// It does NOT block requests, but populates the context with either the userID or an error.
// Individual handlers must call GetUserIDFromContext to enforce authentication.
func AuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 1. Get Configuration
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	supabaseURL := os.Getenv("SUPABASE_URL")

	if jwtSecret == "" && supabaseURL == "" {
		return nil, status.Error(codes.Internal, "Missing JWT Secret or Supabase URL configuration")
	}

	// Initialize JWKS if Supabase URL is available
	jwksOnce.Do(func() {
		if supabaseURL != "" {
			fmt.Printf("Initializing JWKS from Supabase URL: %s\n", supabaseURL)
			jwksURL := fmt.Sprintf("%s/auth/v1/.well-known/jwks.json", supabaseURL)
			var err error
			jwksSet, err = keyfunc.NewDefault([]string{jwksURL})
			if err != nil {
				fmt.Printf("Failed to initialize JWKS: %v\n", err)
			} else {
				fmt.Println("JWKS initialized successfully")
			}
		} else {
			fmt.Println("SUPABASE_URL not set, JWKS initialization skipped")
		}
	})

	var authErr error
	var userID string

	// 2. Extract Token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		authErr = status.Error(codes.Unauthenticated, "No metadata provided")
		goto Continue
	}

	{
		authHeader := md["authorization"]
		if len(authHeader) == 0 {
			authErr = status.Error(codes.Unauthenticated, "No Authorization header provided")
			goto Continue
		}

		tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
		if tokenString == authHeader[0] {
			authErr = status.Error(codes.Unauthenticated, "Invalid Authorization header format")
			goto Continue
		}

		// 3. Parse and Validate Token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Check if it's HMAC (HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
				if jwtSecret == "" {
					return nil, fmt.Errorf("HS256 signing method used but SUPABASE_JWT_SECRET not configured")
				}
				return []byte(jwtSecret), nil
			}

			// Check if it's ECDSA (ES256)
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); ok {
				if jwksSet == nil {
					return nil, fmt.Errorf("ES256 signing method used but JWKS not initialized (check SUPABASE_URL)")
				}
				return jwksSet.Keyfunc(token)
			}

			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		})

		if err != nil {
			authErr = status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
			goto Continue
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 4. Extract User ID (subject)
			sub, ok := claims["sub"].(string)
			if !ok {
				authErr = status.Error(codes.Unauthenticated, "Token missing subject claim")
				goto Continue
			}
			userID = sub
		} else {
			authErr = status.Error(codes.Unauthenticated, "Invalid token claims")
			goto Continue
		}
	}

Continue:
	newCtx := ctx
	if authErr != nil {
		newCtx = context.WithValue(ctx, authErrorKey, authErr)
	} else if userID != "" {
		newCtx = context.WithValue(ctx, userIDKey, userID)
	}

	return handler(newCtx, req)
}

// GetUserIDFromContext retrieves the user ID from the context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	// 1. Check for specific auth error stored by interceptor
	if err, ok := ctx.Value(authErrorKey).(error); ok {
		return "", err
	}

	// 2. Check for userID
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "No user ID in context")
	}
	return userID, nil
}
