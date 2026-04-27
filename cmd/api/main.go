package main

import (
	"Warehouse_service/internal/repository"
	"Warehouse_service/internal/service"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Warehouse_service/internal/config"
	_ "Warehouse_service/internal/docs"
	"Warehouse_service/internal/handler"
)

// @title API сервиса склада
// @version 1.0
// @description Backend API складского мобильного приложения
// @BasePath /
// @description API для мобильных складских сценариев.
func main() {
	cfg := config.MustLoad()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("init db pool: %v", err)
	}
	defer pool.Close()

	markerRepo := repository.NewMarkerRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	userSessionRepo := repository.NewUserSessionRepository(pool)
	storageCellRepo := repository.NewStorageCellRepository(pool)
	palletRepo := repository.NewPalletRepository(pool)
	boxRepo := repository.NewBoxRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	batchRepo := repository.NewBatchRepository(pool)
	scanEventRepo := repository.NewScanEventRepository(pool)
	operationHistoryRepo := repository.NewOperationHistoryRepository(pool)

	objectService := service.NewObjectService(
		markerRepo,
		storageCellRepo,
		palletRepo,
		boxRepo,
		productRepo,
		batchRepo,
	)

	scanEventService := service.NewScanEventService(scanEventRepo)
	operationHistoryService := service.NewOperationHistoryService(operationHistoryRepo)
	labelService := service.NewLabelService(
		markerRepo,
		storageCellRepo,
		palletRepo,
		boxRepo,
		productRepo,
		batchRepo,
	)
	authService := service.NewAuthService(userRepo, userSessionRepo)
	adminService := service.NewAdminService(
		productRepo,
		storageCellRepo,
		boxRepo,
		batchRepo,
		markerRepo,
		userRepo,
		pool,
	)

	scanService := service.NewScanService(objectService, scanEventService)
	moveBoxService := service.NewMoveBoxService(
		markerRepo,
		boxRepo,
		storageCellRepo,
		operationHistoryService,
	)
	moveBatchService := service.NewMoveBatchService(
		markerRepo,
		batchRepo,
		boxRepo,
		storageCellRepo,
		operationHistoryService,
	)

	objectHandler := handler.NewObjectHandler(objectService)
	scanEventHandler := handler.NewScanEventHandler(scanEventService)
	operationHistoryHandler := handler.NewOperationHistoryHandler(operationHistoryService)
	scanHandler := handler.NewScanHandler(scanService)
	moveBoxHandler := handler.NewMoveBoxHandler(moveBoxService)
	moveBatchHandler := handler.NewMoveBatchHandler(moveBatchService)
	labelHandler := handler.NewLabelHandler(labelService)
	authHandler := handler.NewAuthHandler(authService)
	adminHandler := handler.NewAdminHandler(adminService)
	authMiddleware := handler.NewAuthMiddleware(authService)

	mux := http.NewServeMux()

	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	mux.HandleFunc("/healthz", healthzHandler(pool))

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			authHandler.APILogin(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/auth/me", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			authHandler.APIGetCurrentUser(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/auth/logout", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			authHandler.APILogout(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/boxes/move", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			moveBoxHandler.Execute(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/batches/move", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			moveBatchHandler.Execute(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/objects/by-marker", authMiddleware.RequireAuthenticated(objectHandler.GetByMarkerCode))

	mux.HandleFunc("/api/v1/scan-events", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			scanEventHandler.Create(w, r)
		case http.MethodGet:
			scanEventHandler.List(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/operations", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			operationHistoryHandler.Create(w, r)
		case http.MethodGet:
			operationHistoryHandler.List(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/scan", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			scanHandler.Execute(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/labels", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.List(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/workers", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListWorkersAPI(w, r)
		case http.MethodPost:
			adminHandler.CreateWorkerAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/products", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListProductsAPI(w, r)
		case http.MethodPost:
			adminHandler.CreateProductAPI(w, r)
		case http.MethodPut:
			adminHandler.UpdateProductAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/products/import", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			adminHandler.ImportProductsAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/storage-cells", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListStorageCellsAPI(w, r)
		case http.MethodPost:
			adminHandler.CreateStorageCellAPI(w, r)
		case http.MethodPut:
			adminHandler.UpdateStorageCellAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/boxes", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListBoxesAPI(w, r)
		case http.MethodPost:
			adminHandler.CreateBoxAPI(w, r)
		case http.MethodPut:
			adminHandler.UpdateBoxAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/batches", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListBatchesAPI(w, r)
		case http.MethodPost:
			adminHandler.CreateBatchAPI(w, r)
		case http.MethodPut:
			adminHandler.UpdateBatchAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/labels/qr", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.RenderQR(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/labels/pdf", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.DownloadPDF(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("server started on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen and serve: %v", err)
	}
}

// healthzHandler godoc
// @Summary Проверка доступности
// @Description Проверяет готовность API через ping пула подключений PostgreSQL.
// @Tags Система
// @Produce plain
// @Success 200 {string} string "Сервис доступен"
// @Failure 503 {string} string "База данных недоступна"
// @Router /healthz [get]
func healthzHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(pingCtx); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
