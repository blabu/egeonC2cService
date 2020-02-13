package server

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"
)

type writeWrapper struct {
	writeChan chan []byte
	stream    io.WriteCloser
	cntr      int32
}

func NonBlockingWriter(f io.WriteCloser, bufSize int) io.WriteCloser {
	w := writeWrapper{
		stream:    f,
		cntr:      0,
		writeChan: make(chan []byte, bufSize),
	}
	go w.write()
	return &w
}

func (w *writeWrapper) write() {
	defer w.stream.Close()
	for {
		b, ok := <-w.writeChan
		if !ok {
			log.Print("Finish write. Channel was closed")
			atomic.StoreInt32(&w.cntr, -1)
			return
		}
		if _, err := w.stream.Write(b); err != nil {
			log.Print("Finish write ", err.Error())
			atomic.StoreInt32(&w.cntr, -1)
			return
		}
		atomic.AddInt32(&w.cntr, -1)
	}
}

func (w *writeWrapper) Write(data []byte) (int, error) {
	if atomic.LoadInt32(&w.cntr) < 0 {
		return 0, fmt.Errorf("Write fail")
	}
	atomic.AddInt32(&w.cntr, 1)
	w.writeChan <- data
	return len(data), nil
}

func (w *writeWrapper) Close() error {
	for atomic.LoadInt32(&w.cntr) > 0 {
		time.Sleep(time.Microsecond)
	}
	close(w.writeChan)
	return nil
}
