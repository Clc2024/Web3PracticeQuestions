package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	mode := flag.String("mode", "server", "运行模式：server 或 client")

	flag.Parse()
	switch *mode {
	case "01":
		log.Println("Starting block_height_monitor")
		block_height_monitor()
	case "02":
		log.Println("Starting address_tx_monitor")
		address_tx_monitor()

	case "03":
		log.Println("Starting multi_erc20_monitor")
		multi_erc20_monitor()

	default:
		log.Fatalf("Unknown mode: %s . User 'server' or 'client'\n", *mode)
		os.Exit(1)
	}
}
