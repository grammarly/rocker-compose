package util

// TODO: document and write tests

import (
	"fmt"
	"time"
)

type ErrorWaitGroup struct {
	ch chan error
}

func NewErrorWaitGroup(size int) *ErrorWaitGroup {
	return &ErrorWaitGroup{
		ch: make(chan error, size),
	}
}

func (wg *ErrorWaitGroup) Done(err error) {
	wg.ch <- err
	return
}

func (wg *ErrorWaitGroup) Wait() (err error) {
	n := cap(wg.ch)
	if n == 0 {
		return nil
	}
	for {
		if resErr := <-wg.ch; resErr != nil && err == nil {
			err = resErr
		}
		if n -= 1; n == 0 {
			break
		}
	}
	return err
}

func (wg *ErrorWaitGroup) WaitFor(timeout time.Duration) error {
	n := cap(wg.ch)
	if n == 0 {
		return nil
	}
	for {
		select {
		case err := <-wg.ch:
			if err != nil {
				return err
			}
		case <-time.After(timeout):
			return fmt.Errorf("timeout %s", timeout)
		}
		if n -= 1; n == 0 {
			break
		}
	}
	return nil
}
