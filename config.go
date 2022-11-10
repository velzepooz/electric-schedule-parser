package main

import (
	"log"
	"os"
	"time"
)

type Config struct {
	botToken          string
	chatID            string
	addressesToSearch [2]AddressData
	schedulerUrl      string
	streetIDUrl       string
	locale            *time.Location
}

func GetConfig() Config {
	var addressesToSearch = [2]AddressData{
		{streetName: os.Getenv("STREET_NAME_ONE"), houseNumber: os.Getenv("HOUSE_NUMBER_ONE"), houseNumberToSearch: os.Getenv("HOUSE_NUMBER_TO_SEARCH_ONE"), region: os.Getenv("REGION_ONE"), who: os.Getenv("WHO_ONE")},
		{streetName: os.Getenv("STREET_NAME_TWO"), houseNumber: os.Getenv("HOUSE_NUMBER_TWO"), houseNumberToSearch: os.Getenv("HOUSE_NUMBER_TO_SEARCH_TWO"), region: os.Getenv("REGION_TWO"), who: os.Getenv("WHO_TWO")},
	}

	locale, err := time.LoadLocation("Europe/Kiev")
	if err != nil {
		log.Panic(err)
	}

	return Config{
		botToken:          os.Getenv("TELEGRAM_BOT_TOKEN"),
		chatID:            os.Getenv("CHAT_ID"),
		addressesToSearch: addressesToSearch,
		schedulerUrl:      os.Getenv("SCHEDULER_URL"),
		streetIDUrl:       os.Getenv("STREET_ID_URL"),
		locale:            locale,
	}
}
