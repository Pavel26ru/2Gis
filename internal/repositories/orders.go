package repositories

import (
	"fmt"
	"time"

	"example/internal/dto"
	"example/internal/interfaces"
)

type orderMemoryRepository struct {
	orders []*dto.Order

	rooms []dto.Room
}

func NewOrdersMemoryRepository(knownRooms []dto.Room) interfaces.OrdersRepository {
	return &orderMemoryRepository{
		orders: make([]*dto.Order, 0),
		rooms:  knownRooms,
	}
}

func (o *orderMemoryRepository) Create(order *dto.Order) error {
	if !o.isKnownRoom(order.HotelID, order.RoomTypeID) {
		return fmt.Errorf("unknown room")
	}

	if !o.isRoomAvailable(order.HotelID, order.RoomTypeID, order.From, order.To) {
		return fmt.Errorf("room not available")
	}

	o.orders = append(o.orders, order)
	return nil
}

func (o *orderMemoryRepository) isRoomAvailable(hotelID, roomID string, from, to time.Time) bool {
	for _, order := range o.orders {
		if order.HotelID != hotelID || order.RoomTypeID != roomID {
			continue
		}
		if order.From.Before(to) && order.To.After(from) {
			return false
		}
	}
	return true
}

func (o *orderMemoryRepository) isKnownRoom(hotelID, roomID string) bool {
	for _, room := range o.rooms {
		if room.HotelID == hotelID && room.RoomTypeID == roomID {
			return true
		}
	}
	return false
}
