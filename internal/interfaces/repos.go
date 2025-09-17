package interfaces

import (
	"example/internal/dto"
)

type OrdersRepository interface {
	Create(order *dto.Order) error
}
