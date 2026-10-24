---
layout: post
title: "Why settle for one ugly API when you can have two?"
date: 2014-05-06 19:53:28
categories: code
tags: ["ruby", "code", "api"]
---

In my new job I am working exclusively in Ruby at the moment. What that means for this blog is that most of
the code I talk about will be in related to Ruby. I actually have found Ruby to be a beautiful language
as OO languages go. One of the things that has really blown me away is how similar Objective-C and Ruby are.
It is not that much of a surprise once one takes a step back and thinks about the design of the languages,
but Ruby wins in almost every regard from a design perspective. I don't really want to talk about that today
though, I have plans for a series of posts comparing different patterns in the two languages. This is going
to be a Ruby post though.

I was working on a problem where I needed to use [Mechanize][mechanize] to script filling out a form. Actually,
I have 50 subtly different forms that need to be filled out at different times. Fifty should remind you of the
number of states in the US so you might see the domain where this comes from. Anywho, Mechanize has an alright
API for interacting with an individual page or form on a page, but like any general tool, it does not present
the ideal API for my specific purpose. One approach is to just use the raw API directly in the place that you
need it and hope that you don't need to use something similar anywhere else. This is probably the way to go
if you cannot imagine replacing the lower level API (Mechanize in this case) with any other library or if
the semantics are basically the same between your desired use and what API is already presented.

My case involved violations of both of these conditions. The first condition is violated because I know for
a fact that I will need to use [Capybara][capybara] for a small subset of the sites I am interacting with
because of their use of Javascript. For the second condition, I wanted to have a method like
`check_radio_button(form, data)` where `form` is some representation of the form on which the radio button
lives and `data` is sufficient information to select a single radio button. Mechanize presents an API that
lets you find a radio button and check the button, but these are two method calls. The gain from moving to
one call isn't huge, but the real benefit is semantics. My API is more clear about what it is doing, and not
only that, I don't care how it is doing it, I just know that a radio button is getting checked. I have a need
for other related behaviours that take even more effort using the Mechanize API.

Okay, so I am convinced that I need to wrap the given API with my own. It is important to actually think about
this point though before just wrapping the API. When someone creates an API, they put some thought into it
(hopefully), they think about how people are going to use it, and they ideally iterate on it as the API is
used over time. Your wrapper is likely to be as shitty if not shittier because you are just quickly trying to
use the thing that someone else has already thought a lot about. Maybe you have a flash of inspiration and you
come up with the perfect API for a particular use case, but that never happens. Realize that you are usually
trading a shitty general API For a shitty idiosyncratic API. In my case that is kinda what I want, but YMMV.

I read [Metaprogramming Ruby][meta-ruby] when I realized that I would be working in Ruby soon because I'm a
sucker for esoteric language features. I remembered there being a discussion in there about using `instance_eval`
to allow you to use an API without having to call a function directly on an object because of the implicit self
in Ruby. I used this idea to clean up the API wrapper that I eventually came up, but I had to go one step
further because I needed to pass an argument into my block to really make this wrapper shine.

I have a service object, which is some low level library that has an API that already solves some problem I have.
It also has some kind of type of object that it will yield to you. That object is what you actually are going
to interact with.

```ruby
class Service
  def initialize(config)
  end

  def with_object(criteria)
    yield ServiceType.new
  end
end

class ServiceType
  def alpha(a)
    1 + a
  end

  def beta(a)
    2 + a
  end

  def gamma(a)
    3 + a
  end
end
```

The traditional way of using this API would just be to do:

```ruby
service = Service.new('some config')
service.with_object(a: 'something') do |obj|
  obj.alpha(42)
  obj.beta(13)
  obj.gamma(66)
end
```

That doesn't look so bad, but think about if calling `alpha` then `beta` in that
order meant to do something specific that has a particular meaning to you. Then
wrapping up some of these calls into your own interface would be nice:

```ruby
class Interface
  def add(obj, a)
    obj.alpha(a) + obj.gamma(a)
  end

  def times(obj, a)
    obj.beta(a) * obj.beta(a)
  end

  def interact_with(config, &block)
    # keep reading
  end
end
```

This is the interface you want to expose, whatever it means, it takes the service
object as an input and possibly some other parameters and does whatever you need.
Why does it takes the service object as an input? You could actually stash the
service object in an ivar because of where we get it as you will see later, but
I prefer this because it makes this interface pure. That is, you can test these
with no work because they are pure functions.

Now I am going to skip over that one method stub and just show you how you use this:

```ruby
class Worker
  def do_work
    Interface.new.interact_with('config data') do |obj|
      x = add(obj, 1)
      times(obj, x)
    end
  end
end

puts Worker.new.do_work
```

We have another class that uses this interface. I wrote it as a class just to wrap
this up into something self contained, but there is no need for that. Basically,
you call `interact_with` with your configuration data and then use a block
to use your new interface. Inside this object you can call instance methods on
your interface class without specifying an instance. You are yielded the service
object so you can pass it through to the pure functions you have defined. How do
we accomplish this?

```ruby
class Interface
  def interact_with(config, &block)
    Service.new(config).with_object do |obj|
      instance_exec obj, &block
    end
  end
end
```

It is really simple in the end. You just use `instance_exec` which allows you to
execute `block` in the context of the `Interface` class and pass the `obj` as
a variable to the block.

You can "clean" this up a bit if you don't care about pure functions:

```ruby
class Interface
  def add(a)
    @obj.alpha(a) + @obj.gamma(a)
  end

  def times(a)
    @obj.beta(a) * @obj.beta(a)
  end

  def interact_with(config, &block)
    Service.new(config).with_object do |obj|
      @obj = obj
      instance_eval &block
    end
  end
end

class Worker
  def do_work
    Interface.new.interact_with('config data') do
      x = add(1)
      times(x)
    end
  end
end
```

It probably depends on the specific problem which version makes sense. I just threw
this together today when I was trying to get my Mechanize wrapper to be easier to
use. I am not 100% yet whether this is a good pattern, or if the use of `instance_exec`
is a good use of magic or not in this situation. I guess time will tell with this
one, but I had a good time playing around with Ruby to get this working.


[mechanize]:   https://github.com/sparklemotion/mechanize
[capybara]:    https://github.com/jnicklas/capybara
[meta-ruby]:   http://pragprog.com/book/ppmetr/metaprogramming-ruby
