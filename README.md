# Slack Status Manager

Sets status of slack workspaces based on whether or not a zoom call is open.

## Configuration & Installation

First, get a user token from slack [here](https://api.slack.com/custom-integrations/legacy-tokens)

### Configuration

Create a JSON, TOML, YAML, HCL, envfile or Java properties config file, in either your $HOME directory or the directory you're running the compiled go binary in (ie repo root). The file should be prefixed with `.zoom-slack-status`. For example, `~/.zoom-slack-status.yml` if you're creating a YAML file. (See [spf13/viper](https://github.com/spf13/viper) for more details)

```yaml
accounts:
  - name: My slack workspace
    token: xoxp-123456890abcdefghijklmnopqrstuvwxyz
    # Optional
    # meetingStatus:
    #   status_text: "In a meeting"
    #   status_emoji: ":zoom:"
    # noMeetingStatus:
    #   status_text: "I'm available"
    #   status_emoji: ":facepalm:"

# interval for how often to check if a Zoom meeting is in progress (default: 60s)
interval: "60s"
```

### Run

Run with the go toolchain with `go run main.go`. Or `go build . && ./zoom-slack-status`.

### Build "Mac App"

1. Build the app with `./build-app.sh`
2. Launch the app from `/Applications` in your Finder
