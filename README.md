# What is Syncer

`Syncer` is a lib designed primarily for server components conception. 

In Golang, there is no easy way to do a "push to channel A if still open" operation.

This lib gives the primitives to patch this need.

In particular, it let's you quite effortlessly design functions in your component that any caller can call, whether related channels are open, closed, not yet open, in the process of being ready, etc...


See the unit tests as a typical use-case design.

# Behind the hood

When created, a syncer is not yet `active`; It checks its activity, and calls
IfClosed when not active.

Open() creates a semaphore of n boolean resources. `n` represents the number of
IfOpen functions that will be executable in parallel. This semaphore is implemented as a buffered channel of booleans, called `pool`.

Each time IfOpen is executed, a boolean resource of the semaphore is booked, and released after IfOpen execution.

Close() creates a disciplined way to get out of this system, and switch back from a `opened` status to a `closed` status. A go routine, which we will call "finisher routine",  fetches All `n`resources from semaphore: once it is done, we are sure that no IfOpen operation is ongoing, and none will be executed until resources are released.
At this stage, the finisher routine closes the the pool. As immediate effect, IfOpen Is now able to continue; however, from now on, resources fetched from pool are `false`, which indicates that IfOpen should not be executed. In view of these
`false`resources, IfOpen therefore returns immediately without execution of the callback.


The semaphore `pool`, limited by `n`resources, is not considered here as a way to limit the number of parallel queries; It is used as way for the finisher routine to block temporarily operations, and send an unambiguous signal to IfOpen. 

Typically, IfOpen callback will push data to a channel A, and therefore allow parallel calls when the syncer is open. 

Function:
Push(msg string, stream chan string)
    |
    |-> IfOpen : stream <- msg
    |
    |-> IfClosed: return

MainServer:
...
    s := NewSync(10)
    s.Open()

    for {
        select {
            msg:= <- stream:
                //do something with msg
            <- done:
                //we do not break the loop listening on stream
                //before being sure that no messages are being or going to be
                //processed
                s.Close()
                s.WaitClosed()
                return
        }
    }
...
