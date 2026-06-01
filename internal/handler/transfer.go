package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/middleware"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/service"
)

type TransferHandler struct {
	transferService *service.TransferService
	validate        *validator.Validate
}

func NewTransferHandler(transferService *service.TransferService) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
		validate:        validator.New(),
	}
}

func (h *TransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.transferService.Transfer(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInsufficientFunds):
			writeError(w, http.StatusUnprocessableEntity, "insufficient funds")
		case errors.Is(err, service.ErrRecipientNotFound):
			writeError(w, http.StatusNotFound, "recipient not found")
		case errors.Is(err, service.ErrSelfTransfer):
			writeError(w, http.StatusBadRequest, "cannot transfer to yourself")
		case errors.Is(err, service.ErrWalletNotFound):
			writeError(w, http.StatusNotFound, "wallet not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, res)
}
