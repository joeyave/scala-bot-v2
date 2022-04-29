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

func EventEdit(event *entity.Event, user *entity.User, chatID int64, messageID int64, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: txt.Get("button.setlist", lang), WebApp: &gotgbot.WebAppInfo{Url: fmt.Sprintf("%s/web-app/events/%s/edit?messageId=%d&chatId=%d", os.Getenv("HOST"), event.ID.Hex(), messageID, chatID)}},
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
			{Text: txt.Get("button.delete", lang), CallbackData: "todo"},
		},
		{
			{Text: txt.Get("button.back", lang), CallbackData: util.CallbackData(state.EventCB, event.ID.Hex()+":init")},
		},
	}

	return keyboard
}

func SongInit(song *entity.Song, user *entity.User, lang string) [][]gotgbot.InlineKeyboardButton {

	var keyboard [][]gotgbot.InlineKeyboardButton

	if song.BandID == user.BandID {
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{
				{Text: txt.Get("button.edit", lang), CallbackData: util.CallbackData(state.SongCB, song.ID.Hex()+":edit")},
			},
		}

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
			})
		} else {
			keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
				{Text: txt.Get("button.unlike", lang), CallbackData: util.CallbackData(state.SongLike, song.ID.Hex()+":like")},
			})
		}

	} else {
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: txt.Get("button.copyToMyBand", lang), CallbackData: ""}},
			{{Text: txt.Get("button.voices", lang), CallbackData: ""}},
		}
	}

	return keyboard
}

func SongEdit(song *entity.Song, user *entity.User, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: txt.Get("button.docLink", lang), Url: song.PDF.WebViewLink},
		},
		{
			{Text: txt.Get("button.voices", lang), CallbackData: util.CallbackData(state.SongVoices, song.ID.Hex())},
			{Text: txt.Get("button.tags", lang), CallbackData: util.CallbackData(state.SongTags, song.ID.Hex())},
		},
		{
			{Text: txt.Get("button.transpose", lang), CallbackData: "todo"},
			{Text: txt.Get("button.style", lang), CallbackData: "todo"},
		},
		{
			{Text: txt.Get("button.changeBpm", lang), CallbackData: "todo"},
			{Text: txt.Get("button.lyrics", lang), CallbackData: "todo"},
		},
		{
			{Text: txt.Get("button.back", lang), CallbackData: util.CallbackData(state.SongCB, song.ID.Hex()+":init")},
		},
	}

	return keyboard
}
