# Markdown server
# Using markdown, templates and JIT to promote laziness

### Brief
Create something that requires minimal effort to use, and provides only the features you actually use.

Jekyll is great but you have to be bothered to compile after you change something. Why not be lazy and only compile it when someone wants to look at it. There are also certain things that are just better when dynamic, while it is possible to write a search system in client side JavaScript, it isn't very practical.

### Things it is not
A script for generating a bunch of static files; unless you use it like that.

A markdown implementation; I borrowed one of those.

A server side template system; I'm using the built in one.

A perfect solution to everything; unless your universe consists merely of things I'm interested in.

### Things it could be
A blogging platform; when I add rss/atom feeds.

A podcast site; you would need a load of front end work but its doable.

A company website; markdown is so easy the execs could do it.

### Features

- Just in time markdown and template compilation
- css, js and img directories routed

### Todo

1. RSS/atom
2. Caching

### Odd things
It uses extended attributes for the keyword tags, so this will only work on file systems that support them.

### License

This project is licensed under the GPLv3. 
