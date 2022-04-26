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

//
//var SettingsKeyboard = [][]gotgbot.KeyboardButton{
//	{{Text: BandSettings}, {Text: ProfileSettings}},
//	{{Text: Back}},
//}
//
//var ProfileSettingsKeyboard = [][]gotgbot.KeyboardButton{
//	{{Text: ChangeBand}},
//	{{Text: Back}},
//}
//
//var BandSettingsKeyboard = [][]gotgbot.KeyboardButton{
//	{{Text: CreateRole}, {Text: AddAdmin}},
//	{{Text: Back}},
//}
//
//var KeysKeyboard = [][]gotgbot.KeyboardButton{
//	{{Text: "C"}, {Text: "C#"}, {Text: "Db"}},
//	{{Text: "D"}, {Text: "D#"}, {Text: "Eb"}},
//	{{Text: "E"}},
//	{{Text: "F"}, {Text: "F#"}, {Text: "Gb"}},
//	{{Text: "G"}, {Text: "G#"}, {Text: "Ab"}},
//	{{Text: "A"}, {Text: "A#"}, {Text: "Bb"}},
//	{{Text: "B"}},
//}
//
//var TimesKeyboard = [][]gotgbot.KeyboardButton{
//	{{Text: "2/4"}, {Text: "3/4"}, {Text: "4/4"}},
//}
//
//var SearchEverywhereKeyboard = [][]gotgbot.KeyboardButton{
//	{{Text: Cancel}, {Text: SearchEverywhere}},
//}
//
//var ConfirmDeletingEventKeyboard = [][]gotgbot.InlineKeyboardButton{
//	{{Text: Cancel, CallbackData: AggregateCallbackData(EventActionsState, 0, "EditEventKeyboard")}, {Text: Yes, CallbackData: AggregateCallbackData(DeleteEventState, 1, "")}},
//}
