package main

import (
	assistantweb "nws510obtl25p1Il/ai-assistant-web"
	"log"
)

func main() {
	if err := assistantweb.Start(); err != nil {
		log.Fatalf("ai-assistant-web start failed: %v", err)
	}
}
