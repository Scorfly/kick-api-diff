package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rapidloop/skv"
	"github.com/wI2L/jsondiff"
)

func main() {
	currentTime := time.Now()
	fmt.Println("Starting … ", currentTime.Format("2006-01-02 15:04:05"))

	/* get discord webhook url */
	discordWebhook := flag.String("discord", "127.0.0.1", "Discord webhook URL")
	flag.Parse()

	/***************************************************
	 *
	 *      GET API
	 *
	 */
	url := "https://api.kick.com/swagger/v1/doc.json"
	req, err := http.NewRequest("GET", url, nil)
	resp, _ := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respJSON, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	/***************************************************
	 *
	 *      Get last version of the API stored
	 *
	 */

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// get last version form local db
	store, err := skv.Open(dir + "/.kickApi.db")
	if err != nil {
		panic(err)
	}

	var previousAPI string
	err = store.Get("kick::api", &previousAPI)
	if err != nil {
		if err.Error() == "skv: key not found" {
			// init
			err = store.Put("kick::api", string(respJSON))
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	if previousAPI == string(respJSON) {
		return
	}

	/***************************************************
	 *
	 *      New version detected !
	 *          - store it
	 *          - post on discord the update
	 *
	 */

	err = store.Put("kick::api", string(respJSON))
	if err != nil {
		panic(err)
	}

	patch, err := jsondiff.CompareJSON([]byte(previousAPI), respJSON)
	if err != nil {
		panic(err)
	}
	for _, op := range patch {
		fmt.Printf("%s\n", op)
	}

	// Step 1: Unmarshal the compact JSON into an interface{}
	var diff interface{}
	err = json.Unmarshal([]byte(patch.String()), &diff)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	// Step 2: Marshal the data back into a pretty-printed JSON string
	prettyJSON, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	prettyJSONStr := strings.Replace(string(prettyJSON), `"`, `\"`, -1)
	prettyJSONStr = strings.Replace(prettyJSONStr, "\n", `\n`, -1)

	var jsonStr = []byte(fmt.Sprintf(
		`{"content": "**[KICK API update]** \n\n`+"```json"+`\n%s\n`+"```"+`"}`,
		prettyJSONStr,
	))

	url = *discordWebhook
	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Updated …")
}
