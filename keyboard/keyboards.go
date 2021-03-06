package keyboard

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"github.com/joeyave/scala-bot-v2/util"
	"os"
)

func Menu(lang string) [][]gotgbot.KeyboardButton {
	return [][]gotgbot.KeyboardButton{
		{{Text: txt.Get("button.schedule", lang)}},
		{{Text: txt.Get("button.songs", lang)}, {Text: txt.Get("button.stats", lang)}},
		{{Text: txt.Get("button.settings", lang)}},
	}
}

func Settings(user *entity.User, lang string) [][]gotgbot.InlineKeyboardButton {
	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: txt.Get("button.changeBand", lang), CallbackData: util.CallbackData(state.SettingsBands, "")}},
	}
	if user.IsAdmin() {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.addAdmin", lang), CallbackData: util.CallbackData(state.SettingsBandMembers, user.BandID.Hex())}})
	}
	return keyboard
}

func NavigationByToken(nextPageToken *entity.NextPageToken, lang string) [][]gotgbot.KeyboardButton {

	var keyboard [][]gotgbot.KeyboardButton

	// если есть пред стр
	if nextPageToken.GetPrevValue() != "" {
		// если нет след стр
		if nextPageToken.GetValue() != "" {
			keyboard = append(keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.prev", lang)}, {Text: txt.Get("button.menu", lang)}, {Text: txt.Get("button.next", lang)}})
		} else { // если есть след
			keyboard = append(keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.prev", lang)}, {Text: txt.Get("button.menu", lang)}})
		}
	} else { // если нет пред стр
		if nextPageToken.GetValue() != "" {
			keyboard = append(keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.menu", lang)}, {Text: txt.Get("button.next", lang)}})
		} else {
			keyboard = append(keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.menu", lang)}})
		}
	}

	return keyboard
}

func EventInit(event *entity.Event, user *entity.User, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: txt.Get("button.chords", lang), CallbackData: util.CallbackData(state.EventSetlistDocs, event.ID.Hex())},
			{Text: txt.Get("button.metronome", lang), CallbackData: util.CallbackData(state.EventSetlistMetronome, event.ID.Hex())},
		},
	}

	if user.IsAdmin() || user.IsEventMember(event) {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			//{Text: txt.Get("button.edit", lang), WebApp: &gotgbot.WebAppInfo{Url: fmt.Sprintf("%s/web-app/events/%s/edit", os.Getenv("HOST"), event.ID.Hex())}},
			{Text: txt.Get("button.edit", lang), CallbackData: util.CallbackData(state.EventCB, event.ID.Hex()+":edit")},
		})
	}

	return keyboard
}

func EventEdit(event *entity.Event, user *entity.User, chatID, messageID int64, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: txt.Get("button.setlist", lang), WebApp: &gotgbot.WebAppInfo{Url: fmt.Sprintf("%s/web-app/events/%s/edit?messageId=%d&chatId=%d&userId=%d", os.Getenv("HOST"), event.ID.Hex(), messageID, chatID, user.ID)}},
			{Text: txt.Get("button.members", lang), CallbackData: util.CallbackData(state.EventMembers, event.ID.Hex())},
		},
		//{
		//	{Text: txt.Get("button.notes", lang), CallbackData: "todo"},
		//},
		//{
		//	{Text: txt.Get("button.setlist", lang), WebApp: &gotgbot.WebAppInfo{Url: fmt.Sprintf("%s/web-app/events/%s/edit?messageId=%d&chatId=%d", os.Getenv("HOST"), event.ID.Hex(), messageID, chatID)}},
		//},
		{
			//{Text: txt.Get("button.editDate", lang), CallbackData: "todo"},
			{Text: txt.Get("button.delete", lang), CallbackData: util.CallbackData(state.EventDeleteConfirm, event.ID.Hex())},
		},
		{
			{Text: txt.Get("button.back", lang), CallbackData: util.CallbackData(state.EventCB, event.ID.Hex()+":init")},
		},
	}

	return keyboard
}

func SongInit(song *entity.Song, user *entity.User, chatID int64, messageID int64, lang string) [][]gotgbot.InlineKeyboardButton {

	var keyboard [][]gotgbot.InlineKeyboardButton

	if song.BandID == user.BandID {

		liked := false
		for _, userID := range song.Likes {
			if user.ID == userID {
				liked = true
				break
			}
		}

		if liked {
			keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
				{Text: txt.Get("button.like", lang), CallbackData: util.CallbackData(state.SongLike, song.ID.Hex()+":dislike")},
				{Text: txt.Get("button.voices", lang), CallbackData: util.CallbackData(state.SongVoices, song.ID.Hex())}, // todo: enable
				{Text: txt.Get("button.more", lang), CallbackData: util.CallbackData(state.SongCB, song.ID.Hex()+":edit")},
			})
		} else {
			keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
				{Text: txt.Get("button.unlike", lang), CallbackData: util.CallbackData(state.SongLike, song.ID.Hex()+":like")},
				{Text: txt.Get("button.voices", lang), CallbackData: util.CallbackData(state.SongVoices, song.ID.Hex())}, // todo: enable
				{Text: txt.Get("button.more", lang), CallbackData: util.CallbackData(state.SongCB, song.ID.Hex()+":edit")},
			})
		}

		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.edit", lang), WebApp: &gotgbot.WebAppInfo{Url: fmt.Sprintf("%s/web-app/songs/%s/edit?userId=%d&messageId=%d&chatId=%d", os.Getenv("HOST"), song.ID.Hex(), user.ID, messageID, chatID)}}})

	} else {
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: txt.Get("button.copyToMyBand", lang), CallbackData: util.CallbackData(state.SongCopyToMyBand, song.DriveFileID)}},
			//{{Text: txt.Get("button.voices", lang), CallbackData: util.CallbackData(state.SongVoices, song.ID.Hex())}}, // todo: enable
		}

		if user.ID == 195295372 { // todo: remove
			keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.more", lang), CallbackData: util.CallbackData(state.SongCB, song.ID.Hex()+":edit")}})
		}
	}

	return keyboard
}

func SongEdit(song *entity.Song, user *entity.User, chatID, messageID int64, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		//{
		//	{Text: txt.Get("button.edit", lang), WebApp: &gotgbot.WebAppInfo{Url: fmt.Sprintf("%s/web-app/songs/%s/edit?userId=%d&messageId=%d&chatId=%d", os.Getenv("HOST"), song.ID.Hex(), user.ID, messageID, chatID)}},
		//},
		{
			{Text: txt.Get("button.docLink", lang), Url: song.PDF.WebViewLink},
		},
		{
			{Text: txt.Get("button.style", lang), CallbackData: util.CallbackData(state.SongStyle, song.DriveFileID)},
			{Text: txt.Get("button.lyrics", lang), CallbackData: util.CallbackData(state.SongAddLyricsPage, song.DriveFileID)},
		},
		//{
		//	{Text: txt.Get("button.voices", lang), CallbackData: util.CallbackData(state.SongVoices, song.ID.Hex())},
		//},

		//{
		//	{Text: txt.Get("button.delete", lang), CallbackData: util.CallbackData(state.SongDeleteConfirm, song.ID.Hex())},
		//},

	}

	// todo: allow for Admins
	if user.ID == 195295372 {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.delete", lang), CallbackData: util.CallbackData(state.SongDeleteConfirm, song.ID.Hex())}})
	}

	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.back", lang), CallbackData: util.CallbackData(state.SongCB, song.ID.Hex()+":init")}})

	return keyboard
}
