package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/kent-h/stateful-router"
	"github.com/kent-h/stateful-router/example-service/protos/server"
	"github.com/kent-h/stateful-router/example-service/service"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var peerDNSFormat string
var listenAddress string

func init() {
	var have bool
	listenAddress, have = os.LookupEnv("LISTEN_ADDRESS")
	if !have {
		panic("env var LISTEN_ADDRESS not defined")
	}
	peerDNSFormat, have = os.LookupEnv("PEER_DNS_FORMAT")
	if !have {
		panic("env var PEER_DNS_FORMAT not specified")
	}
}

func main() {
	ordinal := router.MustParseOrdinal(os.Getenv("ORDINAL"))

	fmt.Println("ordinal:", ordinal)

	client := service.New(uint32(ordinal), peerDNSFormat, listenAddress)

	//go sendDummyRequests(client, uint32(ordinal))

	go cli(client)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println(<-sigs)
	client.Stop()
}

func sendDummyRequests(client stateful.StatefulServer, ordinal uint32) {
	for device, ctr := uint64(0), 0; true; device, ctr = (device+1)%6, ctr+1 {
		time.Sleep(time.Second * 1)
		if response, err := client.GetData(context.Background(), &stateful.GetDataRequest{Device: strconv.FormatUint(device, 10)}); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Device:", device, "Data:", string(response.Data))
		}

		if _, err := client.SetData(context.Background(), &stateful.SetDataRequest{Device: strconv.FormatUint(device, 10), Data: []byte(fmt.Sprint("some string ", ordinal, " ", ctr))}); err != nil {
			fmt.Println(err)
		}
	}
}

func cli(client stateful.StatefulServer) {
	regex := regexp.MustCompile(`^(set|get) ([^ ]+)(.*)$`)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		array := regex.FindStringSubmatch(scanner.Text())
		if array == nil {
			fmt.Println("not a valid command")
			continue
		}

		deviceId := array[2]

		if array[1] == "set" {
			if _, err := client.SetData(context.Background(), &stateful.SetDataRequest{Device: deviceId, Data: []byte(strings.TrimSpace(array[3]))}); err != nil {
				fmt.Println(">", err)
			} else {
				fmt.Println("> OK")
			}
		} else if array[1] == "get" {
			resp, err := client.GetData(context.Background(), &stateful.GetDataRequest{Device: deviceId})
			if err != nil {
				fmt.Println(">", err)
			} else {
				fmt.Println(">", string(resp.Data))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
