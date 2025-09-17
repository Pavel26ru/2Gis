package interfaces

import (
	"example/internal/dto"
)

type OrdersService interface {
	Create(order *dto.Order) (*dto.Order, error)
}
