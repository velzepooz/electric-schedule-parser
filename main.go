package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"time"
)

var dayNumToUADays = map[int]string{
	0: "Воскресенье",
	1: "Понедельник",
	2: "Вторник",
	3: "Среда",
	4: "Четверг",
	5: "Пятница",
	6: "Суббота",
}

var engMonthToUA = map[int]string{
	1:  "января",
	2:  "февраля",
	3:  "марта",
	4:  "апреля",
	5:  "мая",
	6:  "июня",
	7:  "июля",
	8:  "августа",
	9:  "сентября",
	10: "октября",
	11: "ноября",
	12: "декабря",
}

func main() {
	loadEnv()

	var addressToSearch = [2]AddressData{
		{streetName: os.Getenv("STREET_NAME_ONE"), houseNumber: os.Getenv("HOUSE_NUMBER_ONE"), houseNumberToSearch: os.Getenv("HOUSE_NUMBER_TO_SEARCH_ONE"), region: os.Getenv("REGION_ONE"), who: os.Getenv("WHO_ONE")},
		{streetName: os.Getenv("STREET_NAME_TWO"), houseNumber: os.Getenv("HOUSE_NUMBER_TWO"), houseNumberToSearch: os.Getenv("HOUSE_NUMBER_TO_SEARCH_TWO"), region: os.Getenv("REGION_TWO"), who: os.Getenv("WHO_TWO")},
	}

	loc, err := time.LoadLocation("Europe/Kiev")
	if err != nil {
		log.Panic(err)
	}

	startCroneJob("TZ=Europe/Kiev 05 0 * * 1", func() {
		getWeekScheduleAndSendToTelegram(os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("CHAT_ID"), addressToSearch, os.Getenv("SCHEDULER_URL"), os.Getenv("STREET_ID_URL"), loc)
	})

	startCroneJob("TZ=Europe/Kiev 10 0 * * *", func() {
		getDayScheduleAndSendToTelegram(os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("CHAT_ID"), addressToSearch, os.Getenv("SCHEDULER_URL"), os.Getenv("STREET_ID_URL"), loc)
	})

	log.Println("App is starting...")

	_, err = fmt.Scanln()
	if err != nil {
		return
	}
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func getWeekScheduleAndSendToTelegram(botToken string, chatID string, addressToSearch [2]AddressData, schedulerUrl string, streetIDUrl string, loc *time.Location) {
	log.Println("Sending week schedule")

	schedule := loadScheduleData()
	telegramMessage := fmt.Sprintf("График выключений электроэнергии на неделю, %v\n", getCurrentWeekPeriodInUALocale(loc))

	for _, address := range addressToSearch {
		groupsFromServer, err := requestGroupNumber(schedulerUrl, address, streetIDUrl)

		if err != nil {
			log.Panic(err)
		}

		group := getGroupNumber(address.houseNumber, groupsFromServer)

		if group == 0 {
			log.Fatal("Group not found")
		}

		weekSchedule := getScheduleInfo(group, &schedule)

		telegramMessage += "\n" + address.who + ":\n"

		for dayNumber, daySchedule := range weekSchedule {
			/* make day number like at UA */
			dayNumberUA := dayNumber

			if dayNumberUA == 6 {
				dayNumberUA = 0
			} else {
				dayNumberUA++
			}

			telegramMessage += "\n" + dayNumToUADays[dayNumberUA] + ":\n"
			for _, period := range daySchedule {
				telegramMessage += "\t\t- c " + strconv.Itoa(period.Start) + " до " + strconv.Itoa(period.End) + "\n"
			}
		}
	}

	err := sendDataToTelegram(botToken, chatID, telegramMessage)
	if err != nil {
		log.Panic(err)
	}

	log.Println("Week schedule sent")
}

func getDayScheduleAndSendToTelegram(botToken string, chatID string, addressToSearch [2]AddressData, schedulerUrl string, streetIDUrl string, loc *time.Location) {
	log.Println("Sending day schedule")

	todayDayNumberAtWeek := time.Now().In(loc).Weekday()

	/* make day number like at UA */
	if todayDayNumberAtWeek == 0 {
		todayDayNumberAtWeek = 6
	} else {
		todayDayNumberAtWeek--
	}

	schedule := loadScheduleData()
	telegramMessage := fmt.Sprintf("График выключений электроэнергии на сегодня, %v:\n", getCurrentDateInUALocale(loc))

	for _, address := range addressToSearch {
		groupsFromServer, err := requestGroupNumber(schedulerUrl, address, streetIDUrl)

		if err != nil {
			log.Panic(err)
		}

		group := getGroupNumber(address.houseNumber, groupsFromServer)

		if group == 0 {
			log.Fatal("Group not found")
		}

		weekSchedule := getScheduleInfo(group, &schedule)
		todaySchedule := weekSchedule[todayDayNumberAtWeek]

		telegramMessage += "\n" + address.who + ":\n"

		for _, period := range todaySchedule {
			telegramMessage += "C " + strconv.Itoa(period.Start) + " до " + strconv.Itoa(period.End) + "\n"
		}
	}

	err := sendDataToTelegram(botToken, chatID, telegramMessage)
	if err != nil {
		log.Panic(err)
	}

	log.Println("Day schedule sent")
}

func getCurrentWeekPeriodInUALocale(loc *time.Location) string {
	numberOfDayInWeek := time.Now().In(loc).Weekday()

	firstDayOfWeek := time.Now().In(loc).AddDate(0, 0, int(-numberOfDayInWeek+1))
	lastDayOfWeek := firstDayOfWeek.AddDate(0, 0, 6)

	return strconv.Itoa(int(firstDayOfWeek.Day())) + " " + engMonthToUA[int(firstDayOfWeek.Month())] + " " + strconv.Itoa(firstDayOfWeek.Year()) + " - " + strconv.Itoa(int(lastDayOfWeek.Day())) + " " + engMonthToUA[int(lastDayOfWeek.Month())] + " " + strconv.Itoa(lastDayOfWeek.Year())
}

func getCurrentDateInUALocale(loc *time.Location) string {
	now := time.Now().In(loc)

	return dayNumToUADays[int(now.Weekday())] + ", " + strconv.Itoa(int(now.Day())) + " " + engMonthToUA[int(now.Month())] + " " + strconv.Itoa(now.Year())
}
