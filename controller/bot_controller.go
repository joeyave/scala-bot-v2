package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/joeyave/scala-bot-v2/dto"
	"github.com/joeyave/scala-bot-v2/entities"
	myhandlers "github.com/joeyave/scala-bot-v2/handlers"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/services"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/drive/v3"
	"os"
	"regexp"
	"strings"
	"time"
)

type BotController struct {
	UserService       *services.UserService
	DriveFileService  *services.DriveFileService
	SongService       *services.SongService
	VoiceService      *services.VoiceService
	BandService       *services.BandService
	MembershipService *services.MembershipService
	EventService      *services.EventService
	RoleService       *services.RoleService
	OldHandler        *myhandlers.Handler
}

func (c *BotController) ChooseHandlerOrSearch(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	switch user.State.Name {
	case state.GetEvents:
		return c.GetEvents(user.State.Index)(bot, ctx)
	case state.FilterEvents:
		return c.FilterEvents(user.State.Index)(bot, ctx)
	case state.GetSongs:
		return c.GetSongs(user.State.Index)(bot, ctx)
	case state.FilterSongs:
		return c.FilterSongs(user.State.Index)(bot, ctx)
	}

	return c.Search(user.State.Index)(bot, ctx)
}

func (c *BotController) RegisterUser(bot *gotgbot.Bot, ctx *ext.Context) error {

	user, err := c.UserService.FindOneOrCreateByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	user.Name = strings.TrimSpace(fmt.Sprintf("%s %s", ctx.EffectiveChat.FirstName, ctx.EffectiveChat.LastName))

	if user.BandID == primitive.NilObjectID && user.State.Name != helpers.ChooseBandState && user.State.Name != helpers.CreateBandState {
		user.State = entities.State{
			Name: helpers.ChooseBandState,
		}
	}

	ctx.Data["user"] = user

	return nil
}

func (c *BotController) UpdateUser(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	_, err := c.UserService.UpdateOne(*user)
	return err
}

func (c *BotController) Event(bot *gotgbot.Bot, ctx *ext.Context, event *entities.Event) error {

	user := ctx.Data["user"].(*entities.User)

	html := c.EventService.ToHtmlStringByEvent(*event)

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: helpers.GetEventActionsKeyboard(*user, *event),
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

	user := ctx.Data["user"].(*entities.User)

	eventDate, err := time.Parse("2006-01-02", data.Event.Date)
	if err != nil {
		return err
	}

	event := entities.Event{
		Time:   eventDate,
		Name:   data.Event.Name,
		BandID: user.BandID,
	}
	createdEvent, err := c.EventService.UpdateOne(event)
	if err != nil {
		return err
	}

	user.State.Index = 0
	err = c.Event(bot, ctx, createdEvent)
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

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.GetEvents {
			user.State = entities.State{
				Index: index,
				Name:  state.GetEvents,
			}
			user.Cache = entities.Cache{}
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
					InputFieldPlaceholder: "Фраза из песни или список",
				}

				user.Cache.Buttons = helpers.GetWeekdayButtons(events)
				markup.Keyboard = append(markup.Keyboard, user.Cache.Buttons)
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: "➕ Добавить собрание", WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/create-event"}}})

				for _, event := range events {
					buttonText := helpers.EventButton(event, user, false)
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
				}

				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}})

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseEvent", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.Cache.Filter = "-" // todo: remove

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode), txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					return c.GetEvents(0)(bot, ctx)

				case txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode), txt.Get("button.archive", ctx.EffectiveUser.LanguageCode):
					ctx.Data["buttons"] = user.Cache.Buttons
					return c.FilterEvents(0)(bot, ctx)

				default:
					if helpers.IsWeekdayString(ctx.EffectiveMessage.Text) {
						ctx.Data["buttons"] = user.Cache.Buttons
						return c.FilterEvents(0)(bot, ctx)
					}
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				eventName, eventTime, err := helpers.ParseEventButton(ctx.EffectiveMessage.Text)
				if err != nil {
					return c.Search(0)(bot, ctx)
				}

				foundEvent, err := c.EventService.FindOneByNameAndTimeAndBandID(eventName, eventTime, user.BandID)
				if err != nil {
					return c.GetEvents(0)(bot, ctx)
				}

				//event, err := c.EventService.FindOneByID(foundEvent.ID)
				//if err != nil {
				//	return err
				//}

				err = c.Event(bot, ctx, foundEvent)
				return err
			}
		}
		return nil
	}
}

func (c *BotController) FilterEvents(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.FilterEvents {
			user.State = entities.State{
				Index: index,
				Name:  state.FilterEvents,
			}
			user.Cache = entities.Cache{}
		}

		switch index {
		case 0:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				if (ctx.EffectiveMessage.Text == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) ||
					helpers.IsWeekdayString(ctx.EffectiveMessage.Text)) && user.Cache.Filter != txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) {
					user.Cache.Filter = ctx.EffectiveMessage.Text
				}

				var (
					events []*entities.Event
					err    error
				)

				if user.Cache.Filter == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
					events, err = c.EventService.FindManyFromTodayByBandIDAndUserID(user.BandID, user.ID, user.Cache.PageIndex)
				} else if user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) {
					if helpers.IsWeekdayString(ctx.EffectiveMessage.Text) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, helpers.GetWeekdayFromString(ctx.EffectiveMessage.Text), user.Cache.PageIndex)
						user.Cache.Query = ctx.EffectiveMessage.Text
					} else if helpers.IsWeekdayString(user.Cache.Query) && (ctx.EffectiveMessage.Text == txt.Get("button.next", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode)) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, helpers.GetWeekdayFromString(user.Cache.Query), user.Cache.PageIndex)
					} else if ctx.EffectiveMessage.Text == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndUserIDAndPageNumber(user.BandID, user.ID, user.Cache.PageIndex)
						user.Cache.Query = ctx.EffectiveMessage.Text
					} else if user.Cache.Query == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) && (ctx.EffectiveMessage.Text == txt.Get("button.next", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode)) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndUserIDAndPageNumber(user.BandID, user.ID, user.Cache.PageIndex)
					} else {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndPageNumber(user.BandID, user.Cache.PageIndex)
						user.Cache.Buttons = helpers.GetWeekdayButtons(events)
					}
				} else if helpers.IsWeekdayString(user.Cache.Filter) {
					events, err = c.EventService.FindManyFromTodayByBandIDAndWeekday(user.BandID, helpers.GetWeekdayFromString(user.Cache.Filter))
				}
				if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: "Фраза из песни или список",
				}

				if len(user.Cache.Buttons) == 0 {
					user.Cache.Buttons = ctx.Data["buttons"].([]gotgbot.KeyboardButton)
				}

				var buttons []gotgbot.KeyboardButton
				for _, button := range user.Cache.Buttons {
					buttons = append(buttons, button)
				}

				markup.Keyboard = append(markup.Keyboard, buttons)
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: "➕ Добавить собрание", WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/create-event"}}})

				for i := range markup.Keyboard[0] {
					if markup.Keyboard[0][i].Text == user.Cache.Filter || (markup.Keyboard[0][i].Text == ctx.EffectiveMessage.Text && user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode)) ||
						(markup.Keyboard[0][i].Text == user.Cache.Query && user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) && (ctx.EffectiveMessage.Text == txt.Get("button.next", ctx.EffectiveUser.LanguageCode) || ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode))) {
						markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
					}
				}

				for _, event := range events {

					buttonText := ""
					if user.Cache.Filter == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
						buttonText = helpers.EventButton(event, user, true)
					} else {
						buttonText = helpers.EventButton(event, user, false)
					}

					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
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
					return c.FilterEvents(0)(bot, ctx)
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex++
					return c.FilterEvents(0)(bot, ctx)
				case txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex--
					return c.FilterEvents(0)(bot, ctx)
				default:
					if helpers.IsWeekdayString(ctx.EffectiveMessage.Text) {
						user.Cache.PageIndex = 0
						return c.FilterEvents(0)(bot, ctx)
					}
				}

				if strings.Contains(ctx.EffectiveMessage.Text, "〔") && strings.Contains(ctx.EffectiveMessage.Text, "〕") {
					if user.Cache.Filter == txt.Get("button.archive", ctx.EffectiveUser.LanguageCode) {
						if helpers.IsWeekdayString(strings.ReplaceAll(strings.ReplaceAll(ctx.EffectiveMessage.Text, "〔", ""), "〕", "")) ||
							strings.ReplaceAll(strings.ReplaceAll(ctx.EffectiveMessage.Text, "〔", ""), "〕", "") == txt.Get("button.eventsWithMe", ctx.EffectiveUser.LanguageCode) {
							return c.FilterEvents(0)(bot, ctx)
						} else {
							return c.GetEvents(0)(bot, ctx)
						}
					} else {
						return c.GetEvents(0)(bot, ctx)
					}
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				eventName, eventTime, err := helpers.ParseEventButton(ctx.EffectiveMessage.Text)
				if err != nil {
					return c.Search(0)(bot, ctx)
				}

				foundEvent, err := c.EventService.FindOneByNameAndTimeAndBandID(eventName, eventTime, user.BandID)
				if err != nil {
					return c.GetEvents(0)(bot, ctx)
				}

				err = c.Event(bot, ctx, foundEvent)
				return err
			}
		case 2:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				tags, err := c.SongService.GetTags()
				if err != nil {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode),
				}
				markup.Keyboard = [][]gotgbot.KeyboardButton{
					{
						{Text: txt.Get("button.like", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.tag", ctx.EffectiveUser.LanguageCode)},
					},
				}

				for i := range markup.Keyboard[0] {
					if markup.Keyboard[0][i].Text == user.Cache.Filter {
						markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
						break
					}
				}

				for _, tag := range tags {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: tag}})
				}

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseTag", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.State.Index = 0
				return nil
			}
		}

		return nil
	}
}

func (c *BotController) Search(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.Search {
			user.State = entities.State{
				Index: index,
				Name:  state.Search,
			}
			user.Cache = entities.Cache{}
		}

		switch index {
		case 0:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				var query string

				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode):
					user.Cache.Filter = ctx.EffectiveMessage.Text
					query = user.Cache.Query
				case txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					query = user.Cache.Query
					user.Cache.NextPageToken = user.Cache.NextPageToken.Prev.Prev
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					query = user.Cache.Query
				default:
					query = ctx.EffectiveMessage.Text
					// Обнуляем страницы при новом запросе.
					user.Cache.NextPageToken = nil
				}

				query = helpers.CleanUpQuery(query)
				songNames := helpers.SplitQueryByNewlines(query)

				if len(songNames) > 1 {
					// todo
					user.State = entities.State{
						Index: 0,
						Name:  helpers.SetlistState,
						Next: &entities.State{
							Index: 2,
							Name:  helpers.SearchSongState,
						},
						Context: user.State.Context,
					}
					user.State.Context.SongNames = songNames
					return c.OldHandler.Enter(ctx, user)

				} else if len(songNames) == 1 {
					query = songNames[0]
					user.Cache.Query = query
				} else {
					_, err := ctx.EffectiveChat.SendMessage(bot, "Из запроса удаляются все числа, дефисы и скобки вместе с тем, что в них.", nil)
					return err
				}

				//if user.Cache.nextPageToken == nil {
				//	user.Cache.nextPageToken = &entities.nextPageToken{}
				//}

				var (
					driveFiles    []*drive.File
					nextPageToken string
					err           error
				)
				switch user.Cache.Filter {
				case txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode):
					driveFiles, nextPageToken, err = c.DriveFileService.FindSomeByFullTextAndFolderID(query, "", user.Cache.NextPageToken.GetValue())
				default:
					driveFiles, nextPageToken, err = c.DriveFileService.FindSomeByFullTextAndFolderID(query, user.Band.DriveFolderID, user.Cache.NextPageToken.GetValue())
				}
				if err != nil {
					return err
				}

				user.Cache.NextPageToken = &entities.NextPageToken{
					Value: nextPageToken,
					Prev:  user.Cache.NextPageToken,
				}

				if len(driveFiles) == 0 {
					markup := &gotgbot.ReplyKeyboardMarkup{
						Keyboard: [][]gotgbot.KeyboardButton{
							{{Text: txt.Get("button.cancel", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode)}},
						},
						ResizeKeyboard: true,
					}
					_, err := ctx.EffectiveChat.SendMessage(bot, txt.Get("text.nothingFound", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: query,
				}
				if markup.InputFieldPlaceholder == "" {
					markup.InputFieldPlaceholder = txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode)
				}

				likedSongs, likedSongErr := c.SongService.FindManyLiked(user.ID)

				set := make(map[string]*entities.Band)
				for i, driveFile := range driveFiles {

					if user.Cache.Filter == txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode) {

						for _, parentFolderID := range driveFile.Parents {
							_, exists := set[parentFolderID]
							if !exists {
								band, err := c.BandService.FindOneByDriveFolderID(parentFolderID)
								if err == nil {
									set[parentFolderID] = band
									driveFiles[i].Name += fmt.Sprintf(" (%s)", band.Name)
									break
								}
							} else {
								driveFiles[i].Name += fmt.Sprintf(" (%s)", set[parentFolderID].Name)
							}
						}
					}
					driveFileName := driveFile.Name

					if likedSongErr == nil {
						for _, likedSong := range likedSongs {
							if likedSong.DriveFileID == driveFile.Id {
								driveFileName += " " + txt.Get("button.like", ctx.EffectiveUser.LanguageCode)
							}
						}
					}

					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: driveFileName}})
				}

				if ctx.EffectiveMessage.Text != txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode) {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode)}})
				}

				markup.Keyboard = append(markup.Keyboard, keyboard.Navigation(user.Cache.NextPageToken, ctx.EffectiveUser.LanguageCode)...)

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseSong", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.Cache.DriveFiles = driveFiles

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode), txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					return c.Search(0)(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "upload_document")

				driveFiles := user.Cache.DriveFiles
				var foundDriveFile *drive.File
				for _, driveFile := range driveFiles {
					if driveFile.Name == strings.ReplaceAll(ctx.EffectiveMessage.Text, " "+txt.Get("button.like", ctx.EffectiveUser.LanguageCode), "") {
						foundDriveFile = driveFile
						break
					}
				}

				if foundDriveFile != nil {
					return c.Song(bot, ctx, foundDriveFile.Id)
				} else {
					return c.Search(0)(bot, ctx)
				}
			}
		}
		return nil
	}
}

func (c *BotController) Song(bot *gotgbot.Bot, ctx *ext.Context, driveFileID string) error {

	user := ctx.Data["user"].(*entities.User)

	ctx.EffectiveChat.SendAction(bot, "upload_document")

	song, driveFile, err := c.SongService.FindOrCreateOneByDriveFileID(driveFileID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: helpers.GetSongInitKeyboard(user, song),
	}

	sendDocumentByReader := func() (*gotgbot.Message, error) {
		reader, err := c.DriveFileService.DownloadOneByID(driveFile.Id)
		if err != nil {
			return nil, err
		}

		message, err := bot.SendDocument(ctx.EffectiveChat.Id, gotgbot.NamedFile{
			File:     *reader,
			FileName: fmt.Sprintf("%s.pdf", driveFile.Name),
		}, &gotgbot.SendDocumentOpts{
			Caption:     song.Caption() + "\n" + strings.Join(song.Tags, ", "),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		return message, err
	}

	sendDocumentByFileID := func() (*gotgbot.Message, error) {
		message, err := bot.SendDocument(ctx.EffectiveChat.Id, song.PDF.TgFileID, &gotgbot.SendDocumentOpts{
			Caption:     song.Caption() + "\n" + strings.Join(song.Tags, ", "),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		return message, err
	}

	var msg *gotgbot.Message
	if song.PDF.TgFileID == "" {
		msg, err = sendDocumentByReader()
	} else {
		msg, err = sendDocumentByFileID()
		if err != nil {
			msg, err = sendDocumentByReader()
		}
	}
	if err != nil {
		return err
	}

	song.PDF.TgFileID = msg.Document.FileId

	// todo
	//err = SendSongToChannel(h, c, user, song)
	//if err != nil {
	//	return err
	//}

	song, err = c.SongService.UpdateOne(*song)
	if err != nil {
		return err
	}

	return nil
}

func (c *BotController) GetSongs(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.GetSongs {
			user.State = entities.State{
				Index: index,
				Name:  state.GetSongs,
			}
			user.Cache = entities.Cache{}
		}

		switch index {
		case 0:
			{
				// todo
				if ctx.EffectiveMessage.Text == txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode) {
					user.State = entities.State{
						Name: helpers.CreateSongState,
					}
					return c.OldHandler.Enter(ctx, user)
				}

				ctx.EffectiveChat.SendAction(bot, "typing")

				if ctx.EffectiveMessage.Text == txt.Get("button.prev", ctx.EffectiveUser.LanguageCode) && user.Cache.NextPageToken.GetPrevValue() != "" {
					user.Cache.NextPageToken = user.Cache.NextPageToken.Prev.Prev
				}

				driveFiles, nextPageToken, err := c.DriveFileService.FindAllByFolderID(user.Band.DriveFolderID, user.Cache.NextPageToken.GetValue())
				if err != nil {
					return err
				}

				user.Cache.NextPageToken = &entities.NextPageToken{
					Value: nextPageToken,
					Prev:  user.Cache.NextPageToken,
				}

				if len(driveFiles) == 0 {
					markup := &gotgbot.ReplyKeyboardMarkup{
						Keyboard: [][]gotgbot.KeyboardButton{
							{{Text: txt.Get("button.cancel", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.globalSearch", ctx.EffectiveUser.LanguageCode)}},
						},
						ResizeKeyboard: true,
					}
					_, err := ctx.EffectiveChat.SendMessage(bot, "В папке на Google Диске нет документов.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: "Фраза из песни или список",
				}

				markup.Keyboard = [][]gotgbot.KeyboardButton{
					{{Text: txt.Get("button.like", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.tag", ctx.EffectiveUser.LanguageCode)}},
				}
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode)}})

				likedSongs, likedSongErr := c.SongService.FindManyLiked(user.ID)

				for _, driveFile := range driveFiles {
					driveFileName := driveFile.Name

					if likedSongErr == nil {
						for _, likedSong := range likedSongs {
							if likedSong.DriveFileID == driveFile.Id {
								driveFileName += " " + txt.Get("button.like", ctx.EffectiveUser.LanguageCode)
							}
						}
					}

					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: driveFileName}})
				}

				markup.Keyboard = append(markup.Keyboard, keyboard.Navigation(user.Cache.NextPageToken, ctx.EffectiveUser.LanguageCode)...)

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseSong", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.Cache.DriveFiles = driveFiles

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {

				// todo
				case txt.Get("button.createDoc", ctx.EffectiveUser.LanguageCode):
					user.State = entities.State{
						Name: helpers.CreateSongState,
					}
					return c.OldHandler.Enter(ctx, user)

				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode), txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					return c.GetSongs(0)(bot, ctx)

				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode), txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode), txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode), txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					return c.FilterSongs(0)(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "upload_document")

				driveFiles := user.Cache.DriveFiles
				var foundDriveFile *drive.File
				for _, driveFile := range driveFiles {
					if driveFile.Name == strings.ReplaceAll(ctx.EffectiveMessage.Text, " "+txt.Get("button.like", ctx.EffectiveUser.LanguageCode), "") {
						foundDriveFile = driveFile
						break
					}
				}

				if foundDriveFile != nil {
					return c.Song(bot, ctx, foundDriveFile.Id)
				} else {
					return c.Search(0)(bot, ctx)
				}
			}
		}
		return nil
	}
}

func (c *BotController) FilterSongs(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entities.User)

		if user.State.Name != state.FilterSongs {
			user.State = entities.State{
				Index: index,
				Name:  state.FilterSongs,
			}
			user.Cache = entities.Cache{}
		}

		switch index {
		case 0:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode), txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode), txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode):
					user.Cache.Filter = ctx.EffectiveMessage.Text

				case txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					user.Cache.Filter = ctx.EffectiveMessage.Text
					return c.FilterSongs(2)(bot, ctx)
				}

				var (
					songs []*entities.SongExtra
					err   error
				)

				switch user.Cache.Filter {
				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode):
					songs, err = c.SongService.FindManyExtraLiked(user.ID, user.Cache.PageIndex)
				case txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode):
					songs, err = c.SongService.FindAllExtraByPageNumberSortedByLatestEventDate(user.BandID, user.Cache.PageIndex)
				case txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode):
					songs, err = c.SongService.FindAllExtraByPageNumberSortedByEventsNumber(user.BandID, user.Cache.PageIndex)
				case txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					if strings.Contains(ctx.EffectiveMessage.Text, "〔") {
						return c.GetSongs(0)(bot, ctx)
					}
					if user.Cache.Query == "" {
						user.Cache.Query = ctx.EffectiveMessage.Text
					}
					songs, err = c.SongService.FindManyExtraByTag(user.Cache.Query, user.BandID, user.Cache.PageIndex)
				}
				if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: "Фраза из песни или список",
				}
				markup.Keyboard = [][]gotgbot.KeyboardButton{
					{
						{Text: txt.Get("button.like", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.tag", ctx.EffectiveUser.LanguageCode)},
					},
				}

				for i := range markup.Keyboard[0] {
					if markup.Keyboard[0][i].Text == user.Cache.Filter {
						markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
						break
					}
				}

				for _, songExtra := range songs {
					buttonText := songExtra.Song.PDF.Name
					if songExtra.Caption() != "" {
						buttonText += fmt.Sprintf(" (%s)", songExtra.Caption())
					}

					if user.Cache.Filter != txt.Get("button.like", ctx.EffectiveUser.LanguageCode) {
						for _, userID := range songExtra.Song.Likes {
							if user.ID == userID {
								buttonText += " " + txt.Get("button.like", ctx.EffectiveUser.LanguageCode)
								break
							}
						}
					}

					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
				}

				if user.Cache.PageIndex != 0 {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.prev", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.next", ctx.EffectiveUser.LanguageCode)}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: txt.Get("button.menu", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.next", ctx.EffectiveUser.LanguageCode)}})
				}

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseSong", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.State.Index = 1

				return nil
			}
		case 1:
			{
				switch ctx.EffectiveMessage.Text {
				case txt.Get("button.like", ctx.EffectiveUser.LanguageCode), txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode), txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode), txt.Get("button.tag", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex = 0
					return c.FilterSongs(0)(bot, ctx)
				case txt.Get("button.next", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex++
					return c.FilterSongs(0)(bot, ctx)
				case txt.Get("button.prev", ctx.EffectiveUser.LanguageCode):
					user.Cache.PageIndex--
					return c.FilterSongs(0)(bot, ctx)
				}

				if strings.Contains(ctx.EffectiveMessage.Text, "〔") && strings.Contains(ctx.EffectiveMessage.Text, "〕") {
					return c.GetSongs(0)(bot, ctx)
				}

				ctx.EffectiveChat.SendAction(bot, "upload_document")

				var songName string
				regex := regexp.MustCompile(`\s*\(.*\)\s*(` + txt.Get("button.like", ctx.EffectiveUser.LanguageCode) + `)?\s*`)
				songName = regex.ReplaceAllString(ctx.EffectiveMessage.Text, "")

				song, err := c.SongService.FindOneByName(strings.TrimSpace(songName))
				if err != nil {
					return c.Search(0)(bot, ctx)
				}

				return c.Song(bot, ctx, song.DriveFileID)
			}
		case 2:
			{
				ctx.EffectiveChat.SendAction(bot, "typing")

				tags, err := c.SongService.GetTags()
				if err != nil {
					return err
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					ResizeKeyboard:        true,
					InputFieldPlaceholder: txt.Get("text.defaultPlaceholder", ctx.EffectiveUser.LanguageCode),
				}
				markup.Keyboard = [][]gotgbot.KeyboardButton{
					{
						{Text: txt.Get("button.like", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.calendar", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.numbers", ctx.EffectiveUser.LanguageCode)}, {Text: txt.Get("button.tag", ctx.EffectiveUser.LanguageCode)},
					},
				}

				for i := range markup.Keyboard[0] {
					if markup.Keyboard[0][i].Text == user.Cache.Filter {
						markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
						break
					}
				}

				for _, tag := range tags {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: tag}})
				}

				_, err = ctx.EffectiveChat.SendMessage(bot, txt.Get("text.chooseTag", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				user.State.Index = 0
				return nil
			}
		}

		return nil
	}
}

func (c *BotController) Menu(b *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	user.State = entities.State{}

	replyMarkup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard:       keyboard.Menu(ctx.EffectiveUser.LanguageCode),
		ResizeKeyboard: true,
	}

	_, err := ctx.EffectiveChat.SendMessage(b, txt.Get("text.menu", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return err
	}

	return nil
}
