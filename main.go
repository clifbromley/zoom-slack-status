package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	icon "github.com/caitlinelfring/zoom-slack-status/icons"
	"github.com/fsnotify/fsnotify"

	"github.com/getlantern/systray"
	homedir "github.com/mitchellh/go-homedir"
	ps "github.com/mitchellh/go-ps"
	"github.com/spf13/viper"
)

type SlackStatus struct {
	StatusText  string `json:"status_text"`
	StatusEmoji string `json:"status_emoji"`
}

type slackAccount struct {
	Name            string
	Token           string
	MeetingStatus   *SlackStatus
	NoMeetingStatus *SlackStatus
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
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName(".zoom-slack-status")
	viper.SetDefault("interval", 60*time.Second)

	loadInConfig()

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("Config file changed: %s, operation: %s\n", e.Name, e.Op)
		loadInConfig()
	})

	systray.Run(onReady, onExit)
}

func loadInConfig() {
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	if err := viper.UnmarshalKey("accounts", &slackAccounts); err != nil {
		panic(err)
	}
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

		time.Sleep(viper.GetDuration("interval"))
	}
}

func onExit() {
	setInMeeting(false)
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

// sliceContains checks to see if string s is in the slice
func sliceContains(s string, slice []string) bool {
	for _, sl := range slice {
		if strings.Contains(sl, s) {
			return true
		}
	}
	return false
}
