package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
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

type GetScheduleAndSendToTelegramParams struct {
	startMessage           string
	endMessage             string
	telegramMessage        string
	prepareTelegramMessage func(weekSchedule WeekSchedule, who string, loc *time.Location) (telegramMessage string)
}

type CronJobConfig struct {
	pattern string
	handler func()
}

func main() {
	loadEnv()

	appConfig := GetConfig()
	cronJobsConfigs := [2]CronJobConfig{
		{
			pattern: "TZ=Europe/Kiev 05 0 * * 1",
			handler: func() {
				weeklyParams := GetScheduleAndSendToTelegramParams{
					startMessage:    "Sending week schedule",
					endMessage:      "Week schedule sent",
					telegramMessage: fmt.Sprintf("График выключений электроэнергии на неделю, %v\n", getCurrentWeekPeriodInUALocale(appConfig.locale)),
					prepareTelegramMessage: func(weekSchedule WeekSchedule, who string, loc *time.Location) (telegramMessage string) {
						telegramMessage += "\n" + who + ":\n"

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
						return telegramMessage
					},
				}

				getScheduleAndSendToTelegram(appConfig, weeklyParams)
			},
		},
		{
			//pattern: "TZ=Europe/Kiev 10 0 * * *",
			pattern: "@every 30s",

			handler: func() {
				dailyParams := GetScheduleAndSendToTelegramParams{
					startMessage:    "Sending day schedule",
					endMessage:      "Day schedule sent",
					telegramMessage: fmt.Sprintf("График выключений электроэнергии на сегодня, %v:\n", getCurrentDateInUALocale(appConfig.locale)),
					prepareTelegramMessage: func(weekSchedule WeekSchedule, who string, loc *time.Location) (telegramMessage string) {
						todayDayNumberAtWeek := time.Now().In(loc).Weekday()

						/* make day number like at UA */
						if todayDayNumberAtWeek == 0 {
							todayDayNumberAtWeek = 6
						} else {
							todayDayNumberAtWeek--
						}

						todaySchedule := weekSchedule[todayDayNumberAtWeek]

						telegramMessage += "\n" + who + ":\n"

						for _, period := range todaySchedule {
							telegramMessage += "C " + strconv.Itoa(period.Start) + " до " + strconv.Itoa(period.End) + "\n"
						}

						return telegramMessage
					},
				}

				getScheduleAndSendToTelegram(appConfig, dailyParams)
			},
		},
	}

	initCroneJobs(cronJobsConfigs)

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

func initCroneJobs(cronJobsConfigs [2]CronJobConfig) {
	for _, croneJobConfig := range cronJobsConfigs {
		startCroneJob(croneJobConfig.pattern, croneJobConfig.handler)
	}
}

func getScheduleAndSendToTelegram(config Config, params GetScheduleAndSendToTelegramParams) {
	log.Println(params.startMessage)

	schedule := loadScheduleData()
	telegramMessage := params.telegramMessage

	for _, address := range config.addressesToSearch {
		groupsFromServer, err := requestGroupNumber(config.schedulerUrl, address, config.streetIDUrl)

		if err != nil {
			log.Panic(err)
		}

		group := getGroupNumber(address.houseNumber, groupsFromServer)

		if group == 0 {
			log.Fatal("Group not found")
		}

		weekSchedule := getScheduleInfo(group, &schedule)

		telegramMessage += params.prepareTelegramMessage(weekSchedule, address.who, config.locale)
	}

	err := sendDataToTelegram(config.botToken, config.chatID, telegramMessage)
	if err != nil {
		log.Panic(err)
	}

	log.Println(params.endMessage)
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
