package service

import (
	"context"
	"errors"
	"strings"

	"Warehouse_service/internal/models"
)

var ErrInvalidScanEvent = errors.New("invalid scan event")

type ScanEventRepository interface {
	Create(
		ctx context.Context,
		markerCode string,
		userID *int64,
		deviceInfo *string,
		success bool,
	) (models.ScanEvent, error)

	List(ctx context.Context, limit int32) ([]models.ScanEvent, error)
}

type CreateScanEventInput struct {
	MarkerCode string
	UserID     *int64
	DeviceInfo *string
	Success    *bool
}

type ScanEventService struct {
	repo ScanEventRepository
}

func NewScanEventService(repo ScanEventRepository) *ScanEventService {
	return &ScanEventService{repo: repo}
}

func (s *ScanEventService) Create(ctx context.Context, input CreateScanEventInput) (models.ScanEvent, error) {
	markerCode := strings.TrimSpace(input.MarkerCode)
	if markerCode == "" {
		return models.ScanEvent{}, ErrInvalidScanEvent
	}

	success := true
	if input.Success != nil {
		success = *input.Success
	}

	return s.repo.Create(ctx, markerCode, input.UserID, input.DeviceInfo, success)
}

func (s *ScanEventService) List(ctx context.Context, limit int32) ([]models.ScanEvent, error) {
	normalizedLimit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}

	return s.repo.List(ctx, normalizedLimit)
}
