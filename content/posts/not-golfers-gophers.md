---
layout: post
title: "Not Golfers, Gophers"
date: 2014-05-18 11:01:37
categories: code
tags: ["go", "code", "erlang"]
---

I was doing some language exploration as I am toying with the idea of getting some backend
services up and running, and I have some constraints that Ruby does not meet. I kept hearing
people mention Go, and my only vague notion about it was that it was like Erlang but not
as mature, so I have been putting off really looking into it. Well that changed, I went through
the Go tour and have been playing around with it a bit, and I have to say I am really impressed.

## Channels vs Actors

The actor model in Erlang encourages you to code in a certain style. Basically, you write
single purpose entities with a minimal amount of state, that operate using mostly pure functions.
The only interactions with the world outside of an actor are through message passing, and those
messages are typically tagged tuples. Each actor has a mailbox for getting messages which it typically
processes in a FIFO way, but you can also do selective receives by pattern matching on the messages
in your inbox. In this way, when you want to communicate to another actor, you need a reference
to their process identifier (PID), that you can use to send messages to. You usually get this
reference passed to you in a message. For instance, here is a raw version of a common pattern in
non-OTP code for the main loop of an Erlang actor:

```erlang
% some_guy.erl
live(State) ->
  receive
    {From, {handshake, Person}} ->
      NewState = shake_hands(State, Person)
      From ! {self(), ok},
      live(NewState);
    {From, {punch, Person}} ->
      NewState = get_punched(State, Person)
      From ! {self(), ok},
      live(NewState);
    terminate ->
      ok
  end.
```

This guy's life is pretty boring, he knows how to shake hands, get punched, and die. Here, we do
a selective receive on the mailbox, and the loop will only pull messages out of the mailbox that
are of the form `{SomePID, {Action, Varible}}` where `Action` is in a small set of known atoms.
He also responds to `terminate` which stops him looping. The way you would message this guy
from another actor would look like

```erlang
SomeGuyPid = spawn(some_guy, live, [[]]).
SomeGuyPid ! {self(), {handshake, "Me"}}.
```

In Erlang, `self()` returns your own PID, this allows us to send messages to others and get
replies back without them actually knowing who we are. This allows actors to be highly
decoupled and it makes it really easy to keep them single purpose. There needs to be some actor
at the top who spawns some processes, and maybe has a lookup table from names to PIDs, but
most of the actors you write simply receive a message with enough context to know who to
respond to. In reality you use OTP to build trees of communicating processes, and wrap these
bare message sends in APIs, but the underlying idea is the same. Actors have a mailbox with
an address (PID), and you can stuff their mailbox by sending a message to that address. It is
then up to them to choose what and when to pull out of their mailbox.

Go does not present these same primatives. Instead, Go has a type known as a `Channel`. A
channel can be send-only, receive-only, or bidirectional. Any number of processes can
write to or read from a channel as long as they have a reference to it. If we want to reimplement
the same boring guy as above in Go, it might look something like (pardon my ignorance, I wrote
hello world in Go yesterday):

```go
// some_guy.go
type InMessage interface {
  inMessageType()
}

type OutMessage interface {
  outMessageType()
}

type Request struct {
  From chan OutMessage
  Action string
  Person string
}

type Atom string
type Response struct {
  From chan
  Status Atom
}

func (a *Atom) outMessageType() {}
func (a *Response) outMessageType() {}

func (a *Atom) inMessageType() {}
func (a *Request) inMessageType() {}

func Live(state []string, ch chan InMessage) {
  raw_msg := <-ch
  switch msg := raw_msg.(type) {
  case Request:
    switch msg.Action {
    case 'handshake':
      new_state = shake_hands(state, msg.Person)
      msg.From <- &{ch, "ok"}
      Live(new_state, ch)
    case 'punch':
      new_state = get_punched(state, msg.Person)
      msg.From <- &{ch, "ok"}
      Live(new_state, ch)
    }
  case Atom:
    if msg == "terminate" {
      // "ok"
    }
  }
}
```

Although some of the trickeration I had to go through there with something like an
algebraic data type (ADT) was primiarly because of the constraints imposed by directly
translating from Erlang. I have found that the type of structures you use as communcation
primatives in an untyped language are drastically different than what you would choose
to use in a typed language. For instance, it is common to use a heterogeneous array
in languages like Python to repreesnt multiple return values, and this makes sense
because that is how the language allows you to work. Clearly you wouldn't do this
in a language like Haskell which does not allow heterogeneous lists, instead you
would typically use an ADT. In Erlang, you often use tuples, usually with type tags,
as the communication protocol, because of the nice pattern matching facilities, the
low overhead of these objects, and the complete lack of static types. In Go, we have
types, but not ADTs (well not naturally anyway), but we do have interfaces. Typically
this communication would be done over a more natural set of channels with a more
clearly defined interface for the objects being passed around.

I can imagine building channels in Erlang and Actors in Go using the set of primatives
that are offered, so I don't think they should really be seen as competing technologies
in that sense. However, Go encourages the use of typed channels for communication.
I think this is great. I think channels are more general and allow the type of flexibility
that you usually want. If you want a channel that only one goroutine uses then you can
do that really easily, you just only pass the reference to that function. But the ability
to build more complex communication networks out of channels is super interesting.
Also having some notion of static types on those channels is nice only because of the
flexibility of the type system in Go. Usually you have to jump through a lot of hoops
when you are dealing with making outside data type safe, but it seems like Go makes
an okay tradeoff here. My opinion on that is mostly grounded in my lack of experience
doing that everyday in a strong, statically typed language, it just feels like a ton
of extra work with real, but sometimes marginal benefits.

## Go oddities

So I like the language of Go, but it has some goofy parts too. The one that bugs
me the most, is how it handles pointers, references, values, and interfaces. Okay,
that makes no sense I know, but here is the example (taken from part 55 of the Go tour):

```go{numberLines: true}
package main

import (
    "fmt"
    "math"
)

type Abser interface {
    Abs() float64
}

func main() {
    var a Abser
    f := MyFloat(-math.Sqrt2)
    v := Vertex{3, 4}

    a = f
    a = &v

    // a = v
    // uncomment the line above => no compile
    fmt.Println(v.Abs()) // this compiles just fine

    fmt.Println(a.Abs())
}

type MyFloat float64

func (f MyFloat) Abs() float64 {
    if f < 0 {
        return float64(-f)
    }
    return float64(f)
}

type Vertex struct {
    X, Y float64
}

func (v *Vertex) Abs() float64 {
    return math.Sqrt(v.X*v.X + v.Y*v.Y)
}
```

We are defining an interface called `Abser` which has one functio `Abs()` which returns a `float64`.
At the bottom we define two types and implement the interface for them. However, for one, `MyFloat`
we implement the interface with a value of the type, not a pointer. For the other, `Vertex`, we
implement the interface only for a pointer to that type. Okay, now here is the weird part,
if you uncomment line 20, this will not compile. This is because you cannot assign a value of
type `Vertex` to a variable of the interface type `Abser` because there is no implementation for
`Abs` on a value of type `Vertex`. Now this is perfectly reasonable, it could be that the implementation
mutates the object and therefore only getting a copy via pass-by-value would not be the intended
behaviour. So Go makes the decision that because of that, there is no implementation for that type and
refuses to compile. However, the next line shows clearly that you can call `Abs` on a `Vertex` value.
This works just fine because the value is implicitly converted to a pointer. In many places, because
Go allows you to use the same syntax when working with a pointer, reference, and value, it can be
somewhat confusing to know which one you've got. Most of the time it doesn't really matter, but
the fact that it does enforce some constraints on this is a bit weird. Basically, just because you
see the following code:

```go
type X interface {
  someFunction()
}

func main() {
  var v SomeType
  v.someFunction()
}
```

It does not imply that `v` actually implements the interface `X` which entirely flies in the face
of duck typing intuition. It is only this edge case dealing with pointers and values that I know
of where you can make this happen, but still it is not desireable.

## Will I use it?

I really like the strong standard library, the channel based concurrency primatives, and
the static typing. There is also a nice library called [goworker][goworker] which allows you
to use Go to write workers for jobs using [Resque][resque]. Basically, this means that in a Rails
app, you can push work onto a Redis queue, then have a Go process pull the work off that queue, and
do whatever it needs to do to process that job. Now clearly, you can pick any two languages and
any middle queueing system you want and make the same statement. However, the solid concurrency
primatives of Go, with a low memory footprint, combined with the fact that I am already preconstrained
to be working in a Rails environment, makes this set-up highly appealing. I am currently writing
background services in Ruby, which works and is a great language. But I think there are some services
where having high concurrency and high performance will be paramount and I want to have the ability
to handle those tasks sooner rather than later.


[goworker]:   http://www.goworker.org/
[resque]:     https://github.com/resque/resque
