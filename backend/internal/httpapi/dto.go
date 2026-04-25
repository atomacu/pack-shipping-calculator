package httpapi

import "pack-shipping-calculator/backend/internal/packing"

type packsResponse struct {
	PackSizes []int `json:"pack_sizes"`
}

type replacePacksRequest struct {
	PackSizes []int `json:"pack_sizes"`
}

type calculateOrderRequest struct {
	Items int `json:"items"`
}

type calculateOrderResponse = packing.Plan

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
