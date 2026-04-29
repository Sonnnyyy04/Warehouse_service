package handler

import (
	"encoding/json"
	"io"
)

func decodeJSONBody(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
