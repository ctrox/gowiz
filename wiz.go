package wiz

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Message struct {
	Method string `json:"method"`
	Params Params `json:"params"`
}

type Params struct {
	State   bool `json:"state"`
	White   int  `json:"w,omitempty"`
	Red     int  `json:"r,omitempty"`
	Blue    int  `json:"b,omitempty"`
	Green   int  `json:"g,omitempty"`
	Dimming int  `json:"dimming,omitempty"`
}

type Reply struct {
	Method string `json:"method"`
	Env    string `json:"env"`
	Result Result `json:"result"`
}

type Result struct {
	Success bool `json:"success"`
}

type Light struct {
	conn    net.Conn
	log     *zap.Logger
	timeout time.Duration
}

type Colors struct {
	White int
	Red   int
	Blue  int
	Green int
}

type Option func(l *Light)

func New(address net.Addr, opts ...Option) (*Light, error) {
	l := &Light{
		timeout: time.Second,
	}

	for _, opt := range opts {
		opt(l)
	}

	if l.log == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("error creating logger: %w", err)
		}
		l.log = logger
	}

	conn, err := net.Dial("udp", address.String())
	if err != nil {
		return nil, fmt.Errorf("error dialing light at %s: %w", address.String(), err)
	}

	l.conn = conn

	return l, err
}

func Logger(logger *zap.Logger) Option {
	return func(l *Light) {
		l.log = logger
	}
}

func Timeout(d time.Duration) Option {
	return func(l *Light) {
		l.timeout = d
	}
}

func (l *Light) Pulse(ctx context.Context, colors Colors) {
	const (
		lowDim    = 10
		lowSleep  = 200
		highDim   = 100
		highSleep = 800
	)

	dim := lowDim
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := l.SetColor(colors, dim); err != nil {
				l.log.Error(err.Error())
			}

			sleep := 0
			switch dim {
			case lowDim:
				dim = highDim
				sleep = highSleep
			case highDim:
				dim = lowDim
				sleep = lowSleep
			}

			time.Sleep(time.Millisecond * time.Duration(sleep))
		}
	}
}

func (l *Light) SendMessage(msg *Message) (*Reply, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal message: %w", err)
	}

	if err := l.conn.SetDeadline(time.Now().Add(l.timeout)); err != nil {
		return nil, fmt.Errorf("unable to set connection deadline")
	}

	i, err := l.conn.Write(b)
	if err != nil {
		return nil, fmt.Errorf("unable to write to connection: %w", err)
	}

	l.log.Debug("written to udp connection", zap.Int("bytes", i))

	// TODO: not sure how big a reply can be but from testing 1024 should
	// 	be more than enough.
	buffer := make([]byte, 1024)
	n, err := l.conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("unable to read from udp connection: %w", err)
	}

	reply := &Reply{}
	if err := json.Unmarshal(buffer[:n], reply); err != nil {
		return nil, fmt.Errorf("unable to unmarshal message: %w", err)
	}

	l.log.Debug("message received", zap.Object("reply", reply))

	return reply, nil
}

func (l *Light) TurnOn() error {
	if _, err := l.SendMessage(&Message{
		Method: "setPilot",
		Params: Params{
			State: true,
		},
	}); err != nil {
		return fmt.Errorf("unable to turn on light: %w", err)
	}

	return nil
}

func (l *Light) TurnOff() error {
	if _, err := l.SendMessage(&Message{
		Method: "setPilot",
		Params: Params{
			State: false,
		},
	}); err != nil {
		return fmt.Errorf("unable to turn off light: %w", err)
	}

	return nil
}

func (l *Light) SetColor(colors Colors, dim int) error {
	if _, err := l.SendMessage(&Message{
		Method: "setPilot",
		Params: Params{
			State:   true,
			White:   colors.White,
			Red:     colors.Red,
			Green:   colors.Green,
			Blue:    colors.Blue,
			Dimming: dim,
		},
	}); err != nil {
		return fmt.Errorf("unable to set light color: %w", err)
	}

	return nil
}

func (reply *Reply) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("success", reply.Result.Success)
	enc.AddString("method", reply.Method)
	return nil
}
