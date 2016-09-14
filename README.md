# What is Syncer

`Syncer` is a lib designed primarily for server components conception. 

In Golang, there is no easy way to do a "push to channel A if still open" operation.

This lib gives the primitives to patch this need.

In particular, it let's you quite effortlessly design functions in your component that any caller can call, whether related channels are open, closed, not yet open, in the process of being ready, etc...


See the unit tests as a typical use-case design.
