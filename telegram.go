package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

func sendDataToTelegram(botToken string, chatID string, messageToTelegram string) error {
	telegramUrl := getTelegramUrl(botToken, chatID, messageToTelegram)

	res, err := http.Get(telegramUrl)

	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Panic("Error: ", err)
		}
	}(res.Body)

	return nil
}

func getTelegramUrl(botToken string, chatID string, message string) string {
	return "https://api.telegram.org/bot" + url.QueryEscape(botToken) + "/sendMessage?chat_id=" + url.QueryEscape(chatID) + "&text=" + url.QueryEscape(message)
}
