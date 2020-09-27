package example

import (
	"context"
	"log"
	"time"
	"encoding/json"

	"github.com/andrewchambers/srop"
)

// This file implements a 'greeter' protocol.
// You can make a new greeter, say hello to it, and then destroy it.
// The subfolders client, and server can be built and run from the command line.
// cd server && go build && ./server
// cd client && go build && ./client

// These types are all our custom messages, The numbers are randomly generated, but the idea is that once
// you define a message type, you don't change it in backwards incompatible ways.
// That way the number globally and uniquely identifies your message purpose and structure.

const (
	TYPE_MAKE_GREETER = 0x9685d09cb0114f1f
	TYPE_HELLO        = 0xa79e175dc97ed3ab
)

func JsonUnmarshal(buf []byte, v interface{}) bool {
	err := json.Unmarshal(buf, v)
	if err != nil {
		return false
	}

	return true
}

func JsonMarshal(v interface{}) []byte {
	buf, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return buf
}

type MakeGreeter struct {
	Name string
}

func (m *MakeGreeter) SropType() uint64            { return TYPE_MAKE_GREETER }
func (m *MakeGreeter) SropMarshal() []byte           { return JsonMarshal(m) }
func (m *MakeGreeter) SropUnmarshal(buf []byte) bool { return JsonUnmarshal(buf, m) }

type Hello struct {
	From string
}

func (m *Hello) SropType() uint64            { return TYPE_HELLO }
func (m *Hello) SropMarshal() []byte           { return JsonMarshal(m) }
func (m *Hello) SropUnmarshal(buf []byte) bool { return JsonUnmarshal(buf, m) }

// We must register our types before coolmsg can understand then.
func init() {
	srop.RegisterMessage(TYPE_MAKE_GREETER, func() srop.Message { return &MakeGreeter{} })
	srop.RegisterMessage(TYPE_HELLO, func() srop.Message { return &Hello{} })
}

// The server implementation

// The RootObject will be our bootstrap object, it responds to MakeGreeter messages with MakeGreeterReply messages.
type RootObject struct {
	Name string
}

func (r *RootObject) Message(ctx context.Context, cs *srop.ConnServer, m srop.Message, respond srop.RespondFunc) {
	log.Printf("RootObject got a message: %#v", m)

	switch m := m.(type) {
	case *MakeGreeter:
		g := &Greeter{
			Name: m.Name,
		}
		id := cs.Register(g)
		// Save the object id, so it knows how
		// to remove itself.
		g.Self = id
		log.Printf("I just greated a greeter with id: %d", id)
		respond(&srop.ObjectRef{
			Id: id,
		})
	default:
		respond(&srop.UnexpectedMessage{})
	}
}

func (r *RootObject) UnknownMessage(ctx context.Context, cs *srop.ConnServer, t uint64, buf []byte, respond srop.RespondFunc) {
	log.Printf("got an unknown message: %v %#v", t, buf)
	respond(&srop.UnexpectedMessage{})
}

// Clunk is the cleanup method of an object, the name Clunk comes from the 9p protocol.
// An object is clunked when a server is done with it.
func (r *RootObject) Clunk(cs *srop.ConnServer) {
	log.Printf("RootObject clunked.")
}

type Greeter struct {
	Name string
	Self uint64
}

func (g *Greeter) Message(ctx context.Context, cs *srop.ConnServer, m srop.Message, respond srop.RespondFunc) {
	log.Printf("greeter %d got a message: %#v", g.Self, m)
	switch m := m.(type) {
	case *Hello:
		log.Printf("%s just said hello to me, saying hello back in one second.", m.From)

		// The Message function can block the server, to do work asyncronously, handle it in a goroutine.
		// Using the cs.Go() allows the server to wait until all active requests end on termination for
		// tidy shutdown.
		cs.Go(func() {
			time.Sleep(1 * time.Second)
			respond(&Hello{
				From: g.Name,
			})
		})
	case *srop.Clunk:
		log.Printf("destroying myself.")
		cs.Clunk(g.Self)
		respond(&srop.Ok{})
	default:
		log.Printf("got an unexpected message: %#v", m)
		respond(&srop.UnexpectedMessage{})
	}
}

func (g *Greeter) UnknownMessage(ctx context.Context, cs *srop.ConnServer, t uint64, buf []byte, respond srop.RespondFunc) {
	log.Printf("got an unknown message: %v %#v", t, buf)
	respond(&srop.UnexpectedMessage{})
}

func (g *Greeter) Clunk(cs *srop.ConnServer) {
	log.Printf("Greeter with id %d clunked.", g.Self)
}
