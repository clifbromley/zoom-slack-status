package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caitlinelfring/zoom-slack-status/icons"

	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
	homedir "github.com/mitchellh/go-homedir"
	ps "github.com/mitchellh/go-ps"
	"github.com/spf13/viper"
)

type SlackStatus struct {
	StatusText  string `mapstructure:"status_text" json:"status_text"`
	StatusEmoji string `mapstructure:"status_emoji" json:"status_emoji"`
}

type Account struct {
	Name            string       `mapstructure:"name"`
	Token           string       `mapstructure:"token"`
	MeetingStatus   *SlackStatus `mapstructure:"meetingStatus"`
	NoMeetingStatus *SlackStatus `mapstructure:"noMeetingStatus"`
}

type Config struct {
	Accounts []Account     `mapstructure:"accounts"`
	Interval time.Duration `mapstructure:"interval"`
}

type SlackResponse struct {
	Ok      bool   `json:"ok"`
	Error   string `json:"error"`
	Warning string `json:"warning"`
	// Other fields ignored:
	// 		response_metadata
	//		profile
}

var (
	defaultMeetingStatus = SlackStatus{
		StatusText:  "In a meeting",
		StatusEmoji: ":zoom:",
	}
	defaultNoMeetingStatus               = SlackStatus{}
	defaultInterval        time.Duration = 60 * time.Second

	config        = Config{}
	configChanged = false
)

// Receiver functions for outputting Config and Account structures as strings.
// Custom handling is necessary to output the contents of structs embedded via pointers.
func (c Config) String() string {
	return fmt.Sprintf("{Accounts:%v Interval:%v}", c.Accounts, c.Interval)
}

func (a Account) String() string {
	return fmt.Sprintf("{Name:%v Token:%v MeetingStatus:%+v NoMeetingStatus:%+v}", a.Name, a.Token, a.MeetingStatus, a.NoMeetingStatus)
}

func main() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigName(".zoom-slack-status")

	viper.SetDefault("interval", defaultInterval)

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
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	// Set default status values if not configured.
	for i := range cfg.Accounts {
		if cfg.Accounts[i].MeetingStatus == nil {
			cfg.Accounts[i].MeetingStatus = &defaultMeetingStatus
		}
		if cfg.Accounts[i].NoMeetingStatus == nil {
			cfg.Accounts[i].NoMeetingStatus = &defaultNoMeetingStatus
		}
	}

	// Update global configuration.
	config = cfg
	configChanged = true

	fmt.Printf("Configuration loaded:\n%v\n\n", config)
}

func onReady() {
	systray.SetTooltip("Zoom Status")
	systray.SetIcon(icons.Free)

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
		inMeeting = checkForMeeting()

		if inMeeting {
			if !wasInMeeting || configChanged {
				setInMeeting(true)
			} else {
				fmt.Println("Status already set to in meeting")
			}
		} else {
			if wasInMeeting || configChanged {
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

		configChanged = false

		time.Sleep(config.Interval)
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

	if inMeeting {
		systray.SetIcon(icons.Busy)
	} else {
		systray.SetIcon(icons.Free)
	}

	// Set status for all accounts
	for _, account := range config.Accounts {
		fmt.Println("Setting slack status for " + account.Name)

		var status *SlackStatus

		if inMeeting {
			status = account.MeetingStatus
		} else {
			status = account.NoMeetingStatus
		}

		if err := setSlackProfile(*status, account.Token); err != nil {
			fmt.Printf("Failed to set slack profile for %s: %s\n", account.Name, err.Error())
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
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)

	r := SlackResponse{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return err
	}

	if len(r.Warning) > 0 {
		fmt.Println("Warning: ", r.Warning)
	}

	if !r.Ok {
		err = fmt.Errorf(r.Error)
	}

	return err
}
