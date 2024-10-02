package main

import (
	"fmt"
	"net/http"
	"time"
)

func scheduleLT(target Target) {
	go func() {
		<-time.After(target.StartAt.Sub(time.Now()))
		client := http.Client{}
		for {
			resp, err := client.Get(target.Url)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(resp)
		}
	}()
}
