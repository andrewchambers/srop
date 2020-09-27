# SROP

Simple remote object protocol.

Explicit goals of the protocol are:

- Simplicity
- Security
- Flexibility

The protocol is based on message passing where clients send arbitrary messages to objects on a server and the objects
are able to reply with arbitrary messages. Specific applications may define how the objects it serves will respond to
different application specific messages. There is a small set of messages the reference implementation defines
that other implementations are encouraged to reuse.

Applications may define their own message types by simply selecting a unique 64 bit identifier for that message
and defining the format. There is no need for complex protocol negotiation, objects simply respond to messages
they understand. If an object recieves a message it does not understand, it does not need to do anything.

The allocation of new objects and their corresponding id on the server is an application detail.
For example a message sent to the root object may trigger the allocation of a new object.
Communicating new ids to the client is an application detail,
but would generally be communicated via an application specific response message.

## Example

### Running the example

```
$ go run example/server/main.go 
  listening on 127.0.0.1:4444
  RootObject got a message: &example.MakeGreeter{Name:"bob"}
  I just greated a greeter with id: 1
  greeter 1 got a message: &example.Hello{From:"client"}
  client just said hello to me, saying hello back in one second.
  greeter 1 got a message: &srop.Clunk{}
  destroying myself.
  Greeter with id 1 clunked.
  RootObject clunked.
```

```
$ go run ./example/client/main.go
  Creating a new greeter named bob by contacting the bootstrap object...
  Saying hello to our new greeter...
  Got a reply from: bob
  destroying the greeter...
  closing the connection...
```

### Talking to the example Greeter server from go

```

  client := srop.NewClient(c, srop.ClientOptions{})

  // The server root object responds to MakeGreeter messages by returning us a new object handle.
  reply, _ := client.Send(srop.BOOTSTRAP_OBJECT_ID, &example.MakeGreeter{Name: "bob"})

  greeterId := reply.(*srop.ObjectRef).Id

  // Sending hello to our newly created remote greeter results in a reply hello message.
  reply, _ = client.Send(greeterId, &example.Hello{From: "client"}).(*example.Hello)

  fmt.Printf("greetings from: %s" reply.From)

  client.Close()

```

## Specification

[here](./SPEC.md)

## Implementations

- Go - The reference implementation in this repository.
