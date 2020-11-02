package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andybrewer/mack"
	icon "github.com/caitlinelfring/zoom-slack-status/icons"
	"github.com/getlantern/systray"
	homedir "github.com/mitchellh/go-homedir"
)

type status struct {
	StatusText  string `json:"status_text"`
	StatusEmoji string `json:"status_emoji"`
}

type slackAccount struct {
	Name            string  `json:"name"`
	Token           string  `json:"token"`
	MeetingStatus   *status `json:"meetingStatus,omitempty"`
	NoMeetingStatus *status `json:"noMeetingStatus,omitempty"`
}

type slackProfile struct {
	Profile status `json:"profile"`
}

var (
	activeMeetingWindowName = "Zoom Meeting"

	slackAccounts []slackAccount

	inMeeting  = false
	menuStatus *systray.MenuItem

	defaultMeetingStatus = status{
		StatusText:  "In A Meeting",
		StatusEmoji: ":zoom:",
	}

	defaultNoMeetingStatus = status{
		StatusText:  "",
		StatusEmoji: "",
	}
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTooltip("Zoom Status")
	systray.SetIcon(icon.Data)

	menuStatus = systray.AddMenuItem("Status: Not In Meeting", "Not In Meeting")
	menuStatus.Disable()

	systray.AddSeparator()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit Zoom Status", "Quit Zoom Status")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()

	loadConfig()

	for {
		inMeetingNow := checkForMeeting()

		if inMeetingNow {
			if !inMeeting {
				setInMeeting(true)
			} else {
				fmt.Println("Status already set to in meeting")
			}
		} else {
			if inMeeting {
				setInMeeting(false)
			} else {
				fmt.Printf("Status already set to not in meeting \n")
			}
		}

		time.Sleep(60 * time.Second)
	}
}

func onExit() {
	setInMeeting(false)
}

func loadConfig() {
	fmt.Println("Loading Config...")
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	jsonFile, err := os.Open(filepath.Join(home, ".slack-status-config.json"))
	if err != nil {
		panic(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	_ = json.Unmarshal(byteValue, &slackAccounts)
}

func checkForMeeting() bool {
	fmt.Println("Checking for active meetings...")
	result, err := mack.Tell("System Events", "get the title of every window of every process")
	if err != nil {
		panic(err)
	}

	apps := deleteEmpty(strings.Split(result, ","))
	return hasZoomMeetingWindow(apps)

}

func hasZoomMeetingWindow(apps []string) bool {
	for _, app := range apps {
		if strings.Contains(app, activeMeetingWindowName) {
			return true
		}
	}

	return false
}

func setInMeeting(_inMeeting bool) {
	fmt.Printf("Setting status to in meeting: %t\n", inMeeting)
	inMeeting = _inMeeting

	// Set status for all accounts
	for _, account := range slackAccounts {
		fmt.Println("Setting slack status for " + account.Name)

		var status status

		if inMeeting {
			if account.MeetingStatus != nil {
				status = *account.MeetingStatus
			} else {
				status = defaultMeetingStatus
			}
		} else {
			if account.NoMeetingStatus != nil {
				status = *account.NoMeetingStatus
			} else {
				status = defaultNoMeetingStatus
			}
		}

		setSlackProfile(slackProfile{status}, account.Token)
	}
	if inMeeting {
		menuStatus.SetTitle("Status: In Meeting")
	} else {
		menuStatus.SetTitle("Status: Not In Meeting")
	}
}

func setSlackProfile(profile slackProfile, token string) {
	statusBytes, err := json.Marshal(profile)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/users.profile.set", bytes.NewBuffer(statusBytes))
	if err != nil {
		panic(err)
	}

	// Add proper headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)

	_, _ = http.DefaultClient.Do(req)
}

/*
Removes empty strings from a slice of strings
*/
func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		str = strings.TrimSpace(str)
		if len(str) > 0 {
			r = append(r, str)
		}
	}
	return r
}
