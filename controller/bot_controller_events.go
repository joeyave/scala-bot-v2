package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/joeyave/scala-bot-v2/dto"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/metronome"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"github.com/joeyave/scala-bot-v2/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"strings"
	"time"
)

func (c *BotController) event(bot *gotgbot.Bot, ctx *ext.Context, event *entity.Event) error {

	user := ctx.Data["user"].(*entity.User)

	html := c.EventService.ToHtmlStringByEvent(*event, ctx.EffectiveUser.LanguageCode) // todo: refactor

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboard.EventInit(event, user, ctx.EffectiveUser.LanguageCode),
	}

	_, err := ctx.EffectiveChat.SendMessage(bot, html, &gotgbot.SendMessageOpts{
		ReplyMarkup:           markup,
		DisableWebPagePreview: true,
		ParseMode:             "HTML",
	})
	return err
}

func (c *BotController) CreateEvent(bot *gotgbot.Bot, ctx *ext.Context) error {

	var data *dto.CreateEventData
	err := json.Unmarshal([]byte(ctx.EffectiveMessage.WebAppData.Data), &data)
	if err != nil {
		return err
	}

	user := ctx.Data["user"].(*entity.User)

	eventDate, err := time.Parse("2006-01-02", data.Event.Date)
	if err != nil {
		return err
	}

	event := entity.Event{
		Time:   eventDate,
		Name:   data.Event.Name,
		BandID: user.BandID,
	}
	createdEvent, err := c.EventService.UpdateOne(event)
	if err != nil {
		return err
	}

	user.State.Index = 0
	err = c.event(bot, ctx, createdEvent)
	if err != nil {
		return err
	}
	err = c.GetEvents(0)(bot, ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *BotController) GetEvents(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entity.User)

		if user.State.Name != state.GetEvents {
			user.State = entity.State{
				Index: index,
				Name:  state.GetEvents,
			}
			user.Cache = entity.Cache{}
		}

		switch index {
		case 0:
			{
				events, err := c.EventService.FindManyFromTodayByBandID(user.BandID)
				if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode),
				}

				user.Cache.Buttons = keyboard.GetEventsStateFilterButtons(events, ctx.EffectiveUser.LanguageCode)
				markup.Keyboard = append(markup.Keyboard, user.Cache.Buttons)
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.createEvent", ctx.EffectiveUser.LanguageCode), WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/events/create"}}})

				for _, event := range events {
					markup.Keyboard = append(markup.Keyboard, keyboard.EventButton(event, user, ctx.EffectiveUser.LanguageCode, false))
				}

				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}})

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseEvent", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				//user.Cache.Filter = "-" // todo: remove

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode), txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					return c.GetEvents(0)(bot, ctx)

				case txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode), txt.Get("button.archive", ctx.EffectiveUser.LanguageCode):
					return c.filterEvents(0)(bot, ctx)

				default:
					if keyboard.IsWeekdayButton(ctx.EffectiveMessage.Text) {
						return c.filterEvents(0)(bot, ctx)
					}
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				eventName, eventTime, err := keyboard.ParseEventButton(ctx.EffectiveMessage.Text)
				if err != nil {
					return c.search(0)(bot, ctx)
				}

				foundEvent, err := c.EventService.FindOneByNameAndTimeAndBandID(eventName, eventTime, user.BandID)
				if err != nil {
					return c.search(0)(bot, ctx)
				}

				err = c.event(bot, ctx, foundEvent)
				return err
			}
		}
		return nil
	}
}

func (c *BotController) filterEvents(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entity.User)

		if user.State.Name != state.FilterEvents {
			user.State = entity.State{
				Index: index,
				Name:  state.FilterEvents,
			}
			user.Cache = entity.Cache{
				Buttons: user.Cache.Buttons,
			}
		}

		switch index {
		case 0:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				// todo: refactor - extract to func
				if (ctx.EffectiveMessage.Text == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) ||
					keyboard.IsWeekdayButton(ctx.EffectiveMessage.Text)) && user.Cache.Filter != txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) {
					user.Cache.Filter = ctx.EffectiveMessage.Text
				}

				var (
					events []*entity.Event
					err    error
				)

				if user.Cache.Filter == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
					events, err = c.EventService.FindManyFromTodayByBandIDAndUserID(user.BandID, user.ID, user.Cache.PageIndex)
				} else if user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) {
					if keyboard.IsWeekdayButton(ctx.EffectiveMessage.Text) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, keyboard.ParseWeekdayButton(ctx.EffectiveMessage.Text), user.Cache.PageIndex)
						user.Cache.Query = ctx.EffectiveMessage.Text
					} else if keyboard.IsWeekdayButton(user.Cache.Query) && (ctx.EffectiveMessage.Text == txt.Get("button.next", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode)) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, keyboard.ParseWeekdayButton(user.Cache.Query), user.Cache.PageIndex)
					} else if ctx.EffectiveMessage.Text == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndUserIDAndPageNumber(user.BandID, user.ID, user.Cache.PageIndex)
						user.Cache.Query = ctx.EffectiveMessage.Text
					} else if user.Cache.Query == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) && (ctx.EffectiveMessage.Text == txt.Get("button.next", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode)) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndUserIDAndPageNumber(user.BandID, user.ID, user.Cache.PageIndex)
					} else {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndPageNumber(user.BandID, user.Cache.PageIndex)
						user.Cache.Buttons = keyboard.GetEventsStateFilterButtons(events, ctx.EffectiveUser.LanguageCode)
					}
				} else if keyboard.IsWeekdayButton(user.Cache.Filter) {
					events, err = c.EventService.FindManyFromTodayByBandIDAndWeekday(user.BandID, keyboard.ParseWeekdayButton(user.Cache.Filter))
				}
				if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode),
				}

				var buttons []gotgbot.KeyboardButton
				for _, button := range user.Cache.Buttons {

					if button.Text == user.Cache.Filter ||
						(button.Text == ctx.EffectiveMessage.Text && user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode)) ||
						(button.Text == user.Cache.Query && user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) && (ctx.EffectiveMessage.Text == txt.Get("button.next", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode))) {
						button = keyboard.SelectedButton(button.Text)
					}

					buttons = append(buttons, button)
				}

				markup.Keyboard = append(markup.Keyboard, buttons)
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.createEvent", ctx.EffectiveUser.LanguageCode), WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/events/create"}}})

				for _, event := range events {
					if user.Cache.Filter == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
						markup.Keyboard = append(markup.Keyboard, keyboard.EventButton(event, user, ctx.EffectiveUser.LanguageCode, true))
					} else {
						markup.Keyboard = append(markup.Keyboard, keyboard.EventButton(event, user, ctx.EffectiveUser.LanguageCode, false))
					}
				}

				if user.Cache.PageIndex != 0 {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.prev", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.next", ctx.EffectiveUser.LanguageCode)}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.next", ctx.EffectiveUser.LanguageCode)}})
				}

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseEvent", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode), txt.Get("button.archive", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex = 0
					return c.filterEvents(0)(bot, ctx)
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex++
					return c.filterEvents(0)(bot, ctx)
				case txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex--
					return c.filterEvents(0)(bot, ctx)
				default:
					if keyboard.IsWeekdayButton(ctx.EffectiveMessage.Text) {
						user.Cache.PageIndex = 0
						return c.filterEvents(0)(bot, ctx)
					}
				}

				if keyboard.IsSelectedButton(ctx.EffectiveMessage.Text) {
					if user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) {
						if keyboard.IsWeekdayButton(keyboard.ParseSelectedButton(ctx.EffectiveMessage.Text)) ||
							keyboard.ParseSelectedButton(ctx.EffectiveMessage.Text) == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
							return c.filterEvents(0)(bot, ctx)
						} else {
							return c.GetEvents(0)(bot, ctx)
						}
					} else {
						return c.GetEvents(0)(bot, ctx)
					}
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				eventName, eventTime, err := keyboard.ParseEventButton(ctx.EffectiveMessage.Text)
				if err != nil {
					return c.search(0)(bot, ctx)
				}

				foundEvent, err := c.EventService.FindOneByNameAndTimeAndBandID(eventName, eventTime, user.BandID)
				if err != nil {
					return c.GetEvents(0)(bot, ctx)
				}

				err = c.event(bot, ctx, foundEvent)
				return err
			}
		}

		return nil
	}
}

// ------- Callback controllers -------

func (c *BotController) EventSetlistDocs(bot *gotgbot.Bot, ctx *ext.Context) error {

	eventIDHex := util.ParseCallbackPayload(ctx.CallbackQuery.Data)

	eventID, err := primitive.ObjectIDFromHex(eventIDHex)
	if err != nil {
		return err
	}
	event, err := c.EventService.GetEventWithSongs(eventID)
	if err != nil {
		return err
	}

	var driveFileIDs []string
	for _, song := range event.Songs {
		driveFileIDs = append(driveFileIDs, song.DriveFileID)
	}

	err = c.songsAlbum(bot, ctx, driveFileIDs)
	if err != nil {
		return err
	}

	ctx.CallbackQuery.Answer(bot, nil)
	return nil
}

func (c *BotController) EventSetlistMetronome(bot *gotgbot.Bot, ctx *ext.Context) error {
	{

		eventIDHex := util.ParseCallbackPayload(ctx.CallbackQuery.Data)

		eventID, err := primitive.ObjectIDFromHex(eventIDHex)
		if err != nil {
			return err
		}
		event, err := c.EventService.GetEventWithSongs(eventID)
		if err != nil {
			return err
		}

		var bigAlbum []gotgbot.InputMedia

		for _, song := range event.Songs {
			audio := &gotgbot.InputMediaAudio{
				Media:   metronome.GetMetronomeTrackFileID(song.PDF.BPM, song.PDF.Time),
				Caption: "↑ " + song.PDF.Name,
			}

			bigAlbum = append(bigAlbum, audio)
		}

		chunks := chunkAlbum(bigAlbum, 10)

		for _, album := range chunks {
			_, err := bot.SendMediaGroup(ctx.EffectiveChat.Id, album, nil)
			if err != nil {
				return err
			}
		}

		ctx.CallbackQuery.Answer(bot, nil)
		return nil
	}
}

func (c *BotController) EventCB(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	payload := util.ParseCallbackPayload(ctx.CallbackQuery.Data)
	split := strings.Split(payload, ":")

	hex := split[0]
	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{}

	if len(split) > 1 {
		switch split[1] {
		case "edit":
			markup.InlineKeyboard = keyboard.EventEdit(eventID, user, ctx.EffectiveUser.LanguageCode)
		default:
			event, err := c.EventService.FindOneByID(eventID)
			if err != nil {
				return err
			}
			markup.InlineKeyboard = keyboard.EventInit(event, user, ctx.EffectiveUser.LanguageCode)
		}
	}

	_, _, err = ctx.EffectiveMessage.EditReplyMarkup(bot, &gotgbot.EditMessageReplyMarkupOpts{
		ReplyMarkup: markup,
	})

	return err
}

func (c *BotController) EventSetlist(bot *gotgbot.Bot, ctx *ext.Context) error {

	hex := util.ParseCallbackPayload(ctx.CallbackQuery.Data)

	eventID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return err
	}

	event, err := c.EventService.GetEventWithSongs(eventID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{}

	markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.changeSongsOrder", ctx.EffectiveUser.LanguageCode), CallbackData: "todo"}})
	for _, song := range event.Songs {
		text := song.PDF.Name

		for _, eventSong := range event.Songs {
			if eventSong.ID == song.ID {
				text += " ✅"
				break
			}
		}

		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: text, CallbackData: "todo"}})
	}
	markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.addSong", ctx.EffectiveUser.LanguageCode), CallbackData: "todo"}})
	markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.back", ctx.EffectiveUser.LanguageCode), CallbackData: util.CallbackData(state.EventCB, event.ID.Hex()+":edit")}})

	text := fmt.Sprintf("<b>%s</b>\n\n%s:", event.Alias(ctx.EffectiveUser.LanguageCode), helpers.Setlist)

	text += "<a href=\"t.me/callbackData\">&#8203;</a>"

	_, _, err = ctx.EffectiveMessage.EditText(bot, text, &gotgbot.EditMessageTextOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
		ReplyMarkup:           markup,
	})
	if err != nil {
		return err
	}

	ctx.CallbackQuery.Answer(bot, nil)

	//var songs []*entity.Song
	//if payload != "deleted" {
	//
	//	songsJson, err := json.Marshal(event.Songs)
	//	if err != nil {
	//		return err
	//	}
	//
	//	q := user.State.CallbackData.Query()
	//	q.Set("eventId", eventID.Hex())
	//	q.Set("eventAlias", event.Alias(ctx.EffectiveUser.LanguageCode))
	//	q.Set("songs", string(songsJson))
	//	q.Del("index")
	//	q.Del("driveFileIds")
	//	user.State.CallbackData.RawQuery = q.Encode()
	//
	//	songs = event.Songs
	//} else {
	//	songsJson := user.State.CallbackData.Query().Get("songs")
	//
	//	err := json.Unmarshal([]byte(songsJson), &songs)
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}
