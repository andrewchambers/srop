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

## Specification

[here](./SPEC.md)

## Implementations

- Go - The reference implementation in this repository.

