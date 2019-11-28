package main

import (
	"bytes"
	rice "github.com/GeertJohan/go.rice"
	"GoShort/Config"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
	"time"
)

// Checks whether a box has a specific file
func boxHasFile(box *rice.Box, file string) bool {
	if file == "/" {
		return true
	}
	oo, err := box.Open(file)
	if err != nil {
		return false
	}
	oo.Close()
	return true
}

func isValidUrl(rawurl string) bool {
	_, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return false
	}
	return true
}

// Generates a random string with the given length and charset
func generateStringWithCharset(length int, charset string) string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func trimLeftChar(s string) string {
	for i := range s {
		if i > 0 {
			return s[i:]
		}
	}
	return s[:0]
}

func buildIndex(config *Config.Config) (index string, err error) {
	box, err := rice.FindBox("website")
	if err != nil {
		return
	}
	tmpl, err := box.String("index.html")
	if err != nil {
		return
	}
	var indexTemplate *template.Template
	indexTemplate, err = template.New("index").Parse(tmpl)
	if err != nil {
		return
	}
	var tpl bytes.Buffer
	err = indexTemplate.Execute(&tpl, config)
	if err != nil {
		return
	}
	index = tpl.String()
	return
}

// Prevent returning stuff like http://localhost/XXXX when port != 80
func buildMapping(config *Config.Config, mapping string) string {
	if config.Protocol == "http" && config.Port != 80 {
		mapping = "http://" + config.Domain + ":" + strconv.Itoa(config.Port) + "/" + mapping
	} else if config.Protocol == "https" && config.Port != 443 {
		mapping = "https://" + config.Domain + ":" + strconv.Itoa(config.Port) + "/" + mapping
	} else {
		mapping = config.Protocol + "://" + config.Domain + "/" + mapping
	}
	return mapping
}

// This function check if the host header matches the provided domain:port
// For requests on port 80 or 443 it won't check the port
func comingFromDomain(domain string, port int, r *http.Request) bool {
	if port != 80 && port != 443 {
		if r.Host == domain+":"+strconv.Itoa(port) {
			return true
		}
	} else {
		if r.Host == domain {
			return true
		}
	}
	return false
}