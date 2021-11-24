# gowiz

Go library for WiZ lights.

This just supports a few very basic functions so far. It has only been tested with the LED-Lightstrip from WiZ.

```bash
go get github.com/ctrox/gowiz
```

There is a small sample program in the `cmd/wizctl` directory which will just make your LED strip pulse.

```bash
go run github.com/ctrox/gowiz/cmd/wizctl -addr 192.168.1.194
```