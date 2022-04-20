package handlers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joeyave/scala-bot-v2/entities"
	"github.com/joeyave/scala-bot-v2/helpers"
	"github.com/joeyave/scala-bot-v2/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/url"
	"regexp"

	"strings"
	"time"
)

type Handler struct {
	bot               *gotgbot.Bot
	userService       *services.UserService
	driveFileService  *services.DriveFileService
	songService       *services.SongService
	voiceService      *services.VoiceService
	bandService       *services.BandService
	membershipService *services.MembershipService
	eventService      *services.EventService
	roleService       *services.RoleService
}

func NewHandler(
	bot *gotgbot.Bot,
	userService *services.UserService,
	driveFileService *services.DriveFileService,
	songService *services.SongService,
	voiceService *services.VoiceService,
	bandService *services.BandService,
	membershipService *services.MembershipService,
	eventService *services.EventService,
	roleService *services.RoleService,
) *Handler {

	return &Handler{
		bot:               bot,
		userService:       userService,
		driveFileService:  driveFileService,
		songService:       songService,
		voiceService:      voiceService,
		bandService:       bandService,
		membershipService: membershipService,
		eventService:      eventService,
		roleService:       roleService,
	}
}

func (h *Handler) OnText(b *gotgbot.Bot, ctx *ext.Context) error {

	user, err := h.userService.FindOneByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	// Handle buttons.
	switch ctx.EffectiveMessage.Text {
	case helpers.Cancel, helpers.Back:

		// user.State.Context.MessagesToDelete = append(user.State.Context.MessagesToDelete, c.Message().ID)
		// for _, messageID := range user.State.Context.MessagesToDelete {
		// 	h.bot.Delete(&telebot.Message{
		// 		ID:   messageID,
		// 		Chat: c.Chat(),
		// 	})
		// }
		if user.State.Prev != nil {
			user.State = user.State.Prev
			user.State.Index = 0
		} else {
			user.State = &entities.State{
				Index: 0,
				Name:  helpers.MainMenuState,
			}
		}

	case helpers.Menu:
		user.State = &entities.State{
			Index: 0,
			Name:  helpers.MainMenuState,
		}
	}

	err = h.enter(ctx, user)
	if err != nil {
		return err
	}

	_, err = h.userService.UpdateOne(*user)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) OnVoice(b *gotgbot.Bot, ctx *ext.Context) error {

	user, err := h.userService.FindOneByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	if user.State.Name != helpers.UploadVoiceState {
		user.State = &entities.State{
			Index: 0,
			Name:  helpers.UploadVoiceState,
			Context: entities.Context{
				Voice: &entities.Voice{
					FileID: ctx.EffectiveMessage.Voice.FileId,
				},
			},
			Prev: user.State,
		}
	}

	err = h.enter(ctx, user)
	if err != nil {
		return err
	}

	_, err = h.userService.UpdateOne(*user)
	if err != nil {
		return err
	}

	return err
}

func (h *Handler) OnAudio(b *gotgbot.Bot, ctx *ext.Context) error {

	user, err := h.userService.FindOneByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	if user.State.Name != helpers.UploadVoiceState {
		user.State = &entities.State{
			Index: 0,
			Name:  helpers.UploadVoiceState,
			Context: entities.Context{
				Voice: &entities.Voice{
					FileID: ctx.EffectiveMessage.Audio.FileId,
				},
			},
			Prev: user.State,
		}
	}

	err = h.enter(ctx, user)
	if err != nil {
		return err
	}

	_, err = h.userService.UpdateOne(*user)
	if err != nil {
		return err
	}

	return err
}

func (h *Handler) OnCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	user, err := h.userService.FindOneByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	err = h.enter(ctx, user)
	if err != nil {
		return err
	}

	_, err = h.userService.UpdateOne(*user)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) RegisterUser(b *gotgbot.Bot, ctx *ext.Context) error {

	user, err := h.userService.FindOneOrCreateByID(ctx.EffectiveChat.Id)
	if err != nil {
		return err
	}

	// if user.Name == "" {
	// }
	user.Name = strings.TrimSpace(fmt.Sprintf("%s %s", ctx.EffectiveChat.FirstName, ctx.EffectiveChat.LastName))

	if user.BandID == primitive.NilObjectID && user.State.Name != helpers.ChooseBandState && user.State.Name != helpers.CreateBandState {
		user.State = &entities.State{
			Index: 0,
			Name:  helpers.ChooseBandState,
		}
	}

	ctx.Data["user"] = user

	return nil
}

func (h *Handler) NotifyUser() {
	for range time.Tick(time.Hour * 2) {
		events, err := h.eventService.FindAllFromToday()
		if err != nil {
			return
		}

		for _, event := range events {
			if event.Time.Add(time.Hour*8).Sub(time.Now()).Hours() < 48 {
				for _, membership := range event.Memberships {
					if membership.Notified == true {
						continue
					}

					eventString := h.eventService.ToHtmlStringByEvent(*event)
					text := fmt.Sprintf("Привет. Ты учавствуешь в собрании через несколько дней! Вот план:\n\n%s", eventString)

					_, err := h.bot.SendMessage(membership.UserID, text, &gotgbot.SendMessageOpts{
						ParseMode:             "HTML",
						DisableWebPagePreview: true,
					})
					if err != nil {
						continue
					}

					membership.Notified = true
					h.membershipService.UpdateOne(*membership)
				}
			}
		}
	}
}

func (h *Handler) enter(c *ext.Context, user *entities.User) error {

	if user.State.CallbackData == nil {
		user.State.CallbackData, _ = url.Parse("t.me/callbackData")
	}

	if c.CallbackQuery != nil {
		return h.enterInlineHandler(c, user)
	} else {
		return h.enterReplyHandler(c, user)
	}
}

func (h *Handler) enterInlineHandler(c *ext.Context, user *entities.User) error {

	re := regexp.MustCompile(`t\.me/callbackData.*`)

	for _, entity := range c.CallbackQuery.Message.CaptionEntities {
		if entity.Type == "text_link" {
			matches := re.FindStringSubmatch(entity.Url)

			if len(matches) > 0 {
				u, err := url.Parse(matches[0])
				if err != nil {
					return err
				}

				user.State.CallbackData = u
				break
			}
		}
	}

	for _, entity := range c.CallbackQuery.Message.Entities {
		if entity.Type == "text_link" {
			matches := re.FindStringSubmatch(entity.Url)

			if len(matches) > 0 {
				u, err := url.Parse(matches[0])
				if err != nil {
					return err
				}

				user.State.CallbackData = u
				break
			}
		}
	}

	state, index, _ := helpers.ParseCallbackData(c.CallbackQuery.Data)

	// Handle error.
	handlerFuncs, _ := handlers[state]

	return handlerFuncs[index](h, c, user)
}

func (h *Handler) enterReplyHandler(c *ext.Context, user *entities.User) error {
	handlerFuncs, ok := handlers[user.State.Name]

	if ok == false || user.State.Index < 0 || user.State.Index >= len(handlerFuncs) {
		user.State = &entities.State{Name: helpers.MainMenuState}
		handlerFuncs = handlers[user.State.Name]
	}

	return handlerFuncs[user.State.Index](h, c, user)
}