package main

import "log"

func LogInfo(msg string) {
	log.Println("[INFO]", msg)
}

func LogError(msg string, err error) {
	if err != nil {
		log.Println("[ERROR]", msg, ":", err)
	} else {
		log.Println("[ERROR]", msg)
	}
}
