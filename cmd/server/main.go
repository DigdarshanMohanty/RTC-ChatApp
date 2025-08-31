package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chat-backend/config"
	"chat-backend/handlers"
	"chat-backend/repository"
	"chat-backend/services"
	"chat-backend/ws"
)

// loggingMiddleware adds request logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions, Sec-WebSocket-Key, Sec-WebSocket-Version, Upgrade, Connection")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// --- config/env ---
	cfg := config.Load()

	log.Printf("Starting chat server on port %s", cfg.Port)

	// --- repos (in-memory for now) ---
	userRepo := repository.NewInMemoryUserRepo()
	messageRepo := repository.NewInMemoryMessageRepo()
	chatRepo := repository.NewInMemoryChatRepo()

	// --- create default room ---
	defaultRoom, err := chatRepo.Create("General")
	if err != nil {
		log.Printf("Warning: could not create default room: %v", err)
	} else {
		log.Printf("Created default room: %s (ID: %d)", defaultRoom.Name, defaultRoom.ID)
	}

	// --- websocket hub ---
	hub := ws.NewHub()
	go hub.Run()

	// --- services ---
	authSvc := services.NewAuthService(userRepo, &cfg)
	msgSvc := services.NewMessageService(messageRepo, chatRepo, userRepo, hub, &cfg)
	chatSvc := services.NewChatService(chatRepo, userRepo)

	// --- handlers ---
	authH := handlers.NewAuthHandler(authSvc)
	msgH := handlers.NewMessageHandler(msgSvc, authSvc)
	chatH := handlers.NewChatHandler(hub, chatSvc, authSvc, msgSvc)

	// --- mux and routes ---
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// API routes
	mux.HandleFunc("/api/register", authH.Register)
	mux.HandleFunc("/api/login", authH.Login)
	mux.HandleFunc("/api/rooms", chatH.WithAuth(chatH.Rooms))         // GET list rooms
	mux.HandleFunc("/api/rooms/create", chatH.WithAuth(chatH.Create)) // POST create room
	mux.HandleFunc("/api/messages", msgH.WithAuth(msgH.ListMessages)) // GET ?roomId=1
	mux.HandleFunc("/ws", chatH.WS)                                   // WS ?roomId=1&token=<token>

	// Apply middleware
	handler := withCORS(loggingMiddleware(mux))

	// --- server setup ---
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- graceful shutdown ---
	go func() {
		log.Printf("Chat server running on http://localhost:%s", cfg.Port)
		log.Printf("WS endpoint: ws://localhost:%s/ws?roomId=<id>&token=<token>", cfg.Port)
		log.Printf("Default room ID: %d", defaultRoom.ID)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
