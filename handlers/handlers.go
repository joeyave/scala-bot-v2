package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/txt"
	"github.com/klauspost/lctime"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/drive/v3"
	"gopkg.in/telebot.v3"
	"html"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func mainMenuHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:              helpers.MainMenuKeyboard,
			ResizeKeyboard:        true,
			InputFieldPlaceholder: helpers.Placeholder,
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Основное меню:", &gotgbot.SendMessageOpts{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			ReplyMarkup:           markup,
		})
		if err != nil {
			return err
		}
		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		switch c.EffectiveMessage.Text {

		case helpers.Schedule:
			user.State = entity.State{
				Name: helpers.GetEventsState,
			}

		case helpers.Songs:
			user.State = entity.State{
				Name: helpers.SearchSongState,
			}

		case helpers.Stats:
			users, err := h.userService.FindManyExtraByBandID(user.BandID)
			if err != nil {
				return err
			}

			usersStr := ""
			event, err := h.eventService.FindOneOldestByBandID(user.BandID)
			if err == nil {
				usersStr = fmt.Sprintf("Статистика ведется с %s", lctime.Strftime("%d %B, %Y", event.Time))
			}

			for _, user := range users {
				if user.User == nil || user.User.Name == "" || len(user.Events) == 0 {
					continue
				}

				usersStr = fmt.Sprintf("%s\n\n%v", usersStr, user.String())
			}

			_, err = c.EffectiveChat.SendMessage(h.bot, usersStr, &gotgbot.SendMessageOpts{
				ParseMode: "HTML",
			})
			return err

		case helpers.Settings:
			user.State = entity.State{
				Name: helpers.SettingsState,
			}

		default:
			user.State = entity.State{
				Name: helpers.SearchSongState,
			}
		}

		return h.Enter(c, user)
	})

	return helpers.MainMenuState, handlerFuncs
}

func settingsHandler() (int, []HandlerFunc) {

	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {
		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       helpers.SettingsKeyboard,
			ResizeKeyboard: true,
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, helpers.Settings+":", &gotgbot.SendMessageOpts{
			ReplyMarkup: markup,
		})
		if err != nil {
			return err
		}
		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Prev = &entity.State{
			Index: 0,
			Name:  helpers.SettingsState,
		}

		switch c.EffectiveMessage.Text {
		case helpers.BandSettings:
			markup := &gotgbot.ReplyKeyboardMarkup{
				ResizeKeyboard: true,
				Keyboard:       helpers.BandSettingsKeyboard,
			}

			_, err := c.EffectiveChat.SendMessage(h.bot, helpers.BandSettings+":", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			return err

		case helpers.ProfileSettings:

			markup := &gotgbot.ReplyKeyboardMarkup{
				ResizeKeyboard: true,
				Keyboard:       helpers.ProfileSettingsKeyboard,
			}

			_, err := c.EffectiveChat.SendMessage(h.bot, helpers.ProfileSettings+":", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			return err

		case helpers.ChangeBand:
			user.State = entity.State{
				Name: helpers.ChooseBandState,
			}

		case helpers.CreateRole:
			user.State = entity.State{
				Name: helpers.CreateRoleState,
			}

		case helpers.AddAdmin:
			user.State = entity.State{
				Name: helpers.AddBandAdminState,
			}
		}

		return h.Enter(c, user)
	})

	return helpers.SettingsState, handlerFuncs
}

func createRoleHandler() (int, []HandlerFunc) {

	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь название новой роли. Например, лид-вокал, проповедник и т. д.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Context.Role = &entity.Role{
			Name: c.EffectiveMessage.Text,
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		if len(user.Band.Roles) == 0 {
			user.State.Context.Role.Priority = 1
			user.State.Index++
			return h.Enter(c, user)
		}

		for _, role := range user.Band.Roles {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: role.Name}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Cancel}})

		_, err := c.EffectiveChat.SendMessage(h.bot, "После какой роли должна быть эта роль?", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		if user.State.Context.Role.Priority == 0 {

			var foundRole *entity.Role
			for _, role := range user.Band.Roles {
				if c.EffectiveMessage.Text == role.Name {
					foundRole = role
					break
				}
			}

			if foundRole == nil {
				user.State.Index--
				return h.Enter(c, user)
			}

			user.State.Context.Role.Priority = foundRole.Priority + 1

			for _, role := range user.Band.Roles {
				if role.Priority > foundRole.Priority {
					role.Priority++
					h.roleService.UpdateOne(*role)
				}
			}
		}

		role, err := h.roleService.UpdateOne(
			entity.Role{
				Name:     user.State.Context.Role.Name,
				BandID:   user.BandID,
				Priority: user.State.Context.Role.Priority,
			})
		if err != nil {
			return err
		}

		_, err = c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Добавлена новая роль: %s.", role.Name), nil)
		if err != nil {
			return err
		}

		user.State = entity.State{Name: helpers.MainMenuState}
		return h.Enter(c, user)
	})

	return helpers.CreateRoleState, handlerFuncs
}

func getEventsHandler() (int, []HandlerFunc) {

	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		events, err := h.eventService.FindManyFromTodayByBandID(user.BandID)

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard:        true,
			InputFieldPlaceholder: helpers.Placeholder,
		}

		user.State.Context.WeekdayButtons = helpers.GetWeekdayButtons(events)
		markup.Keyboard = append(markup.Keyboard, user.State.Context.WeekdayButtons)
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: "➕ Добавить собрание", WebApp: &gotgbot.WebAppInfo{Url: os.Getenv("HOST") + "/web-app/create-event"}}})

		for _, event := range events {
			buttonText := helpers.EventButton(event, user, false)
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
		}

		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}})

		msg, err := c.EffectiveChat.SendMessage(h.bot, txt.Get("text.chooseEvent", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		for _, messageID := range user.State.Context.MessagesToDelete {
			h.bot.DeleteMessage(c.EffectiveChat.Id, messageID)
		}

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)

		user.State.Context.QueryType = "-"
		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		text := c.EffectiveMessage.Text

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

		if strings.Contains(text, "〔") && strings.Contains(text, "〕") {
			if helpers.IsWeekdayString(strings.ReplaceAll(strings.ReplaceAll(text, "〔", ""), "〕", "")) && user.State.Context.QueryType == helpers.Archive {
				text = helpers.Archive
			} else {
				user.State.Index--
				return h.Enter(c, user)
			}
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard:        true,
			InputFieldPlaceholder: helpers.Placeholder,
		}

		if text == helpers.CreateEvent {
			user.State = entity.State{
				Name: helpers.CreateEventState,
				Prev: &user.State,
			}
			user.State.Prev.Index = 0
			return h.Enter(c, user)
		} else if text == helpers.GetEventsWithMe || text == helpers.Archive || text == helpers.PrevPage || text == helpers.NextPage || helpers.IsWeekdayString(text) {

			c.EffectiveChat.SendAction(h.bot, "typing")

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
					(markup.Keyboard[0][i].Text == user.State.Context.PrevText && user.State.Context.QueryType == helpers.Archive && (c.EffectiveMessage.Text == helpers.NextPage || c.EffectiveMessage.Text == helpers.PrevPage)) {
					markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
				}
			}

			var events []*entity.Event
			var err error
			switch user.State.Context.QueryType {
			case helpers.Archive:
				if helpers.IsWeekdayString(text) {
					events, err = h.eventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, helpers.GetWeekdayFromString(text), user.State.Context.PageIndex)
					user.State.Context.PrevText = text
				} else if helpers.IsWeekdayString(user.State.Context.PrevText) && (c.EffectiveMessage.Text == helpers.NextPage || c.EffectiveMessage.Text == helpers.PrevPage) {
					events, err = h.eventService.FindManyUntilTodayByBandIDAndWeekdayAndPageNumber(user.BandID, helpers.GetWeekdayFromString(user.State.Context.PrevText), user.State.Context.PageIndex)
				} else {
					events, err = h.eventService.FindManyUntilTodayByBandIDAndPageNumber(user.BandID, user.State.Context.PageIndex)
				}
			case helpers.GetEventsWithMe:
				events, err = h.eventService.FindManyFromTodayByBandIDAndUserID(user.BandID, user.ID, user.State.Context.PageIndex)
			default:
				if helpers.IsWeekdayString(user.State.Context.QueryType) {
					events, err = h.eventService.FindManyFromTodayByBandIDAndWeekday(user.BandID, helpers.GetWeekdayFromString(user.State.Context.QueryType))
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

			msg, err := c.EffectiveChat.SendMessage(h.bot, txt.Get("text.chooseEvent", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			if err != nil {
				return err
			}

			for _, messageID := range user.State.Context.MessagesToDelete {
				h.bot.DeleteMessage(c.EffectiveChat.Id, messageID)
			}

			user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)

			return nil
		} else {

			c.EffectiveChat.SendAction(h.bot, "typing")

			eventName, eventTime, err := helpers.ParseEventButton(text)
			if err != nil {
				user.State = entity.State{
					Name: helpers.SearchSongState,
				}
				return h.Enter(c, user)
			}

			foundEvent, err := h.eventService.FindOneByNameAndTimeAndBandID(eventName, eventTime, user.BandID)
			if err != nil {
				user.State.Index--
				return h.Enter(c, user)
			}

			user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

			user.State = entity.State{
				Name: helpers.EventActionsState,
				Context: entity.Context{
					EventID: foundEvent.ID,
				},
				Prev: &user.State,
			}
			user.State.Prev.Index = 1
			return h.Enter(c, user)
		}
	})

	return helpers.GetEventsState, handlerFuncs
}

func createEventHandler() (int, []HandlerFunc) {

	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		names, err := h.eventService.GetMostFrequentEventNames()
		if err != nil {
			return err
		}

		for i := range names {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: names[i].Name}})
			if i == 4 {
				break
			}
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Cancel}})

		_, err = c.EffectiveChat.SendMessage(h.bot, "Введи название этого собрания или выбери:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := gotgbot.InlineKeyboardMarkup{}

		now := time.Now()
		var monthFirstDayDate time.Time
		var monthLastDayDate time.Time

		if c.CallbackQuery != nil {
			_, _, monthFirstDateStr := helpers.ParseCallbackData(c.CallbackQuery.Data)

			monthFirstDayDate, _ = time.Parse(time.RFC3339, monthFirstDateStr)
			monthLastDayDate = monthFirstDayDate.AddDate(0, 1, -1)
		} else {
			user.State.Context.Map = map[string]string{"eventName": c.EffectiveMessage.Text}

			monthFirstDayDate = time.Now().AddDate(0, 0, -now.Day()+1)
			monthLastDayDate = time.Now().AddDate(0, 1, -now.Day())
		}

		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{})
		for d := time.Date(2000, 1, 3, 0, 0, 0, 0, time.Local); d != time.Date(2000, 1, 10, 0, 0, 0, 0, time.Local); d = d.AddDate(0, 0, 1) {
			markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
				append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{
					Text: lctime.Strftime("%a", d), CallbackData: "-",
				})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{})

		for d := monthFirstDayDate; d.After(monthLastDayDate) == false; d = d.AddDate(0, 0, 1) {
			timeStr := lctime.Strftime("%d", d)

			if now.Day() == d.Day() && now.Month() == d.Month() && now.Year() == d.Year() {
				timeStr = helpers.Today
			}

			if d.Weekday() == time.Monday {
				markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{})
			}

			wd := int(d.Weekday())
			if wd == 0 {
				wd = 7
			}
			wd = wd - len(markup.InlineKeyboard[len(markup.InlineKeyboard)-1])
			for k := 1; k < wd; k++ {
				markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
					append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{Text: " ", CallbackData: "-"})
			}

			markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
				append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{
					Text:         timeStr,
					CallbackData: helpers.AggregateCallbackData(helpers.CreateEventState, 2, d.Format(time.RFC3339)),
				})
		}

		for len(markup.InlineKeyboard[len(markup.InlineKeyboard)-1]) != 7 {
			markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
				append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{Text: " ", CallbackData: "-"})
		}

		prevMonthLastDate := monthFirstDayDate.AddDate(0, 0, -1)
		prevMonthFirstDateStr := prevMonthLastDate.AddDate(0, 0, -prevMonthLastDate.Day()+1).Format(time.RFC3339)
		nextMonthFirstDate := monthLastDayDate.AddDate(0, 0, 1)
		nextMonthFirstDateStr := monthLastDayDate.AddDate(0, 0, 1).Format(time.RFC3339)
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
			{
				Text:         lctime.Strftime("◀️ %B", prevMonthLastDate),
				CallbackData: helpers.AggregateCallbackData(helpers.CreateEventState, 1, prevMonthFirstDateStr),
			},
			{
				Text:         lctime.Strftime("%B ▶️", nextMonthFirstDate),
				CallbackData: helpers.AggregateCallbackData(helpers.CreateEventState, 1, nextMonthFirstDateStr),
			},
		})

		msg := fmt.Sprintf("Выбери дату:\n\n<b>%s</b>", lctime.Strftime("%B %Y", monthFirstDayDate))
		if c.CallbackQuery != nil {

			c.EffectiveMessage.EditText(h.bot, msg, &gotgbot.EditMessageTextOpts{
				ParseMode:   "HTML",
				ReplyMarkup: markup,
			})

			c.CallbackQuery.Answer(h.bot, nil)
		} else {
			c.EffectiveChat.SendMessage(h.bot, msg, &gotgbot.SendMessageOpts{
				ReplyMarkup: markup,
				ParseMode:   "HTML",
			})
		}

		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, _, eventTime := helpers.ParseCallbackData(c.CallbackQuery.Data)

		parsedTime, err := time.Parse(time.RFC3339, eventTime)
		if err != nil {
			user.State = entity.State{Name: helpers.CreateEventState}
			return h.Enter(c, user)
		}

		event, err := h.eventService.FindOneByNameAndTimeAndBandID(user.State.Context.Map["eventName"], parsedTime, user.BandID)
		if err != nil || event == nil {
			err = nil
			event, err = h.eventService.UpdateOne(entity.Event{
				Name:   user.State.Context.Map["eventName"],
				Time:   parsedTime,
				BandID: user.BandID,
			})
			if err != nil {
				return err
			}
		}

		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.EventActionsState, 0, "")
		q := user.State.CallbackData.Query()
		q.Set("eventId", event.ID.Hex())
		user.State.CallbackData.RawQuery = q.Encode()

		h.enterInlineHandler(c, user)

		user.State = entity.State{
			Name: helpers.GetEventsState,
		}
		return h.enterReplyHandler(c, user)
	})

	return helpers.CreateEventState, handlerFuncs
}

func eventActionsHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		var eventID primitive.ObjectID
		var keyboard string
		if c.CallbackQuery != nil {
			eventIDFromCallback, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
			if err != nil {
				return err
			}
			eventID = eventIDFromCallback

			_, _, keyboard = helpers.ParseCallbackData(c.CallbackQuery.Data)
		} else {
			eventID = user.State.Context.EventID
			keyboard = user.State.Context.Map["keyboard"]
		}

		eventString, event, err := h.eventService.ToHtmlStringByID(eventID)
		if err != nil {
			return err
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		switch keyboard {
		case "EditEventKeyboard":
			markup.InlineKeyboard = helpers.GetEditEventKeyboard(*user)
		default:
			markup.InlineKeyboard = helpers.GetEventActionsKeyboard(*user, *event)
		}

		q := user.State.CallbackData.Query()
		q.Set("eventId", eventID.Hex())
		q.Set("eventAlias", event.Alias())
		q.Del("index")
		q.Del("driveFileIds")
		user.State.CallbackData.RawQuery = q.Encode()

		if c.CallbackQuery != nil {
			c.EffectiveMessage.EditText(h.bot, helpers.AddCallbackData(eventString, user.State.CallbackData.String()), &gotgbot.EditMessageTextOpts{
				ReplyMarkup:           markup,
				DisableWebPagePreview: true,
				ParseMode:             telebot.ModeHTML,
			})
			c.CallbackQuery.Answer(h.bot, nil)
			return nil
		} else {
			_, err := c.EffectiveChat.SendMessage(h.bot, helpers.AddCallbackData(eventString, user.State.CallbackData.String()), &gotgbot.SendMessageOpts{
				ReplyMarkup:           markup,
				DisableWebPagePreview: true,
				ParseMode:             telebot.ModeHTML,
			})
			if err != nil {
				return err
			}

			if user.State.Next != nil {
				user.State = *user.State.Next
				return h.Enter(c, user)
			} else {
				user.State = *user.State.Prev
				return nil
			}
		}
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "upload_document")

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		event, err := h.eventService.FindOneByID(eventID)
		if err != nil {
			return err
		}

		var driveFileIDs []string
		for _, song := range event.Songs {
			driveFileIDs = append(driveFileIDs, song.DriveFileID)
		}

		err = sendDriveFilesAlbum(h, c, user, driveFileIDs)
		if err != nil {
			return err
		}

		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {
		c.EffectiveChat.SendAction(h.bot, "upload_audio")

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		event, err := h.eventService.FindOneByID(eventID)
		if err != nil {
			return err
		}

		var bigAlbum []gotgbot.InputMedia

		for _, song := range event.Songs {

			audio := &gotgbot.InputMediaAudio{
				Media:   helpers.GetMetronomeTrackFileID(song.PDF.BPM, song.PDF.Time),
				Caption: "↑ " + song.PDF.Name,
			}

			bigAlbum = append(bigAlbum, audio)
		}

		const chunkSize = 10
		chunks := chunkAlbumBy(bigAlbum, chunkSize)

		for _, album := range chunks {
			_, err := h.bot.SendMediaGroup(c.EffectiveChat.Id, album, nil)
			if err != nil {
				return err
			}
			if err != nil {
				return err
			}
		}

		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	return helpers.EventActionsState, handlerFuncs
}

func changeSongOrderHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, chosenDriveFileID := helpers.ParseCallbackData(c.CallbackQuery.Data)

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		if user.State.CallbackData.Query().Get("index") == "" {

			event, err := h.eventService.GetEventWithSongs(eventID)
			if err != nil {
				return err
			}

			q := user.State.CallbackData.Query()
			for _, song := range event.Songs {
				q.Add("driveFileIds", song.DriveFileID)
			}

			q.Set("eventAlias", event.Alias())
			q.Set("index", "-1")
			user.State.CallbackData.RawQuery = q.Encode()
		}

		songIndex, err := strconv.Atoi(user.State.CallbackData.Query().Get("index"))
		if err != nil {
			return err
		}

		if chosenDriveFileID != "" {
			q := user.State.CallbackData.Query()
			for i, driveFileID := range user.State.CallbackData.Query()["driveFileIds"] {
				if driveFileID == chosenDriveFileID {
					q["driveFileIds"] = append(q["driveFileIds"][:i], q["driveFileIds"][i+1:]...)
					user.State.CallbackData.RawQuery = q.Encode()
					break
				}
			}

			song, _, err := h.songService.FindOrCreateOneByDriveFileID(chosenDriveFileID)
			if err != nil {
				return err
			}

			err = h.eventService.ChangeSongIDPosition(eventID, song.ID, songIndex)
			if err != nil {
				return err
			}
		}

		if len(user.State.CallbackData.Query()["driveFileIds"]) == 0 {
			c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.DeleteEventSongState, 0, "")
			return h.Enter(c, user)
		}

		event, err := h.eventService.GetEventWithSongs(eventID)
		if err != nil {
			return err
		}

		songsStr := fmt.Sprintf("<b>%s:</b>\n", helpers.Setlist)
		for i, song := range event.Songs {
			if i == songIndex+1 {
				break
			}

			songName := fmt.Sprintf("%d. <a href=\"%s\">%s</a>  (%s)",
				i+1, song.PDF.WebViewLink, song.PDF.Name, song.Caption())
			songsStr += songName + "\n"
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		songs, err := h.songService.FindManyByDriveFileIDs(user.State.CallbackData.Query()["driveFileIds"])
		if err != nil {
			return err
		}

		for _, song := range songs {
			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: fmt.Sprintf("%s (%s)", song.PDF.Name, song.Caption()), CallbackData: helpers.AggregateCallbackData(state, index, song.DriveFileID)}})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.End, CallbackData: helpers.AggregateCallbackData(helpers.DeleteEventSongState, 0, "")}})

		q := user.State.CallbackData.Query()
		q.Set("index", strconv.Itoa(songIndex+1))
		user.State.CallbackData.RawQuery = q.Encode()

		c.EffectiveMessage.EditText(h.bot, helpers.AddCallbackData(fmt.Sprintf("<b>%s</b>\n\n%s\nВыбери %d песню:", user.State.CallbackData.Query().Get("eventAlias"), songsStr, songIndex+2),
			user.State.CallbackData.String()), &gotgbot.EditMessageTextOpts{
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			ReplyMarkup:           markup,
		})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	return helpers.ChangeSongOrderState, handlerFuncs
}

func changeEventDateHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {
		markup := gotgbot.InlineKeyboardMarkup{}

		now := time.Now()
		var monthFirstDayDate time.Time
		var monthLastDayDate time.Time

		_, _, monthFirstDateStr := helpers.ParseCallbackData(c.CallbackQuery.Data)
		if monthFirstDateStr != "" {
			monthFirstDayDate, _ = time.Parse(time.RFC3339, monthFirstDateStr)
			monthLastDayDate = monthFirstDayDate.AddDate(0, 1, -1)
		} else {
			monthFirstDayDate = time.Now().AddDate(0, 0, -now.Day()+1)
			monthLastDayDate = time.Now().AddDate(0, 1, -now.Day())
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{})
		for d := time.Date(2000, 1, 3, 0, 0, 0, 0, time.Local); d != time.Date(2000, 1, 10, 0, 0, 0, 0, time.Local); d = d.AddDate(0, 0, 1) {
			markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
				append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{
					Text: lctime.Strftime("%a", d), CallbackData: "-",
				})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{})

		for d := monthFirstDayDate; d.After(monthLastDayDate) == false; d = d.AddDate(0, 0, 1) {
			timeStr := lctime.Strftime("%d", d)

			if now.Day() == d.Day() && now.Month() == d.Month() && now.Year() == d.Year() {
				timeStr = helpers.Today
			}

			if d.Weekday() == time.Monday {
				markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{})
			}

			wd := int(d.Weekday())
			if wd == 0 {
				wd = 7
			}
			wd = wd - len(markup.InlineKeyboard[len(markup.InlineKeyboard)-1])
			for k := 1; k < wd; k++ {
				markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
					append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{Text: " ", CallbackData: "-"})
			}

			markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
				append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{
					Text:         timeStr,
					CallbackData: helpers.AggregateCallbackData(helpers.ChangeEventDateState, 1, d.Format(time.RFC3339)),
				})
		}

		for len(markup.InlineKeyboard[len(markup.InlineKeyboard)-1]) != 7 {
			markup.InlineKeyboard[len(markup.InlineKeyboard)-1] =
				append(markup.InlineKeyboard[len(markup.InlineKeyboard)-1], gotgbot.InlineKeyboardButton{Text: " ", CallbackData: "-"})
		}

		prevMonthLastDate := monthFirstDayDate.AddDate(0, 0, -1)
		prevMonthFirstDateStr := prevMonthLastDate.AddDate(0, 0, -prevMonthLastDate.Day()+1).Format(time.RFC3339)
		nextMonthFirstDate := monthLastDayDate.AddDate(0, 0, 1)
		nextMonthFirstDateStr := monthLastDayDate.AddDate(0, 0, 1).Format(time.RFC3339)
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
			{
				Text:         lctime.Strftime("◀️ %B", prevMonthLastDate),
				CallbackData: helpers.AggregateCallbackData(helpers.ChangeEventDateState, 0, prevMonthFirstDateStr),
			},
			{
				Text:         lctime.Strftime("%B ▶️", nextMonthFirstDate),
				CallbackData: helpers.AggregateCallbackData(helpers.ChangeEventDateState, 0, nextMonthFirstDateStr),
			},
		})

		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.EventActionsState, 0, "EditEventKeyboard")}})

		msg := fmt.Sprintf("<b>%s</b>\n\nВыбери новую дату:\n\n<b>%s</b>", user.State.CallbackData.Query().Get("eventAlias"), lctime.Strftime("%B %Y", monthFirstDayDate))
		c.EffectiveMessage.EditText(h.bot, helpers.AddCallbackData(msg, user.State.CallbackData.String()), &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   "HTML",
		})
		c.CallbackQuery.Answer(h.bot, nil)

		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {
		_, _, eventTime := helpers.ParseCallbackData(c.CallbackQuery.Data)

		parsedTime, err := time.Parse(time.RFC3339, eventTime)
		if err != nil {
			user.State = entity.State{Name: helpers.CreateEventState}
			return h.Enter(c, user)
		}

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		event, err := h.eventService.FindOneByID(eventID)
		if err != nil {
			return err
		}
		event.Time = parsedTime

		event, err = h.eventService.UpdateOne(*event)
		if err != nil {
			return err
		}

		eventString := h.eventService.ToHtmlStringByEvent(*event)

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: helpers.GetEventActionsKeyboard(*user, *event),
		}

		c.EffectiveMessage.EditText(h.bot, helpers.AddCallbackData(eventString, user.State.CallbackData.String()), &gotgbot.EditMessageTextOpts{
			ReplyMarkup:           markup,
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		})
		c.CallbackQuery.Answer(h.bot, nil)

		return nil
	})

	return helpers.ChangeEventDateState, handlerFuncs
}

func addEventMemberHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, _ := helpers.ParseCallbackData(c.CallbackQuery.Data)

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		event, err := h.eventService.FindOneByID(eventID)
		if err != nil {
			return err
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		for _, role := range event.Band.Roles {
			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: role.Name, CallbackData: helpers.AggregateCallbackData(state, index+1, fmt.Sprintf("%s", role.ID.Hex()))}})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.Back, CallbackData: helpers.AggregateCallbackData(helpers.DeleteEventMemberState, 0, "")}})

		text := fmt.Sprintf("<b>%s</b>\n\n", event.Alias())
		if event.Roles() != "" {
			text += fmt.Sprintf("%s\n\n", event.Roles())
		}
		text += "Выбери роль для нового участника:"

		c.EffectiveMessage.EditText(h.bot, helpers.AddCallbackData(text, user.State.CallbackData.String()), &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   "HTML",
		})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, payload := helpers.ParseCallbackData(c.CallbackQuery.Data)

		parsedPayload := strings.Split(payload, ":")
		roleIDHex := parsedPayload[0]
		loadMore := false
		if len(parsedPayload) > 1 && parsedPayload[1] == "LoadMore" {
			loadMore = true
		}

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		event, err := h.eventService.FindOneByID(eventID)
		if err != nil {
			return err
		}

		roleID, err := primitive.ObjectIDFromHex(roleIDHex)
		if err != nil {
			return err
		}

		usersExtra, err := h.userService.FindManyByBandIDAndRoleID(event.BandID, roleID)
		if err != nil {
			return err
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		if loadMore == false {
			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
				{Text: helpers.LoadMore, CallbackData: helpers.AggregateCallbackData(state, index, fmt.Sprintf("%s:%s", roleIDHex, "LoadMore"))},
			})
		}

		for _, userExtra := range usersExtra {
			var buttonText string
			if len(userExtra.Events) == 0 {
				buttonText = userExtra.User.Name
			} else {
				buttonText = fmt.Sprintf("%s (%v, %d)", userExtra.User.Name, lctime.Strftime("%d %b", userExtra.Events[0].Time), len(userExtra.Events))
			}

			for _, eventMembership := range event.Memberships {
				if eventMembership.RoleID == roleID && eventMembership.UserID == userExtra.User.ID {
					buttonText = helpers.AppendTickSymbol(buttonText)
					break
				}
			}

			if (len(userExtra.Events) > 0 && time.Now().Sub(userExtra.Events[0].Time) < 24*364/3*time.Hour) || loadMore == true {
				markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
					{Text: buttonText, CallbackData: helpers.AggregateCallbackData(state, index+1, fmt.Sprintf("%s:%d", roleIDHex, userExtra.User.ID))},
				})
			}
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.Back, CallbackData: helpers.AggregateCallbackData(helpers.AddEventMemberState, 0, "")}})

		c.EffectiveMessage.EditReplyMarkup(h.bot, &gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup: markup})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, _, payload := helpers.ParseCallbackData(c.CallbackQuery.Data)

		parsedPayload := strings.Split(payload, ":")

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		eventMemberships, err := h.membershipService.FindMultipleByEventID(eventID)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}

		userID, err := strconv.ParseInt(parsedPayload[1], 10, 0)
		if err != nil {
			return err
		}

		user2, err := h.userService.FindOneByID(userID)
		if err != nil {
			return err
		}

		roleID, err := primitive.ObjectIDFromHex(parsedPayload[0])
		if err != nil {
			return err
		}

		var foundEventMembership *entity.Membership
		for _, eventMembership := range eventMemberships {
			if eventMembership.RoleID == roleID && eventMembership.UserID == user2.ID {
				foundEventMembership = eventMembership
				break
			}
		}

		if foundEventMembership == nil {
			_, err = h.membershipService.UpdateOne(entity.Membership{
				EventID: eventID,
				UserID:  userID,
				RoleID:  roleID,
			})
			if err != nil {
				return err
			}
			go func() {
				role, err := h.roleService.FindOneByID(roleID)
				if err != nil {
					return
				}

				event, err := h.eventService.FindOneByID(eventID)
				if err != nil {
					return
				}

				now := time.Now().Local()
				if event.Time.After(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
					h.bot.SendMessage(userID, fmt.Sprintf("Привет. %s только что добавил тебя как %s в собрание %s!\n\nБолее подробную информацию можешь посмотреть в меню Расписание 📆.",
						user.Name, role.Name, event.Alias()), &gotgbot.SendMessageOpts{
						ParseMode:             "HTML",
						DisableWebPagePreview: true,
					})
				}
			}()
		} else {
			err := h.membershipService.DeleteOneByID(foundEventMembership.ID)
			if err != nil {
				return err
			}
		}

		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.AddEventMemberState, 1, payload)
		return h.Enter(c, user)
	})

	return helpers.AddEventMemberState, handlerFuncs
}

func deleteEventMemberHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, payload := helpers.ParseCallbackData(c.CallbackQuery.Data)

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		eventMemberships, err := h.membershipService.FindMultipleByEventID(eventID)
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return err
		}

		eventAlias, err := h.eventService.GetAlias(context.Background(), eventID)
		if err != nil {
			return err
		}

		var memberships []*entity.Membership
		if payload != "deleted" {

			membershipsJson, err := json.Marshal(eventMemberships)
			if err != nil {
				return err
			}

			q := user.State.CallbackData.Query()
			q.Set("eventId", eventID.Hex())
			q.Set("eventAlias", eventAlias)
			q.Set("memberships", string(membershipsJson))
			q.Del("index")
			q.Del("driveFileIds")
			user.State.CallbackData.RawQuery = q.Encode()

			memberships = eventMemberships
		} else {
			membershipsJson := user.State.CallbackData.Query().Get("memberships")

			err := json.Unmarshal([]byte(membershipsJson), &memberships)
			if err != nil {
				return err
			}
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		for _, membership := range memberships {
			user, err := h.userService.FindOneByID(membership.UserID)
			if err != nil {
				continue
			}

			text := fmt.Sprintf("%s (%s)", user.Name, membership.Role.Name)

			for _, m := range eventMemberships {
				if m.ID == membership.ID {
					text += " ✅"
					break
				}
			}

			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: text, CallbackData: helpers.AggregateCallbackData(state, index+1, fmt.Sprintf("%s", membership.ID.Hex()))}})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.AddMember, CallbackData: helpers.AggregateCallbackData(helpers.AddEventMemberState, 0, "")}})
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.Back, CallbackData: helpers.AggregateCallbackData(helpers.EventActionsState, 0, "EditEventKeyboard")}})

		c.EffectiveMessage.EditText(h.bot, helpers.AddCallbackData(fmt.Sprintf("<b>%s</b>\n\n%s:", eventAlias, helpers.Members), user.State.CallbackData.String()),
			&gotgbot.EditMessageTextOpts{
				ReplyMarkup: markup,
				ParseMode:   "HTML",
			})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, _, membershipHex := helpers.ParseCallbackData(c.CallbackQuery.Data)

		membershipID, err := primitive.ObjectIDFromHex(membershipHex)
		if err != nil {
			return err
		}

		membership, err := h.membershipService.FindOneByID(membershipID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = nil
				membershipsJson := user.State.CallbackData.Query().Get("memberships")

				var memberships []*entity.Membership
				err := json.Unmarshal([]byte(membershipsJson), &memberships)
				if err != nil {
					return err
				}

				var foundMembership *entity.Membership
				for _, m := range memberships {
					if m.ID == membershipID {
						foundMembership = m
						break
					}
				}

				_, err = h.membershipService.UpdateOne(*foundMembership)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			err = h.membershipService.DeleteOneByID(membershipID)
			if err != nil {
				return err
			}

			go func() {

				role, err := h.roleService.FindOneByID(membership.RoleID)
				if err != nil {
					return
				}

				event, err := h.eventService.FindOneByID(membership.EventID)
				if err != nil {
					return
				}

				now := time.Now().Local()
				if event.Time.After(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
					h.bot.SendMessage(membership.UserID, fmt.Sprintf("Привет. %s только что удалил тебя как %s из собрания %s ☹️",
						user.Name, role.Name, event.Alias()), &gotgbot.SendMessageOpts{
						ParseMode:             "HTML",
						DisableWebPagePreview: true,
					})
				}
			}()

		}

		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.DeleteEventMemberState, 0, "deleted")
		return h.Enter(c, user)
	})

	return helpers.DeleteEventMemberState, handlerFuncs
}

func addEventSongHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		var eventID primitive.ObjectID
		if c.CallbackQuery != nil {
			eventIDFromCallback, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
			if err != nil {
				return err
			}
			eventID = eventIDFromCallback
			c.CallbackQuery.Answer(h.bot, nil)
		} else {
			eventID = user.State.Context.EventID
		}

		markup := gotgbot.ReplyKeyboardMarkup{
			Keyboard:              [][]gotgbot.KeyboardButton{{{Text: helpers.End}}},
			ResizeKeyboard:        true,
			InputFieldPlaceholder: "Название песни или список",
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Введи название песни или список (название каждой песни с новой строки):", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State = entity.State{
			Index: 1,
			Name:  helpers.AddEventSongState,
			Context: entity.Context{
				EventID: eventID,
			},
		}
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		if c.EffectiveMessage.Text == helpers.End {
			user.State = entity.State{
				Name:    helpers.DeleteEventSongState,
				Context: user.State.Context,
			}
			user.State.Next = &entity.State{
				Name: helpers.GetEventsState,
			}
			return h.Enter(c, user)
		}

		c.EffectiveChat.SendAction(h.bot, "typing")

		query := helpers.CleanUpQuery(c.EffectiveMessage.Text)
		songNames := helpers.SplitQueryByNewlines(query)

		if len(songNames) > 1 {
			user.State = entity.State{
				Index:   0,
				Name:    helpers.SetlistState,
				Context: user.State.Context,
				Next: &entity.State{
					Name:    helpers.AddEventSongState,
					Index:   3,
					Context: user.State.Context,
				},
			}
			user.State.Context.SongNames = songNames
			return h.Enter(c, user)
		}

		driveFiles, _, err := h.driveFileService.FindSomeByFullTextAndFolderID(query, user.Band.DriveFolderID, "")
		if err != nil {
			return err
		}

		if len(driveFiles) == 0 {

			markup := gotgbot.ReplyKeyboardMarkup{
				Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.End}}},
				ResizeKeyboard: true,
			}

			_, err := c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("По запросу \"%s\" ничего не найдено.", c.EffectiveMessage.Text), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			return err
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		// TODO: some sort of pagination.
		for _, song := range driveFiles {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: song.Name}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.End}})

		_, err = c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Выбери песню по запросу \"%s\" или введи другое название:", c.EffectiveMessage.Text), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		if c.EffectiveMessage.Text == helpers.End {
			user.State = entity.State{
				Name:    helpers.DeleteEventSongState,
				Context: user.State.Context,
			}
			user.State.Next = &entity.State{
				Name: helpers.GetEventsState,
			}
			return h.Enter(c, user)
		}

		c.EffectiveChat.SendAction(h.bot, "typing")

		foundDriveFile, err := h.driveFileService.FindOneByNameAndFolderID(c.EffectiveMessage.Text, user.Band.DriveFolderID)
		if err != nil {
			user.State.Index--
			return h.Enter(c, user)
		}

		song, _, err := h.songService.FindOrCreateOneByDriveFileID(foundDriveFile.Id)
		if err != nil {
			return err
		}

		err = h.eventService.PushSongID(user.State.Context.EventID, song.ID)
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.EffectiveChat.SendMessage(h.bot, "Вероятнее всего, эта песня уже есть в списке.", nil)
		} else if err != nil {
			return err
		}

		user.State.Index = 0
		return h.Enter(c, user)
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		for _, id := range user.State.Context.FoundDriveFileIDs {
			song, _, err := h.songService.FindOrCreateOneByDriveFileID(id)
			if err != nil {
				return err
			}

			err = h.eventService.PushSongID(user.State.Context.EventID, song.ID)
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.EffectiveChat.SendMessage(h.bot, "Вероятнее всего, эта песня уже есть в списке.", nil)
			} else if err != nil {
				return err
			}
		}

		user.State.Index = 0
		return h.Enter(c, user)
	})

	return helpers.AddEventSongState, handlerFuncs
}

func changeEventNotesHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}
		c.CallbackQuery.Answer(h.bot, nil)

		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}

		_, err = c.EffectiveChat.SendMessage(h.bot, "Заметки:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		// user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)

		user.State = entity.State{
			Index: 1,
			Name:  helpers.ChangeEventNotesState,
			Context: entity.Context{
				EventID:          eventID,
				MessagesToDelete: user.State.Context.MessagesToDelete,
			},
			Prev: &user.State,
		}

		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		// user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

		c.EffectiveChat.SendAction(h.bot, "typing")

		event, err := h.eventService.FindOneByID(user.State.Context.EventID)
		if err != nil {
			return err
		}

		text := html.EscapeString(c.EffectiveMessage.Text)

		for _, entity := range c.EffectiveMessage.Entities {
			textWithoutHTML := c.EffectiveMessage.ParseEntity(entity)

			var textWithHTML string
			switch entity.Type {
			case "bold":
				textWithHTML = fmt.Sprintf("<b>%s</b>", textWithoutHTML)
			case "italic":
				textWithHTML = fmt.Sprintf("<i>%s</i>", textWithoutHTML)
			case "underline":
				textWithHTML = fmt.Sprintf("<u>%s</u>", textWithoutHTML)
			case "strikethrough":
				textWithHTML = fmt.Sprintf("<s>%s</s>", textWithoutHTML)
			case "spoiler":
				textWithHTML = fmt.Sprintf("<tg-spoiler>%s</tg-spoiler>", textWithoutHTML)
			case "code":
				textWithHTML = fmt.Sprintf("<code>%s</code>", textWithoutHTML)
			}

			text = strings.ReplaceAll(text, textWithoutHTML.Text, textWithHTML)
		}

		event.Notes = text

		fmt.Println(c.EffectiveMessage.Entities)

		event, err = h.eventService.UpdateOne(*event)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Index: 0,
			Name:  helpers.EventActionsState,
			Context: entity.Context{
				EventID: user.State.Context.EventID,
			},
			Next: &entity.State{
				Name: helpers.GetEventsState,
			},
		}

		return h.Enter(c, user)
	})

	return helpers.ChangeEventNotesState, handlerFuncs
}

func deleteEventSongHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		var eventID primitive.ObjectID
		var payload string
		if c.CallbackQuery != nil {
			eventIDFromCallback, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
			if err != nil {
				return err
			}
			eventID = eventIDFromCallback

			_, _, payload = helpers.ParseCallbackData(c.CallbackQuery.Data)
		} else {
			eventID = user.State.Context.EventID
		}

		event, err := h.eventService.GetEventWithSongs(eventID)
		if err != nil {
			return err
		}

		var songs []*entity.Song
		if payload != "deleted" {

			songsJson, err := json.Marshal(event.Songs)
			if err != nil {
				return err
			}

			q := user.State.CallbackData.Query()
			q.Set("eventId", eventID.Hex())
			q.Set("eventAlias", event.Alias())
			q.Set("songs", string(songsJson))
			q.Del("index")
			q.Del("driveFileIds")
			user.State.CallbackData.RawQuery = q.Encode()

			songs = event.Songs
		} else {
			songsJson := user.State.CallbackData.Query().Get("songs")

			err := json.Unmarshal([]byte(songsJson), &songs)
			if err != nil {
				return err
			}
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.SongsOrder, CallbackData: helpers.AggregateCallbackData(helpers.ChangeSongOrderState, 0, "")}})
		for _, song := range songs {
			// driveFile, err := h.driveFileService.FindOneByID(song.DriveFileID)
			// if err != nil {
			// 	continue
			// }

			text := song.PDF.Name

			for _, eventSong := range event.Songs {
				if eventSong.ID == song.ID {
					text += " ✅"
					break
				}
			}

			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: text, CallbackData: helpers.AggregateCallbackData(helpers.DeleteEventSongState, 1, song.ID.Hex())}})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.AddSong, CallbackData: helpers.AggregateCallbackData(helpers.AddEventSongState, 0, "")}})
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.Back, CallbackData: helpers.AggregateCallbackData(helpers.EventActionsState, 0, "EditEventKeyboard")}})

		str := fmt.Sprintf("<b>%s</b>\n\n%s:", event.Alias(), helpers.Setlist)
		// todo
		c.EffectiveMessage.EditText(h.bot, helpers.AddCallbackData(str, user.State.CallbackData.String()), &gotgbot.EditMessageTextOpts{
			ReplyMarkup:           markup,
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		})
		c.CallbackQuery.Answer(h.bot, nil)

		if c.CallbackQuery == nil && user.State.Next != nil {
			user.State = *user.State.Next
			return h.Enter(c, user)
		} else if c.CallbackQuery == nil {
			user.State = *user.State.Prev
			return nil
		}
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, _, songIDHex := helpers.ParseCallbackData(c.CallbackQuery.Data)

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}

		songID, err := primitive.ObjectIDFromHex(songIDHex)
		if err != nil {
			return err
		}

		event, err := h.eventService.GetEventWithSongs(eventID)
		if err != nil {
			return err
		}

		found := false
		for _, eventSong := range event.Songs {
			if songID == eventSong.ID {
				found = true
				break
			}
		}

		if found {
			err = h.eventService.PullSongID(eventID, songID)
			if err != nil {
				return err
			}
		} else {
			songsJson := user.State.CallbackData.Query().Get("songs")

			var songs []*entity.Song
			err := json.Unmarshal([]byte(songsJson), &songs)
			if err != nil {
				return err
			}

			index := 0
			for _, song := range songs {

				for _, eventSong := range event.Songs {
					if song.ID == eventSong.ID {
						index++
						break
					}
				}

				if song.ID == songID {
					break
				}
			}

			err = h.eventService.PushSongID(eventID, songID)
			if err != nil {
				return err
			}
			err = h.eventService.ChangeSongIDPosition(eventID, songID, index)
			if err != nil {
				return err
			}
		}

		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.DeleteEventSongState, 0, "deleted")
		return h.Enter(c, user)
	})

	return helpers.DeleteEventSongState, handlerFuncs
}

func deleteEventHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := gotgbot.InlineKeyboardMarkup{}
		markup.InlineKeyboard = helpers.ConfirmDeletingEventKeyboard
		msg := helpers.AddCallbackData(fmt.Sprintf("<b>%s</b>\n\nТы уверен, что хочешь удалить это собрание?", user.State.CallbackData.Query().Get("eventAlias")),
			user.State.CallbackData.String())
		_, _, err := c.EffectiveMessage.EditText(h.bot, msg, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: markup,
			ParseMode:   "HTML",
		})
		return err

	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		eventID, err := primitive.ObjectIDFromHex(user.State.CallbackData.Query().Get("eventId"))
		if err != nil {
			return err
		}
		err = h.eventService.DeleteOneByID(eventID)
		if err != nil {
			return err
		}

		_, _, err = c.EffectiveMessage.EditText(h.bot, "Удаление завершено.", nil)
		if err != nil {
			return err
		}
		return err
	})

	return helpers.DeleteEventState, handlerFuncs
}

func chooseBandHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {
		bands, err := h.bandService.FindAll()
		if err != nil {
			return err
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.CreateBand}})
		for _, band := range bands {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: band.Name}})
		}

		_, err = c.EffectiveChat.SendMessage(h.bot, "Выбери свою группу:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Context.Bands = bands
		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {
		switch c.EffectiveMessage.Text {
		case helpers.CreateBand:
			user.State = entity.State{
				Index: 0,
				Name:  helpers.CreateBandState,
			}
			return h.Enter(c, user)

		default:
			bands := user.State.Context.Bands
			var foundBand *entity.Band
			for _, band := range bands {
				if band.Name == c.EffectiveMessage.Text {
					foundBand = band
					break
				}
			}

			if foundBand != nil {
				_, err := c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Ты добавлен в группу %s.", foundBand.Name), nil)
				if err != nil {
					return err
				}

				user.BandID = foundBand.ID
				user.State = entity.State{
					Index: 0,
					Name:  helpers.MainMenuState,
				}
			} else {
				user.State.Index--
			}

			return h.Enter(c, user)
		}
	})

	return helpers.ChooseBandState, handlerFuncs
}

func createBandHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Введи название своей группы:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		user.State.Context.Band = &entity.Band{
			Name: c.EffectiveMessage.Text,
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}
		_, err := c.EffectiveChat.SendMessage(h.bot, "Теперь добавь имейл scala-drive@scala-chords-bot.iam.gserviceaccount.com в папку на Гугл Диске как редактора. После этого отправь мне ссылку на эту папку.",
			&gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		re := regexp.MustCompile(`(/folders/|id=)(.*?)(/|\?|$)`)
		matches := re.FindStringSubmatch(c.EffectiveMessage.Text)
		if matches == nil || len(matches) < 3 {
			user.State.Index--
			return h.Enter(c, user)
		}
		user.State.Context.Band.DriveFolderID = matches[2]
		user.Role = helpers.Admin
		band, err := h.bandService.UpdateOne(*user.State.Context.Band)
		if err != nil {
			return err
		}

		user.BandID = band.ID

		_, err = c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Ты добавлен в группу \"%s\" как администратор.", band.Name), nil)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Name: helpers.MainMenuState,
		}
		return h.Enter(c, user)
	})

	return helpers.CreateBandState, handlerFunc
}

func addBandAdminHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		users, err := h.userService.FindMultipleByBandID(user.BandID)
		if err != nil {
			return err
		}

		for _, user := range users {
			buttonText := user.Name
			if user.Role == helpers.Admin {
				buttonText += " (админ)"
			}
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Cancel}})

		_, err = c.EffectiveChat.SendMessage(h.bot, "Выбери пользователя, которого ты хочешь сделать администратором:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		regex := regexp.MustCompile(` \(админ\)$`)
		query := regex.ReplaceAllString(c.EffectiveMessage.Text, "")

		chosenUser, err := h.userService.FindOneByName(query)
		if err != nil {
			user.State.Index--
			return h.Enter(c, user)
		}

		chosenUser.Role = helpers.Admin
		_, err = h.userService.UpdateOne(*chosenUser)
		if err != nil {
			return err
		}

		_, err = c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Пользователь '%s' повышен до администратора.", chosenUser.Name), nil)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Name: helpers.MainMenuState,
		}

		return h.Enter(c, user)
	})

	return helpers.AddBandAdminState, handlerFunc
}

func getSongsFromMongoHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

		switch c.EffectiveMessage.Text {
		case helpers.SongsByNumberOfPerforming, helpers.SongsByLastDateOfPerforming, helpers.LikedSongs:
			user.State.Context.QueryType = c.EffectiveMessage.Text
		case helpers.TagsEmoji:
			user.State.Context.QueryType = c.EffectiveMessage.Text
			user.State.Index = 2
			return h.Enter(c, user)
		}

		var songs []*entity.SongWithEvents
		var err error
		switch user.State.Context.QueryType {
		case helpers.SongsByLastDateOfPerforming:
			songs, err = h.songService.FindAllExtraByPageNumberSortedByLatestEventDate(user.BandID, user.State.Context.PageIndex)
		case helpers.SongsByNumberOfPerforming:
			songs, err = h.songService.FindAllExtraByPageNumberSortedByEventsNumber(user.BandID, user.State.Context.PageIndex)
		case helpers.LikedSongs:
			songs, err = h.songService.FindManyExtraLiked(user.ID, user.State.Context.PageIndex)
		case helpers.TagsEmoji:
			songs, err = h.songService.FindManyExtraByTag(c.EffectiveMessage.Text, user.BandID, user.State.Context.PageIndex)
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard:        true,
			InputFieldPlaceholder: helpers.Placeholder,
		}
		markup.Keyboard = [][]gotgbot.KeyboardButton{
			{
				{Text: helpers.LikedSongs}, {Text: helpers.SongsByLastDateOfPerforming}, {Text: helpers.SongsByNumberOfPerforming}, {Text: helpers.TagsEmoji},
			},
		}

		for i := range markup.Keyboard[0] {
			if markup.Keyboard[0][i].Text == user.State.Context.QueryType {
				markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
				break
			}
		}

		for _, songExtra := range songs {
			buttonText := songExtra.Song.PDF.Name
			if songExtra.Caption() != "" {
				buttonText += fmt.Sprintf(" (%s)", songExtra.Caption())
			}

			if user.State.Context.QueryType != helpers.LikedSongs {
				for _, userID := range songExtra.Song.Likes {
					if user.ID == userID {
						buttonText += " " + helpers.Like
						break
					}
				}
			}

			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: buttonText}})
		}

		if user.State.Context.PageIndex != 0 {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.PrevPage}, {Text: helpers.Menu}, {Text: helpers.NextPage}})
		} else {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Menu}, {Text: helpers.NextPage}})
		}

		msg, err := c.EffectiveChat.SendMessage(h.bot, txt.Get("text.chooseSong", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		for _, messageID := range user.State.Context.MessagesToDelete {
			h.bot.DeleteMessage(c.EffectiveChat.Id, messageID)
		}

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)

		user.State.Index++
		return nil
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

		switch c.EffectiveMessage.Text {
		case helpers.SongsByLastDateOfPerforming, helpers.SongsByNumberOfPerforming, helpers.LikedSongs, helpers.TagsEmoji:
			user.State = entity.State{
				Name:    helpers.GetSongsFromMongoState,
				Context: user.State.Context,
			}
			return h.Enter(c, user)
		case helpers.NextPage:
			user.State.Context.PageIndex++
			user.State.Index--
			return h.Enter(c, user)
		case helpers.PrevPage:
			user.State.Context.PageIndex--
			user.State.Index--
			return h.Enter(c, user)
		}

		c.EffectiveChat.SendAction(h.bot, "upload_document")

		var songName string
		regex := regexp.MustCompile(`\s*\(.*\)\s*(` + helpers.Like + `)?\s*`)
		songName = regex.ReplaceAllString(c.EffectiveMessage.Text, "")

		song, err := h.songService.FindOneByName(strings.TrimSpace(songName))
		if err != nil {
			user.State = entity.State{
				Name:    helpers.SearchSongState,
				Context: user.State.Context,
			}
			return h.Enter(c, user)
		}

		user.State = entity.State{
			Name: helpers.SongActionsState,
			Context: entity.Context{
				DriveFileID: song.DriveFileID,
			},
			Prev: &user.State,
		}
		return h.Enter(c, user)
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

		tags, err := h.songService.GetTags()
		if err != nil {
			return err
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard:        true,
			InputFieldPlaceholder: helpers.Placeholder,
		}
		markup.Keyboard = [][]gotgbot.KeyboardButton{
			{
				{Text: helpers.LikedSongs}, {Text: helpers.SongsByLastDateOfPerforming}, {Text: helpers.SongsByNumberOfPerforming}, {Text: helpers.TagsEmoji},
			},
		}

		for i := range markup.Keyboard[0] {
			if markup.Keyboard[0][i].Text == user.State.Context.QueryType {
				markup.Keyboard[0][i].Text = fmt.Sprintf("〔%s〕", markup.Keyboard[0][i].Text)
				break
			}
		}

		for _, tag := range tags {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: tag}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Back}})

		msg, err := c.EffectiveChat.SendMessage(h.bot, txt.Get("text.chooseTag", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}
		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)

		user.State.Index = 0
		return nil
	})
	return helpers.GetSongsFromMongoState, handlerFuncs
}

func searchSongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	// Print list of found songs.
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		c.EffectiveChat.SendAction(h.bot, "typing")

		// user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

		var query string
		if c.EffectiveMessage.Text == helpers.CreateDoc {
			user.State = entity.State{
				Name: helpers.CreateSongState,
			}
			return h.Enter(c, user)
		} else if c.EffectiveMessage.Text == helpers.SearchEverywhere || c.EffectiveMessage.Text == helpers.Songs || c.EffectiveMessage.Text == helpers.SongsByLastDateOfPerforming {
			user.State.Context.QueryType = c.EffectiveMessage.Text
			query = user.State.Context.Query
		} else if strings.Contains(c.EffectiveMessage.Text, "〔") && strings.Contains(c.EffectiveMessage.Text, "〕") {
			user.State.Context.QueryType = helpers.Songs
			query = user.State.Context.Query
		} else if c.EffectiveMessage.Text == helpers.PrevPage || c.EffectiveMessage.Text == helpers.NextPage {
			query = user.State.Context.Query
		} else {
			user.State.Context.NextPageToken = nil
			query = c.EffectiveMessage.Text
		}

		query = helpers.CleanUpQuery(query)
		songNames := helpers.SplitQueryByNewlines(query)

		if len(songNames) > 1 {
			user.State = entity.State{
				Index: 0,
				Name:  helpers.SetlistState,
				Next: &entity.State{
					Index: 2,
					Name:  helpers.SearchSongState,
				},
				Context: user.State.Context,
			}
			user.State.Context.SongNames = songNames
			return h.Enter(c, user)

		} else if len(songNames) == 1 {
			query = songNames[0]
			user.State.Context.Query = query
		} else {
			_, err := c.EffectiveChat.SendMessage(h.bot, "Из запроса удаляются все числа, дефисы и скобки вместе с тем, что в них.", nil)
			if err != nil {
				return err
			}

			user.State = entity.State{
				Name: helpers.MainMenuState,
			}

			return h.Enter(c, user)
		}

		var driveFiles []*drive.File
		var nextPageToken string
		var err error

		if c.EffectiveMessage.Text == helpers.PrevPage {
			if user.State.Context.NextPageToken != nil &&
				user.State.Context.NextPageToken.PrevPageToken != nil {
				user.State.Context.NextPageToken = user.State.Context.NextPageToken.PrevPageToken.PrevPageToken
			}
		}

		if user.State.Context.NextPageToken == nil {
			user.State.Context.NextPageToken = &entity.PageToken{}
		}

		filters := true
		if user.State.Context.QueryType == helpers.SearchEverywhere {
			filters = false
			_driveFiles, _nextPageToken, _err := h.driveFileService.FindSomeByFullTextAndFolderID(query, "", user.State.Context.NextPageToken.Token)
			driveFiles = _driveFiles
			nextPageToken = _nextPageToken
			err = _err
		} else if user.State.Context.QueryType == helpers.Songs && user.State.Context.Query == "" {
			_driveFiles, _nextPageToken, _err := h.driveFileService.FindAllByFolderID(user.Band.DriveFolderID, user.State.Context.NextPageToken.Token)
			driveFiles = _driveFiles
			nextPageToken = _nextPageToken
			err = _err
		} else {
			filters = false
			_driveFiles, _nextPageToken, _err := h.driveFileService.FindSomeByFullTextAndFolderID(query, user.Band.DriveFolderID, user.State.Context.NextPageToken.Token)
			driveFiles = _driveFiles
			nextPageToken = _nextPageToken
			err = _err
		}

		if err != nil {
			return err
		}

		user.State.Context.NextPageToken = &entity.PageToken{
			Token:         nextPageToken,
			PrevPageToken: user.State.Context.NextPageToken,
		}

		if len(driveFiles) == 0 {
			markup := &gotgbot.ReplyKeyboardMarkup{
				Keyboard:       helpers.SearchEverywhereKeyboard,
				ResizeKeyboard: true,
			}
			_, err := c.EffectiveChat.SendMessage(h.bot, txt.Get("text.nothingFound", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			return err
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard:        true,
			InputFieldPlaceholder: query,
		}
		if markup.InputFieldPlaceholder == "" {
			markup.InputFieldPlaceholder = helpers.Placeholder
		}

		if filters {
			markup.Keyboard = [][]gotgbot.KeyboardButton{
				{{Text: helpers.LikedSongs}, {Text: helpers.SongsByLastDateOfPerforming}, {Text: helpers.SongsByNumberOfPerforming}, {Text: helpers.TagsEmoji}},
			}
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.CreateDoc}})
		}

		likedSongs, likedSongErr := h.songService.FindManyLiked(user.ID)

		set := make(map[string]*entity.Band)
		for i, driveFile := range driveFiles {

			if user.State.Context.QueryType == helpers.SearchEverywhere {

				for _, parentFolderID := range driveFile.Parents {
					_, exists := set[parentFolderID]
					if !exists {
						band, err := h.bandService.FindOneByDriveFolderID(parentFolderID)
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

		if c.EffectiveMessage.Text != helpers.SearchEverywhere || c.EffectiveMessage.Text != helpers.Songs {
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

		msg, err := c.EffectiveChat.SendMessage(h.bot, txt.Get("text.chooseSong", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		for _, messageID := range user.State.Context.MessagesToDelete {
			h.bot.DeleteMessage(c.EffectiveChat.Id, messageID)
		}

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)

		user.State.Context.DriveFiles = driveFiles
		user.State.Index++
		return nil

	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		switch c.EffectiveMessage.Text {
		case helpers.CreateDoc:
			user.State = entity.State{
				Name: helpers.CreateSongState,
			}
			return h.Enter(c, user)

		case helpers.SearchEverywhere, helpers.NextPage:
			user.State.Index--
			return h.Enter(c, user)

		case helpers.SongsByLastDateOfPerforming, helpers.SongsByNumberOfPerforming, helpers.LikedSongs, helpers.TagsEmoji:
			user.State = entity.State{
				Name:    helpers.GetSongsFromMongoState,
				Context: user.State.Context,
			}
			return h.Enter(c, user)

		default:
			c.EffectiveChat.SendAction(h.bot, "upload_document")

			driveFiles := user.State.Context.DriveFiles
			var foundDriveFile *drive.File
			for _, driveFile := range driveFiles {
				if driveFile.Name == strings.ReplaceAll(c.EffectiveMessage.Text, " "+helpers.Like, "") {
					foundDriveFile = driveFile
					break
				}
			}

			if foundDriveFile != nil {
				user.State = entity.State{
					Name: helpers.SongActionsState,
					Context: entity.Context{
						DriveFileID: foundDriveFile.Id,
					},
					Prev: &user.State,
				}
				return h.Enter(c, user)
			} else {
				user.State.Index--
				return h.Enter(c, user)
			}
		}
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		for _, messageID := range user.State.Context.MessagesToDelete {
			h.bot.DeleteMessage(c.EffectiveChat.Id, messageID)
		}

		err := sendDriveFilesAlbum(h, c, user, user.State.Context.FoundDriveFileIDs)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Name: helpers.MainMenuState,
		}
		return h.Enter(c, user)
	})

	return helpers.SearchSongState, handlerFunc
}

func songActionsHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		var driveFileID string

		if c.CallbackQuery != nil {
			driveFileID = user.State.CallbackData.Query().Get("driveFileId")
		} else {
			c.EffectiveChat.SendAction(h.bot, "upload_document")
			driveFileID = user.State.Context.DriveFileID
		}

		err := SendDriveFileToUser(h, c, user, driveFileID)
		if err != nil {
			return err
		}

		for _, messageID := range user.State.Context.MessagesToDelete {
			h.bot.DeleteMessage(c.EffectiveChat.Id, messageID)
		}

		if c.CallbackQuery != nil {
			c.CallbackQuery.Answer(h.bot, nil)
		} else {
			if user.State.Next != nil {
				user.State = *user.State.Next
				return h.Enter(c, user)
			} else {
				user.State.Prev.Context.MessagesToDelete = user.State.Prev.Context.MessagesToDelete[:0]
				user.State = *user.State.Prev
				return nil
			}
		}
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		song, driveFile, err :=
			h.songService.FindOrCreateOneByDriveFileID(user.State.CallbackData.Query().Get("driveFileId"))
		if err != nil {
			return err
		}

		markup := gotgbot.InlineKeyboardMarkup{}
		markup.InlineKeyboard = helpers.GetSongActionsKeyboard(*user, *song, *driveFile)

		c.EffectiveMessage.EditReplyMarkup(h.bot, &gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup: markup})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		song, _, err :=
			h.songService.FindOrCreateOneByDriveFileID(user.State.CallbackData.Query().Get("driveFileId"))
		if err != nil {
			return err
		}

		_, _, action := helpers.ParseCallbackData(c.CallbackQuery.Data)

		if action == "like" {
			err := h.songService.Like(song.ID, user.ID)
			if err != nil {
				return err
			}

			song.Likes = append(song.Likes, user.ID)

		} else if action == "dislike" {
			err := h.songService.Dislike(song.ID, user.ID)
			if err != nil {
				return err
			}

			song.Likes = song.Likes[:0]
		}

		markup := gotgbot.InlineKeyboardMarkup{}
		markup.InlineKeyboard = helpers.GetSongInitKeyboard(user, song)

		c.EffectiveMessage.EditReplyMarkup(h.bot, &gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup: markup})
		c.CallbackQuery.Answer(h.bot, nil)

		return nil
	})

	return helpers.SongActionsState, handlerFunc
}

func addSongTagHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, _ := helpers.ParseCallbackData(c.CallbackQuery.Data)

		song, _, err :=
			h.songService.FindOrCreateOneByDriveFileID(user.State.CallbackData.Query().Get("driveFileId"))
		if err != nil {
			return err
		}

		tags, err := h.songService.GetTags()
		if err != nil {
			return err
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		for _, tag := range tags {
			text := tag

			for _, songTag := range song.Tags {
				if songTag == tag {
					text += " ✅"
					break
				}
			}

			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: text, CallbackData: helpers.AggregateCallbackData(state, index+2, tag)}})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.CreateTag, CallbackData: helpers.AggregateCallbackData(state, index+1, "")}})
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")}})

		c.EffectiveMessage.EditReplyMarkup(h.bot, &gotgbot.EditMessageReplyMarkupOpts{ReplyMarkup: markup})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		fmt.Println(c.EffectiveMessage.Text)
		state, index, _ := helpers.ParseCallbackData(c.CallbackQuery.Data)

		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
			ResizeKeyboard: true,
		}
		_, err := c.EffectiveChat.SendMessage(h.bot, "Введи название тега:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State = entity.State{
			Index: index + 1,
			Name:  state,
			Context: entity.Context{
				DriveFileID: user.State.CallbackData.Query().Get("driveFileId"),
			},
		}
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.Context.DriveFileID
		tag := c.EffectiveMessage.Text

		if c.CallbackQuery != nil {
			driveFileID = user.State.CallbackData.Query().Get("driveFileId")
			_, _, tag = helpers.ParseCallbackData(c.CallbackQuery.Data)
		}

		song, _, err :=
			h.songService.FindOrCreateOneByDriveFileID(driveFileID)
		if err != nil {
			return err
		}

		song, err = h.songService.TagOrUntag(tag, song.ID)
		if err != nil {
			return err
		}

		tags, err := h.songService.GetTags()
		if err != nil {
			return err
		}

		markup := gotgbot.InlineKeyboardMarkup{}

		for _, tag := range tags {
			text := tag

			for _, songTag := range song.Tags {
				if songTag == tag {
					text += " ✅"
					break
				}
			}

			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: text, CallbackData: helpers.AggregateCallbackData(helpers.AddSongTagState, 2, tag)}})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.CreateTag, CallbackData: helpers.AggregateCallbackData(helpers.AddSongTagState, 1, "")}})
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")}})

		if c.CallbackQuery != nil {
			_, _, err = c.EffectiveMessage.EditCaption(h.bot,
				&gotgbot.EditMessageCaptionOpts{
					Caption:     helpers.AddCallbackData(song.Caption()+"\n"+strings.Join(song.Tags, ", "), user.State.CallbackData.String()),
					ReplyMarkup: markup,
					ParseMode:   "HTML",
				})
			c.CallbackQuery.Answer(h.bot, nil)
		} else {
			user.State = entity.State{
				Name:    helpers.SongActionsState,
				Context: user.State.Context,
				Next: &entity.State{
					Name: helpers.MainMenuState,
				},
			}
			return h.Enter(c, user)
		}
		return nil
	})

	return helpers.AddSongTagState, handlerFunc
}

func transposeSongHandler() (int, []HandlerFunc) {

	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, _ := helpers.ParseCallbackData(c.CallbackQuery.Data)

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{Text: "C (Am)", CallbackData: helpers.AggregateCallbackData(state, index+1, "C")},
					{Text: "C# (A#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "C#")},
					{Text: "Db (Bbm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Db")},
				},
				{
					{Text: "D (Bm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "D")},
					{Text: "D# (Cm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "D#")},
					{Text: "Eb (Cm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Eb")},
				},
				{
					{Text: "E (C#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "E")},
				},
				{
					{Text: "F (Dm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "F")},
					{Text: "F# (D#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "F#")},
					{Text: "Gb (Ebm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Gb")},
				},
				{
					{Text: "G (Em)", CallbackData: helpers.AggregateCallbackData(state, index+1, "G")},
					{Text: "G# (Fm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "G#")},
					{Text: "Ab (Fm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Ab")},
				},
				{
					{Text: "A (F#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "A")},
					{Text: "A# (Gm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "A#")},
					{Text: "Bb (Gm)", CallbackData: helpers.AggregateCallbackData(state, index+1, "Bb")},
				},
				{
					{Text: "B (G#m)", CallbackData: helpers.AggregateCallbackData(state, index+1, "B")},
				},
				{
					{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")},
				},
			},
		}

		_, _, err := c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{
			Caption:     helpers.AddCallbackData("Выбери новую тональность:", user.State.CallbackData.String()),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		if err != nil {
			return err
		}

		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, key := helpers.ParseCallbackData(c.CallbackQuery.Data)

		q := user.State.CallbackData.Query()
		q.Set("key", key)
		user.State.CallbackData.RawQuery = q.Encode()

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{Text: helpers.AppendSection, CallbackData: helpers.AggregateCallbackData(state, index+1, "-1")},
				},
			},
		}

		sectionsNumber, err := h.driveFileService.GetSectionsNumber(user.State.CallbackData.Query().Get("driveFileId"))
		if err != nil {
			return err
		}

		for i := 0; i < sectionsNumber; i++ {
			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
				{Text: fmt.Sprintf("Вместо %d-й секции", i+1), CallbackData: helpers.AggregateCallbackData(state, index+1, fmt.Sprintf("%d", i))},
			})
		}
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
			{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")},
		})

		c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{
			Caption:     helpers.AddCallbackData("Куда ты хочешь вставить новую тональность?", user.State.CallbackData.String()),
			ReplyMarkup: markup,
			ParseMode:   "HTML",
		})

		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "upload_document")

		_, _, sectionIndexStr := helpers.ParseCallbackData(c.CallbackQuery.Data)

		sectionIndex, _ := strconv.Atoi(sectionIndexStr)

		driveFile, err := h.driveFileService.TransposeOne(
			user.State.CallbackData.Query().Get("driveFileId"),
			user.State.CallbackData.Query().Get("key"),
			sectionIndex)
		if err != nil {
			return err
		}

		song, err := h.songService.FindOneByDriveFileID(driveFile.Id)
		if err != nil {
			return err
		}

		fakeTime, _ := time.Parse("2006", "2006")
		song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

		_, err = h.songService.UpdateOne(*song)

		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")
		return h.enterInlineHandler(c, user)
	})

	return helpers.TransposeSongState, handlerFunc
}

func styleSongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	// Print list of found songs.
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.CallbackData.Query().Get("driveFileId")

		driveFile, err := h.driveFileService.StyleOne(driveFileID)
		if err != nil {
			return err
		}

		song, err := h.songService.FindOneByDriveFileID(driveFile.Id)
		if err != nil {
			return err
		}

		fakeTime, _ := time.Parse("2006", "2006")
		song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

		_, err = h.songService.UpdateOne(*song)
		if err != nil {
			return err
		}

		// c.CallbackQuery.Answer(h.bot, nil)
		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")
		return h.enterInlineHandler(c, user)
	})
	return helpers.StyleSongState, handlerFunc
}

func addLyricsPageHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	// Print list of found songs.
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.CallbackData.Query().Get("driveFileId")

		driveFile, err := h.driveFileService.AddLyricsPage(driveFileID)
		if err != nil {
			return err
		}

		song, err := h.songService.FindOneByDriveFileID(driveFile.Id)
		if err != nil {
			return err
		}

		fakeTime, _ := time.Parse("2006", "2006")
		song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

		_, err = h.songService.UpdateOne(*song)
		if err != nil {
			return err
		}

		// c.CallbackQuery.Answer(h.bot, nil)
		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")
		return h.enterInlineHandler(c, user)
	})
	return helpers.AddLyricsPageState, handlerFunc
}

func changeSongBPMHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.CallbackData.Query().Get("driveFileId")

		user.State = entity.State{
			Index: 1,
			Name:  helpers.ChangeSongBPMState,
			Context: entity.Context{
				DriveFileID: driveFileID,
			},
		}

		markup := gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
			Keyboard: [][]gotgbot.KeyboardButton{
				{{Text: "60"}, {Text: "65"}, {Text: "70"}, {Text: "75"}, {Text: "80"}, {Text: "85"}},
				{{Text: "90"}, {Text: "95"}, {Text: "100"}, {Text: "105"}, {Text: "110"}, {Text: "115"}},
				{{Text: "120"}, {Text: "125"}, {Text: "130"}, {Text: "135"}, {Text: "140"}, {Text: "145"}},
				{{Text: helpers.Cancel}},
			},
		}
		c.EffectiveChat.SendMessage(h.bot, "Введи новый темп:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		c.CallbackQuery.Answer(h.bot, nil)

		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		_, err := h.driveFileService.ReplaceAllTextByRegex(user.State.Context.DriveFileID, regexp.MustCompile(`(?i)bpm:(.*?);`), fmt.Sprintf("BPM: %s;", c.EffectiveMessage.Text))
		if err != nil {
			return err
		}

		song, err := h.songService.FindOneByDriveFileID(user.State.Context.DriveFileID)
		if err != nil {
			return err
		}

		song.PDF.BPM = c.EffectiveMessage.Text

		fakeTime, _ := time.Parse("2006", "2006")
		song.PDF.ModifiedTime = fakeTime.Format(time.RFC3339)

		song, err = h.songService.UpdateOne(*song)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Index:   0,
			Name:    helpers.SongActionsState,
			Context: user.State.Context,
			Next:    &entity.State{Name: helpers.MainMenuState, Index: 0},
		}
		return h.Enter(c, user)
	})

	return helpers.ChangeSongBPMState, handlerFunc
}

func copySongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileID := user.State.CallbackData.Query().Get("driveFileId")

		c.EffectiveChat.SendAction(h.bot, "typing")

		file, err := h.driveFileService.FindOneByID(driveFileID)
		if err != nil {
			return err
		}

		file = &drive.File{
			Name:    file.Name,
			Parents: []string{user.Band.DriveFolderID},
		}

		copiedSong, err := h.driveFileService.CloneOne(driveFileID, file)
		if err != nil {
			return err
		}

		song, _, err := h.songService.FindOrCreateOneByDriveFileID(copiedSong.Id)
		if err != nil {
			return err
		}

		q := user.State.CallbackData.Query()
		q.Set("driveFileId", copiedSong.Id)
		user.State.CallbackData.RawQuery = q.Encode()

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: helpers.GetSongInitKeyboard(user, song),
		}
		c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{
			Caption:     helpers.AddCallbackData("Скопировано", user.State.CallbackData.String()),
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	return helpers.CopySongState, handlerFunc
}

func createSongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	markup := gotgbot.ReplyKeyboardMarkup{
		Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
		ResizeKeyboard: true,
	}
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь название:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		markup := &gotgbot.ReplyKeyboardMarkup{
			Keyboard:       helpers.CancelOrSkipKeyboard,
			ResizeKeyboard: true,
		}

		user.State.Context.CreateSongPayload.Name = c.EffectiveMessage.Text
		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь слова:", &gotgbot.SendMessageOpts{
			ReplyMarkup: markup,
		})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		switch c.EffectiveMessage.Text {
		case helpers.Skip:
		default:
			user.State.Context.CreateSongPayload.Lyrics = c.EffectiveMessage.Text
		}

		markup := gotgbot.ReplyKeyboardMarkup{
			Keyboard:       append(helpers.KeysKeyboard, helpers.CancelOrSkipKeyboard...),
			ResizeKeyboard: true,
		}
		_, err := c.EffectiveChat.SendMessage(h.bot, "Выбери или отправь тональность:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		switch c.EffectiveMessage.Text {
		case helpers.Skip:
		default:
			user.State.Context.CreateSongPayload.Key = c.EffectiveMessage.Text
		}

		markup := gotgbot.ReplyKeyboardMarkup{
			Keyboard:       helpers.CancelOrSkipKeyboard,
			ResizeKeyboard: true,
		}
		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь темп:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		switch c.EffectiveMessage.Text {
		case helpers.Skip:
		default:
			user.State.Context.CreateSongPayload.BPM = c.EffectiveMessage.Text
		}

		markup := gotgbot.ReplyKeyboardMarkup{
			Keyboard:       append(helpers.TimesKeyboard, helpers.CancelOrSkipKeyboard...),
			ResizeKeyboard: true,
		}
		_, err := c.EffectiveChat.SendMessage(h.bot, "Выбери или отправь размер:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		switch c.EffectiveMessage.Text {
		case helpers.Skip:
		default:
			user.State.Context.CreateSongPayload.Time = c.EffectiveMessage.Text
		}

		c.EffectiveChat.SendAction(h.bot, "upload_document")

		file := &drive.File{
			Name:     user.State.Context.CreateSongPayload.Name,
			Parents:  []string{user.Band.DriveFolderID},
			MimeType: "application/vnd.google-apps.document",
		}
		newFile, err := h.driveFileService.CreateOne(
			file,
			user.State.Context.CreateSongPayload.Lyrics,
			user.State.Context.CreateSongPayload.Key,
			user.State.Context.CreateSongPayload.BPM,
			user.State.Context.CreateSongPayload.Time,
		)

		if err != nil {
			return err
		}

		newFile, err = h.driveFileService.StyleOne(newFile.Id)
		if err != nil {
			return err
		}

		user.State = entity.State{
			Index: 0,
			Name:  helpers.SongActionsState,
			Context: entity.Context{
				DriveFileID: newFile.Id,
			},
			Next: &entity.State{
				Name: helpers.MainMenuState,
			},
		}

		return h.Enter(c, user)
	})

	return helpers.CreateSongState, handlerFunc
}

func deleteSongHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		if user.Role == helpers.Admin {
			err := h.songService.DeleteOneByDriveFileID(user.State.CallbackData.Query().Get("driveFileId"))
			if err != nil {
				return err
			}

			c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{Caption: "Удалено"})
		}

		return nil
	})

	return helpers.DeleteSongState, handlerFunc
}

func getVoicesHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, _ := helpers.ParseCallbackData(c.CallbackQuery.Data)

		song, driveFileID, err := h.songService.FindOrCreateOneByDriveFileID(user.State.CallbackData.Query().Get("driveFileId"))
		if err != nil {
			return err
		}

		if song.Voices == nil || len(song.Voices) == 0 {
			markup := gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: helpers.GetSongActionsKeyboard(*user, *song, *driveFileID),
			}
			c.EffectiveMessage.EditCaption(h.bot, &gotgbot.EditMessageCaptionOpts{
				Caption:     helpers.AddCallbackData("У этой песни нет партий. Чтобы добавить, отправь мне голосовое сообщение.", user.State.CallbackData.String()),
				ParseMode:   "HTML",
				ReplyMarkup: markup,
			})
			return nil
		} else {
			markup := gotgbot.InlineKeyboardMarkup{}

			for _, voice := range song.Voices {
				markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
					{Text: voice.Name, CallbackData: helpers.AggregateCallbackData(state, index+1, voice.ID.Hex())},
				})
			}

			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{
				{Text: helpers.Back, CallbackData: helpers.AggregateCallbackData(helpers.SongActionsState, 0, "")},
				{Text: "➕ Добавить партию", CallbackData: helpers.AggregateCallbackData(helpers.UploadVoiceState, 4, "")},
			})

			c.EffectiveMessage.EditMedia(h.bot, &gotgbot.InputMediaDocument{
				Media:     song.PDF.TgFileID,
				Caption:   helpers.AddCallbackData("Выбери партию:", user.State.CallbackData.String()),
				ParseMode: "HTML",
			}, &gotgbot.EditMessageMediaOpts{
				ReplyMarkup: markup,
			})

			return nil
		}
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		state, index, voiceIDHex := helpers.ParseCallbackData(c.CallbackQuery.Data)

		voiceID, err := primitive.ObjectIDFromHex(voiceIDHex)
		if err != nil {
			return err
		}

		voice, err := h.voiceService.FindOneByID(voiceID)
		if err != nil {
			return err
		}

		markup := gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{Text: helpers.Back, CallbackData: helpers.AggregateCallbackData(state, index-1, "")},
					{Text: helpers.Delete, CallbackData: helpers.AggregateCallbackData(helpers.DeleteVoiceState, index-1, voiceIDHex)},
				},
			},
		}

		song, driveFile, err := h.songService.FindOrCreateOneByDriveFileID(user.State.CallbackData.Query().Get("driveFileId"))
		getPerformer := func() string {
			if driveFile != nil {
				return driveFile.Name
			} else {
				return ""
			}
		}
		getCaption := func() string {
			if song != nil {
				return song.Caption() + "\n" + strings.Join(song.Tags, ", ")
			} else {
				return "-"
			}
		}

		if voice.AudioFileID == "" {
			f, err := h.bot.GetFile(voice.FileID)
			if err != nil {
				return err
			}

			reader, err := helpers.File(h.bot, f)
			if err != nil {
				return err
			}

			msg, _, err := c.EffectiveMessage.EditMedia(h.bot, &gotgbot.InputMediaAudio{
				Media: gotgbot.NamedFile{
					File:     reader,
					FileName: voice.Name,
				},
				Caption:   helpers.AddCallbackData(getCaption(), user.State.CallbackData.String()),
				ParseMode: "HTML",
				Performer: getPerformer(),
				Title:     voice.Name,
			}, &gotgbot.EditMessageMediaOpts{
				ReplyMarkup: markup,
			})
			if err != nil {
				return err
			}

			if err != nil {
				c.CallbackQuery.Answer(h.bot, nil)
				return err
			}
			voice.AudioFileID = msg.Audio.FileId
			h.voiceService.UpdateOne(*voice)
		} else {

			c.EffectiveMessage.EditMedia(h.bot, &gotgbot.InputMediaAudio{
				Media:     voice.AudioFileID, // todo
				Caption:   helpers.AddCallbackData(getCaption(), user.State.CallbackData.String()),
				ParseMode: "HTML",
				Performer: getPerformer(),
				Title:     voice.Name,
			}, &gotgbot.EditMessageMediaOpts{
				ReplyMarkup: markup,
			})
		}

		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		switch c.EffectiveMessage.Text {
		case helpers.Back:
			user.State.Index = 0
			return h.Enter(c, user)
		case helpers.Members:
			// TODO: handle delete
			return nil
		default:
			_, err := c.EffectiveChat.SendMessage(h.bot, "Я тебя не понимаю. Нажми на кнопку.", nil)
			return err
		}
	})

	return helpers.GetVoicesState, handlerFunc
}

func uploadVoiceHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	markup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard:       [][]gotgbot.KeyboardButton{{{Text: helpers.Cancel}}},
		ResizeKeyboard: true,
	}

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, err := c.EffectiveChat.SendMessage(h.bot, "Введи название песни, к которой ты хочешь прикрепить эту партию:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		driveFiles, _, err := h.driveFileService.FindSomeByFullTextAndFolderID(c.EffectiveMessage.Text, user.Band.DriveFolderID, "")
		if err != nil {
			return err
		}

		if len(driveFiles) == 0 {
			_, err := c.EffectiveChat.SendMessage(h.bot, "Ничего не найдено. Попробуй другое название.", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			return err
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
		}

		// TODO: some sort of pagination.
		for _, driveFile := range driveFiles {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: driveFile.Name}})
		}
		markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: helpers.Cancel}})

		_, err = c.EffectiveChat.SendMessage(h.bot, txt.Get("text.chooseSong", c.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "upload_document")

		foundDriveFile, err := h.driveFileService.FindOneByNameAndFolderID(c.EffectiveMessage.Text, user.Band.DriveFolderID)
		if err != nil {
			user.State.Index--
			return h.Enter(c, user)
		}

		song, _, err := h.songService.FindOrCreateOneByDriveFileID(foundDriveFile.Id)
		if err != nil {
			return err
		}

		user.State.Context.DriveFileID = song.DriveFileID

		markup := markup
		_, err = c.EffectiveChat.SendMessage(h.bot, "Отправь мне название этой партии:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Context.Voice.Name = c.EffectiveMessage.Text

		song, err := h.songService.FindOneByDriveFileID(user.State.Context.DriveFileID)
		if err != nil {
			return err
		}

		user.State.Context.Voice.SongID = song.ID

		_, err = h.voiceService.UpdateOne(*user.State.Context.Voice)
		if err != nil {
			return err
		}

		c.EffectiveChat.SendMessage(h.bot, "Добавление завершено.", nil)

		user.State = entity.State{
			Name: helpers.SongActionsState,
			Context: entity.Context{
				DriveFileID: user.State.Context.DriveFileID,
			},
		}
		return h.Enter(c, user)
	})

	// Upload voice from song menu.
	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State = entity.State{
			Name:    helpers.UploadVoiceState,
			Index:   4,
			Context: entity.Context{DriveFileID: user.State.CallbackData.Query().Get("driveFileId")},
		}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь мне аудио или голосовое сообщение:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		c.EffectiveChat.SendAction(h.bot, "typing")

		fileID := c.EffectiveMessage.Voice.FileId
		if fileID == "" {
			fileID = c.EffectiveMessage.Audio.FileId
		}
		user.State.Context.Voice = &entity.Voice{FileID: fileID}

		_, err := c.EffectiveChat.SendMessage(h.bot, "Отправь мне название этой партии:", &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		user.State.Context.Voice.Name = c.EffectiveMessage.Text

		song, err := h.songService.FindOneByDriveFileID(user.State.Context.DriveFileID)
		if err != nil {
			return err
		}

		user.State.Context.Voice.SongID = song.ID

		_, err = h.voiceService.UpdateOne(*user.State.Context.Voice)
		if err != nil {
			return err
		}

		c.EffectiveChat.SendMessage(h.bot, "Добавление завершено.", nil)

		user.State = entity.State{
			Name: helpers.SongActionsState,
			Context: entity.Context{
				DriveFileID: user.State.Context.DriveFileID,
			},
			Next: &entity.State{Name: helpers.MainMenuState},
		}
		return h.Enter(c, user)
	})

	return helpers.UploadVoiceState, handlerFunc
}

func deleteVoiceHandler() (int, []HandlerFunc) {
	handlerFuncs := make([]HandlerFunc, 0)

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, index, voiceIDHex := helpers.ParseCallbackData(c.CallbackQuery.Data)

		markup := gotgbot.InlineKeyboardMarkup{}
		markup.InlineKeyboard = [][]gotgbot.InlineKeyboardButton{
			{
				{Text: helpers.Cancel, CallbackData: helpers.AggregateCallbackData(helpers.GetVoicesState, 0, "")},
				{Text: helpers.Yes, CallbackData: helpers.AggregateCallbackData(helpers.DeleteVoiceState, index+1, voiceIDHex)},
			},
		}

		_, _, err := c.EffectiveMessage.EditCaption(h.bot,
			&gotgbot.EditMessageCaptionOpts{
				Caption:     helpers.AddCallbackData("Ты уверен, что хочешь удалить эту партию?", user.State.CallbackData.String()),
				ReplyMarkup: markup,
				ParseMode:   "HTML",
			})
		return err
	})

	handlerFuncs = append(handlerFuncs, func(h *Handler, c *ext.Context, user *entity.User) error {

		_, _, voiceIDHex := helpers.ParseCallbackData(c.CallbackQuery.Data)

		voiceID, err := primitive.ObjectIDFromHex(voiceIDHex)
		if err != nil {
			return err
		}

		err = h.voiceService.DeleteOne(voiceID)
		if err != nil {
			return err
		}

		c.CallbackQuery.Data = helpers.AggregateCallbackData(helpers.GetVoicesState, 0, "")
		return h.enterInlineHandler(c, user)
	})

	return helpers.DeleteVoiceState, handlerFuncs
}

func setlistHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		if len(user.State.Context.SongNames) < 1 {
			user.State.Index = 2
			return h.Enter(c, user)
		}

		songNames := user.State.Context.SongNames

		currentSongName := songNames[0]
		user.State.Context.SongNames = songNames[1:]

		c.EffectiveChat.SendAction(h.bot, "typing")

		driveFiles, _, err := h.driveFileService.FindSomeByFullTextAndFolderID(currentSongName, user.Band.DriveFolderID, "")
		if err != nil {
			return err
		}

		if len(driveFiles) == 0 {
			markup := &gotgbot.ReplyKeyboardMarkup{
				Keyboard:       helpers.CancelOrSkipKeyboard,
				ResizeKeyboard: true,
			}

			msg, err := c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("По запросу \"%s\" ничего не найдено. Напиши новое название или пропусти эту песню.", currentSongName), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
			if err != nil {
				return err
			}

			user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)
			user.State.Index++
			return err
		}

		markup := &gotgbot.ReplyKeyboardMarkup{
			ResizeKeyboard:        true,
			InputFieldPlaceholder: currentSongName,
		}

		// TODO: some sort of pagination.
		for _, song := range driveFiles {
			markup.Keyboard = append(markup.Keyboard, []gotgbot.KeyboardButton{{Text: song.Name}})
		}
		markup.Keyboard = append(markup.Keyboard, helpers.CancelOrSkipKeyboard...)

		msg, err := c.EffectiveChat.SendMessage(h.bot, fmt.Sprintf("Выбери песню по запросу \"%s\" или введи другое название:", currentSongName), &gotgbot.SendMessageOpts{ReplyMarkup: markup})
		if err != nil {
			return err
		}

		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, msg.MessageId)
		user.State.Index++
		return nil
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {
		user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.EffectiveMessage.MessageId)

		switch c.EffectiveMessage.Text {
		case helpers.Skip:
			user.State.Index = 0
			return h.Enter(c, user)
		}

		foundDriveFile, err := h.driveFileService.FindOneByNameAndFolderID(c.EffectiveMessage.Text, user.Band.DriveFolderID)
		if err != nil {
			user.State.Context.SongNames = append([]string{c.EffectiveMessage.Text}, user.State.Context.SongNames...)
		} else {
			user.State.Context.FoundDriveFileIDs = append(user.State.Context.FoundDriveFileIDs, foundDriveFile.Id)
		}

		user.State.Index = 0
		return h.Enter(c, user)
	})

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		driveFileIDs := user.State.Context.FoundDriveFileIDs
		messagesToDelete := user.State.Context.MessagesToDelete
		if user.State.Next != nil {
			user.State = *user.State.Next
			user.State.Context.FoundDriveFileIDs = driveFileIDs
			user.State.Context.MessagesToDelete = messagesToDelete
			return h.Enter(c, user)
		} else {
			user.State = *user.State.Prev
			return nil
		}

		// user.State = user.State.Prev
		// user.State.Index = 0
		//
		// return h.enter(c, user)
	})

	return helpers.SetlistState, handlerFunc
}

func editInlineKeyboardHandler() (int, []HandlerFunc) {
	handlerFunc := make([]HandlerFunc, 0)

	handlerFunc = append(handlerFunc, func(h *Handler, c *ext.Context, user *entity.User) error {

		markup := gotgbot.InlineKeyboardMarkup{}
		markup.InlineKeyboard = helpers.GetEditEventKeyboard(*user)
		c.EffectiveMessage.EditReplyMarkup(h.bot, &gotgbot.EditMessageReplyMarkupOpts{
			ReplyMarkup: markup,
		})
		c.CallbackQuery.Answer(h.bot, nil)
		return nil
	})

	return helpers.EditInlineKeyboardState, handlerFunc
}

func chunkAlbumBy(items []gotgbot.InputMedia, chunkSize int) (chunks [][]gotgbot.InputMedia) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}
