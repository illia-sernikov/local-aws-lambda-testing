package main

import (
	"log"
	"time"
)

func main() {
	var last []ApiInfo
	log.Println("traefik generator started")

	for {

		apis, err := DiscoverAPIs()
		if err != nil {
			log.Println("discovery error:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("discovered %d api-stage entries", len(apis))

		normalized := normalizeAPIs(apis)

		if Changed(last, normalized) {
			log.Printf("api set changed: %d -> %d entries, regenerating config", len(last), len(normalized))
			cfg := BuildConfig(normalized)

			err = WriteConfig(cfg)
			if err != nil {
				log.Println("config write error:", err)
			} else {
				log.Printf("config updated: %d routers, %d services, %d middlewares", len(cfg.HTTP.Routers), len(cfg.HTTP.Services), len(cfg.HTTP.Middlewares))
			}
			last = normalized
		} else {
			log.Println("no api changes detected, config left unchanged")
		}

		time.Sleep(30 * time.Second)
	}
}
