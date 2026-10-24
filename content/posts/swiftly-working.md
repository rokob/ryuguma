---
layout: post
title: "Swiftly Working on iOS"
date: 2014-06-02 23:37:09
categories: code
tags: ["code", "objc", "swift"]
---

This post is my mind dump from a few minutes after "Swift" was announced. As I perused the [documentation][swift-docs]
I made a few observations about this new language. After spending a little bit more time with it, I have some
slightly different opinions, but most of these initial thoughts are still valid. I am not going to just put
these up here unedited, and then add a section at the bottom of further thoughts.

## First Thoughts

Swift is pretty sick. It borrows some syntax from Go including types after names which has really started
to grow on me, forced braces with conditionals and no parentheses required, and multiple return types.

Type annotations must be explicitly marked as nullable which probably gives the compiler
a big boost by being able to use the fast path for message sends that avoid nil checks.

String interpolation is HUGE, using Ruby for a while I can't believe people are not going
nuts with `stringWithFormat:`.

Switch statements that extend beyond simple integer types is amazing for different types of true
pattern matching. Still no pattern matching in function declarations which is what makes functional
languages (Haskell, Erlang, ML, etc.) really shine, but still better switch statements are better for
the world. No Implicit Fallthrough in switch statements also, i.e. they fixed their "goto fail;" bug
at the language level. Nice. Go also has this. Also assignments and where statements in case
statements is pretty nice.

Generics.

Array mutability is a bit broken: "Immutability has a slightly different meaning for arrays, however.
You are still not allowed to perform any action that has the potential to change the size of an immutable
array, but you are allowed to set a new value for an existing index in the array." I mean they go on to
say it provides optimizations, but calling it immutable and then saying you can mutate it is classic Apple.

Ruby style range syntax, i.e. 1...theEnd, is a nice addition. Strings are enumerable by character
which is also a nice convenience feature.

Much nicer function syntax, including external/internal parameters, better closure syntax which
means your functions that return functions no longer look like someone sat on your keyboard.

Enums which have associated values which are basically really fancy unions. A dubious feature,
considering their example uses QR codes.

Sadly Swift still supports classes with inheritance. But there are now structs that look like classes
but are value types which don't have any reference counting, so much easier to have real value objects
floating around. Also, proper constructor/destructor semantics called initializer/deinitializer. You
don't return a value from the initializer like you do in objc, and the deinitializer is called before
dealloc so you can have a consistent view of your object state. Wow.

Builtin KVO, "property observers", seems like an odd way to go.

Override the [] operator, who feels like writing a matrix class???

Unowned references which are described as non-optional weak references are just unsafe_unretained, why
for the love of god? Also this whole section of the reference will be the cause of the most bugs people
write. Two-Phase initialization combined with implicitly unwrapped optional references is just weird and
very error prone. But,

Closure Capture Lists!!!

The optional chaining support is pretty similar to some of the approaches to the [Maybe monad
in Ruby][ruby-maybe], it's a hard problem without biting the bullet and accepting that you really
are dealing with monads. Swift seems to take the approach that optionality (Maybe) will be the only
monad you care about.

Protocol conformance checks are much cleaner now: `dataSource?.incrementForCount?(count)` where
incrementForCount is an optional method in a protocol that dataSource conforms to.

Overall it seems like a huge leap forward from Objective-C, but not quite all the way to a truly
functional language. Pragmatism and runtime constraints probably win out. The most glaringly obvious
omission is anything related to making concurrency easier. Why would you make a new language today
without thinking about concurrency? At least something like channels or actors, something plz.
No new imperative languages!

## Later Thoughts

Some interesting features I found in the grammar defintion. For instance, you can override
operators and even add your own, so this becomes possible:

```objc
@infix func >>=<T: Any, K: Any> (m: T[], b: T -> K[]) -> K[] {
  var result = K[]()
  for x in m {
    for y in b(x) {
      result.append(y)
    }
  }
  return result
}

@infix func >><T: Any, K: Any> (m: T[], b: K[]) -> K[] {
  return m >>= {(t: T) -> K[] in b}
}

func lift<T: Any>(a: T) -> T[] {
  return [a]
}

func fail<T: Any>(_s: String) -> T[] {
  return []
}
```

Here we have the implementation of the Monad typeclass for a generic array type in Swift.
The Tuple type I somehow glossed over almost completely on my first pass through, but it
is actually pretty huge. It is what allows multiple return types which I saw initially,
but I didn't catch that it was a full blown type that basically gives you a fixed size
container with optional names for the values at the different positions. This is weird
that this exists because of the broken Array immutability because it makes me think that
the optimizations that exist for an immutable array is really just converting the object
to be a mutable tuple.

Where clauses in Generic type parameter lists is pretty huge for bringing real type
safe generic programming to iOS.

The most important part of the language that I missed is the bit on attributes.
This allows for quite a bit of language annotations that everyone I know has wanted
in iOS and Objective-C for a long time. For instance, being able to mark classes and
methods as final, and language support for lazy properties. These are pretty big deals.

Another thing that I have found is missing from the documentation is a clear discussion
of code visibility. If you create a new iOS app with the Swift templates in the new
version of Xcode, you will find that all of the classes your create are basically visible to everything in
your application. You don't have to import anything. This is terrible. It seems great
for a small project, but not being able to understand the dependency structure of
your project is pretty bad. I am assuming that there is something I am just missing,
but the docs are sparse on the topic of Modules. There is a construct, it just isn't
really clear what the deal is.

[swift-docs]:    https://itunes.apple.com/us/book/swift-programming-language/id881256329?mt=11
[ruby-maybe]:    https://github.com/bhb/maybe
