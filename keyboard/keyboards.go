package keyboard

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"github.com/joeyave/scala-bot-v2/util"
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
			{Text: txt.Get("button.edit", lang), CallbackData: util.CallbackData(state.EditEventKeyboard, event.ID.Hex())},
		})
	}

	return keyboard
}

func EventEdit(user *entity.User, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: txt.Get("button.setlist", lang), CallbackData: "todo"},
			{Text: txt.Get("button.members", lang), CallbackData: "todo"},
		},
		{
			{Text: txt.Get("button.notes", lang), CallbackData: "todo"},
		},
		{
			{Text: txt.Get("button.editDate", lang), CallbackData: "todo"},
			{Text: txt.Get("button.delete", lang), CallbackData: "todo"},
		},
		{
			{Text: txt.Get("button.back", lang), CallbackData: "todo"},
		},
	}

	return keyboard
}

func SongInit(song *entity.Song, user *entity.User, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: txt.Get("button.edit", lang), CallbackData: "todo"},
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
			{Text: txt.Get("button.like", lang), CallbackData: "todo"},
		})
	} else {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: txt.Get("button.unlike", lang), CallbackData: "todo"},
		})
	}

	return keyboard
}
