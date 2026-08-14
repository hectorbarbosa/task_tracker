package server

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"task_tracker/internal/config"
	"task_tracker/internal/handler"
	"task_tracker/internal/middleware"
	"task_tracker/internal/repository"
	"task_tracker/internal/service"

	_ "task_tracker/docs/swagger"
)

// Server wires together the HTTP router, middlewares, and handlers.
type Server struct {
	cfg        *config.Config
	router     *gin.Engine
	httpServer *http.Server
	db         *sql.DB
	rdb        *redis.Client
}

// New constructs the server with all routes registered.
func New(cfg *config.Config) (*Server, error) {
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Open database connection
	db, err := sql.Open("mysql", cfg.Database.DSN())
	if err != nil {
		return nil, err
	}

	// Ping database to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr(),
	})

	// Ping Redis to verify connection
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := rdb.Ping(ctx2).Err(); err != nil {
		return nil, err
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	cacheRepo := repository.NewCacheRepository(rdb)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	teamService := service.NewTeamService(teamRepo, userRepo)
	taskService := service.NewTaskService(taskRepo, teamRepo, cacheRepo)
	commentService := service.NewCommentService(commentRepo, taskRepo, teamRepo)
	statsService := service.NewStatsService(statsRepo, teamRepo)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger())

	// Custom validator: surface binding errors with field names.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v // available for custom registrations
	}

	// Health check (unauthenticated).
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes.
	public := router.Group("/api/v1")
	authH := handler.NewAuthHandler(authService)
	public.POST("/register", authH.Register)
	public.POST("/login", authH.Login)

	// Protected routes.
	protected := router.Group("/api/v1")
	protected.Use(middleware.Auth(cfg.JWT.Secret))
	{
		// Teams.
		teamH := handler.NewTeamHandler(teamService)
		protected.POST("/teams", teamH.CreateTeam)
		protected.GET("/teams", teamH.ListTeams)
		protected.POST("/teams/:id/invite", teamH.Invite)

		// Tasks.
		taskH := handler.NewTaskHandler(taskService)
		protected.POST("/tasks", taskH.Create)
		protected.GET("/tasks", taskH.List)
		protected.PUT("/tasks/:id", taskH.Update)
		protected.GET("/tasks/:id/history", taskH.History)

		// Comments.
		commentH := handler.NewCommentHandler(commentService)
		protected.POST("/tasks/:id/comments", commentH.Create)
		protected.GET("/tasks/:id/comments", commentH.List)

		// Stats.
		statsH := handler.NewStatsHandler(statsService)
		protected.GET("/teams/:team_id/stats", statsH.Get)
	}

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		cfg:        cfg,
		router:     router,
		httpServer: httpServer,
		db:         db,
		rdb:        rdb,
	}, nil
}

// Run starts the HTTP server and blocks until SIGINT/SIGTERM, then shuts down
// gracefully with a 5-second timeout.
func (s *Server) Run() error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("starting server on %s (env=%s)", s.httpServer.Addr, s.cfg.Server.Env)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("received signal %v, shutting down", sig)
	case err := <-errCh:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	// Close database connection
	if err := s.db.Close(); err != nil {
		log.Printf("error closing database: %v", err)
	}

	// Close Redis connection
	if err := s.rdb.Close(); err != nil {
		log.Printf("error closing redis: %v", err)
	}

	return nil
}

// requestLogger is a minimal structured request logger.
// TODO: replace with a proper structured logger (zerolog / zap / slog).
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %d %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}
