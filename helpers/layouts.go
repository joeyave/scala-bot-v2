package helpers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joeyave/scala-bot-v2/entity"
	"google.golang.org/api/drive/v3"
)

func GetSongActionsKeyboard(user entity.User, song entity.Song, driveFile drive.File) [][]gotgbot.InlineKeyboardButton {
	if song.BandID == user.BandID {
		return [][]gotgbot.InlineKeyboardButton{
			{{Text: LinkToTheDoc, Url: driveFile.WebViewLink}},
			{
				{Text: Voices, CallbackData: AggregateCallbackData(GetVoicesState, 0, "")},
				{Text: Tags, CallbackData: AggregateCallbackData(AddSongTagState, 0, "")},
			},
			{
				{Text: Transpose, CallbackData: AggregateCallbackData(TransposeSongState, 0, "")},
				{Text: Style, CallbackData: AggregateCallbackData(StyleSongState, 0, "")},
			},
			{
				{Text: ChangeSongBPM, CallbackData: AggregateCallbackData(ChangeSongBPMState, 0, "")},
				{Text: AddLyricsPage, CallbackData: AggregateCallbackData(AddLyricsPageState, 0, "")},
			},
		}
	} else {
		return [][]gotgbot.InlineKeyboardButton{
			{{Text: driveFile.Name, Url: driveFile.WebViewLink}},
			{{Text: CopyToMyBand, CallbackData: AggregateCallbackData(CopySongState, 0, "")}},
			{{Text: Voices, CallbackData: AggregateCallbackData(GetVoicesState, 0, "")}},
		}
	}
}

func GetEventActionsKeyboard(user entity.User, event entity.Event) [][]gotgbot.InlineKeyboardButton {
	member := false
	for _, membership := range event.Memberships {
		if user.ID == membership.UserID {
			member = true
		}
	}

	if user.Role == entity.AdminRole || member {
		return [][]gotgbot.InlineKeyboardButton{
			{
				{Text: FindChords, CallbackData: AggregateCallbackData(EventActionsState, 1, "")},
				{Text: Metronome, CallbackData: AggregateCallbackData(EventActionsState, 2, "")},
			},
			{
				{Text: Edit, CallbackData: AggregateCallbackData(EditInlineKeyboardState, 0, "")},
			},
		}
	}

	return [][]gotgbot.InlineKeyboardButton{
		{
			{Text: FindChords, CallbackData: AggregateCallbackData(EventActionsState, 1, "")},
			{Text: Metronome, CallbackData: AggregateCallbackData(EventActionsState, 2, "")},
		},
	}
}

func GetEditEventKeyboard(user entity.User) [][]gotgbot.InlineKeyboardButton {
	if user.Role == entity.AdminRole {
		return [][]gotgbot.InlineKeyboardButton{
			{
				{Text: Setlist, CallbackData: AggregateCallbackData(DeleteEventSongState, 0, "")},
				{Text: Members, CallbackData: AggregateCallbackData(DeleteEventMemberState, 0, "")},
			},
			{
				{Text: Notes, CallbackData: AggregateCallbackData(ChangeEventNotesState, 0, "")},
			},
			{
				{Text: Date, CallbackData: AggregateCallbackData(ChangeEventDateState, 0, "")},
				{Text: Delete, CallbackData: AggregateCallbackData(DeleteEventState, 0, "")},
			},
			{
				{Text: Back, CallbackData: AggregateCallbackData(EventActionsState, 0, "")},
			},
		}
	}

	return [][]gotgbot.InlineKeyboardButton{
		{
			{Text: Setlist, CallbackData: AggregateCallbackData(DeleteEventSongState, 0, "")},
			//{Text: AddSong, CallbackData: AggregateCallbackData(AddEventSongState, 0, "")},
		},
		//{
		//	{Text: SongsOrder, CallbackData: AggregateCallbackData(ChangeSongOrderState, 0, "")},
		//},
		{
			{Text: Back, CallbackData: AggregateCallbackData(EventActionsState, 0, "")},
		},
	}
}

var MainMenuKeyboard = [][]gotgbot.KeyboardButton{
	{{Text: Schedule}},
	{{Text: Songs}, {Text: Stats}},
	{{Text: Settings}},
}

var SettingsKeyboard = [][]gotgbot.KeyboardButton{
	{{Text: BandSettings}, {Text: ProfileSettings}},
	{{Text: Back}},
}

var ProfileSettingsKeyboard = [][]gotgbot.KeyboardButton{
	{{Text: ChangeBand}},
	{{Text: Back}},
}

var BandSettingsKeyboard = [][]gotgbot.KeyboardButton{
	{{Text: CreateRole}, {Text: AddAdmin}},
	{{Text: Back}},
}

var KeysKeyboard = [][]gotgbot.KeyboardButton{
	{{Text: "C"}, {Text: "C#"}, {Text: "Db"}},
	{{Text: "D"}, {Text: "D#"}, {Text: "Eb"}},
	{{Text: "E"}},
	{{Text: "F"}, {Text: "F#"}, {Text: "Gb"}},
	{{Text: "G"}, {Text: "G#"}, {Text: "Ab"}},
	{{Text: "A"}, {Text: "A#"}, {Text: "Bb"}},
	{{Text: "B"}},
}

var TimesKeyboard = [][]gotgbot.KeyboardButton{
	{{Text: "2/4"}, {Text: "3/4"}, {Text: "4/4"}},
}

var SearchEverywhereKeyboard = [][]gotgbot.KeyboardButton{
	{{Text: Cancel}, {Text: SearchEverywhere}},
}

var ConfirmDeletingEventKeyboard = [][]gotgbot.InlineKeyboardButton{
	{{Text: Cancel, CallbackData: AggregateCallbackData(EventActionsState, 0, "EditEventKeyboard")}, {Text: Yes, CallbackData: AggregateCallbackData(DeleteEventState, 1, "")}},
}
