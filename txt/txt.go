package txt

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var locales = map[string]map[string]string{
	"button.schedule": {
		"ru": "🗓️ Расписание",
	},
	"button.menu": {
		"ru": "💻 Меню",
	},
	"button.songs": {
		"ru": "🎵 Песни",
	},
	"button.stats": {
		"ru": "📈 Статистика",
	},
	"button.settings": {
		"ru": "⚙ Настройки",
	},
	"button.next": {
		"ru": "→",
	},
	"button.prev": {
		"ru": "←",
	},
	"button.eventsWithMe": {
		"ru": "🙋‍♂️",
	},
	"button.archive": {
		"ru": "📥",
	},
	"button.like": {
		"ru": "❤️‍🔥",
	},
	"button.calendar": {
		"ru": "📆",
	},
	"button.numbers": {
		"ru": "🔢",
	},
	"button.tag": {
		"ru": "🔖",
	},
	"button.globalSearch": {
		"ru": "🔎 Искать во всех группах",
	},
	"button.cancel": {
		"ru": "🚫 Отмена",
	},
	"button.createDoc": {
		"ru": "➕ Создать документ",
	},

	"text.defaultPlaceholder": {
		"ru": "Фраза из песни или список",
	},
	"text.chooseEvent": {
		"ru": "Выбери собрание:",
	},
	"text.chooseTag": {
		"ru": "Выбери тег:",
	},
	"text.chooseSong": {
		"ru": "Выбери песню:",
	},
	"text.nothingFound": {
		"ru": "Ничего не найдено. Попробуй еще раз.",
	},
	"text.menu": {
		"ru": "Меню:",
	},
}

func init() {
	for key, langToMsgMap := range locales {
		for lang, msg := range langToMsgMap {
			message.SetString(language.Make(lang), key, msg)
		}
	}
}

func Get(key, lang string, a ...interface{}) string {
	//switch lang {
	//case "en":
	//	return enPrinter.Sprintf(key, a)
	//case "uk":
	//	return ukPrinter.Sprintf(key, a)
	//}
	return ruPrinter.Sprintf(key, a)
}

//var ukPrinter = message.NewPrinter(language.Ukrainian)
//var enPrinter = message.NewPrinter(language.English)
var ruPrinter = message.NewPrinter(language.Russian)
