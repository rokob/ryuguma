---
layout: post
title:  "The New Blog"
date:   2013-12-30 19:11:00
categories: update
tags: ["blog", "jekyll"]
---

Two years ago I said that I was going to blog every single day as my New Year's
resolution. I managed 168 days and then nothing but silence. I am not really
sure why I just stopped so suddenly, one possibility being that I used to use
the commute home to do my writing, and my last entry was the day we moved out
of San Francisco which altered my commute considerably. Whatever the reason, I
have not written for the past 18 months or so and that is too bad. The reason I
wanted to write every day was so that I could look back and see how I was
feeling or what I was thinking about during certain periods of time without
having to try and remember. The last 18 months have been decently crazy and it
would have been really nice to see my thoughts during that time.

I also really wish I had access to those 168 posts but that is partially the
story for this new set-up. I got an email from Webfaction, my hosting provider,
telling me that the machine that my site was dying and thus they were going to
migrate me to a new server. They were telling me because this meant that the IP
address for my A record would need to be updated, but they said all of my data
would be moved over without me having to do anything. As it turns out, when I
finally got around to checking in on the new machine and changing my A record,
I noticed that all of my files were gone. The directory structure was all
there, but just no files. Weird. The code was backed up because I have it in a
private Bitbucket repo, but what was not backed up was the CouchDB files where
my blog posts were stored. So I sent an email to Webfaction asking about this
and seeing if they could restore my data from their backups. Here is their
response:

>I am sorry but we were not able to recover your files from our backup system.
>Our system only keeps up to 10 days of backups and since it has been over 30
>days there is no way to recover your files.
>
>Regarding backups, we do make daily backups, but we do not guarantee their availability.
>Sometimes problems with the backup servers can prevent backups from being made
>successfully. Like all reputable web hosts, we do recommend to all our customers that
>they make their own backups independent of ours for any mission-critical data that they
>cannot afford to lose.

Well that is just clowny. I understand it I guess, but still it is ridiculous
for a hosting provider to blow away all your files and then not be able to
recover them from their backups. Yes I should have backed up my DB but I never
got around to it so I have to pay the price for that.

I also tried to rely on the [Wayback Machine][wayback] and Google caches to try
to restore my blog posts, but I only managed to get 3 out of the 168. I will be putting
those up here soon and back dating them, they are all stupid, but what the hell.
I actually wrote a tiny script to scrape the Wayback Machine which is how I got
the 3, but it did not have archives for anything else.

All this led me to the decision to get off
Webfaction and try something new. I also decided to try to incorporate the
backup of my posts in a natural way. This led me to [Github Pages][github-pages]
and [Jekyll][jekyll]. I considered going to Hakyll route which I read about [here][blaenkdenum],
but in the end I decided to take some of the ideas from [blaenk's][blaenk-gh] set-up
and just mix it in with Jekyll. In particular, I have this blog set-up using
Github pages with two different branches, `master` and `source`. The `source`
branch has all of the actual code and content that Jekyll needs to create
this site, whereas the `master` branch just has the generated output. I use a
shell script to run Jekyll, perform a commit, and push the new content to
the right branch. This was copied directly from the aforementioned site.

One interesting thing that he did which I wanted to copy was to leverage
the fact that I am using `git` to have an automatic way of tracking the
history of each post. On his site he used a single commit hash for
every single post when he was using Jekyll because it requires you to regeneate everything
every time. One nice think about Hakyll is that you only need to recompile
what has changed, so it is easy to get a git commit hash for each post
that only changes when the past actually changes. I wanted the best of both
worlds so I modified his commit plugin to the following:

```ruby
module Jekyll
  class Commit < Liquid::Tag
    MATCHER = /\A(\S+)\Z/
    def render(context)
      markup = @markup.strip
 
      page_file = "#{context.registers[:site].source}"
      file_path = "#{context.environments.first["page"]["path"]}"
      page_file += "/#{file_path}"
 
      sha = `git log -n 1 --pretty=format:%H -- #{page_file}`
      message = `git log -n 1 --pretty=format:%s -- #{page_file}`
 
      history_url = "<a href='https://github.com/rokob/rokob.github.com/commits/source/"
      history_url += file_path
      history_url += "'>History</a>"
 
      commit_url = "<span class='commit' title=\"#{message}\">"
      if not markup.empty?
        repo = markup.match(MATCHER)[1]
        commit_url += "<a href=\"https://github.com/#{repo}/commit/#{sha}\""
        commit_url += " title=\"#{message}\">#{sha[0...8]}</a>"
      else
        commit_url += "#{sha[0...8]}"
      end
      commit_url += "</span>"
 
      history_url + " -- " + commit_url
    end
  end
end
 
Liquid::Template.register_tag('commit', Jekyll::Commit)
```

The trick here is that I used the fact that `git log` can tell you the last
commit when a particular file changed, which is exactly what I want. I just
have to do some trickeration with the `context` variable that is passed in
to get a file path for the current post, but after that it is pretty much
the same as his original plugin.

Jekyll is a pretty nice tool, I can't imagine I would outgrow it for quite some
time (where outgrow would mean generate too many pages to make the static site
unwieldy or desparately need server side logic).

I am going to try to post on here more frequently and not try to force myself
into writing everyday which will ultimately lead me to want to stop writing
if I miss too many times. I already have some ideas for a few posts that will
be meant to fill in the gaps of what I have been up to for the past 18 months,
and then probably some even more boring stuff.

[jekyll]:    http://jekyllrb.com
[github-pages]: http://pages.github.com/
[blaenkdenum]: http://blaenkdenum.com/posts/the-switch-to-hakyll/
[blaenk-gh]: https://github.com/blaenk/blaenk.github.io/
[wayback]: https://archive.org/web/
