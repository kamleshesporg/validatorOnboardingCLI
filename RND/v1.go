	// Get the data
	response, err := http.Get(URL)
	if err != nil {
		fmt.Println(err)
	}
	defer response.Body.Close()

	if response.StatusCode == 200 {
		// Create the file
		out, err := os.Create(gopherName + ".png")
		if err != nil {
			fmt.Println(err)
		}
		defer out.Close()

		// Writer the body to file
		_, err = io.Copy(out, response.Body)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Perfect! Just saved in " + out.Name() + "!")
	} else {
		fmt.Println("Error: " + gopherName + " not exists! :-(")
	}

	/****************JSON read-write*****************/

	resp, err := http.Get(URL)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// //Convert the body to type string
	// sb := string(body)
	// log.Printf(sb)
