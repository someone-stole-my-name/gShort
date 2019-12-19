# gShort

![Build](https://github.com/someone-stole-my-name/gShort/workflows/Build/badge.svg)
![Dependencies](https://img.shields.io/librariesio/github/someone-stole-my-name/gShort)
![License](https://img.shields.io/github/license/someone-stole-my-name/gShort?color=green)

![Screenshot](https://github.com/someone-stole-my-name/gShort/raw/master/Examples/Screenshot.png)

## Overview

gShort is a URL shortener that you can selfhost or easily deploy to a PaaS. ([Demo Site on Heroku][gshort_demo_site])

### Features

 * Password protected links
 * One Time Links

## Configuration

gShort requires a json configuration file, you can check the one that demo site uses [here][example_config].

#### General

 * **Domain**: The domain or IP where gShort will be accesed from. (**Required**)
 * **Port**: The port where gShort will listen for incoming requests. (**Required but can be overridden**)
 * **Protocol**: The protocol that users will use to access gShort. **This is not the protocol that gShort will use**, only HTTP is supported. Eg: If running on Heroku or behind a reverse proxy configured with SSL this should be HTTPS. (**Required**)
 * **SiteName**: HTML Title of your page. (**Required**)
 * **TagLine**: (**Required**)
  
#### MongoDB

 * **URI**: Format `mongodb+srv://$USER:$PASSWORD@cluster1-agata.mongodb.net/` (**Required but can be overridden**)
 * **DataBase**: MongoDB Database to use. (**Required but can be overridden**)
 * **Collection**: MongoDB Collection to use. (**Required but can be overridden**)

#### RandomStringGenerator

 * **Charset**: Charset used when generating short URLs. (**Required**)
 * **Length**: Length of the generated random strings. (**Required**)

#### ReCaptcha
 * **SiteKey**: Google's reCAPTCHAv3 Key, if you don't have one of theese just leave it as `""`.  (**Optional and can be overridden**)
 * **SecretKey**: Google's reCAPTCHAv3 Secret Key, if you don't have one of theese just leave it as `""` (**Optional and can be overridden**)

## Heroku (or other PaaS)

Deployment to Heroku should be pretty straightforward:
 * Fork
 * Modify the example `config.json` file
 * Set the following environment variables:
    ```
    MongoDB_Collection
    MongoDB_Database
    MongoDB_URI
    ReCaptcha_SecretKey
    ReCaptcha_SiteKey
    ```
 * Deploy master branch
 
 ## Getting Started (self Host)
 TODO

[gshort_demo_site]:https://gshort.christiansegundo.com
[example_config]:https://github.com/someone-stole-my-name/gShort/blob/master/config.json
