---
layout: post
title: "Eloquent Objective-C: Write Code That Looks Like Objective-C"
date: 2014-06-01 00:20:04
categories: code
tags: ["code", "objc", "eloquent"]
---

When I look at someone's code, I immediately make a value judgment about the likelihood
that their code will work and that the person knows what they are doing based on the
style of the their code. Basically, if the code matches my expectations for the language
and it is clear then I have a feeling that this person is probably legit. I know this is
not a great thing to do for some reasons, mostly that this should not be a sign of positive
quality but more a negative signal when the code is missing some of the qualities that
one expects. This is a similar filter to FizzBuzz. Most people who write code have no idea
what they are doing, they don't understand code design or architecture,
they have terrible naming and syntax usage, and they typically cannot solve the most basic
problems. Poor style is a strong indicator of a lack of skills in the same way as struggling
to complete FizzBuzz. This post is about what Objective-C is supposed to look like and why.

## Know the Style Guidelines

The most important set of guidelines for Objective-C that the community has accepted are
those handed down [from above][apple-guidelines]. You should read through all of these guidelines
at least once, if not multiple times as you encounter questions during development.
The set of things covered include:

* Code Naming Basics
* Naming Methods
* Naming Functions
* Naming Properties and Data Types
* Acceptable Abbreviations and Acronyms

For instance, these guides settle the debate that every language has to undergo about whether
it uses snake_case or camelCase. Objective-C has chosen the camel. Don't fight it, consistent
naming is incredibly important for writing code that is maintainable. The rest of the advice
in the guides is pretty straightforward and you should try to follow it as closely as possible.

There are however, some things that most people in the community have settled on that are
either not directly stated in the style guide, or which are actually contradictory. One example
stems from the following Apple guideline:

> Names of most private methods in the Cocoa frameworks have an underscore prefix (for example, _fooData ) to mark them as private. From this fact follow two recommendations.
>
> - Don't use the underscore character as a prefix for your private methods. Apple reserves this convention.
>
> - If you are subclassing a large Cocoa framework class (such as NSView or UIView) and you want to be absolutely sure
>that your private methods have names different from those in the superclass, you can add your own prefix to your
>private methods. The prefix should be as unique as possible, perhaps one based on your company or project and
>of the form "XX_". So if your project is called Byte Flogger, the prefix might be BF_addObject:

At Facebook, pretty much everyone used `_foo:` as the style for a private method. Some care
may have been taken by the more conscientious of the bunch when subclassing a system class to use
a different convention to avoid accidentally overriding a private method of a superclass. Part of this stems
from the lack of real encapsulation in Objective-C even though it makes you feel like it is there. Because of
the way method resolution works in the inheritance hierarchy, there is no such thing as a truly private
method. Some people choose to just name things differently inside the implementation and then hope no
one sends that message which is not declared in the interface. Others choose to use categories such as
`MyClass (Private)` to define private methods which are less than private for some other use cases, testing
being a common one. Others use a class extension to declare their private methods. This can be useful to see
them all in one place and to possibly have a better place for documentation. Some people go so far as to
use static C functions declared in the implementation file. This gives you probably the closest thing
to a real private "method", but it does not seem to be common practice in the community at large.
The modern LLVM compiler does not require you to explicitly declare methods within an implementation
in order to call them, so frequently you will just not see a declaration separately from the definition.

I have settled on a mix of static C functions, methods defined in an "Internal" header, and
then just properly named methods without a leading underscore which are just not declared anywhere
outside of the implementation. Each of these have different use cases. I rarely define many methods in subclasses
of system classes, so I rarely worry about name conflicts, plus the implementation looks much cleaner without
all those underscore prefixed methods floating around. I use the interface as the source of truth for what
is public so I don't really need the distinction within the implementation. Do use `pragmas` to separate
sections of your implementations so that you can have one clear "Public API" section

My use of static C functions is confined to pure functions which do not require `self`, for example

```objc
static NSArray* filterArrayWithBlock(NSArray *input, BOOL(^predicate)(id)) {
  if (!predicate) {
    return input;
  }
  NSMutableArray *result = [NSMutableArray arrayWithCapacity:[input count]];
  [input enumerateObjectsUsingBlock:^(id obj, NSUInteger idx, BOOL *stop) {
    if (predicate(obj)) {
      [result addObject:obj];
    }
  }];
  return [result copy];
}
```

This is just a toy function that would probably be better suited as a member of a category on `NSArray`
which adds functional programming primitives like `map`, `reduce`, `filter`, etc. But you can declare
pure functions like this within your implementation file, and then call it possibly passing in
instance variables as arguments. There is no way that a subclass can override this behaviour, plus you trivially
can guarantee that you are not modifying the state of your object because the C function only has access
to it's inputs, and you never should make `self` one of those inputs.

I use methods in an "Internal" header to expose some things for tests. This is usually a smell I know,
but sometimes it is the only bridge between tested and untested code before you go through a big refactor.
I see this often with async interfaces that need to expose some synchronization to make
tests reliable.

<br />
<h2>Dots and Brackets</h2>

If you are coming from Lisp then all the brackets everywhere should not make you too anxious, otherwise
you probably were a bit thrown off initially about all the brackets. The dot syntax is used in a variety of
other languages so it might innately feel like the right thing to use in many cases. Like all other parts
of this post, consistency within a codebase is more important than your preferences, so even if you disagree
with a certain use in a particular project, I suggest you be flexible rather than be "that guy". Nonetheless,
there is a correct style to use here. The dot syntax should only be used for simple property getter/setters,
and typically this usually means only outside of an implementation because you should be using instance
variables directly inside an implementation (the one exception being to remain KVO compliant, however since you
should never use KVO this is moot). The bracket syntax should be used for everything else. The only alternative
style is to never use the dot syntax and use brackets everywhere. So, using the dot syntax to
call a method is a bad thing.

<div class="incorrect">
```objc
@interface SomeClass : NSObject
@property (nonatomic, readwrite, copy) NSString *string;
- (void)doIt;
@end

// Somewhere else
SomeClass *someClass = SomeClass.alloc.init;
someClass.doIt;

// The next line is okay in isolation
[someClass setString:@"Nope"];
// but mixing the method call setter with the dot getter is inconsistent
NSLog(@"%@", someClass.string);

// WTF?
UIColor *myColor = UIColor.blackColor;
```
</div>

<div class="correct">
```objc
@interface SomeClass : NSObject
@property (nonatomic, readwrite, copy) NSString *string;
- (void)doIt;
@end

// Somewhere else
SomeClass *someClass = [[SomeClass alloc] init];

someClass.string = @"Yeah";
NSLog(@"%@", someClass.string);

[someClass doIt];
```
</div>

## Embrace Selector Keywords and Verbosity

Objective-C is a verbose language, method calls can easily take up a lot of editor columns:

```objc
- (void)sendAction:(SEL)aSelector toObject:(id)anObject forAllCells:(BOOL)flag;
```

that is not a bad thing if you use it to your advantage. The parts of a selector
that describe the argument is known as a keyword (this terminology comes from Smalltalk).
In many other languages, keyword arguments are the exception not the rule. Have you ever
seen a function call in C that looks like this:

```c
sendMessage(obj, 0, false, false, NULL);
```

Sometimes people will throw in comments everywhere they call a function like this to explain
what those random `false` arguments mean. This is terrible. This is partly just a product of
poor design, but in Objective-C you are forced to make it look like:

```objc
[obj sendMessageWithIndex:0 shouldReply:NO isHappy:YES callback:NULL];
```

## Do Not Use "new"

Use `[[SomeClass alloc] init]` instead of `[SomeClass new]`. Why? It is idiomatic Objective-C,
and you can't extend `new` to do more for you like `initWithStuff:`.

## Use Pragmas Liberally

There is a preprocessor directive called "pragma" which Xcode picks up to define sections
in the code navigator. The style is to use `#pragma` followed by the word `mark` and then
some text. You can place a single dash to create a horizontal rule in the code navigator.
If you put some text then it will show up as a bold header. It helps you to separate your methods
in to useful sections without adding a bunch of comments, and it nicely gets picked up by
Xcode.

```objc
#pragma mark -
#pragma mark NSObject

- (id)initWith...
{
  ...
}

#pragma mark -
#pragma mark UIAlertViewDelegate

- (void)...
```

## What Others Have To Say

If you Google for Objective-C style guides, you will find quite a lot of people have opinions. Most of
them are consistent with one another, and focus on syntax (i.e. how many spaces, tabs vs spaces, dots vs brackets), and spend
less time on semantics. All of the style guides that come out of Google for various languages are top notch,
and the [Objective-C one][google-objc] does not disappoint. Two other good ones come from the [NY Times][ny-times-objc]
and [Github][github-objc]. The most important bit of style advice is to choose something and be consistent
within a project. However, if you are stepping into someone else's codebase, adapt to how they do things,
don't waste your time arguing the merits of the style you used at your last job. The only caveat to this is
if there is clearly no consistent style, then work to get one enforced and the code converted as quickly
as possible. I have my own style that I use on personal projects, which differed from the style at Facebook,
but I always just kept the two separate. I think it is good to expose yourself to multiple styles because
you can pick up the little bits that work better from each and merge them into something better.

## And then Swift happened

So I was watching the WWDC keynote this morning and Apple announced a new langauge, Swift. This will be the
language for iOS and Mac development for the forseeable future. So all this talk about how to write
Objective-C is going down the drain. So although there is a right way to write Objective-C, it doesn't
matter much anymore. I'll have to start working on "Eloquent Swift"...

[apple-guidelines]:  https://developer.apple.com/library/mac/documentation/Cocoa/Conceptual/CodingGuidelines/CodingGuidelines.html
[google-objc]:       http://google-styleguide.googlecode.com/svn/trunk/objcguide.xml
[ny-times-objc]:     https://github.com/NYTimes/objective-c-style-guide
[github-objc]:       https://github.com/github/objective-c-conventions
