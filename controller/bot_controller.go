package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/dto"
	"github.com/joeyave/scala-bot-v2/entities"
	myhandlers "github.com/joeyave/scala-bot-v2/handlers"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/services"
	"github.com/joeyave/scala-bot-v2/state"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/drive/v3"
	"os"
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
		return c.GetEvents(bot, ctx)
	case state.GetSongs:
		return c.GetSongs(bot, ctx)
	}

	return c.Search(bot, ctx)
}

func (c *BotController) RegisterUser(bot *gotgbot.Bot, ctx *ext.Context) error {

	user, err := c.UserService.FindOneOrCreateByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	user.Name = strings.TrimSpace(fmt.Sprintf("%s %s", ctx.EffectiveChat.FirstName, ctx.EffectiveChat.LastName))

	if user.State == nil {
		user.State = &entities.State{Name: 0}
	}

	if user.BandID == primitive.NilObjectID && user.State.Name != helpers.ChooseBandState && user.State.Name != helpers.CreateBandState {
		user.State = &entities.State{
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
	err = c.GetEvents(bot, ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *BotController) GetEvents(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	if ctx.EffectiveMessage.Text == "🗓️ Расписание" {
		user.State.Index = 0
	}

	switch user.State.Index {
	case 0:
		{
			events, err := c.EventService.FindManyFromTodayByBandID(user.BandID)
			if err != nil {
				return err
			}

			markup := &gotgbot.ReplyKeyboardMarkup{
				ResizeKeyboard:        true,
				InputFieldPlaceholder: "Фраза из песни или список",
			}

			user.State.Context.WeekdayButtons = helpers.GetWeekdayButtons(events)
			markup.Keyboard = append(markup.Keyboard, user.State.Context.WeekdayButtons)
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: "➕ Добавить собрание", WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/create-event"}}})

			for _, event := range events {
				buttonText := helpers.EventButton(event, user, false)
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
			}

			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}})

			_, err = ctx.EffectiveChat.SendMessage(bot, "Выбери собрание:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			if err != nil {
				return err
			}

			user.State.Context.QueryType = "-"

			user.State = &entities.State{
				Name:    state.GetEvents,
				Index:   1,
				Context: user.State.Context,
			}

			return nil
		}
	case 1:
		{
			text := ctx.EffectiveMessage.Text

			if strings.Contains(text, "〔") && strings.Contains(text, "〕") {
				if helpers.IsWeekdayString(strings.ReplaceAll(strings.ReplaceAll(text, "〔", ""), "〕", "")) && user.State.Context.QueryType == helpers.Archive {
					text = helpers.Archive
				} else {
					user.State.Index = 0
					ctx.Data["user"] = user
					return c.GetEvents(bot, ctx)
				}
			}

			markup := &gotgbot.ReplyKeyboardMarkup{
				ResizeKeyboard:        true,
				InputFieldPlaceholder: helpers.Placeholder,
			}

			if text == helpers.GetEventsWithMe || text == helpers.Archive || text == helpers.PrevPage || text == helpers.NextPage || helpers.IsWeekdayString(text) {

				ctx.EffectiveChat.SendAction(bot, "typing")

				if text == helpers.NextPage {
					user.State.Context.PageIndex++
				} else if text == helpers.PrevPage {
					user.State.Context.PageIndex--
				} else {
					if user.State.Context.QueryType == helpers.Archive && helpers.IsWeekdayString(text) {
						// todo
					} else {
						user.State.Context.QueryType = text
						if user.State.Context.QueryType == helpers.ByWeekday {
							user.State.Context.QueryType = helpers.GetWeekdayString(time.Now())
						}
					}
				}

				var buttons []gotgbot.KeyboardButton
				for _, button := range user.State.Context.WeekdayButtons {
					buttons = append(buttons, button)
				}

				markup.Keyboard = append(markup.Keyboard, buttons)
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: "➕ Добавить собрание", WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/create-event"}}})

				for i := range markup.Keyboard[0] {
					if markup.Keyboard[0][i].Text == user.State.Context.QueryType || (markup.Keyboard[0][i].Text == text && user.State.Context.QueryType == helpers.Archive) ||
						(markup.Keyboard[0][i].Text == user.State.Context.PrevText && user.State.Context.QueryType == helpers.Archive && (ctx.EffectiveMessage.Text == helpers.NextPage || ctx.EffectiveMessage.Text == helpers.PrevPage)) {
						markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
					}
				}

				var events []*entities.Event
				var err error
				switch user.State.Context.QueryType {
				case helpers.Archive:
					if helpers.IsWeekdayString(text) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, helpers.GetWeekdayFromString(text), user.State.Context.PageIndex)
						user.State.Context.PrevText = text
					} else if helpers.IsWeekdayString(user.State.Context.PrevText) && (ctx.EffectiveMessage.Text == helpers.NextPage || ctx.EffectiveMessage.Text == helpers.PrevPage) {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, helpers.GetWeekdayFromString(user.State.Context.PrevText), user.State.Context.PageIndex)
					} else {
						events, err = c.EventService.FindManyUntilTodayByBandIDAndPageNumber(user.BandID, user.State.Context.PageIndex)
					}
				case helpers.GetEventsWithMe:
					events, err = c.EventService.FindManyFromTodayByBandIDAndUserID(user.BandID, user.ID, user.State.Context.PageIndex)
				default:
					if helpers.IsWeekdayString(user.State.Context.QueryType) {
						events, err = c.EventService.FindManyFromTodayByBandIDAndWeekday(user.BandID, helpers.GetWeekdayFromString(user.State.Context.QueryType))
					}
				}
				if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
					return err
				}

				for _, event := range events {

					buttonText := ""
					if user.State.Context.QueryType == helpers.GetEventsWithMe {
						buttonText = helpers.EventButton(event, user, true)
					} else {
						buttonText = helpers.EventButton(event, user, false)
					}

					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
				}
				if user.State.Context.PageIndex != 0 {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.PrevPage}, {Text: helpers.Menu}, {Text: helpers.NextPage}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}, {Text: helpers.NextPage}})
				}

				_, err = ctx.EffectiveChat.SendMessage(bot, "Выбери собрание:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				if err != nil {
					return err
				}

				return nil
			} else {

				ctx.EffectiveChat.SendAction(bot, "typing")

				eventName, eventTime, err := helpers.ParseEventButton(text)
				if err != nil {
					user.State = &entities.State{
						Name: helpers.SearchSongState,
					}
					return nil
				}

				foundEvent, err := c.EventService.FindOneByNameAndTimeAndBandID(eventName, eventTime, user.BandID)
				if err != nil {
					user.State.Index = 0
					ctx.Data["user"] = user
					return c.GetEvents(bot, ctx)
				}

				event, err := c.EventService.FindOneByID(foundEvent.ID)
				if err != nil {
					return err
				}

				err = c.Event(bot, ctx, event)
				return err
			}
		}
	}
	return nil
}

func (c *BotController) Search(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	switch user.State.Index {
	case 0:
		{
			ctx.EffectiveChat.SendAction(bot, "typing")

			var query string
			if ctx.EffectiveMessage.Text == helpers.SearchEverywhere {
				user.State.Context.QueryType = ctx.EffectiveMessage.Text
				query = user.State.Context.Query
			} else if ctx.EffectiveMessage.Text == helpers.PrevPage || ctx.EffectiveMessage.Text == helpers.NextPage {
				query = user.State.Context.Query
			} else {
				user.State.Context.NextPageToken = nil
				query = ctx.EffectiveMessage.Text
			}

			query = helpers.CleanUpQuery(query)
			songNames := helpers.SplitQueryByNewlines(query)

			if len(songNames) > 1 {
				// todo
				user.State = &entities.State{
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
				user.State.Context.Query = query
			} else {
				_, err := ctx.EffectiveChat.SendMessage(bot, "Из запроса удаляются все числа, дефисы и скобки вместе с тем, что в них.", nil)
				return err
			}

			var driveFiles []*drive.File
			var nextPageToken string
			var err error

			if ctx.EffectiveMessage.Text == helpers.PrevPage {
				if user.State.Context.NextPageToken != nil &&
					user.State.Context.NextPageToken.PrevPageToken != nil {
					user.State.Context.NextPageToken = user.State.Context.NextPageToken.PrevPageToken.PrevPageToken
				}
			}

			if user.State.Context.NextPageToken == nil {
				user.State.Context.NextPageToken = &entities.NextPageToken{}
			}

			if user.State.Context.QueryType == helpers.SearchEverywhere {
				_driveFiles, _nextPageToken, _err := c.DriveFileService.FindSomeByFullTextAndFolderID(query, "", user.State.Context.NextPageToken.Token)
				driveFiles = _driveFiles
				nextPageToken = _nextPageToken
				err = _err
			} else {
				_driveFiles, _nextPageToken, _err := c.DriveFileService.FindSomeByFullTextAndFolderID(query, user.Band.DriveFolderID, user.State.Context.NextPageToken.Token)
				driveFiles = _driveFiles
				nextPageToken = _nextPageToken
				err = _err
			}

			if err != nil {
				return err
			}

			user.State.Context.NextPageToken = &entities.NextPageToken{
				Token:         nextPageToken,
				PrevPageToken: user.State.Context.NextPageToken,
			}

			if len(driveFiles) == 0 {
				markup := &gotgbot.ReplyKeyboardMarkup{
					Keyboard:       helpers.SearchEverywhereKeyboard,
					ResizeKeyboard: true,
				}
				_, err := ctx.EffectiveChat.SendMessage(bot, "Ничего не найдено. Попробуй еще раз.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				return err
			}

			markup := &gotgbot.ReplyKeyboardMarkup{
				ResizeKeyboard:        true,
				InputFieldPlaceholder: query,
			}
			if markup.InputFieldPlaceholder == "" {
				markup.InputFieldPlaceholder = helpers.Placeholder
			}

			likedSongs, likedSongErr := c.SongService.FindManyLiked(user.ID)

			set := make(map[string]*entities.Band)
			for i, driveFile := range driveFiles {

				if user.State.Context.QueryType == helpers.SearchEverywhere {

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
							driveFileName += " " + helpers.Like
						}
					}
				}

				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: driveFileName}})
			}

			if ctx.EffectiveMessage.Text != helpers.SearchEverywhere {
				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.SearchEverywhere}})
			}

			if user.State.Context.NextPageToken.Token != "" {
				if user.State.Context.NextPageToken.PrevPageToken != nil && user.State.Context.NextPageToken.PrevPageToken.Token != "" {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.PrevPage}, {Text: helpers.Menu}, {Text: helpers.NextPage}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}, {Text: helpers.NextPage}})
				}
			} else {
				if user.State.Context.NextPageToken.PrevPageToken.Token != "" {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.PrevPage}, {Text: helpers.Menu}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}, {Text: helpers.NextPage}})
				}
			}

			_, err = ctx.EffectiveChat.SendMessage(bot, "Выбери песню:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			if err != nil {
				return err
			}

			user.State.Context.DriveFiles = driveFiles

			user.State = &entities.State{
				Name:    state.Search,
				Index:   1,
				Context: user.State.Context,
			}

			return nil
		}
	case 1:
		{

			switch ctx.EffectiveMessage.Text {
			case helpers.SearchEverywhere, helpers.NextPage:
				user.State.Index = 0
				return c.Search(bot, ctx)
			}

			ctx.EffectiveChat.SendAction(bot, "upload_document")

			driveFiles := user.State.Context.DriveFiles
			var foundDriveFile *drive.File
			for _, driveFile := range driveFiles {
				if driveFile.Name == strings.ReplaceAll(ctx.EffectiveMessage.Text, " "+helpers.Like, "") {
					foundDriveFile = driveFile
					break
				}
			}

			if foundDriveFile != nil {
				return c.Song(bot, ctx, foundDriveFile.Id)
			} else {
				user.State.Index = 0
				return c.Search(bot, ctx)
			}
		}
	}
	return nil
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

func (c *BotController) GetSongs(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	switch user.State.Index {
	case 0:
		{

			//todo
			if ctx.EffectiveMessage.Text == helpers.CreateDoc {
				user.State = &entities.State{
					Name: helpers.CreateSongState,
				}
				return c.OldHandler.Enter(ctx, user) // todo: remove
			}

			ctx.EffectiveChat.SendAction(bot, "typing")

			user.State.Context.QueryType = helpers.Songs

			var driveFiles []*drive.File
			var nextPageToken string
			var err error

			if ctx.EffectiveMessage.Text == helpers.PrevPage {
				if user.State.Context.NextPageToken != nil && user.State.Context.NextPageToken.PrevPageToken != nil {
					user.State.Context.NextPageToken = user.State.Context.NextPageToken.PrevPageToken.PrevPageToken
				}
			}

			if user.State.Context.NextPageToken == nil {
				user.State.Context.NextPageToken = &entities.NextPageToken{}
			}

			if user.State.Context.QueryType == helpers.Songs {
				_driveFiles, _nextPageToken, _err := c.DriveFileService.FindAllByFolderID(user.Band.DriveFolderID, user.State.Context.NextPageToken.Token)
				driveFiles = _driveFiles
				nextPageToken = _nextPageToken
				err = _err
			}

			if err != nil {
				return err
			}

			user.State.Context.NextPageToken = &entities.NextPageToken{
				Token:         nextPageToken,
				PrevPageToken: user.State.Context.NextPageToken,
			}

			if len(driveFiles) == 0 {
				markup := &gotgbot.ReplyKeyboardMarkup{
					Keyboard:       helpers.SearchEverywhereKeyboard,
					ResizeKeyboard: true,
				}
				_, err := ctx.EffectiveChat.SendMessage(bot, "Ничего не найдено. Попробуй еще раз.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
				return err
			}

			markup := &gotgbot.ReplyKeyboardMarkup{
				ResizeKeyboard: true,
			}
			if markup.InputFieldPlaceholder == "" {
				markup.InputFieldPlaceholder = helpers.Placeholder
			}

			markup.Keyboard = [][]gotgbot.KeyboardButton{
				{{Text: helpers.LikedSongs}, {Text: helpers.SongsByLastDateOfPerforming}, {Text: helpers.SongsByNumberOfPerforming}, {Text: helpers.TagsEmoji}},
			}
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.CreateDoc}})

			likedSongs, likedSongErr := c.SongService.FindManyLiked(user.ID)

			set := make(map[string]*entities.Band)
			for i, driveFile := range driveFiles {

				if user.State.Context.QueryType == helpers.SearchEverywhere {

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
							driveFileName += " " + helpers.Like
						}
					}
				}

				markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: driveFileName}})
			}

			if user.State.Context.NextPageToken.Token != "" {
				if user.State.Context.NextPageToken.PrevPageToken != nil && user.State.Context.NextPageToken.PrevPageToken.Token != "" {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.PrevPage}, {Text: helpers.Menu}, {Text: helpers.NextPage}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}, {Text: helpers.NextPage}})
				}
			} else {
				if user.State.Context.NextPageToken.PrevPageToken.Token != "" {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.PrevPage}, {Text: helpers.Menu}})
				} else {
					markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}, {Text: helpers.NextPage}})
				}
			}

			_, err = ctx.EffectiveChat.SendMessage(bot, "Выбери песню:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			if err != nil {
				return err
			}

			user.State.Context.DriveFiles = driveFiles

			user.State = &entities.State{
				Name:    state.GetSongs,
				Index:   1,
				Context: user.State.Context,
			}

			return nil
		}
	case 1:
		{
			switch ctx.EffectiveMessage.Text {
			case helpers.CreateDoc:
				user.State = &entities.State{
					Name: helpers.CreateSongState,
				}
				return c.OldHandler.Enter(ctx, user)

			case helpers.NextPage, helpers.PrevPage:
				user.State.Index = 0
				return c.GetSongs(bot, ctx)

				// todo
				//case helpers.SongsByLastDateOfPerforming, helpers.SongsByNumberOfPerforming, helpers.LikedSongs, helpers.TagsEmoji:
				//	user.State = &entities.State{
				//		Name:    helpers.GetSongsFromMongoState,
				//		Context: user.State.Context,
				//	}
				//	return h.Enter(c, user)
			}

			ctx.EffectiveChat.SendAction(bot, "upload_document")

			driveFiles := user.State.Context.DriveFiles
			var foundDriveFile *drive.File
			for _, driveFile := range driveFiles {
				if driveFile.Name == strings.ReplaceAll(ctx.EffectiveMessage.Text, " "+helpers.Like, "") {
					foundDriveFile = driveFile
					break
				}
			}

			if foundDriveFile != nil {
				return c.Song(bot, ctx, foundDriveFile.Id)
			} else {
				user.State.Index = 0
				return c.Search(bot, ctx)
			}
		}
	}
	return nil
}

func (c *BotController) Menu(b *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entities.User)

	replyMarkup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard:       keyboard.Menu,
		ResizeKeyboard: true,
	}

	_, err := ctx.EffectiveChat.SendMessage(b, "Меню:", &gotgbot.SendMessageOpts{
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return err
	}

	user.State = &entities.State{}

	return nil
}
