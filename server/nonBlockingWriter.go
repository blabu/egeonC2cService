package server

import (
	log "blabu/c2cService/logWrapper"
	"errors"
	"io"
	"sync/atomic"
	"time"
)

type writeWrapper struct {
	writeChan chan []byte
	stream    io.WriteCloser
	cntr      int32
	err       error
}

// NonBlockingWriter - wrap any writer into noblocking write operation
func NonBlockingWriter(f io.WriteCloser, bufSize int) io.WriteCloser {
	w := writeWrapper{
		stream:    f,
		cntr:      0,
		writeChan: make(chan []byte, bufSize),
		err:       nil,
	}
	go w.write()
	return &w
}

// write - write in another goroutine
func (w *writeWrapper) write() {
	defer w.stream.Close()
	for {
		b, ok := <-w.writeChan
		if !ok {
			w.err = errors.New("Finish write. Channel was closed")
			atomic.StoreInt32(&w.cntr, -1)
			log.Error(w.err.Error())
			return
		}
		if _, err := w.stream.Write(b); err != nil {
			w.err = err
			atomic.StoreInt32(&w.cntr, -1)
			log.Error(w.err.Error())
			return
		}
		atomic.AddInt32(&w.cntr, -1)
	}
}

func (w *writeWrapper) Write(data []byte) (int, error) {
	if atomic.LoadInt32(&w.cntr) < 0 {
		return 0, errors.New("Write fail: " + w.err.Error())
	}
	atomic.AddInt32(&w.cntr, 1)
	var dst []byte
	copy(dst, data)
	w.writeChan <- dst
	return len(data), nil
}

func (w *writeWrapper) Close() error {
	for atomic.LoadInt32(&w.cntr) > 0 {
		time.Sleep(time.Microsecond)
	}
	close(w.writeChan)
	return w.err
}
