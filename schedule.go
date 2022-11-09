package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
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
	streetName          string
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

func getStreetID(streetName string, streetIDUrl string) (streetID int, err error) {
	res, err := http.Get(streetIDUrl + url.QueryEscape(streetName))

	if err != nil {
		return 0, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Panic("Error: ", err)
		}
	}(res.Body)

	var streetData []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	err = json.NewDecoder(res.Body).Decode(&streetData)
	if err != nil {
		return 0, err
	}

	for _, data := range streetData {
		if data.Name == streetName {
			streetID = data.ID
		}
	}

	return streetID, nil
}

func requestGroupNumber(schedulerUrl string, addressData AddressData, streetIDUrl string) (groupsFromServer []GroupsFromServer, err error) {
	streetID, err := getStreetID(addressData.streetName, streetIDUrl)

	if err != nil {
		return nil, err
	}

	if streetID == 0 {
		return nil, errors.New("street id not found")
	}

	res, err := http.Get(getYasnoUrl(schedulerUrl, addressData.region, strconv.Itoa(streetID), addressData.houseNumberToSearch))

	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Panic("Error: ", err)
		}
	}(res.Body)

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
