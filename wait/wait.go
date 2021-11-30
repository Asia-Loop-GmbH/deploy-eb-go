package wait

import (
	"log"
	"time"
)

type ExecuteFunction func() bool

func Wait(fn ExecuteFunction, interval time.Duration, times int) {
	log.Println("wait")
	for i := 0; i < times; i++ {
		log.Printf("try: %d", i+1)
		result := fn()
		if result {
			log.Printf("success")
			break
		} else {
			if i == times-1 {
				log.Panicf("max tried: %d", i+1)
			}
			log.Printf("sleep %v", interval)
			time.Sleep(interval)
		}
	}
}
