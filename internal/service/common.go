package service

import "errors"

var ErrInvalidLimit = errors.New("invalid limit")

func normalizeLimit(limit int32) (int32, error) {
	if limit == 0 {
		return 50, nil
	}
	if limit < 0 {
		return 0, ErrInvalidLimit
	}
	if limit > 200 {
		return 200, nil
	}
	return limit, nil
}
