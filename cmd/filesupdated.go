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
	"strings"

	"github.com/manifoldco/promptui"
)

func updateGenesis(mynode string) {

	// genesisURL := "https://ipfs.io/ipfs/bafkreiap6fsih5kdo3ixssmfw6lcnjzo6fybux3pguovzkgmed2bwsnue4"
	genesisURL := "http://3.110.16.39/genesis.json"

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

	// confiToml := "https://ipfs.io/ipfs/bafkreie7me3gxmun26g7fzb5ncnp2kiqcilmzy5gn67ayylovagzoyk2mu"
	confiToml := "http://3.110.16.39/config.txt"

	resp, err := http.Get(confiToml)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	sb := string(body)
	// log.Printf(sb)

	getPeerId := getPersistentPeers()
	newgetPeerId := string(getPeerId)

	update := strings.Replace(sb, "persistent_peers = \"\"", "persistent_peers = \""+newgetPeerId+"\"", 1)

	// Create the file
	err1 := os.WriteFile(mynode+"/config/config.toml", []byte(update), os.ModePerm)
	if err1 != nil {
		fmt.Println(err1)
	}

	fmt.Println("Config.toml updated.")
}

type Cryptoresponse struct {
	PersistentPeers string `json:"persistent_peers"`
}

func getPersistentPeers() string {
	// https://plum-skinny-bedbug-581.mypinata.cloud/
	// confiToml := "https://ipfs.io/ipfs/bafkreiaz3gotvqz4llrctggfvpfwdzdmaqfxaggnv53sk6rveqvjh52oha"
	confiToml := "http://3.110.16.39/persistent_peers.json"

	resp, err := http.Get(confiToml)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// //Create a variable of the same type as our model
	var cResp Cryptoresponse

	err = json.Unmarshal(body, &cResp)
	if err != nil {
		log.Fatal(err)
	}
	return cResp.PersistentPeers
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
