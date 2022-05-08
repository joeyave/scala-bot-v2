package helpers

const SongsPageSize = 50
const EventsPageSize = 25

const (
	SongActionsState = iota
	UploadVoiceState
	MainMenuState
	TransposeSongState
	StyleSongState
	AddLyricsPageState
	CreateBandState
	CopySongState
	DeleteSongState
	AddBandAdminState
	CreateRoleState
	DeleteEventState
)

// Buttons constants.
const (
	Skip                        string = "⏩ Пропустить"
	AppendSection               string = "В конец документа"
	CreateBand                  string = "Создать свою группу"
	CopyToMyBand                string = "🖨 Копировать песню в свою группу"
	ChangeBand                  string = "Изменить группу"
	AddAdmin                    string = "➕ Добавить администратора"
	CreateRole                  string = "Создать роль"
	AddMember                   string = "➕ Добавить участника"
	ByWeekday                   string = "День недели"
	BandSettings                string = "Настройки группы"
	ProfileSettings             string = "Настройки профиля"
	SongsByNumberOfPerforming   string = "🔢"
	SongsByLastDateOfPerforming string = "📆"
	LikedSongs                  string = "❤️‍🔥"
	TagsEmoji                   string = "🔖"
	NextPage                    string = "→"
	PrevPage                    string = "←"
	Today                       string = "⏰"
	Setlist                     string = "📝 Список"
	Like                        string = "❤️‍🔥"
	Placeholder                 string = "Фраза из песни или список"
)

var FilesChannelID int64
var LogsChannelID int64
