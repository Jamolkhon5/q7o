package main

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"os"
	"os/signal"
	"q7o/config"
	"q7o/internal/auth"
	"q7o/internal/call"
	"q7o/internal/common/database"
	"q7o/internal/email"
	"q7o/internal/meeting"
	"q7o/internal/user"
	"q7o/pkg/logger"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load config
	cfg := config.Load()

	// Initialize logger
	log := logger.New(cfg.AppEnv)

	// Initialize database
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()

	// Run migrations
	log.Info("Running database migrations...")
	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}
	log.Info("Migrations completed successfully")

	// Initialize Redis
	redis, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatal("Failed to connect to Redis: ", err)
	}
	defer redis.Close()

	// Initialize WebSocket Hub для звонков
	wsHub := call.NewWSHub(redis)
	go wsHub.Run()
	log.Info("WebSocket Hub started")

	// Initialize services
	emailService := email.NewService(cfg.SMTP)

	// Initialize repositories
	userRepo := user.NewRepository(db)
	authRepo := auth.NewRepository(db, redis)
	callRepo := call.NewRepository(db)
	meetingRepo := meeting.NewRepository(db)

	// Initialize services (передаем wsHub в callService)
	userService := user.NewService(userRepo, emailService)
	authService := auth.NewService(authRepo, userRepo, emailService, cfg.JWT)
	callService := call.NewService(callRepo, userRepo, cfg.LiveKit, redis, wsHub) // ДОБАВЛЕН wsHub
	meetingService := meeting.NewService(meetingRepo, userRepo, cfg.LiveKit, redis)

	// Start cleanup goroutine for expired meetings
	go meetingService.CleanupExpiredMeetings(context.Background())

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
	}))

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Routes
	api := app.Group("/api/v1")

	// Auth routes
	authHandler := auth.NewHandler(authService)
	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/refresh", authHandler.RefreshToken)
	authGroup.Post("/verify-email", authHandler.VerifyEmail)
	authGroup.Post("/resend-verification", authHandler.ResendVerification)
	authGroup.Post("/logout", auth.RequireAuth(cfg.JWT), authHandler.Logout)
	authGroup.Post("/validate", auth.RequireAuth(cfg.JWT), authHandler.ValidateToken)
	authGroup.Get("/check-username", authHandler.CheckUsername)
	authGroup.Post("/check-username", authHandler.CheckUsername)
	authGroup.Post("/suggest-usernames", authHandler.SuggestUsernames)
	// User routes
	userHandler := user.NewHandler(userService)
	userGroup := api.Group("/users", auth.RequireAuth(cfg.JWT))
	userGroup.Get("/me", userHandler.GetMe)
	userGroup.Put("/me", userHandler.UpdateProfile)
	userGroup.Get("/search", userHandler.SearchUsers)
	userGroup.Get("/:id", userHandler.GetUser)

	// Call routes (передаем wsHub в handler)
	callHandler := call.NewHandler(callService, wsHub)
	callGroup := api.Group("/calls", auth.RequireAuth(cfg.JWT))
	callGroup.Post("/token", callHandler.GetCallToken)
	callGroup.Post("/initiate", callHandler.InitiateCall)
	callGroup.Post("/answer", callHandler.AnswerCall)
	callGroup.Post("/reject", callHandler.RejectCall)
	callGroup.Post("/end", callHandler.EndCall)
	callGroup.Get("/history", callHandler.GetCallHistory)

	// Meeting routes
	meetingHandler := meeting.NewHandler(meetingService)
	meetingGroup := api.Group("/meetings")

	// Public endpoints
	meetingGroup.Post("/validate", meetingHandler.ValidateMeetingCode)
	meetingGroup.Post("/join", meetingHandler.JoinMeeting)

	// Authenticated endpoints
	meetingAuthGroup := meetingGroup.Group("", auth.RequireAuth(cfg.JWT))
	meetingAuthGroup.Post("/create", meetingHandler.CreateMeeting)
	meetingAuthGroup.Post("/join-auth", meetingHandler.JoinMeetingAuth)
	meetingAuthGroup.Post("/:id/leave", meetingHandler.LeaveMeeting)
	meetingAuthGroup.Post("/:id/end", meetingHandler.EndMeeting)
	meetingAuthGroup.Get("/:id/participants", meetingHandler.GetMeetingParticipants)
	meetingAuthGroup.Put("/:id/participant-status", meetingHandler.UpdateParticipantStatus)
	meetingAuthGroup.Get("/history", meetingHandler.GetUserMeetings)

	// WebSocket for call signaling (обновлено для работы с wsHub)
	app.Get("/ws/call", websocket.New(func(c *websocket.Conn) {
		callHandler.HandleWebSocket(c, wsHub)
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Graceful shutdown
	go func() {
		if err := app.Listen(":" + cfg.AppPort); err != nil {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Info("Server shutdown complete")
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
