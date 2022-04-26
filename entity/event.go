package entity

import (
	"fmt"
	"github.com/joeyave/scala-bot-v2/util"
	"github.com/klauspost/lctime"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"time"
)

type Event struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Time time.Time          `bson:"time,omitempty" json:"time"`
	Name string             `bson:"name,omitempty" json:"name"`

	Memberships []*Membership `bson:"memberships,omitempty" json:"memberships"`

	BandID primitive.ObjectID `bson:"bandId,omitempty" json:"bandId"`
	Band   *Band              `bson:"band,omitempty" json:"band"`

	SongIDs []primitive.ObjectID `bson:"songIds,omitempty" json:"songIds"`
	Songs   []*Song              `bson:"songs,omitempty" json:"songs"`

	Notes string `bson:"notes,omitempty" json:"notes"`
}

func (e *Event) Alias(lang string) string {
	t, _ := lctime.StrftimeLoc(util.IetfToIsoLangCode(lang), "%A, %d.%m.%Y", e.Time)
	return fmt.Sprintf("%s (%s)", e.Name, t)
}

func (e *Event) Roles() string {
	str := ""

	var currRoleID primitive.ObjectID
	for _, membership := range e.Memberships {
		if membership.User == nil {
			continue
		}

		if currRoleID != membership.RoleID {
			currRoleID = membership.RoleID
			str = fmt.Sprintf("%s\n\n<b>%s:</b>", str, membership.Role.Name)
		}

		str = fmt.Sprintf("%s\n - <a href=\"tg://user?id=%d\">%s</a>", str, membership.User.ID, membership.User.Name)
	}

	return strings.TrimSpace(str)
}

type EventNameFrequencies struct {
	Name  string `bson:"_id"`
	Count int    `bson:"count"`
}