package main

import (
	"log"
	"net/http"

	"medcon/internal/api"
	"medcon/internal/auth"
	"medcon/internal/config"
	"medcon/internal/db"
	"medcon/internal/repository"
	"medcon/internal/service"
)

func main() {
	cfg := config.Load()

	mongoClient, err := db.NewMongoClient(cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(mongoClient)
	postRepo := repository.NewPostRepository(mongoClient)
	commentRepo := repository.NewCommentRepository(mongoClient)
	bookingRepo := repository.NewBookingRepository(mongoClient)
	matchRepo := repository.NewMatchRepository(mongoClient)
	chatRepo := repository.NewChatRepository(mongoClient)
	ambulanceRepo := repository.NewAmbulanceRepository(mongoClient)
	vitalsRepo := repository.NewVitalsRepository(mongoClient)

	// Initialize services
	emailService := service.NewEmailService(cfg)
	tokenStore := service.NewPasswordResetTokenStore()
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours, emailService, tokenStore)

	// Initialize services that don't depend on each other first
	userService := service.NewUserService(userRepo, nil)
	postService := service.NewPostService(postRepo, userRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, userRepo)
	chatService := service.NewChatService(chatRepo, userRepo)
	bookingService := service.NewBookingService(bookingRepo, userRepo, chatRepo)

	// Initialize match service (needs email service)
	matchService := service.NewMatchService(matchRepo, userRepo, bookingService, chatService, emailService)

	// Now update user service with match service reference
	userService.SetMatchService(matchService)

	ambulanceService := service.NewAmbulanceService(ambulanceRepo, userRepo)
	vitalsService := service.NewVitalsService(vitalsRepo, userRepo)
	adminService := service.NewAdminService(userRepo, postRepo, commentRepo, bookingRepo, matchRepo, ambulanceRepo)

	// Initialize handlers
	handler := &api.Handler{
		AuthService:      authService,
		UserService:      userService,
		PostService:      postService,
		CommentService:   commentService,
		BookingService:   bookingService,
		MatchService:     matchService,
		ChatService:      chatService,
		AmbulanceService: ambulanceService,
		VitalsService:    vitalsService,
		AdminService:     adminService,
		JWTSecret:        cfg.JWTSecret,
	}

	mux := http.NewServeMux()

	// Public Auth Routes
	mux.HandleFunc("POST /api/auth/register", handler.Register)
	mux.HandleFunc("POST /api/auth/login", handler.Login)
	mux.HandleFunc("POST /api/auth/refresh", handler.RefreshToken)
	mux.HandleFunc("POST /api/auth/forgot-password", handler.RequestPasswordReset)
	mux.HandleFunc("POST /api/auth/reset-password", handler.ResetPassword)

	// Protected User Routes
	mux.Handle("GET /api/me", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.Me)))
	mux.Handle("PATCH /api/me", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.UpdateProfile)))

	// Admin Routes
	adminMiddleware := auth.Middleware(cfg.JWTSecret)
	adminRoleMiddleware := auth.RequireRole(string(config.RoleAdmin))
	mux.Handle("GET /api/admin/stats", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminStats))))
	mux.Handle("GET /api/admin/users", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminListUsers))))
	mux.Handle("GET /api/admin/users/{id}", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminGetUser))))
	mux.Handle("PATCH /api/admin/users/{id}", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminUpdateUser))))
	mux.Handle("DELETE /api/admin/users/{id}", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminDeleteUser))))
	mux.Handle("GET /api/admin/posts", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminListPosts))))
	mux.Handle("PATCH /api/admin/posts/{id}", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminUpdatePost))))
	mux.Handle("DELETE /api/admin/posts/{id}", adminMiddleware(adminRoleMiddleware(http.HandlerFunc(handler.AdminDeletePost))))

	// Forum Routes
	mux.Handle("POST /api/posts", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.CreatePost)))
	mux.Handle("GET /api/posts", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.ListPosts)))
	mux.Handle("GET /api/posts/{id}", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.GetPost)))
	mux.Handle("PATCH /api/posts/{id}", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.UpdatePost)))
	mux.Handle("DELETE /api/posts/{id}", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.DeletePost)))
	mux.Handle("POST /api/posts/{id}/comments", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.CreateComment)))
	mux.Handle("GET /api/posts/{id}/comments", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.ListComments)))
	mux.Handle("PATCH /api/comments/{id}/accept", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.AcceptComment)))
	mux.Handle("POST /api/posts/{id}/view", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.IncrementPostView)))

	// Matching
	mux.Handle("POST /api/match", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.CreateMatch)))
	mux.Handle("GET /api/professionals", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.ListProfessionals)))

	// Bookings
	mux.Handle("POST /api/bookings", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.CreateBooking)))
	mux.Handle("GET /api/bookings", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.ListBookings)))
	mux.Handle("PATCH /api/bookings", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.UpdateBooking)))

	// Chat
	mux.Handle("GET /api/rooms/{roomId}/messages", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.GetMessages)))
	mux.Handle("POST /api/rooms/{roomId}/messages", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.PostMessage)))

	// Ambulance
	mux.Handle("POST /api/ambulance", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.DispatchAmbulance)))
	mux.Handle("GET /api/ambulance", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.ListDispatches)))
	mux.Handle("PATCH /api/ambulance", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.UpdateDispatch)))

	// Vitals
	mux.Handle("POST /api/vitals", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.CreateVitals)))
	mux.Handle("GET /api/vitals", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.ListVitals)))
	mux.Handle("GET /api/vitals/latest", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.GetLatestVitals)))
	mux.Handle("GET /api/vitals/{id}", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.GetVitals)))
	mux.Handle("PATCH /api/vitals/{id}", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.UpdateVitals)))
	mux.Handle("DELETE /api/vitals/{id}", auth.Middleware(cfg.JWTSecret)(http.HandlerFunc(handler.DeleteVitals)))

	corsHandler := withCORS(cfg.FrontendURL, mux)

	log.Printf("[server] listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, corsHandler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func withCORS(frontendURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
