package keyboard

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joeyave/scala-bot-v2/entities"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/txt"
)

func Menu(lang string) [][]gotgbot.KeyboardButton {
	return [][]gotgbot.KeyboardButton{
		{{Text: txt.Get("button.schedule", lang)}},
		{{Text: txt.Get("button.songs", lang)}, {Text: txt.Get("button.stats", lang)}},
		{{Text: txt.Get("button.settings", lang)}},
	}
}

func NavigationByToken(nextPageToken *entities.NextPageToken, lang string) [][]gotgbot.KeyboardButton {

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

func EventInit(user *entities.User, event *entities.Event, lang string) [][]gotgbot.InlineKeyboardButton {

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: txt.Get("button.chords", lang), CallbackData: "eventChords:" + event.ID.Hex()},
			{Text: txt.Get("button.metronome", lang), CallbackData: "eventMetronome:" + event.ID.Hex()},
		},
	}

	member := false
	for _, membership := range event.Memberships {
		if user.ID == membership.UserID {
			member = true
			break
		}
	}

	if user.Role == helpers.Admin || member {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: txt.Get("button.edit", lang), CallbackData: "todo"},
		})
	}

	return keyboard
}
