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

	scanService := service.NewScanService(objectService, scanEventService)
	moveBoxService := service.NewMoveBoxService(
		markerRepo,
		boxRepo,
		storageCellRepo,
		operationHistoryService,
	)

	objectHandler := handler.NewObjectHandler(objectService)
	scanEventHandler := handler.NewScanEventHandler(scanEventService)
	operationHistoryHandler := handler.NewOperationHistoryHandler(operationHistoryService)
	scanHandler := handler.NewScanHandler(scanService)
	moveBoxHandler := handler.NewMoveBoxHandler(moveBoxService)
	labelHandler := handler.NewLabelHandler(labelService)

	mux := http.NewServeMux()

	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	mux.HandleFunc("/healthz", healthzHandler(pool))

	mux.HandleFunc("/api/v1/boxes/move", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			moveBoxHandler.Execute(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/objects/by-marker", objectHandler.GetByMarkerCode)

	mux.HandleFunc("/api/v1/scan-events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			scanEventHandler.Create(w, r)
		case http.MethodGet:
			scanEventHandler.List(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/operations", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			operationHistoryHandler.Create(w, r)
		case http.MethodGet:
			operationHistoryHandler.List(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/scan", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			scanHandler.Execute(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/labels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.List(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/labels/qr", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.RenderQR(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/labels/print", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.PrintPage(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/labels/pdf", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.DownloadPDF(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/labels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			labelHandler.AdminPage(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/labels", http.StatusFound)
	})

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
