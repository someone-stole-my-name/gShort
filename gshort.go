package main

import (
	"gShort/Config"
	"gShort/DataBase"
	"gShort/reCAPTCHAv3"
	"encoding/json"
	"fmt"
	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

// This is how are request to the backend looks like
type gShortPutRequest struct {
	Url         string `json:"url"`		// url to 'short'
	Token		string `json:"token"`	// captcha token (if active)
}

// This is how are response to the backend looks like
type gShortGetResponse struct {
	Url 	string `json:"url"`	// 'shorted' url
	Mapping string `json:"mapping"` // mapping is just the random string associated with that url
}

func main() {
	var (
		config *Config.Config //json config
		index string // templated index
		err error
	)

	args := *Config.ParseArgs()
	config, err = config.LoadConfigFrom(args.ConfigFile)
	if err != nil {
		log.Fatalln(err)
	}

	index, err = buildIndex(config)
	if err != nil {
		log.Fatalln(err)
	}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/short", func(w http.ResponseWriter, r *http.Request) {
		if !comingFromDomain(config.Domain, config.Port, r) { // make sure user is coming from configurated domain
			http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port), http.StatusMovedPermanently)
			return
		}

		gShortPut(config, w, r)
	}).Methods("POST")


	router.PathPrefix("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// index request
			if !comingFromDomain(config.Domain, config.Port, r) { // make sure user is coming from configurated domain
				http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port), http.StatusMovedPermanently)
				return
			}

			if r.RequestURI == "/" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = fmt.Fprint(w, index)
				return
			}

			// requesting something else?
			box := rice.MustFindBox("website")
			if boxHasFile(box, r.RequestURI) {
				http.FileServer(box.HTTPBox()).ServeHTTP(w, r)
			} else { // if requested file is not in box try to redirect
				gShortGet(config, w, r) // it will redirect to homepage if not found in db
			}
		}).Methods("GET")

		// CORS Headers
	router.PathPrefix("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if !comingFromDomain(config.Domain, config.Port, r) { // make sure user is coming from configurated domain
				http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port), http.StatusMovedPermanently)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", config.Protocol+"://"+config.Domain)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		}).Methods("OPTIONS")

	ListenAndServe(config, router)
}

func gShortPut(config *Config.Config, w http.ResponseWriter, r *http.Request) {
	var a gShortPutRequest

	reqBody, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(reqBody, &a)
	if err != nil || len(a.Url) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !isValidUrl(a.Url) {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Bad URL")
		return
	}

	// If there are reCaptcha keys in the config we check:
	// if returned token from frontend is valid
	// if returned hostname matches the domain in config file
	if len(config.ReCaptcha.SecretKey) > 0 && len(config.ReCaptcha.SiteKey) > 0 {
		err = reCAPTCHAv3.ValidateReCaptcha(config.ReCaptcha.SecretKey, a.Token, config.Domain)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("Invalid reCaptcha: %v\n", err)
			return
		}
	}

	mappingInDB, err := DataBase.FilterFromURL(config.MongoDB, a.Url)
	if err == nil {
		mapping := buildMapping(config, mappingInDB)
		resBody := gShortGetResponse{Url:a.Url, Mapping:mapping}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resBody)
		return
	}

	// Generate a new random string
	mapping := generateStringWithCharset(config.RandomStringGenerator.Length, config.RandomStringGenerator.Charset)
	for {
		_, err = DataBase.FilterFromMapping(config.MongoDB, mapping) // Check if that string is already is DB
		if err != nil {
			break // If it is not break the loop
		}
		mapping = generateStringWithCharset(config.RandomStringGenerator.Length, config.RandomStringGenerator.Charset)
	}

	_, err = DataBase.Insert(config.MongoDB, a.Url, mapping)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error writing to database: %v", err)
		return
	}

	// Prevent returning stuff like http://localhost/XXXX when port != 80
	mapped := buildMapping(config, mapping)

	resBody := gShortGetResponse{Url:a.Url, Mapping:mapped}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resBody)
}

func gShortGet(config *Config.Config, w http.ResponseWriter, r *http.Request) {
	mapping := trimLeftChar(r.RequestURI)
	mapsTo, err := DataBase.FilterFromMapping(config.MongoDB, mapping)
	if err != nil {
		log.Printf("Error: %v", err)
		http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port),
			http.StatusMovedPermanently)
		return
	}

	http.Redirect(w, r, mapsTo, http.StatusMovedPermanently)
	return
}

func ListenAndServe(config *Config.Config, router *mux.Router){
	log.Printf("Using DB: %v\nUsing Col: %v\n", config.MongoDB.DataBase, config.MongoDB.Collection)
	// Since config.Port is used in many places ...
	port := os.Getenv("PORT") // heroku
	if port != "" { // if env exists
		log.Printf("Listening on: %v", port)
		log.Fatal(http.ListenAndServe(":"+port, router))
	} else {
		log.Printf("Listening on: %v", config.Port)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), router))
	}
}
