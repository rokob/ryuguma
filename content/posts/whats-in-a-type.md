---
layout: post
title: "What's in a type?"
date: 2014-05-17 17:53:27
categories: code
tags: ["go", "code", "ruby", "erlang", "types", "objc"]
excerpt: This post is mostly going to be about my experiences and where I currently stand on the subject of types in programming languages. It is a bit rambling and not necessarily informative, but I am trying to categorize some of the things I have heard and experienced working with different languages and different people over time. Some topics discussed being what object oriented really means, where it is done well, and where it breaks down. Also a bit about interfaces being, in my mind, the most important thing to make easy to analyze statically.
---

### Disclaimer:

I need to put a giant disclaimer up front here about how I am not an expert in type theory. I read a lot of books and have written code in many different languages,
both professionally and in my spare time. I enjoy thinking about some of the underlying concepts in computer science, and I have sat in on quite a few graduate
courses in CS while I was doing my PhD in something that is not computer science. Therefore I may certainly make statements in this post that are strictly wrong from
a pure theoretical standpoint, although I certainly hope not. This post is mostly going to be about my experiences and where I currently stand on the subject of types in programming languages.


## Object Oriented Type Systems

I think that object oriented programming can be a simple way of organizing your thoughts in code. It works well when you have a clear set of real-world objects
with a single purpose that generates their identity and relationships to one another, and you want to have a conversation with others without any code involved. Often
times the decision to use an object oriented language comes from:

* Legacy
* Hiring concerns
* Previous experience
* Existence of domain experts
* Unwillingness to learn
* Lack of knowledge of alternatives
* Risk assessments

I think that object oriented type systems breakdown when programming in the small and the large, but can be quite natural in the middle ground. One problem with this
is that there is much pain upfront when prototyping, which leads to an okay state in the middle where you feel like your hard work has paid off, only to be followed by
an eternal death spiral where you don't want to change course because of the debt you've already paid. This sunk cost colors your decision making and makes you put in
even more work to keep your giant ball of `AbstractFactoryFactories` humming. You start to write code because of the language, to satisfy the type checker's opinion
you are forced to do unnatural things like forcing unrelated things to have a common super class. Eventually you start to use `void*` or `id` or `Object` to write
"generic" functions and at that point you've lost.

### Object Oriented Done Right

Fundamentally, OO is about single purpose objects that communicate via message passing. This has been lost over time with different implementations and interpretations
of what being object oriented means. This usually happens with the interpretation of what "everything is an object" means. To me, that phrase means that every construct
in your program has a set of messages that it understands. If you send it a message it understands, then it will do something. If you send it a message it doesn't
understand, well the world might explode in firey exceptions, or it might try to interpret your message in another way by deferring to other objects that it knows about.
This is how the real world works, in some sense. The problem enters when you start to define what an object is by a static identifier rather than as a set of messages
that it knows how to handle. Erlang is an OO language with each process representing an object. If you send a message to an Erlang process, it will either act on it
because it understands it, or it will tell you it has no idea what you are talking about. Conceptually this is what OO is about, it is just the implementations of these
ideas where people start to be led astray.

Ruby is an overtly OO language, but it also fundamentally starts from the perspective of objects as actors that either respond to a message or don't. Erlang and Ruby
have in common that they are dynamically, strongly typed. I need to emphasize what I mean by strong because it is not a well-defined term. I will stick with the statement
by Liskov "whenever an object is passed from a calling function to a called function, its type must be compatible with the type declared in the called function".
Translate this roughly to mean that if an object only needs to be compatible with a function call, so for instance in Ruby this means that if you send a message to an
object, it simply needs to respond to that message in some way that does not throw an exception or otherwise corrupt the state of the running program. The type checking
is done at runtime, so you only find out that things blow up when they actually blow up. But that being said, they do blow up if something is wrong. Static type checking
is independent of type strength. In these so-called duck type languages, you code against what an object can do, not what it's name is. In this way, it is much more
akin to the real world where you ask someone to do a job because they are capable, not because they have a particular title or pedigree (or wait...)

### Object Oriented Done Wrong

There are OO languages with a more strict view on what a type is. C++, for instance, has a static, nominal type system. You must know at compile time exactly whether an
object is compatible with a function call, and you know this based on the declaration of the object. That is, the exact name of the object is what is important. Now,
you can say that this comes along with a set of messages that the object knows how to respond to. But it puts you into a different frame of mind. You start to think
about the name of an object, `Shape`, `Rectangle`, as opposed to an object which responds to the message `area`. Now, clearly you can write what basically I am
calling "good" OO code in C++, most of the time this devolves into templates and the newer concept of Concepts. However, in my experience, C++ is neither taught nor
practiced this way. C++ is used like Java, and that is why they are often seen as competing technologies. Also, as a sidebar, clearly anything
can be done in any language, but I am talking about both what is natural in a language, what is intended/built-in, and how a language is actually used in practice. I
love C++ for certain low level programming tasks. I enjoy the static type checker, especially when it comes to type safe containers. But, I find that the benefits you
get from these features really only shine when you are programming in the large, but at that point you end up with so many different people touching the code that
you end up a slave to what the language requires to get something to compile rather than really focusing on solving the problem at hand.

## Interfaces / Protocols / Typeclasses

In a language like Ruby, where there are no formal type declarations and hence no static analysis, you are still coding against an interface between the classes you write.
This informal API driven style works in Ruby mostly by convention, and hence it only works with high quality, thoughtful engineers. If you have someone slapping things
together, they are just going to add methods as they need them and move on. No static analysis means that it is up to the programmer to ensure that API changes propagate
throughout the codebase. Nonetheless, an interface is all that is really important when you are writing code. If you have objects in your system, all you really care
about is what can I tell them to do. What messages can I send between different objects. This is one notion that has been codified decently well in some languages, and
has been bolted on by some older languages. This is the notion of a Typeclass in Haskell, a Protocol in Objective-C, an Interface in Go, etc. Now at this point, the
Haskell guys will be jumping up and down saying things like "typeclasses in Haskell are only superficial similar to interfaces in Go" and "you really mean structural
subtyping, because Haskell typeclasses are so much more powerful". (For a bit on Haskell and OOP in general check [this out][haskelloop])
My response to that is you are probably right, but it doesn't matter much day to day, because all
that I am referring to is a way to codify a set of messages that are allowed. The question becomes whether this is possible to do statically or whether it must happen
at runtime. Go interfaces can most of the time be checked statically, Haskell has a much more powerful type system so it can do a shitload more at compile time.
For the purpose of writing code, what really matters is a clean way to define an interface. Objective-C has protocols which start to get a bit hairy because you have
to explicitly qualify the fact that a class conforms to a given protocol. Go does this really nicely by handling this implicitly based on the functions that have
been defined for a particular type. Long story short, the interface is the important piece about a type system that you really care about, so a good language must
make interfaces easy to handle, preferably statically. Ruby fails here, so do most other highly dynamic duck-typed languages. Objective-C kinda works here as long
as you don't bend the rules too much. Haskell always wins these battles, but I find Go's approach to be quite pleasant, even if strictly less powerful.

## Mixins vs Inheritance

One of the key issues with interfaces is how to implement a generic version of the functionality. How does one write code for the base case that is the same for most
objects that conform to that interface? Essentially, how does one write generic code? The traditional OO approach would be to define the interface through a base
class, put the generic version of the code in the base implementation, and then use inheritance to specialize those methods while maintaining the same API. What
happens when you want to inherit from multiple classes to build up your interface from a set of already defined interfaces? In C++ you can reach for multiple
inheritance, usually in the form of some mix of interface inheritance and implementation inheritance by using a mix of public/private inheritance. This actually works
alright if you know what you are doing. The alternative is templates which are the lifeblood of C++ generic programming. They are really quite nice when you get them
right, but they have their own quirks and issues. In Objective-C, you either drop into Objective-C++ to use templates, or you use standard inheritance. Often times
you want to add a set of instance methods to a group of related classes, and have the interface to those instance methods be defined by a protocol. There just is no
easy way to do this. Ruby gets this right with a feature that makes the language incredibly nice to work with, the mixin. I don't know enough about this, but I
believe there is a notion like this in Javascript as well. This is one of the main features of Ruby that allow you to reuse code, define thin interfaces, and not
get bogged down with inheritance. A language that forces you into inheritance to DRY up code is broken. The entire philosophy of OO is premised on using inheritance
to define is-a relationships. This is completely broken when you are just using it to share code. In Objective-C, I usually end up writing service objects, or classes
which are just a collection of static methods, and then use composition to delegate methods defined in a protocol to an inner object that is perhaps configurable.
This ends up with a similar consequence as the mixin, but it is significantly less clean and leads to unclear code as an object ends up being composed of too many
other objects.

# To be continued

I have a lot more to say on this subject, but I don't want to make this post take too long, and also I sometimes need motivation to keep writing. So I am going to push
a few of the sections onto another post. The next couple sections that I want to discuss are:

* Refactoring vs Building
* Static vs Dynamic

I will hopefully return to this topic very shortly and finish up my ramblings on this subject. Briefly glancing at this, I don't think this was that informative,
so I will probably also use the next post to summarize a bit and make some more definitive statements.


[haskelloop]:    http://www.haskell.org/haskellwiki/OOP_vs_type_classes
