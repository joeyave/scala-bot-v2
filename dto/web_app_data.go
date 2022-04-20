package dto

type CreateEventData struct {
	Event struct {
		Name string `json:"name"`
		Date string `json:"date"`
	} `json:"event"`
}
