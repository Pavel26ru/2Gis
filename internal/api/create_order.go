package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"example/internal/dto"
	"example/internal/interfaces"
)

type CreateOrderHandler struct {
	ordersService interfaces.OrdersService
}

func NewCreateOrderHandler(
	ordersService interfaces.OrdersService,
) *CreateOrderHandler {
	return &CreateOrderHandler{ordersService: ordersService}
}

func (h *CreateOrderHandler) Handle(w http.ResponseWriter, r *http.Request) {
	orderRequest, err := parseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := validateCreateOrder(orderRequest); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	order, err := h.ordersService.Create(&orderRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = orderCreatedResponse(w, order)
	if err != nil {
		log.Default().Print(err)
	}
}

func validateCreateOrder(order dto.Order) error {
	if order.From.After(order.To) {
		return errors.New("from after to")
	}
	return nil
}

func parseRequest(r *http.Request) (dto.Order, error) {
	var newOrder dto.Order
	if err := json.NewDecoder(r.Body).Decode(&newOrder); err != nil {
		return dto.Order{}, err
	}
	return newOrder, nil
}

func orderCreatedResponse(w http.ResponseWriter, order *dto.Order) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err := json.NewEncoder(w).Encode(order)
	if err != nil {
		return err
	}
	return nil
}
