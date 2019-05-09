package roll

import "testing"

func TestWiki(t *testing.T) {
	bot, _ := newConnectedTestBot(t)
	client := getTestHttpClient()

	url := "https://" + bot.Config.HTTPSAddr + "/wiki/test.md"
	_, err := client.Get(url)
	if err != nil {
		t.Errorf("Got error getting %s: %v", url, err)
	}
}
