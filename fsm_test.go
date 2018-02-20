package fsm_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nbio/st"
	"github.com/utilitywarehouse/fsm"
)

// Thing is a minimal struct that is an fsm.Stater
type Thing struct {
	State fsm.State
}

func (t *Thing) CurrentState() fsm.State { return t.State }
func (t *Thing) SetState(s fsm.State)    { t.State = s }

func TestRulesetTransitions(t *testing.T) {
	rules := fsm.CreateRuleset(
		fsm.T{"pending", "started"},
		fsm.T{"started", "finished"},
	)

	examples := []struct {
		subject fsm.Stater
		goal    fsm.State
		outcome []error
	}{
		// A Stater is responsible for setting its default state
		{&Thing{}, "started", []error{fsm.ErrInvalidTransition{fsm.T{"", "started"}}}},
		{&Thing{}, "pending", []error{fsm.ErrInvalidTransition{fsm.T{"", "pending"}}}},
		{&Thing{}, "finished", []error{fsm.ErrInvalidTransition{fsm.T{"", "finished"}}}},

		{&Thing{State: "pending"}, "started", nil},
		{&Thing{State: "pending"}, "pending", []error{fsm.ErrInvalidTransition{fsm.T{"pending", "pending"}}}},
		{&Thing{State: "pending"}, "finished", []error{fsm.ErrInvalidTransition{fsm.T{"pending", "finished"}}}},

		{&Thing{State: "started"}, "started", []error{fsm.ErrInvalidTransition{fsm.T{"started", "started"}}}},
		{&Thing{State: "started"}, "pending", []error{fsm.ErrInvalidTransition{fsm.T{"started", "pending"}}}},
		{&Thing{State: "started"}, "finished", nil},
	}

	for i, ex := range examples {
		st.Expect(t, rules.IsValidTransition(ex.subject, ex.goal), ex.outcome, i)
	}
}

func TestRulesetParallelGuarding(t *testing.T) {
	rules := fsm.Ruleset{}
	rules.AddTransition(fsm.T{"pending", "started"})
	rules.AddTransition(fsm.T{"started", "finished"})

	// Add two failing rules, the slow should be caught first
	rules.AddRule(fsm.T{"started", "finished"}, func(subject fsm.Stater, goal fsm.State) error {
		time.Sleep(1 * time.Second)
		return errors.New("error")
	})

	rules.AddRule(fsm.T{"started", "finished"}, func(subject fsm.Stater, goal fsm.State) error {
		return nil
	})

	st.Expect(t, rules.IsValidTransition(&Thing{State: "started"}, "finished"), []error{errors.New("error")})

}

func TestMachineTransition(t *testing.T) {
	rules := fsm.Ruleset{}
	rules.AddTransition(fsm.T{"pending", "started"})
	rules.AddTransition(fsm.T{"started", "finished"})

	some_thing := Thing{State: "pending"}
	the_machine := fsm.New(fsm.WithRules(rules), fsm.WithSubject(&some_thing))

	// should not be able to transition to the current state
	err := the_machine.Transition("pending")
	st.Expect(t, err[0], fsm.ErrInvalidTransition{fsm.T{"pending", "pending"}})
	st.Expect(t, some_thing.State, fsm.State("pending"))

	// should not be able to skip states
	err = the_machine.Transition("finished")
	st.Expect(t, err[0], fsm.ErrInvalidTransition{fsm.T{"pending", "finished"}})
	st.Expect(t, some_thing.State, fsm.State("pending"))

	// should be able to transition to the next valid state
	err = the_machine.Transition("started")
	st.Expect(t, some_thing.State, fsm.State("started"))
}

func BenchmarkRulesetParallelGuarding(b *testing.B) {
	rules := fsm.Ruleset{}
	rules.AddTransition(fsm.T{"pending", "started"})
	rules.AddTransition(fsm.T{"started", "finished"})

	// Add two failing rules, one very slow and the other terribly fast
	rules.AddRule(fsm.T{"started", "finished"}, func(subject fsm.Stater, goal fsm.State) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	rules.AddRule(fsm.T{"started", "finished"}, func(subject fsm.Stater, goal fsm.State) error {
		return nil
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rules.IsValidTransition(&Thing{State: "started"}, "finished")
	}
}

func BenchmarkRulesetTransitionPermitted(b *testing.B) {
	// Permitted a transaction requires the transition to be valid and all of its
	// guards to pass. Since we have to run every guard and there won't be any
	// short-circuiting, this should actually be a little bit slower as a result,
	// depending on the number of guards that must pass.
	rules := fsm.Ruleset{}
	rules.AddTransition(fsm.T{"pending", "started"})
	rules.AddTransition(fsm.T{"started", "finished"})

	some_thing := &Thing{State: "started"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rules.IsValidTransition(some_thing, "finished")
	}

}

func BenchmarkRulesetTransitionInvalid(b *testing.B) {
	// This should be incredibly fast, since fsm.T{"pending", "finished"}
	// doesn't exist in the Ruleset. We expect some small overhead from creating
	// the transition to check the internal map, but otherwise, we should be
	// bumping up against the speed of a map lookup itself.

	rules := fsm.Ruleset{}
	rules.AddTransition(fsm.T{"pending", "started"})
	rules.AddTransition(fsm.T{"started", "finished"})

	some_thing := &Thing{State: "pending"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rules.IsValidTransition(some_thing, "finished")
	}
}

func BenchmarkRulesetRuleForbids(b *testing.B) {
	// Here, we explicity create a transition that is forbidden. This simulates an
	// otherwise valid transition that would be denied based on a user role or the like.
	// It should be slower than a standard invalid transition, since we have to
	// actually execute a function to perform the check. The first guard to
	// fail (returning false) will short circuit the execution, getting some some speed.

	rules := fsm.Ruleset{}
	rules.AddTransition(fsm.T{"pending", "started"})

	rules.AddRule(fsm.T{"started", "finished"}, func(subject fsm.Stater, goal fsm.State) error {
		return nil
	})

	some_thing := &Thing{State: "started"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rules.IsValidTransition(some_thing, "finished")
	}
}
