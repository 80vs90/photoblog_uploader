package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"net/http"
	"os"
)

type BlogConfig struct {
	url      string
	password string
}

type PhotoResponse struct {
	Id int64
}

func ParseConfig(config_path string) (error, BlogConfig) {
	config, err := ini.Load(config_path)
	if err != nil {
		return err, BlogConfig{}
	}

	main_section, err := config.GetSection("")
	if err != nil {
		return err, BlogConfig{}
	}

	password := main_section.Key("PASSWORD").Value()
	url := main_section.Key("URL").Value()

	return nil, BlogConfig{url: url, password: password}
}

func Authenticate(config BlogConfig) (error, string) {
	client := &http.Client{}

	auth_json, err := json.Marshal(map[string]string{"password": config.password})
	if err != nil {
		return err, ""
	}
	auth_request, err := http.NewRequest("POST", config.url+"/api/authenticate", bytes.NewBuffer(auth_json))
	auth_request.Header.Set("Content-Type", "application/json")
	auth_token, err := client.Do(auth_request)
	if err != nil {
		return err, ""
	}

	if auth_token.StatusCode != 200 {
		return errors.New("Authentication request returned status code: " + auth_token.Status), ""
	}

	auth_token_buf := new(bytes.Buffer)
	auth_token_buf.ReadFrom(auth_token.Body)
	return nil, auth_token_buf.String()
}

func UploadPhoto(photo_path, title, description string, config BlogConfig) error {
	err, auth_token := Authenticate(config)
	if err != nil {
		return err
	}

	client := &http.Client{}

	image, err := ioutil.ReadFile(photo_path)
	if err != nil {
		return err
	}

	new_photo_request, err := http.NewRequest("POST", config.url+"/api/photos", bytes.NewBuffer(image))
	new_photo_request.Header.Set("X-Auth-Token", auth_token)
	new_photo_request.Header.Set("Content-Type", "image/jpeg")
	new_photo_response, err := client.Do(new_photo_request)
	if err != nil {
		return err
	}

	if new_photo_response.StatusCode != 200 {
		return errors.New("Error POSTing new photo: " + new_photo_response.Status)
	}

	// THIS NEEDS TO PARSE THE JSON RESPONSE
	new_photo_info := new(bytes.Buffer)
	new_photo_info.ReadFrom(new_photo_response.Body)
	var new_photo PhotoResponse
	err = json.Unmarshal(new_photo_info.Bytes(), &new_photo)
	if err != nil {
		return err
	}

	photo_json, err := json.Marshal(map[string]string{"name": title, "description": description})
	if err != nil {
		return err
	}
	new_photo_url := fmt.Sprintf("%s/api/photo/%d", config.url, new_photo.Id)
	edit_photo_info_request, err := http.NewRequest("PUT", new_photo_url, bytes.NewBuffer(photo_json))
	if err != nil {
		return err
	}
	edit_photo_info_request.Header.Set("X-Auth-Token", auth_token)
	edit_photo_info_request.Header.Set("Content-Type", "application/json")
	_, err = client.Do(edit_photo_info_request)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	title := flag.String("title", "", "Title for photo entry.")
	description := flag.String("description", "", "Description of photo.")
	photo_path := flag.String("file", "", "Path to photo file.")
	config_path := flag.String("config", "/home/nrubin/.photoblog/config.ini", "Path to config file (defaults to ~/.photoblog/config.ini).")

	flag.Parse()

	if *photo_path == "" {
		fmt.Fprintf(os.Stderr, "Error: No file path specified.\n")
		os.Exit(1)
	}

	err, config := ParseConfig(*config_path)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = UploadPhoto(*photo_path, *title, *description, config)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
