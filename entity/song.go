package entity

import (
	"fmt"
	"github.com/joeyave/scala-bot-v2/util"
	"github.com/klauspost/lctime"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Song struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`

	DriveFileID string `bson:"driveFileId,omitempty" json:"-"`

	BandID primitive.ObjectID `bson:"bandId,omitempty" json:"-"`
	Band   *Band              `bson:"band,omitempty" json:"-"`

	PDF PDF `bson:"pdf,omitempty" json:"pdf"`

	Voices []*Voice `bson:"voices,omitempty" json:"-"`

	Likes []int64  `bson:"likes,omitempty" json:"-"`
	Tags  []string `bson:"tags,omitempty" json:"-"`
}

type PDF struct {
	ModifiedTime string `bson:"modifiedTime,omitempty"`

	TgFileID           string `bson:"tgFileId,omitempty"`
	TgChannelMessageID int    `bson:"tgChannelMessageId,omitempty"`

	Name string `bson:"name,omitempty"`
	Key  string `bson:"key,omitempty"`
	BPM  string `bson:"bpm,omitempty"`
	Time string `bson:"time,omitempty"`

	WebViewLink string `bson:"webViewLink,omitempty"`
}

func (s *Song) Caption() string {
	return fmt.Sprintf("%s, %s, %s", s.PDF.Key, s.PDF.BPM, s.PDF.Time)
}

type SongTagFrequencies struct {
	Tag   string `bson:"_id"`
	Count int    `bson:"count"`
}

type SongWithEvents struct {
	Song `bson:",inline"`

	Events []*Event `bson:"events,omitempty"`
}

func (s *SongWithEvents) Stats(lang string) string {
	if len(s.Events) == 0 {
		return "-, 0"
	}
	t, _ := lctime.StrftimeLoc(util.IetfToIsoLangCode(lang), "%d %b", s.Events[0].Time)
	return fmt.Sprintf("%v, %d", t, len(s.Events))
}
