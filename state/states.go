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
	EventCB = iota + 1

	EventSetlistDocs
	EventSetlistMetronome

	EventSetlist
	EventSetlistDeleteOrRecoverSong

	EventMembers
	EventMembersDeleteOrRecoverMember
)
