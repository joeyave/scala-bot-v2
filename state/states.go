package state

// Plain keyboard states.
const (
	GetEvents = iota + 1
	FilterEvents

	Search
	SearchSetlist

	GetSongs
	FilterSongs

	SongVoicesAddVoice
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
	EventMembersAddMemberChooseRole
	EventMembersAddMemberChooseUser
	EventMembersAddMember
	EventMembersDeleteMember

	SongCB
	SongLike

	SongVoices
	SongVoicesAddVoiceAskForAudio
	SongVoice
	SongVoiceDeleteConfirm
	SongVoiceDelete
)
