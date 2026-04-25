package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"pack-shipping-calculator/backend/internal/packing"
	"pack-shipping-calculator/backend/internal/packs"
)

type packService interface {
	GetPackSizes(context.Context) ([]int, error)
	ReplacePackSizes(context.Context, []int) ([]int, error)
	CalculateOrder(context.Context, int) (packing.Plan, error)
}

type handler struct {
	packService packService
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h handler) getPacks(w http.ResponseWriter, r *http.Request) {
	sizes, err := h.packService.GetPackSizes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load pack sizes")
		return
	}

	writeJSON(w, http.StatusOK, packsResponse{PackSizes: sizes})
}

func (h handler) replacePacks(w http.ResponseWriter, r *http.Request) {
	var req replacePacksRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_json", "request body must be valid JSON")
		return
	}

	sizes, err := h.packService.ReplacePackSizes(r.Context(), req.PackSizes)
	if err != nil {
		if errors.Is(err, packs.ErrInvalidPackSizes) {
			writeError(w, http.StatusBadRequest, "invalid_pack_sizes", "pack sizes must contain at least one positive integer")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not save pack sizes")
		return
	}

	writeJSON(w, http.StatusOK, packsResponse{PackSizes: sizes})
}

func (h handler) calculateOrder(w http.ResponseWriter, r *http.Request) {
	var req calculateOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_json", "request body must be valid JSON")
		return
	}

	plan, err := h.packService.CalculateOrder(r.Context(), req.Items)
	if err != nil {
		if errors.Is(err, packing.ErrInvalidOrderSize) || errors.Is(err, packing.ErrOrderTooLarge) {
			writeError(w, http.StatusBadRequest, "invalid_order", invalidOrderMessage())
			return
		}
		if errors.Is(err, packs.ErrInvalidPackSizes) ||
			errors.Is(err, packing.ErrNoPackSizes) ||
			errors.Is(err, packing.ErrInvalidPackSize) {
			writeError(w, http.StatusBadRequest, "invalid_pack_sizes", "pack sizes must contain at least one positive integer")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not calculate order")
		return
	}

	writeJSON(w, http.StatusOK, calculateOrderResponse(plan))
}

func decodeJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(value); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("request body must contain a single JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, errorResponse{
		Error: apiError{
			Code:    code,
			Message: message,
		},
	})
}

func invalidOrderMessage() string {
	return fmt.Sprintf("items must be a whole number from 1 to %d", packing.MaxOrderSize)
}
