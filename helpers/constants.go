package helpers

const SongsPageSize = 50
const EventsPageSize = 25

const (
	SearchSongState = iota
	SetlistState
	SongActionsState
	GetVoicesState
	AddSongTagState
	UploadVoiceState
	DeleteVoiceState
	MainMenuState
	TransposeSongState
	StyleSongState
	AddLyricsPageState
	ChangeSongBPMState
	ChooseBandState
	CreateBandState
	CopySongState
	CreateSongState
	DeleteSongState
	AddBandAdminState
	GetEventsState
	CreateEventState
	EventActionsState
	CreateRoleState
	AddEventSongState
	DeleteEventState
	ChangeSongOrderState
	AddEventMemberState
	ChangeEventDateState
	ChangeEventNotesState
	DeleteEventMemberState
	DeleteEventSongState
	GetSongsFromMongoState
	EditInlineKeyboardState
	SettingsState
)

// Buttons constants.
const (
	Skip                        string = "⏩ Пропустить"
	Voices                      string = "Партии"
	Tags                        string = "🔖 Теги"
	CreateTag                   string = "➕ Создать тег"
	Transpose                   string = "🎛 Транспонировать"
	Style                       string = "🎨 Стилизовать"
	ChangeSongBPM               string = "🥁 Изменить BPM"
	AddLyricsPage               string = "🔤 Слова"
	Yes                         string = "✅ Да"
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
	LinkToTheDoc                string = "📎 Ссылка на документ"
	Setlist                     string = "📝 Список"
	Like                        string = "❤️‍🔥"
	Placeholder                 string = "Фраза из песни или список"
)

var FilesChannelID int64
var LogsChannelID int64
