package main

import (
	"encoding/json"
	"fmt"
	"gShort/Config"
	"gShort/DataBase"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"github.com/someone-stole-my-name/reCAPTCHAv3"
	"strconv"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
)

// This is how are request to the backend looks like
type gShortPutRequest struct {
	Url   string `json:"url"`   // url to 'short'
	Token string `json:"token"` // captcha token (if active)
	Password string `json:"password"` // url password if set
	MaxHitCount int `json:"maxhitcount"`
}

// This is how are response to the backend looks like
type gShortGetResponse struct {
	Url     string `json:"url"`     // 'shorted' url
	Mapping string `json:"mapping"` // mapping is just the random string associated with that url
	Password string `json:"password"`
}

func main() {
	var (
		config *Config.Config //json config
		index  string         // templated index
		err    error
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

	if args.JustTemplate {
		_ = os.Mkdir("./_templates", 0770)

		f, err := os.Create("./_templates/index.html")
		if err != nil {
			log.Fatalln(err)
		}
		_, err = f.WriteString(index)
		if err != nil {
			fmt.Println(err)
		}
		err = f.Close()
		if err != nil {
			log.Fatalln(err)
		}

		return
	}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/short", func(w http.ResponseWriter, r *http.Request) {
		if !comingFromDomain(config.Domain, config.Port, r) { // make sure user is coming from configurated domain
			http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port), http.StatusMovedPermanently)
			return
		}

		gShortPut(config, w, r)
	}).Methods("POST")

	router.PathPrefix("/password/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if !comingFromDomain(config.Domain, config.Port, r) { // make sure user is coming from configurated domain
				http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port)+r.RequestURI, http.StatusMovedPermanently)
				return
			}

			box := rice.MustFindBox("website")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			password, _ := box.String("password.html")
			_, _ = fmt.Fprint(w, password)
			return
		}).Methods("GET")

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

	if len(a.Password) == 0 || a.MaxHitCount == 0 { // Password protected urls will return false bypassing the check, always create a new mapping
		mappingInDB, err := DataBase.FilterFromURL(config.MongoDB, a.Url)
		if err == nil {
			mapping := buildMapping(config, mappingInDB)
			resBody := gShortGetResponse{Url: a.Url, Mapping: mapping}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resBody)
			return
		}
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

	_, err = DataBase.Insert(config.MongoDB, a.Url, mapping, a.Password, a.MaxHitCount)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error writing to database: %v", err)
		return
	}

	// Prevent returning stuff like http://localhost/XXXX when port != 80
	mapped := buildMapping(config, mapping)

	resBody := gShortGetResponse{Url: a.Url, Mapping: mapped}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resBody)
}

func gShortGet(config *Config.Config, w http.ResponseWriter, r *http.Request) {
	mapping := trimLeftChar(r.RequestURI)
	log.Printf("Requested %v\n", mapping)
	var a gShortGetResponse

	b, p, err := DataBase.IsPasswordProtected(config.MongoDB, mapping)
	reqBody, _ := ioutil.ReadAll(r.Body)
	if len(reqBody) > 0 {
		err = json.Unmarshal(reqBody, &a)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if b && len(r.Header.Get("Key")) == 0 {
		log.Printf("Password protected mapping %v and no password provided, redirecting to password page.", mapping)
		http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port)+"/password/"+mapping, http.StatusFound)
		return
	}

	if r.Header.Get("Key") != p {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if b && r.Header.Get("Key") == p {
		mapsTo, err := DataBase.FilterFromMapping(config.MongoDB, mapping)
		w.Header().Set("Location", mapsTo)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port),
				http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		err = hitCounter(config, mapping)
		if err != nil {
			log.Printf("Error in hitCounter: %v", err)
		}
		return
	}

	mapsTo, err := DataBase.FilterFromMapping(config.MongoDB, mapping)
	if err != nil {
		log.Printf("Error: %v", err)
		http.Redirect(w, r, config.Protocol+"://"+config.Domain+":"+strconv.Itoa(config.Port),
			http.StatusFound)
		return
	}

	http.Redirect(w, r, mapsTo, http.StatusFound)
	err = hitCounter(config, mapping)
	if err != nil {
		log.Printf("Error in hitCounter: %v", err)
	}
	return
}

func ListenAndServe(config *Config.Config, router *mux.Router) {
	log.Printf("Using DB: %v\n", config.MongoDB.DataBase)
	log.Printf("Using Col: %v\n", config.MongoDB.Collection)
	// Since config.Port is used in many places ...
	port := os.Getenv("PORT") // heroku
	if port != "" {           // if env exists
		log.Printf("Listening on: %v", port)
		log.Fatal(http.ListenAndServe(":"+port, router))
	} else {
		log.Printf("Listening on: %v", config.Port)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), router))
	}
}
