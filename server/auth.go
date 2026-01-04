package server

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// key for context value
type authContextKey string

const (
	userIDKey authContextKey = "userID"
)

// AuthInterceptor verifies the JWT token from Supabase
func AuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 1. Get Secret
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if jwtSecret == "" {
		return nil, status.Error(codes.Internal, "Missing JWT Secret configuration")
	}

	// 2. Extract Token
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "No metadata provided")
	}

	authHeader := md["authorization"]
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "No Authorization header provided")
	}

	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	if tokenString == authHeader[0] {
		return nil, status.Error(codes.Unauthenticated, "Invalid Authorization header format")
	}

	// 3. Parse and Validate Token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// 4. Extract User ID (subject)
		sub, ok := claims["sub"].(string)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "Token missing subject claim")
		}

		// 5. Inject into Context
		newCtx := context.WithValue(ctx, userIDKey, sub)
		return handler(newCtx, req)
	}

	return nil, status.Error(codes.Unauthenticated, "Invalid token claims")
}

// GetUserIDFromContext retrieves the user ID from the context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "No user ID in context")
	}
	return userID, nil
}
