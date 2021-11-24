package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"

	wiz "github.com/ctrox/gowiz"
	"go.uber.org/zap"
)

var (
	addr = flag.String("addr", "127.0.0.1", "address of the wiz light device")
	port = flag.Int("port", 38899, "port of the wiz light device")
)

func main() {
	flag.Parse()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}

	light, err := wiz.New(
		&net.UDPAddr{IP: net.ParseIP(*addr), Port: *port},
		wiz.Logger(logger),
	)
	if err != nil {
		logger.Fatal(err.Error())
	}

	go func() {
		select {
		case <-signalChan:
			cancel()
		case <-ctx.Done():

		}
	}()

	light.Pulse(ctx, wiz.Colors{White: 0, Red: 255, Blue: 0, Green: 100})

	if err := light.TurnOff(); err != nil {
		logger.Fatal(err.Error())
	}

	logger.Info("received interrupt, shutting down")

	os.Exit(0)
}
