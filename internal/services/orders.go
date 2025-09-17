package services

import (
	"example/internal/dto"
	"example/internal/interfaces"
)

type ordersService struct {
	ordersRepository interfaces.OrdersRepository
}

func NewOrdersService(
	ordersRepository interfaces.OrdersRepository,
) interfaces.OrdersService {
	return &ordersService{
		ordersRepository: ordersRepository,
	}
}

func (o *ordersService) Create(order *dto.Order) (*dto.Order, error) {
	if err := o.ordersRepository.Create(order); err != nil {
		return nil, err
	}

	return order, nil
}
