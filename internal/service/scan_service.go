package service

import (
	"context"
	"errors"
	"strings"

	"Warehouse_service/internal/models"
)

var ErrInvalidScanPayload = errors.New("invalid scan payload")

type ScanObjectLookup interface {
	GetByMarkerCode(ctx context.Context, markerCode string) (models.ObjectCard, error)
}

type ScanEventLogger interface {
	Create(ctx context.Context, input CreateScanEventInput) (models.ScanEvent, error)
}

type ScanObjectInput struct {
	MarkerCode string
	UserID     *int64
	DeviceInfo *string
}

type ScanService struct {
	objectLookup ScanObjectLookup
	scanLogger   ScanEventLogger
}

func NewScanService(
	objectLookup ScanObjectLookup,
	scanLogger ScanEventLogger,
) *ScanService {
	return &ScanService{
		objectLookup: objectLookup,
		scanLogger:   scanLogger,
	}
}

func (s *ScanService) Execute(ctx context.Context, input ScanObjectInput) (models.ScanResult, error) {
	markerCode := strings.TrimSpace(input.MarkerCode)
	if markerCode == "" {
		return models.ScanResult{}, ErrInvalidScanPayload
	}

	objectCard, err := s.objectLookup.GetByMarkerCode(ctx, markerCode)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			failed := false
			_, logErr := s.scanLogger.Create(ctx, CreateScanEventInput{
				MarkerCode: markerCode,
				UserID:     input.UserID,
				DeviceInfo: input.DeviceInfo,
				Success:    &failed,
			})
			if logErr != nil {
				return models.ScanResult{}, logErr
			}

			return models.ScanResult{}, ErrObjectNotFound
		}

		if errors.Is(err, ErrInvalidMarkerCode) {
			return models.ScanResult{}, ErrInvalidScanPayload
		}

		return models.ScanResult{}, err
	}

	success := true
	event, err := s.scanLogger.Create(ctx, CreateScanEventInput{
		MarkerCode: markerCode,
		UserID:     input.UserID,
		DeviceInfo: input.DeviceInfo,
		Success:    &success,
	})
	if err != nil {
		return models.ScanResult{}, err
	}

	return models.ScanResult{
		Object:    objectCard,
		ScanEvent: event,
	}, nil
}
