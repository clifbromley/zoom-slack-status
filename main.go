package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	icon "github.com/caitlinelfring/zoom-slack-status/icons"

	"github.com/getlantern/systray"
	homedir "github.com/mitchellh/go-homedir"
	ps "github.com/mitchellh/go-ps"
)

type SlackStatus struct {
	StatusText  string `json:"status_text"`
	StatusEmoji string `json:"status_emoji"`
}

type slackAccount struct {
	Name            string       `json:"name"`
	Token           string       `json:"token"`
	MeetingStatus   *SlackStatus `json:"meetingStatus,omitempty"`
	NoMeetingStatus *SlackStatus `json:"noMeetingStatus,omitempty"`
}

var (
	slackAccounts []slackAccount

	defaultNoMeetingStatus = SlackStatus{}
	defaultMeetingStatus   = SlackStatus{
		StatusText:  "In a meeting",
		StatusEmoji: ":zoom:",
	}
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTooltip("Zoom Status")
	systray.SetIcon(icon.Data)

	menuStatus := systray.AddMenuItem("Status: Not In Meeting", "Not In Meeting")
	menuStatus.Disable()

	systray.AddSeparator()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit Zoom Status", "Quit Zoom Status")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
		os.Exit(0)
	}()

	err := loadConfig()
	if err != nil {
		panic(err)
	}

	inMeeting := false

	for {
		wasInMeeting := inMeeting
		inMeeting := checkForMeeting()

		if inMeeting {
			if !wasInMeeting {
				setInMeeting(true)
			} else {
				fmt.Println("Status already set to in meeting")
			}
		} else {
			if wasInMeeting {
				setInMeeting(false)
			} else {
				fmt.Println("Status already set to not in meeting")
			}
		}

		if inMeeting {
			menuStatus.SetTitle("Status: In Meeting")
		} else {
			menuStatus.SetTitle("Status: Not In Meeting")
		}

		time.Sleep(60 * time.Second)
	}
}

func onExit() {
	setInMeeting(false)
}

func loadConfig() error {
	fmt.Println("Loading Config...")
	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	jsonFile, err := os.Open(filepath.Join(home, ".slack-status-config.json"))
	if err != nil {
		return err
	}

	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(byteValue, &slackAccounts)
}

func checkForMeeting() bool {
	fmt.Println("Checking for active meetings...")

	processes, err := ps.Processes()
	if err != nil {
		fmt.Printf("Could not get running process list: %s\n", err)
		return false
	}
	for _, proc := range processes {
		// NOTE: This is the process that is running when a zoom meeting is
		// in progress on Mac. It might not be the same for other systems
		if strings.ToLower(proc.Executable()) == "cpthost" {
			return true
		}
	}
	return false
}

func setInMeeting(inMeeting bool) {
	fmt.Printf("Setting status to in meeting: %t\n", inMeeting)

	// Set status for all accounts
	for _, account := range slackAccounts {
		fmt.Println("Setting slack status for " + account.Name)

		var status SlackStatus

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

		if err := setSlackProfile(status, account.Token); err != nil {
			fmt.Printf("Failed to set slack profile for %s: %s", account.Name, err.Error())
		}
	}
}

func setSlackProfile(status SlackStatus, token string) error {
	var profile = struct {
		Profile SlackStatus `json:"profile"`
	}{status}

	statusBytes, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/users.profile.set", bytes.NewBuffer(statusBytes))
	if err != nil {
		return err
	}

	// Add proper headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// TODO: human-friendly output
	fmt.Println("response Status:", resp.Status)
	_, _ = io.Copy(os.Stdout, resp.Body)
	return err
}

// deleteEmpty Removes empty strings from a slice of strings
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

// sliceContains checks to see if string s is in the slice
func sliceContains(s string, slice []string) bool {
	for _, sl := range slice {
		if strings.Contains(sl, s) {
			return true
		}
	}
	return false
}
