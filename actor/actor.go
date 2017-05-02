package actor

import (
	"log"
)

type Actor struct {
	box chan msg
}

type msg struct {
	fn  func()
	res chan interface{}
}

func NewActor() *Actor {
	a := &Actor{
		box: make(chan msg, 5),
	}
	go a.loop()
	return a
}

func (a *Actor) Run(f func()) {
	res := make(chan interface{}, 1)
	a.box <- msg{f, res}
	if err := <-res; err != nil {
		panic(err)
	}
}

func (a *Actor) Post(f func()) {
	a.box <- msg{
		fn: f,
	}
}

func (a *Actor) Kill() {
	close(a.box)
}

func (a *Actor) loop() {
	for m := range a.box {
		m.safeRun()
	}
}

func (m msg) safeRun() {
	defer func() {
		err := recover()
		if m.res != nil {
			m.res <- err
		} else if err != nil {
			log.Printf("Recovered from panic! %s \n", err)
		}
	}()
	m.fn()
}
