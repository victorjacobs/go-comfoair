package main

import (
	"log"
	"time"
)

func loopSafely(f func()) {
	defer func() {
		if v := recover(); v != nil {
			log.Printf("Panic: %v, restarting", v)
			time.Sleep(time.Second)
			go loopSafely(f)
		}
	}()

	for {
		f()
	}
}
