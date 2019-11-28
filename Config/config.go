package Config

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
)

type Args struct {
	ConfigFile string
}

type Config struct {
	MongoDB            *MongoDB
	RandomStringGenerator *RandomStringGenerator
	Domain 	string `json:"Domain"`
	Protocol string `json:"Protocol"`
	ReCaptcha *ReCaptcha
	SiteName string `json:"SiteName"`
	TagLine string `json:"TagLine"`
	Port int `json:"Port"`
}

type ReCaptcha struct {
	SiteKey string `json:"SiteKey"`
	SecretKey string `json:"SecretKey"`
}

type MongoDB struct {
	URI        string `json:"URI"`
	DataBase   string `json:"DataBase"`
	Collection string `json:"Collection"`
}

type RandomStringGenerator struct {
	Length	int `json:"Length"`
	Charset string `json:"Charset"`
}

func ParseArgs() *Args {
	a := &Args{}
	flag.StringVar(&a.ConfigFile, "config", "", "JSON Config File")
	flag.Parse()
	return a
}

func (*Config) LoadConfigFrom(file string) (config *Config, err error) {
	ConfigFileByteArr, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	if err = json.Unmarshal(ConfigFileByteArr, &config); err != nil {
		return
	}
	config.checkENV()
	return
}

// Override some settings using ENV (heroku)
func (config *Config) checkENV () *Config {
	// TODO REWORK THIS CRAP
	i := os.Getenv("ReCaptcha_SiteKey") // heroku
	if i != "" { // if env exists
		config.ReCaptcha.SiteKey = i
	}

	i = os.Getenv("ReCaptcha_SecretKey") // heroku
	if i != "" { // if env exists
		config.ReCaptcha.SecretKey = i
	}

	i = os.Getenv("MongoDB_URI") // heroku
	if i != "" { // if env exists
		config.MongoDB.URI = i
	}

	i = os.Getenv("MongoDB_DataBase") // heroku
	if i != "" { // if env exists
		config.MongoDB.DataBase = i
	}

	i = os.Getenv("MongoDB_Collection") // heroku
	if i != "" { // if env exists
		config.MongoDB.Collection = i
	}

	return config
}