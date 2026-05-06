package main

import (
	"Warehouse_service/internal/repository"
	"Warehouse_service/internal/service"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Warehouse_service/internal/config"
	"Warehouse_service/internal/handler"
)

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
	rackRepo := repository.NewRackRepository(pool)
	storageCellRepo := repository.NewStorageCellRepository(pool)
	boxRepo := repository.NewBoxRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	batchRepo := repository.NewBatchRepository(pool)
	productAliasRepo := repository.NewProductAliasRepository(pool)
	inboundShipmentRepo := repository.NewInboundShipmentRepository(pool)
	scanEventRepo := repository.NewScanEventRepository(pool)
	operationHistoryRepo := repository.NewOperationHistoryRepository(pool)

	objectService := service.NewObjectService(
		markerRepo,
		rackRepo,
		storageCellRepo,
		boxRepo,
		productRepo,
		batchRepo,
	)

	scanEventService := service.NewScanEventService(scanEventRepo)
	operationHistoryService := service.NewOperationHistoryService(operationHistoryRepo)
	labelService := service.NewLabelService(
		markerRepo,
		rackRepo,
		storageCellRepo,
		boxRepo,
		productRepo,
		batchRepo,
	)
	authService := service.NewAuthService(userRepo, userSessionRepo)
	productInventoryService := service.NewProductInventoryService(productRepo)
	adminService := service.NewAdminService(
		productRepo,
		rackRepo,
		storageCellRepo,
		boxRepo,
		batchRepo,
		markerRepo,
		userRepo,
		productAliasRepo,
		inboundShipmentRepo,
		pool,
	)

	scanService := service.NewScanService(objectService, scanEventService)
	moveBoxService := service.NewMoveBoxService(
		markerRepo,
		boxRepo,
		storageCellRepo,
		operationHistoryService,
		pool,
	)
	moveBatchService := service.NewMoveBatchService(
		markerRepo,
		batchRepo,
		boxRepo,
		storageCellRepo,
		operationHistoryService,
		pool,
	)
	outboundShipmentService := service.NewOutboundShipmentService(pool)

	objectHandler := handler.NewObjectHandler(objectService)
	scanEventHandler := handler.NewScanEventHandler(scanEventService)
	operationHistoryHandler := handler.NewOperationHistoryHandler(operationHistoryService)
	scanHandler := handler.NewScanHandler(scanService)
	moveBoxHandler := handler.NewMoveBoxHandler(moveBoxService)
	moveBatchHandler := handler.NewMoveBatchHandler(moveBatchService)
	outboundShipmentHandler := handler.NewOutboundShipmentHandler(outboundShipmentService)
	labelHandler := handler.NewLabelHandler(labelService)
	authHandler := handler.NewAuthHandler(authService)
	productInventoryHandler := handler.NewProductInventoryHandler(productInventoryService)
	adminHandler := handler.NewAdminHandler(adminService)
	authMiddleware := handler.NewAuthMiddleware(authService)

	mux := http.NewServeMux()

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

	mux.HandleFunc("/api/v1/outbound/shipments/complete", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			outboundShipmentHandler.Complete(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/objects/by-marker", authMiddleware.RequireAuthenticated(objectHandler.GetByMarkerCode))

	mux.HandleFunc("/api/v1/scan-events", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			scanEventHandler.List(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/operations", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
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

	mux.HandleFunc("/api/v1/products/search", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			productInventoryHandler.SearchProducts(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/products/locations", authMiddleware.RequireAuthenticated(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			productInventoryHandler.GetProductLocations(w, r)
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
		case http.MethodDelete:
			adminHandler.DeleteWorkerAPI(w, r)
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
		case http.MethodDelete:
			adminHandler.DeleteProductAPI(w, r)
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

	mux.HandleFunc("/api/v1/admin/shipments", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("id") != "" {
				adminHandler.GetInboundShipmentAPI(w, r)
				return
			}
			adminHandler.ListInboundShipmentsAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/shipments/import", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			adminHandler.ImportInboundShipmentAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/shipments/items/link-product", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			adminHandler.LinkInboundShipmentItemAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/shipments/items/create-product", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			adminHandler.CreateProductForInboundShipmentItemAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/shipments/generate", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			adminHandler.GenerateInboundShipmentAPI(w, r)
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
		case http.MethodDelete:
			adminHandler.DeleteStorageCellAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/admin/racks", authMiddleware.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			adminHandler.ListRacksAPI(w, r)
		case http.MethodPost:
			adminHandler.CreateRackAPI(w, r)
		case http.MethodPut:
			adminHandler.UpdateRackAPI(w, r)
		case http.MethodDelete:
			adminHandler.DeleteRackAPI(w, r)
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
		case http.MethodDelete:
			adminHandler.DeleteBoxAPI(w, r)
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
		case http.MethodDelete:
			adminHandler.DeleteBatchAPI(w, r)
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
