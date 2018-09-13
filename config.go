package roll

import (
	"encoding/json"
	"io/ioutil"
)

// Config is the bot's configuration
type Config struct {
	BotUsername string `json:"bot_username"`
	Channel     string `json:"channel"`
	APIOAuth    string `json:"api_oauth"`
	IRCOAuth    string `json:"irc_oauth"`
	IRCAddress  string `json:"irc_addr"`
	APIURLBase  string `json:"api_url_base"`
	AdminUser   string `json:"admin_user"`

	ClientID string `json:"client_id"`

	HTTPAddr         string `json:"http_addr"`
	HTTPSAddr        string `json:"https_addr"`
	HTTPRedirectBase string `json:"http_redirect_base"`
	KeyFile          string `json:"key_file"`
	CertFile         string `json:"cert_file"`
}

func LoadConfig(fileName string) (*Config, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
