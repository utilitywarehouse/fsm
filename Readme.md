FSM
===

FSM provides a lightweight finite state machine for Golang. It runs allows any number of transition checks you'd like the it runs them in parallel. It's tested and benchmarked too.

## Install

```
go get github.com/utilitywarehouse/fsm
```

## Usage

```go
package main

import (
    "log"
    "fmt"
    "github.com/utilitywarehouse/fsm"
)

type Thing struct {
    State fsm.State

    // our machine cache
    machine *fsm.Machine
}

// Add methods to comply with the fsm.Stater interface
func (t *Thing) CurrentState() fsm.State { return t.State }
func (t *Thing) SetState(s fsm.State)    { t.State = s }

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine. While not
// required, it's something I like to have.
func (t *Thing) Apply(r *fsm.Ruleset) *fsm.Machine {
    if t.machine == nil {
        t.machine = &fsm.Machine{Subject: t}
    }

    t.machine.Rules = r
    return t.machine
}

func main() {

    some_thing := Thing{State: "pending"} // Our subject
    fmt.Println(some_thing)

    // Establish some rules for our FSM
    rules := fsm.Ruleset{}
    rules.AddTransition(fsm.T{"pending", "started"})
    rules.AddTransition(fsm.T{"started", "finished"})

    errs := some_thing.Apply(&rules).Transition("started")
    if err != nil {
        log.Fatal(errs)
    }

    fmt.Println(some_thing)
}

```

*Note:* FSM makes no effort to determine the default state for any ruleset. That's your job.

The `Apply(r *fsm.Ruleset) *fsm.Machine` method is absolutely optional. I like having it though. It solves a pretty common problem I usually have when working with permissions - some users aren't allowed to transition between certain states.

Since the rules are applied to the the subject (through the machine) I can have a simple lookup to determine the ruleset that the subject has to follow for a given user. As a result, I rarely need to use any complicated guards but I can if need be. I leave the lookup and the maintaining of independent rulesets as an exercise of the user.