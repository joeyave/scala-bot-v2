package state

// Plain keyboard states.
const (
	GetEvents = iota + 1
	FilterEvents

	Search
	SearchSetlist

	GetSongs
	FilterSongs
)

// Inline states.
const (
	EventSetlistDocs = iota + 1
)
