package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/manifoldco/promptui"
)

func updateGenesis(mynode string) {

	genesisURL := configCliParams.GenesisUrl //"https://web3sports.s3.ap-south-1.amazonaws.com/blockchain/genesis.json"

	resp, err := http.Get(genesisURL)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// Create the file
	err1 := os.WriteFile(mynode+"/config/genesis.json", body, os.ModePerm)
	if err1 != nil {
		fmt.Println(err1)
	}
	fmt.Println("Genesis updated.")
}

func updateConfigToml(mynode string) {

	confiToml := configCliParams.ConfigTomlUrl //"https://web3sports.s3.ap-south-1.amazonaws.com/blockchain/config.toml"

	resp, err := http.Get(confiToml)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// sb := string(body)
	// log.Printf(sb)

	// getPeerId := getPersistentPeers()
	// newgetPeerId := string(getPeerId)

	// update := strings.Replace(sb, "persistent_peers = \"\"", "persistent_peers = \""+newgetPeerId+"\"", 1)

	// Create the file
	err1 := os.WriteFile(mynode+"/config/config.toml", []byte(body), os.ModePerm)
	if err1 != nil {
		fmt.Println(err1)
	}

	fmt.Println("Config.toml updated.")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return false
}

func yesNo(msg string) bool {
	prompt := promptui.Select{
		Label: msg + "[Yes/No]",
		Items: []string{"Yes", "No"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}
	return result == "Yes"
}

/** TEMP - UNUSED FUNCTION */

func getConfigCliParams() ConfigCliParams {

	fmt.Println("Config parameters fetching...")
	// configParams := "https://web3sports.s3.ap-south-1.amazonaws.com/blockchain/mrmintChainCLIconfig.json"

	// resp, err := http.Get(configParams)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	//We Read the response body on the line below.
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	body := []byte(`{
    "persistent_peers": "521c3b33982d9246bf76f12377d4842f696e92f2@3.110.16.39:26656",
    "genesisUrl": "https://web3sports.s3.ap-south-1.amazonaws.com/blockchain/server/genesis.json",
    "configToml": "https://web3sports.s3.ap-south-1.amazonaws.com/blockchain/server/config.toml",
    "chindId": "os_9000-1",
    "minStakeFund": 50,
    "bootNodeRpc":"http://3.110.16.39:26657"
}`)
	// //Create a variable of the same type as our model
	var cResp ConfigCliParams

	err := json.Unmarshal(body, &cResp)
	if err != nil {
		log.Fatal(err)
	}
	return cResp
}
