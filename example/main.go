package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"time"

	"mvdan.cc/xurls/v2"

	"golang.org/x/net/html"

	ts3plugin "github.com/k-przybyszewski/go-ts3plugin"
	"github.com/k-przybyszewski/go-ts3plugin/teamlog"
	"github.com/k-przybyszewski/go-ts3plugin/teamspeak"
)

const (
	Name    = "Test Go Plugin"
	Author  = "Carl Kittelberger"
	Version = "0.0.0"
)

func catchPanic() {
	if err := recover(); err != nil {
		if ts3plugin.Functions() != nil {
			log(fmt.Sprintf("%s\n%s",
				err,
				string(debug.Stack())), teamlog.Critical)
		}
	}
}

func log(msg string, severity teamlog.LogLevel) {
	ts3plugin.Functions().LogMessage(
		msg,
		severity, Name, 0)
}

func isTitleElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "title"
}

func traverse(n *html.Node) (string, bool) {
	if isTitleElement(n) {
		return n.FirstChild.Data, true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := traverse(c)
		if ok {
			return result, ok
		}
	}

	return "", false
}

func GetHtmlTitle(r io.Reader) (string, bool) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", false
	}

	return traverse(doc)
}

var bbTagsRegex = regexp.MustCompile(`\[/?[A-Za-z0-9_]+\]`)

func stripBBTags(text string) string {
	return bbTagsRegex.ReplaceAllString(text, "")
}

func init() {
	ts3plugin.Name = Name
	ts3plugin.Author = Author
	ts3plugin.Version = Version

	ts3plugin.OnTextMessageEvent = func(
		serverConnectionHandlerID uint64,
		targetMode teamspeak.AnyID,
		toID teamspeak.AnyID,
		fromID teamspeak.AnyID,
		fromName string,
		fromUniqueIdentifier string,
		message string,
		ffIgnored bool) int {
		// If there is a URL to an HTML page, show its title.
		// Does this contain a URL?
		u := xurls.Strict().FindString(stripBBTags(message))
		if len(u) > 0 {
			resp, err := http.Get(u)
			if err != nil {
				log(err.Error(), teamlog.Warning)
				return 0
			}
			defer resp.Body.Close()

			if title, ok := GetHtmlTitle(resp.Body); ok {
				ts3plugin.Functions().PrintMessageToCurrentTab(fmt.Sprintf("[B]Title:[/B] %s", title))
			} else {
				log(fmt.Sprintf("Failed to get HTML title from %s", u), teamlog.Warning)
			}
		}

		return 0
	}

	ts3plugin.OnUserLoggingMessageEvent = func(logMessage string, logLevel teamlog.LogLevel, logChannel string, logID uint64, logTime time.Time, completeLogString string) {
		// Print all log messages to the current chat tab
		ts3plugin.Functions().PrintMessageToCurrentTab(
			fmt.Sprintf("[COLOR=gray][I]%s[/I]\t%s\t[B]%s[/B]\t%s[/COLOR]",
				logTime,
				logLevel,
				logChannel,
				logMessage))
	}

	ts3plugin.Init = func() (ok bool) {
		defer catchPanic()

		version, errCode := ts3plugin.Functions().GetClientLibVersion()
		if errCode == teamspeak.ErrorOK {
			log(
				fmt.Sprintf("TS3::Init - plugin ID %s running on %s!",
					ts3plugin.GetPluginID(),
					version),
				teamlog.Debug)
			ok = true
		} else {
			msg := ""
			if errMsg, errCode2 := ts3plugin.Functions().GetErrorMessage(errCode); errCode2 == 0 {
				msg = errMsg
			} else {
				msg = fmt.Sprintf("Error code %d", errCode)
			}
			log(
				fmt.Sprintf("Could not get client lib version: %s",
					msg),
				teamlog.Critical)
		}

		return
	}

	ts3plugin.Shutdown = func() {
		defer catchPanic()
		ts3plugin.Functions().LogMessage("TS3::Shutdown", teamlog.Debug, Name, 0)
	}
}

// This will never be run!
func main() {
	fmt.Println("=======================================")
	fmt.Println("This is a TeamSpeak3 plugin, do not run this as a CLI application!")
	fmt.Println("Args were: ", os.Args)
	fmt.Println("=======================================")
}
