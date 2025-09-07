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
	"q7o/internal/contact"
	"q7o/internal/email"
	"q7o/internal/meeting"
	"q7o/internal/push"
	"q7o/internal/settings"
	"q7o/internal/upload"
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

	// Создаем папку для загрузки файлов
	avatarDir := "./uploads/avatars"
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		log.Fatal("Failed to create avatar directory: ", err)
	}

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

	// Initialize email service
	emailService := email.NewService(cfg.SMTP)

	// Initialize upload service
	uploadConfig := config.LoadUploadConfig()
	uploadService := upload.NewService(uploadConfig)

	// Initialize repositories
	userRepo := user.NewRepository(db)
	authRepo := auth.NewRepository(db, redis)
	callRepo := call.NewRepository(db)
	meetingRepo := meeting.NewRepository(db)
	contactRepo := contact.NewRepository(db)
	settingsRepo := settings.NewRepository(db)
	pushRepo := push.NewRepository(db)

	// Initialize services
	userService := user.NewService(userRepo, emailService, uploadService)
	authService := auth.NewService(authRepo, userRepo, emailService, cfg.JWT)
	meetingService := meeting.NewService(meetingRepo, userRepo, cfg.LiveKit, redis)
	settingsService := settings.NewService(settingsRepo)
	pushService := push.NewService(pushRepo, cfg.Push)

	// Contact service без зависимости от call service
	contactService := contact.NewService(contactRepo, userRepo, wsHub)

	// Call service с contact service и push service
	callService := call.NewService(callRepo, userRepo, cfg.LiveKit, cfg.JWT, redis, wsHub)
	// 🔥 КРИТИЧЕСКИ ВАЖНО: Устанавливаем зависимости в callService ПЕРЕД созданием handlers
	callService.SetContactService(contactService)
	callService.SetPushService(pushService)

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

	// User routes - ПЕРЕДАЕМ contactService
	userHandler := user.NewHandler(userService, contactService)
	userGroup := api.Group("/users", auth.RequireAuth(cfg.JWT))
	userGroup.Get("/me", userHandler.GetMe)
	userGroup.Put("/me", userHandler.UpdateProfile)
	userGroup.Post("/me/avatar", userHandler.UploadAvatar)
	userGroup.Delete("/me/avatar", userHandler.DeleteAvatar)
	userGroup.Put("/me/password", userHandler.ChangePassword)
	userGroup.Get("/search", userHandler.SearchUsers)
	userGroup.Get("/:id", userHandler.GetUser)

	// 🚀 КРИТИЧЕСКИ ВАЖНО: Call handler создается ПОСЛЕ установки всех зависимостей
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

	// Contact routes - ВСЕ ЭНДПОИНТЫ КОТОРЫЕ НУЖНЫ ФРОНТЕНДУ
	contactHandler := contact.NewHandler(contactService)
	contactGroup := api.Group("/contacts", auth.RequireAuth(cfg.JWT))

	// Эти эндпоинты нужны фронтенду для ContactsScreen
	contactGroup.Get("/", contactHandler.GetContacts)                // GET /api/v1/contacts
	contactGroup.Get("/requests", contactHandler.GetContactRequests) // GET /api/v1/contacts/requests

	// Остальные эндпоинты
	contactGroup.Post("/request", contactHandler.SendContactRequest)
	contactGroup.Post("/accept/:request_id", contactHandler.AcceptContactRequest)
	contactGroup.Post("/reject/:request_id", contactHandler.RejectContactRequest)
	contactGroup.Delete("/:contact_id", contactHandler.RemoveContact)
	contactGroup.Get("/check/:user_id", contactHandler.CheckContact)

	// Settings routes
	settingsHandler := settings.NewHandler(settingsService)
	settingsGroup := api.Group("/settings", auth.RequireAuth(cfg.JWT))
	settingsGroup.Get("/", settingsHandler.GetSettings)
	settingsGroup.Put("/", settingsHandler.UpdateSettings)
	settingsGroup.Delete("/", settingsHandler.DeleteSettings)

	// Push notification routes
	pushHandler := push.NewHandler(pushService)
	pushGroup := api.Group("/push", auth.RequireAuth(cfg.JWT))
	pushGroup.Post("/register", pushHandler.RegisterToken)
	pushGroup.Post("/deactivate", pushHandler.DeactivateToken)

	// Static files для аватаров
	app.Static("/uploads", "./uploads")

	// WebSocket for call signaling
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
