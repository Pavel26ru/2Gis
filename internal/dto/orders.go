package dto

import "time"

type Order struct {
	HotelID    string    `json:"hotel_id"`
	RoomTypeID string    `json:"room_id"`
	UserEmail  string    `json:"email"`
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
}
