package controller

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/joeyave/scala-bot-v2/entity"
	"github.com/joeyave/scala-bot-v2/keyboard"
	"github.com/joeyave/scala-bot-v2/state"
	"github.com/joeyave/scala-bot-v2/txt"
	"github.com/joeyave/scala-bot-v2/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"regexp"
	"strconv"
	"strings"
)

func (c *BotController) SettingsChooseBand(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	hex := util.ParseCallbackPayload(ctx.CallbackQuery.Data)
	bandID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return err
	}

	band, err := c.BandService.FindOneByID(bandID)

	user.BandID = bandID

	_, _, err = bot.EditMessageText(txt.Get("text.addedToBand", ctx.EffectiveUser.LanguageCode, band.Name), &gotgbot.EditMessageTextOpts{
		ChatId:    ctx.EffectiveChat.Id,
		MessageId: ctx.EffectiveMessage.MessageId,
		//ReplyMarkup: gotgbot.InlineKeyboardMarkup{},
	})
	if err != nil {
		return err
	}

	return c.Menu(bot, ctx)
}

func (c *BotController) Settings(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboard.Settings(user, ctx.EffectiveUser.LanguageCode),
	}

	text := txt.Get("button.settings", ctx.EffectiveUser.LanguageCode) + ":"
	_, err := ctx.EffectiveChat.SendMessage(bot, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: markup,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *BotController) SettingsCB(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	markup := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: keyboard.Settings(user, ctx.EffectiveUser.LanguageCode),
	}

	text := txt.Get("button.settings", ctx.EffectiveUser.LanguageCode) + ":"
	_, _, err := ctx.EffectiveMessage.EditText(bot, text, &gotgbot.EditMessageTextOpts{
		ReplyMarkup: markup,
	})
	if err != nil {
		return err
	}

	ctx.CallbackQuery.Answer(bot, nil)

	return nil
}

func (c *BotController) BandCreate_AskForName(bot *gotgbot.Bot, ctx *ext.Context) error {

	user := ctx.Data["user"].(*entity.User)

	markup := &gotgbot.ReplyKeyboardMarkup{
		Keyboard:       [][]gotgbot.KeyboardButton{{{Text: txt.Get("button.cancel", ctx.EffectiveUser.LanguageCode)}}},
		ResizeKeyboard: true,
	}

	_, err := ctx.EffectiveChat.SendMessage(bot, txt.Get("text.sendBandName", ctx.EffectiveUser.LanguageCode), &gotgbot.SendMessageOpts{
		ReplyMarkup: markup,
	})
	if err != nil {
		return err
	}

	user.State = entity.State{
		Name:  state.BandCreate,
		Index: 0,
	}

	_, err = c.UserService.UpdateOne(*user)
	if err != nil {
		return err
	}

	ctx.CallbackQuery.Answer(bot, nil)

	return nil
}

func (c *BotController) BandCreate(index int) handlers.Response {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {

		user := ctx.Data["user"].(*entity.User)

		if user.State.Name != state.BandCreate {
			user.State = entity.State{
				Index: index,
				Name:  state.BandCreate,
			}
			user.Cache = entity.Cache{}
		}

		switch index {
		case 0:
			{
				user.Cache.Band = &entity.Band{
					Name: ctx.EffectiveMessage.Text,
				}

				markup := &gotgbot.ReplyKeyboardMarkup{
					Keyboard:       [][]gotgbot.KeyboardButton{{{Text: txt.Get("button.cancel", ctx.EffectiveUser.LanguageCode)}}},
					ResizeKeyboard: true,
				}

				_, err := ctx.EffectiveChat.SendMessage(bot, "Теперь добавь имейл scala-drive@scala-chords-bot.iam.gserviceaccount.com в папку на Гугл Диске как редактора. После этого отправь мне ссылку на эту папку.", &gotgbot.SendMessageOpts{
					ReplyMarkup: markup,
				})
				if err != nil {
					return err
				}

				user.State.Index = 1
				return nil
			}
		case 1:
			{
				re := regexp.MustCompile(`(/folders/|id=)(.*?)(/|\?|$)`)
				matches := re.FindStringSubmatch(ctx.EffectiveMessage.Text)
				if len(matches) < 3 {
					return c.BandCreate(0)(bot, ctx)
				}

				user.Cache.Band.DriveFolderID = matches[2]
				user.Role = entity.AdminRole // todo
				band, err := c.BandService.UpdateOne(*user.Cache.Band)
				if err != nil {
					return err
				}

				user.BandID = band.ID

				text := fmt.Sprintf("Ты добавлен в группу \"%s\" как администратор.", band.Name)
				_, err = ctx.EffectiveChat.SendMessage(bot, text, nil)
				if err != nil {
					return err
				}

				return c.Menu(bot, ctx)
			}
		}
		return nil
	}
}

func (c *BotController) SettingsBands(bot *gotgbot.Bot, ctx *ext.Context) error {

	//user := ctx.Data["user"].(*entity.User)

	markup := gotgbot.InlineKeyboardMarkup{}

	bands, err := c.BandService.FindAll()
	if err != nil {
		return err
	}
	for _, band := range bands {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: band.Name, CallbackData: util.CallbackData(state.SettingsChooseBand, band.ID.Hex())}})
	}
	markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.createBand", ctx.EffectiveUser.LanguageCode), CallbackData: util.CallbackData(state.BandCreate_AskForName, "")}})
	markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.back", ctx.EffectiveUser.LanguageCode), CallbackData: util.CallbackData(state.SettingsCB, "")}})

	text := txt.Get("text.chooseBand", ctx.EffectiveUser.LanguageCode)
	_, _, err = ctx.EffectiveMessage.EditText(bot, text, &gotgbot.EditMessageTextOpts{
		ReplyMarkup: markup,
	})
	if err != nil {
		return err
	}

	ctx.CallbackQuery.Answer(bot, nil)

	return nil
}

func (c *BotController) SettingsBandMembers(bot *gotgbot.Bot, ctx *ext.Context) error {

	//user := ctx.Data["user"].(*entity.User)

	hex := util.ParseCallbackPayload(ctx.CallbackQuery.Data)
	bandID, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return err
	}

	return c.settingsBandMembers(bot, ctx, bandID)
}

func (c *BotController) settingsBandMembers(bot *gotgbot.Bot, ctx *ext.Context, bandID primitive.ObjectID) error {

	//user := ctx.Data["user"].(*entity.User)

	members, err := c.UserService.FindMultipleByBandID(bandID)
	if err != nil {
		return err
	}

	markup := gotgbot.InlineKeyboardMarkup{}

	for _, member := range members {
		text := member.Name
		if member.Role == entity.AdminRole {
			text += " ✔️"
			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: text, CallbackData: util.CallbackData(state.SettingsBandAddAdmin, fmt.Sprintf("%s:%d:delete", bandID.Hex(), member.ID))}})
		} else {
			markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: text, CallbackData: util.CallbackData(state.SettingsBandAddAdmin, fmt.Sprintf("%s:%d:add", bandID.Hex(), member.ID))}})
		}
	}

	markup.InlineKeyboard = util.SplitKeyboardToColumns(markup.InlineKeyboard, 2)

	markup.InlineKeyboard = append(markup.InlineKeyboard, []gotgbot.InlineKeyboardButton{{Text: txt.Get("button.back", ctx.EffectiveUser.LanguageCode), CallbackData: util.CallbackData(state.SettingsCB, "")}})

	text := txt.Get("text.chooseMemberToMakeAdmin", ctx.EffectiveUser.LanguageCode)
	_, _, err = ctx.EffectiveMessage.EditText(bot, text, &gotgbot.EditMessageTextOpts{
		ReplyMarkup: markup,
	})
	if err != nil {
		return err
	}

	ctx.CallbackQuery.Answer(bot, nil)

	return nil
}

// todo: BUG! move role from User to Band

func (c *BotController) SettingsBandAddAdmin(bot *gotgbot.Bot, ctx *ext.Context) error {

	//user := ctx.Data["user"].(*entity.User)

	payload := util.ParseCallbackPayload(ctx.CallbackQuery.Data)
	split := strings.Split(payload, ":")

	bandID, err := primitive.ObjectIDFromHex(split[0])
	if err != nil {
		return err
	}

	userID, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return err
	}

	user, err := c.UserService.FindOneByID(userID)
	if err != nil {
		return err
	}

	switch split[2] {
	case "delete":
		user.Role = ""
	case "add":
		user.Role = entity.AdminRole
	}
	_, err = c.UserService.UpdateOne(*user)
	if err != nil {
		return err
	}

	return c.settingsBandMembers(bot, ctx, bandID)
}
