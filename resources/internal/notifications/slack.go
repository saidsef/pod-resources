package notifications

import (
	"github.com/saidsef/pod-resources/resources/utils"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var (
	slackToken   = utils.GetEnv("SLACK_TOKEN", "", utils.Logger())
	slackChannel = utils.GetEnv("SLACK_CHANNEL", "k8s-alerts", utils.Logger())
)

// NewSlackClient creates a new Slack client
func NewSlackClient() *slack.Client {
	return slack.New(slackToken)
}

// SlackEnabled checks if Slack notifications are enabled
func SlackEnabled() bool {
	return slackToken != "" && slackChannel != ""
}

// SendSlackNotification sends a message to a Slack channel
func SendSlackNotification(api *slack.Client, message string) {
	_, _, err := api.PostMessage(
		slackChannel,
		slack.MsgOptionText(message, false),
	)
	if err != nil {
		utils.LogWithFields(logrus.ErrorLevel, []string{}, "Failed to send Slack notification", err)
	}
}
