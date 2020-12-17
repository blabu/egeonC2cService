package server

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/blabu/egeonC2cService/dto"
)

type panicCover struct {
	errorWriter io.Writer
	base        MainLogicIO
}

func CreatePanicCoverLogic(base MainLogicIO) MainLogicIO {
	return panicCover{os.Stderr, base}
}

func (p panicCover) cover() {
	if err := recover(); err != nil {
		if p.errorWriter != nil {
			p.errorWriter.Write([]byte(fmt.Sprintf("PANIC %v", err)))
		}
	}
}

func (p panicCover) Read(ctx context.Context, handler dto.ServerReadHandler) {
	p.base.Read(ctx, func(data []byte, err error) error {
		defer p.cover()
		return handler(data, err)
	})
}

func (p panicCover) Write(data []byte) (int, error) {
	defer p.cover()
	return p.base.Write(data)
}

func (p panicCover) Close() error {
	defer p.cover()
	return p.base.Close()
}
