package helpers

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"io"
	"net/http"
)

func File(bot *gotgbot.Bot, file *gotgbot.File) (io.ReadCloser, error) {

	url := bot.GetAPIURL() + "/file/bot" + bot.Token + "/" + file.FilePath

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := bot.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("telebot: expected status 200 but got %s", resp.Status)
	}

	return resp.Body, nil
}
