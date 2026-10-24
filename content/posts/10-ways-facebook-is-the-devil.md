---
layout: post
title: "10 ways Facebook is actually the devil"
date: 2014-07-04 15:33:27
categories: code
tags: ["life", "facebook", "clowntown"]
excerpt: There recently was a bit of controversy over a paper that was published by some researchers at Facebook. In the press there was mostly misinformation and alarmist pieces of yellow journalism. This post is about how my name was used to further the agenda of a disingenuous Wall Street Journal reporter by taking what I said out of context.
---

## Read This First (added later)

This post was never intended to be a justification of Facebook or of
the arbitrary experimentation that is done at all web service companies.
The underlying theme was to ask the question why are the people who are making
the most noise making that noise now? Other experiments and questionable
practices from inside Google, Amazon, and Facebook have been made public
before and the level of furor was nothing like this. This is a small fraction
of the news that the recent NSA revelations has gotten, but it is still
pretty close. That is something that many people have basically known was
going on for years and no one said anything about it until someone from
the inside brought out proof. Maybe that is metaphorically why this is
easier to hold on to and point at, but what I say below is more of a
question of why the people who claim
to care the most about this have not been louder. Maybe they have been trying
to and finally people are listening. Either way, from my point of view, this
public discourse on the ethics of experimentation seems new, and I am just
trying to say why now? I do understand why people are upset, I don't think
this particular test was unethical because of the minimal risk of real harm
being to done to participants which is generally accepted by university
internal review boards as one case where consent is unnecessary. This post
was never intended to say that something is okay (or not okay), it was only
intended to put the words attributed to me within the context that I said
them which I guess is even less clear than I thought.

### On the title:

Apparently it is non-obvious, but the title of this post is a jab at the way
"news" is presented nowadays.

### Disclaimer:

Anything contained within this post that resembles something that may or may not have
actually happened at Facebook while I worked there is purely coincidental and does not
constitute a breach of a non-disclosure agreement which may or may not exist between
myself and said company.

## The Short Version

A reporter reached out to me about this experiment that was the subject of a [recent paper][pnas-paper]
published by some members of the Data Science team at Facebook. I don't know how he found me originally,
but we talked a few weeks prior about Facebook in general which did not end up in the newspaper in any
way, so I did not think much of talking to him again. He told me he was just looking for background information
on a story about the data science team in general about how they work and some things related to how this
experiment was run. So I said that sounds reasonable and we talked for at least 45 minutes about testing
and experimentation, which led to a bunch of other stuff. If you know me or have talked to me you know
how I tend to meander around when talking about something. I thought I was even overly positive about
the work being done, talking about how everyone is trying really hard to make the product better
through understanding.

Because of all the talk about this experiment, I thought it was pretty reasonable to write something
more general about the data science team independent of just this experiment. He focused a little bit
on whether there was a review board for running tests and I honestly said no, not back in 2012, maybe now,
but I doubt it, I didn't think much of that, I tried to clarify that I wasn't really on the data science
team, that I worked on SI, so my experience with running tests was quite different as they were never
intended for publication.

So anyway, he told me their editors don't allow them to show things before they are published, but he told
me he could tell me the overall theme of what he was writing, but he refused to do it via email,
and I eventually felt like I expressed that if I cant see it then I don't think he should be
quoting me, which led to him saying well it is going to be published anyway and he thinks he
accurately captured what I had told him.

## The Real Context

All of the quotes from me are out of context, although they are pretty much actual quotes. Some of them
were twisted around and basically pieced together from two different sentences to sound like one, but
all of them got the tone and nuance completely opposite of what I actually said. I talked to the guy who wrote
that story for more than half an hour from which he pulled that handful of quotes. He definitely spun the words
I technically did say to further his ends of painting Facebook in a negative light. I am going to try and
speak here to how I actually feel about things and if you recognize some quotes you will see the context
from which they came.

While I was at Facebook, there was no institutional review board that scrutinized the decision to
run an experiment for internal purposes. Once someone had a result that they decided they wanted to
submit for publication to a journal, there definitely was a back and forth with PR and legal
over what could be published. If you want to run a test to see if people will click on a
green button instead of a blue button you don't need approval. In the same way, if you want to
test a new ad targeting system to see if people click on more ads and revenue goes up, you don't
need institutional approval. Further, if you want to see how people react to changes in certain
systems like the content in news feed, you don't need approval for that experiment, even if it is
just to help inform an internal ranking system. Nonetheless, people aren't just running experiments
willy-nilly. There are only a small subset of people on the data science team who have social science
research as part of their primary job duties. Everyone else would not run an experiment like the one
people are talking about because it would not help the product they are working on. Those who do run
such experiments, have very high ethical standards and experience running experiments both inside and
outside academia. They are trying to make Facebook a better product for over a billion people, and the
way they know how to do that is to understand human behaviour and the human-computer interaction.
Experiments are run on every user at some point in their tenure on the site. Whether that is seeing
different size ad copy, or different marketing messages, or different call to action buttons, or having
their feeds generated by different ranking algorithms, etc. The fundamental purpose of most people
at Facebook working on data is to influence and alter people's moods and behaviour. They are doing
it all the time to make you like stories more, to click on more ads, to spend more time on the site.
This is just how a website works, everyone does this and everyone knows that everyone does this, I
don't see why people are all up in arms over this thing all of a sudden.

I personally never collaborated with outside researchers while they were in academics, I worked
with a couple while they were Facebook employees on sabbatical, so the normal standards of data access
with outside academics did not apply. The decision to work with an outside researcher does go through
some scrutiny but it is not centralized in anyway, for instance any team that knows someone in academics
that could help solve a problem that Facebook has or better understand something can choose to reach
out albeit with a friendly heads up to the academic partnership team, legal, and PR. No user identifiable
information is ever revealed to external researchers, but there is a decent amount of anonymized data
that is shared for the express purpose of helping researchers understand human behaviour. For example,
[this event][compassion] is an example of an event that has happened several times which is basically
a joint effort between internal and external researchers to understand how Facebook and social networks
in general can be more compassionate. I am a bit taken aback by the fact that the most recent paper has
gotten as much press as it has, when the work done as part of the compassion research days has never been
mentioned. Some of these papers are based on experiments that influence people's behavior in similar ways,
but I guess they do not have as much cachet for whatever reason.

There has always been a lot of ethical discussions around experiments being run. From the simplistic,
i.e. will this be disruptive in a way that would simply annoy users, to the more complex, i.e.
is the perceived benefit from the understanding that we will get from this test worth the potential
negative effect on users as well as any potential negative PR from a misunderstanding of the
experiment. Every data scientist at Facebook that I have ever interacted with has been deeply
passionate about making the lives of people using Facebook better, but with the pragmatic
understanding that sometimes you need to hurt the experience for a small number of users to help
make things better for 1+ billion others. That being said, all of this hubbub over 700k users like
it is a large number of people is a bit strange coming from the inside where that is a very tiny
fraction of the user base (less than 0.1%), and even that number is likely inflated to include a
control group. It truly is easy to get desensitized to the fact that those are nearly 1M real people
interacting with the site, and it is something that people constantly are trying to remind everyone
of when working on the product. There is a lot of work to try to increase the empathy towards the
people using Facebook internally, so the backlash against Facebook is always doubly painful because
these really are good people trying to make things better. The PR team maybe does not get in front
of these issues enough, but the only thing I see changing from this is not whether similar experiments
will be run, but rather will they be published. Similar experiments have been and will continue to be
run, but you probably just won't see a paper about it anymore. That is a real shame because Facebook
and the data it has has been able to advance social psychology by quantum leaps over the past decade
and this furor will undoubtedly slow that advance.

I don't mind being quoted or not, in fact I would probably prefer not to be as I really don't want to
be putting my name out there attached to something that could be misconstrued by my friends who still
work at Facebook as me trying to go behind their backs and get publicity out of something bad happening
to them. I quit Facebook in April of this year for a variety of reasons, but I still think they are doing
good things in the world, so above all I don't want to come across as bad mouthing them. That being said,
the plain truth is that the experiment that generated this paper is over 2 years old, so any discussion on
either side about a review board is missing the point. Facebook as a corporation has to be weak in what
they say, but this study was the right thing to do, it was an interesting behavioural question, the
experimental design was solid, and the results were pretty clear and interesting. Furthermore, any
experiments are covered by the TOS and not only that, advertising is equivalent to trying to alter
your mood and behaviour. Every ad based company (Facebook, Google, Yahoo, Twitter, etc.) exists to
alter how you perceive the world. The fact that a human stood up and said that is what they are doing
is the only thing that is different here. So I have no idea why people are talking about this in the first place.

## Further thoughts

I understand reading this how I was really an idiot saying anything like this to a reporter.
It is pretty clear that anything sounds bad out of context, but I really said some bad things out
of context. I did give an example to the reporter of something that I worked on as an example of
how people did care about ethics even in situations where no one would ever know about the test.
This was a further mistake as it may or may not have violated some agreement that I may or may not
have about what I can say about what went on internally. The part that annoys me the most about
the article as it is written is how ridiculous I sound. Everyone I know who actually knows me who
reached out to me, basically said the same thing, it sounds like you got duped. It is clear if you know
me how I was represented is not accurate, but most people don't know me, so that sucks.

Consider reading the following articles for more reasonable opinions about this whole mess:

* [Refriending Facebook][refriending]
* [On the ethics of Facebook experiments][ethics-of-fb]

## Emails

Here are the emails between myself and the WSJ reporter if you want even more detailed information
around the context of the day that this story was published. The majority of our disucussion
was on the phone, so I don't have word for word transcripts of that. Note, some lines have been
removed from these emails as I did not feel they actually added any extra information, these
are in fact not the exact emails, but nothing has been altered other than removing a couple lines.
The lines I removed had to do with the fact that I asked to be quoted as "Dr. Ledvina" instead
of "Mr. Ledvina" as well as the fact that I reached out to other former data scientists but
told the reporter that they did not want to speak as they thought it might be seen as going
behind the back of the people who currently work at FB. The only question I asked these people
was "A guy from the WSJ wants to talk to data scientists from FB, you interested?" They meant
simply that talking to a reporter at all was going behind someone's back, not that they nor I
actually knew the actually gist of the story that was to be written.

* From: Me
* To: reed.albergotti@wsj.com
* July 2, 2014 11:41am

>I don't have time to talk on the phone, but if you have any questions you want answered
>I would be happy to do so via email. I only am willing to say anything more if I can see
>the entirety of whatever is written prior to publication.
>
>I should also clarify that working as a data scientist at fb and being on the data science
>team, the one that publishes papers including the most recent one, are two distinct things.
>I was not on the data science team although I was a data scientist. There is a bit of a
>misconception both internally and externally about the distinction, so whatever you want
>to make of that is up to you.

* From: reed.albergotti@wsj.com
* To: Me
* July 2, 2014 11:48am

>ok, thanks for getting back to me. I’m planning on using more of the material we talked
>about the other night. I can’t send you any article in its entirety before it is published,
>as that’s a hard and fast rule at the WSJ, and really any newspaper.
>
>What I almost always do is talk through all the points and context of the article with
>the people I quote, so they aren’t surprised in any way by the article and understand the
>key points of the piece. If you’d rather not do that, I understand.
>
>I’m not totally clear on your point about the data science team. Did you work with the
>data science team or were you totally separate from that team?

* From: Me
* To: reed.albergotti@wsj.com
* 12:02pm

>There was a larger organization known as Analytics, contained in this org were analysts
>and data scientists that worked on different product areas, as well as a team known just
>as 'Data Science' which also has data scientists. I was not on the Data Science team,
>which has about 10 people focused mostly on social science research and writing papers
>as part of their job while the rest of their job is the same as the rest of the data
>scientists working with different products.
>
>It is a bit convoluted and comes from historical jockeying for power.
>But it is just to clarify that there is a team formerly led by Cameron Marlow that
>exists someone differently from other data scientists although the majority of the
>time the work is the same.

* From: reed.albergotti@wsj.com
* To: Me
* 12:07pm

>Thanks for that. Facebook seems to refer to the larger analytics team as the “data science”
>team, which they say has almost 40 people now. It also may have changed since you left.
>
>What about talking through the article with you over the phone? I understand you don’t
>want to give me additional information, but I’d prefer to brief you on everything we might
>use from our previous conversation before it runs in the paper.

* From: reed.albergotti@wsj.com
* To: Me
* 2:12pm

>There’s some chance this story could run on page one of tomorrow’s paper.
>If you have a free minute, I’d love to run it by you for fact checking.

* From: reed.albergotti@wsj.com
* To: Me
* 3:28pm

>Facebook’s public relations team is saying that the experiment you worked on
>that tested how users react when they’re locked out of the Facebook system never happened.
>I told them I discussed it with you in detail and I’m ready to publish. But if
>you want to discuss, please give me a ring in the next half hour or so.
>I’m on cell at 212-327-3430.

* From: Me
* To: reed.albergotti@wsj.com
* 3:59pm

>I don't really have time to take a call basically any time today. All I can say to
>that is they are lying or more likely there is some miscommunication about what you
>are asking about. I did this as part of the site integrity team, it was not a piece of
>research done as part of the Data Science team with the intention of writing a paper or
>anything like that, so if they just asked someone on that team they never would have
>heard of it. There was no intention of the test being made public, so from a PR standpoint
>I could see why they don't know about it.
>
>I am starting to think this is not the kind of information that I should be sharing
>publicly. I don't really know the details of the non-disclosure rules that I am still
>under, although I thought it only related to proprietary technology and unreleased
>products. I really would need to now the whole gist of what you are trying to say
>about it before I would feel comfortable signing off.

* From: reed.albergotti@wsj.com
* To: Me
* 4:02pm

>Ok, I totally understand where you’re coming from, and I know you’re busy.
>Literally the fastest way to do this would be to get on the phone.
>I just ironed out the details on that site integrity thing with Facebook, so I am all good on that.
>
>But there will some quotes from you in the story that basically elaborate on
>what you were already quoted saying in the other story.
>
>The central theme of the story (other than being a general piece
>about Facebook’s data scientists), focus on the fact that there was
>really no formal review process for these studies. You were really
>helpful in illustrating that. now, there are more controls.
>
>I really think we could both benefit from a very brief phone discussion. I know you’re busy.

* From: reed.albergotti@wsj.com
* To: Me
* 6:26pm

>Hey, if you get any kind of response from Facebook, I want to hear about it, ok?
>I really appreciate you talking to me. I know it didn't turn out to be that
>pleasant of a day, but I think you really added a lot to the public's knowledge
>about this and I really respect that.

* From: Me
* To: reed.albergotti@wsj.com
* 8:42pm

>You really took what I said out of context to put your own spin on things that
>was very different than the sentiment that I expressed. I understand now why no
>one wants to be honest with a reporter.

* From: reed.albergotti@wsj.com
* To: Me
* 9:16pm

>Sorry you feel that way.


### Copyright

All rights reserved. No part of this post may be reproduced or transmitted in
any form or by any means, electronic or mechanical, without the express written
consent from the author.

In other words, stop quoting me and stop quoting this blog post to further
your agenda. If you want to do so, you have to ask me first. This includes
not-for-profit as well as for-profit uses.


[compassion]:    https://www.facebook.com/events/230652223766114/
[pnas-paper]:    http://www.pnas.org/content/111/24/8788.full
[refriending]:   http://www.technologyreview.com/view/528756/refriending-facebook/
[ethics-of-fb]:  http://m.washingtonpost.com/blogs/monkey-cage/wp/2014/07/03/on-the-ethics-of-facebook-experiments/
