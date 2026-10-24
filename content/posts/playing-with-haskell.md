---
layout: post
title:  "Playing with Haskell"
date:   2014-01-12 23:17:19
categories: code
tags: ["haskell", "moving"]
---

Before I get in to the coding part of this post I figured I would talk a little
bit about what is going on because I have not posted in a little over a week.
Nothing too crazy has happend lately, although we did go see one apartment
place, Lauren liked it a lot, I also liked it, so we are moving next weekend.
We are going to pay pretty close to the same rent (which is way too high still)
but get 2 bedrooms and 2 bathrooms instead of 1 bedroom and 1 bathroom. We will
also be in walking distance of Lauren's work which will dramatically change our
commuting situation. Right now we end up having to coordinate using our car and
when we go to and from work because I am often lazy and don't feel like riding
my bike. This new situation will allow her to walk and me to walk to the train
station if I am feeling lazy, so no more annoying coordination. We will also be
getting tv again and legitimate internet which will be a huge win. The downside
is that we have to go through the moving process which is always just terrible.
We are getting pretty good at it considering this will be each of our roughly
tenth place. We went through all of our stuff today and threw out 10+ garbage
bags worth of random crap and clothes we don't wear anymore. The troubling part
being the place does not look that much different. It is amazing how easy it is
to accumulate crap, I know I will continue to accumulate it most likely, but it
is good to clean things up now and then. I will be more willing to do this type
of cleaning in the future because it felt good to unburden myself from a lot of
junk. Now, on to the code.

I have been getting back into Haskell again recently and for whatever reason I
like it a lot more this time around. I have always been much happier coding in
functional languages, but I have spent most of that time working with Erlang
which is basically untyped whereas Haskell is strongly typed. I don't think I
really appreciated the benefits of such a full featured type system until I
spent a lot of time working in Objective-C. If you have ever spent time trying
to prevent shooting yourself in the foot in Obj-C then you know what I am
talking about.

Interestingly enough I have been working on a side project that involves both
languages. I have been prototyping the concept of generating Obj-C with Haskell
to make a lot of the boilerplate that goes into creating an app less annoying,
but more importantly making it declarative and statically verified. Anyway, I
have been spending some time playing around with parsing in Haskell, which
leads to [Parsec][parsec]. Just for fun, I wrote a little parser for JSON just
to get my feet wet.

I am going to walk through it because there is only a small amount of code
which is pretty interesting. First the preamble because I hate reading code on
blogs which isn't easy to get running.

```haskell
import Control.Applicative hiding ((<|>), many)
import Text.ParserCombinators.Parsec
import Text.ParserCombinators.Parsec.Language (haskellDef)
import qualified Text.ParserCombinators.Parsec.Token as P

lexer = P.makeTokenParser haskellDef
brackets = P.brackets lexer
braces = P.braces lexer
stringLit = P.stringLiteral lexer
naturalOrFloat = P.naturalOrFloat lexer
commaSep = P.commaSep lexer
symbol = P.symbol lexer
colon = P.colon lexer
```

These are some pretty standard imports, and then we define some simple parsers
at the top level of this module that we will use to build up the parser.

The entry point into the parser we will call `json` and which will recognize
the two top level types that make up the json spec:

```haskell
json :: Parser Integer
json = (object <|> array) <* eof
```

Okay so this makes a little less sense if you know what is going on here. We
are defining a parser which returns an `Integer` because for fun I am going to
make this thing sum up all of the numbers that show up in the input. This is
just because we have to do something meaningful, but I don't really feel like
doing too much more with it. So with that in mind, let's go through what is
going on here. JSON is either an object or an array. We need the `<* eof` to
say basically that we want to parse all of the input and fail if there is any
trailing characters that do not properly parse. So basically
`{"key":1},[1,2,3]` should not parse as valid, but without the extra bit it
would parse just ignorning the `,[1,2,3]`. Ok well what is an object?

```haskell
object :: Parser Integer
object = summedListOfStuff braces keyValue <?> "object"

summedListOfStuff :: (CharParser () [[Integer]] -> CharParser () [[Integer]])
                  -> Parser Integer
                  -> Parser Integer
summedListOfStuff f v =
  do
    vs <- f . commaSep $ (many1 v)
    return (sum . concat $ vs)
```

An object is key-value pairs between a set of braces, and because we are trying
to sum up the numbers we find we use an auxiliary function to create a parser
that does just that when given the outer delimeters and a parser. It turns out
that array is very similar

```haskell
array :: Parser Integer
array = summedListOfStuff brackets value <?> "array"
```

Basically the same thing just instead of braces we have brackets and instead of
key-value pairs we just have values. This highlights the declarative nature of
combinator parsers where you just describe what something is and out comes a
parser that does what you want. So as long as we define what a key-value pair
is and what a value is then we are done.

```haskell
keyValue :: Parser Integer
keyValue = stringLit >>= (\_ -> colon >>= (\_ -> value)) <?> "key-value pair"

value :: Parser Integer
value =
  do
    try stringLit
    return 0
  <|> do
    e <- naturalOrFloat
    case e of
      Left  i -> return i
      Right f -> return (truncate f)
  <|> do
    symbol "true"
    return 1
  <|> do
    symbol "false"
    return 0
  <|> do
    symbol "null"
    return 0
  <|> try array
  <|> object
  <?> "value"
```

So for key-value pairs I was playing around with using bind instead of the do
syntax, you can just ignore that. A key-value pair is a string literal followed
by a colon followed by a value, pretty clear from the definition that is what
is going on.

Then what is a value? Well, it is a string, a number, an array, an object, or
one of the literals: `true`, `false`, or `null`. This is where we get the
actual integers that we end up summing up. I decided to make strings, `false`,
and `null` equal to zero, `true` equal to one, integers equal to themselves,
and to truncate floats. In the end this allows us to reduce the whole JSON
object to a single integer.

Technically this parser does not
parse to the exact spec. For instance `stringLiteral` will not match strings with
unicode code points of the form \uXXXX directly in the string. This would not
be too difficult to fix but I don't particularly care for this toy. Also
naturalOrFloat does not exactly match the number type in the spec but again
whatever.

This gets us to the following in GHCi:

```
λ> parse json "" "[1,2,3]"
Right 6
λ> parse json "" "{\"key\":[1,3,5],\"keyz\":[{\"arrKey\":10}, 1, 4],\"whoa\":null}"
Right 24
λ> parse json "" "{\"key\":1},{}"
Left (line 1, column 10):
unexpected ','
expecting end of input
```

[parsec]:    http://legacy.cs.uu.nl/daan/parsec.html
