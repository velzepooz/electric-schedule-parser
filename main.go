package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"gopkg.in/robfig/cron.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type ScheduleHours struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type WeekSchedule [7][3]ScheduleHours

type Schedule struct {
	Schedule struct {
		Group1 WeekSchedule `json:"group_1"`
		Group2 WeekSchedule `json:"group_2"`
		Group3 WeekSchedule `json:"group_3"`
		Group4 WeekSchedule `json:"group_4"`
	} `json:"schedule"`
}

type AddressData struct {
	streetID            string
	houseNumber         string
	houseNumberToSearch string
	region              string
	who                 string
}

type GroupsFromServer struct {
	StreetId int    `json:"street_id"`
	Name     string `json:"name"`
	Group    int    `json:"group"`
}

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
		{streetID: os.Getenv("STREET_ID_ONE"), houseNumber: os.Getenv("HOUSE_NUMBER_ONE"), houseNumberToSearch: os.Getenv("HOUSE_NUMBER_TO_SEARCH_ONE"), region: os.Getenv("REGION_ONE"), who: os.Getenv("WHO_ONE")},
		{streetID: os.Getenv("STREET_ID_TWO"), houseNumber: os.Getenv("HOUSE_NUMBER_TWO"), houseNumberToSearch: os.Getenv("HOUSE_NUMBER_TO_SEARCH_TWO"), region: os.Getenv("REGION_TWO"), who: os.Getenv("WHO_TWO")},
	}

	startCroneJob("TZ=Europe/Kiev 10 0 * * *", func() {
		getGroupFromServerAndSendDayScheduleToTelegram(os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("CHAT_ID"), addressToSearch, os.Getenv("SCHEDULER_URL"))
	})

	log.Println("App is starting...")

	_, err := fmt.Scanln()
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

func startCroneJob(pattern string, cb func()) {
	cronJob := cron.New()

	_, err := cronJob.AddFunc(pattern, cb)
	if err != nil {
		return
	}

	cronJob.Start()
}

func getGroupFromServerAndSendDayScheduleToTelegram(botToken string, chatID string, addressToSearch [2]AddressData, schedulerUrl string) {
	log.Println("Sending schedule")
	loc, err := time.LoadLocation("Europe/Kiev")
	if err != nil {
		log.Panic(err)
	}

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
		groupsFromServer, err := requestGroupNumber(schedulerUrl, address)

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

	err = sendDataToTelegram(botToken, chatID, telegramMessage)
	if err != nil {
		log.Panic(err)
	}

	log.Println("Schedule sent")
}

func getCurrentDateInUALocale(loc *time.Location) string {
	now := time.Now().In(loc)

	return dayNumToUADays[int(now.Weekday())] + ", " + strconv.Itoa(int(now.Day())) + " " + engMonthToUA[int(now.Month())] + " " + strconv.Itoa(now.Year())
}

func loadScheduleData() Schedule {
	content, err := ioutil.ReadFile("./data/schedule.json")

	var payload Schedule
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	return payload
}

func requestGroupNumber(schedulerUrl string, addressData AddressData) (groupsFromServer []GroupsFromServer, err error) {
	res, err := http.Get(getYasnoUrl(schedulerUrl, addressData.region, addressData.streetID, addressData.houseNumberToSearch))

	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Panic("Error: ", err)
		}
	}(res.Body)

	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&groupsFromServer)
	if err != nil {
		return nil, err
	}

	return groupsFromServer, nil
}

func getYasnoUrl(mainUrl string, region string, streetID string, homeNumber string) string {
	return mainUrl + "?region=" + region + "&street_id=" + streetID + "&query=" + homeNumber
}

func getGroupNumber(houseNumber string, groups []GroupsFromServer) int {
	var group int

	for _, groupData := range groups {
		if groupData.Name == houseNumber {
			group = groupData.Group
		}
	}

	return group
}

func getScheduleInfo(groupNumber int, schedule *Schedule) WeekSchedule {
	switch groupNumber {
	case 1:
		return schedule.Schedule.Group1
	case 2:
		return schedule.Schedule.Group2
	case 3:
		return schedule.Schedule.Group3
	case 4:
		return schedule.Schedule.Group4
	}

	return schedule.Schedule.Group1
}

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
