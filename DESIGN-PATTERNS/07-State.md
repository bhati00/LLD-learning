# State Pattern

> **Represent an object's states as distinct behavior and delegate operations to the current state.**

State is useful when an object's legal operations and transitions depend on its lifecycle state. It replaces scattered conditionals with explicit state-specific behavior.

---

## The Core Idea

The context stores its current state. An operation delegates to that state, which can perform the operation and select the next state. State transitions become explicit rather than distributed across unrelated methods.

---

## Example: Vending Machine

```go
package vending

import "errors"

type machineState interface {
	insertCoin(*Machine) error
	selectItem(*Machine) error
	dispense(*Machine) error
}

type Machine struct {
	state machineState
	stock int
}

func NewMachine(stock int) *Machine {
	m := &Machine{stock: stock}
	m.state = noCreditState{}
	return m
}

func (m *Machine) InsertCoin() error { return m.state.insertCoin(m) }
func (m *Machine) SelectItem() error { return m.state.selectItem(m) }
func (m *Machine) Dispense() error   { return m.state.dispense(m) }

type noCreditState struct{}

func (noCreditState) insertCoin(m *Machine) error {
	if m.stock == 0 {
		return errors.New("out of stock")
	}
	m.state = hasCreditState{}
	return nil
}
func (noCreditState) selectItem(*Machine) error { return errors.New("insert coin first") }
func (noCreditState) dispense(*Machine) error   { return errors.New("insert coin first") }

type hasCreditState struct{}

func (hasCreditState) insertCoin(*Machine) error { return errors.New("coin already inserted") }
func (hasCreditState) selectItem(m *Machine) error {
	m.state = dispensingState{}
	return nil
}
func (hasCreditState) dispense(*Machine) error { return errors.New("select an item first") }

type dispensingState struct{}

func (dispensingState) insertCoin(*Machine) error { return errors.New("dispensing") }
func (dispensingState) selectItem(*Machine) error { return errors.New("dispensing") }
func (dispensingState) dispense(m *Machine) error {
	m.stock--
	m.state = noCreditState{}
	return nil
}
```

This example keeps transitions visible, but a production machine would also need payment/refund handling, item selection, and protection if called concurrently.

---

## When It Helps

- A lifecycle has meaningful states and state-dependent operations.
- Conditional branches are repeated across many methods.
- Invalid transitions should be explicit and testable.

## Costs and Pitfalls

- A separate type per state can be excessive for a small, stable state machine.
- Transitions may be difficult to discover if they are spread across state methods.
- Avoid putting all shared context behavior into every state.
- Define how invalid operations are reported; do not silently ignore them.
- State objects do not make a context concurrency-safe. Synchronize transitions if operations can race.

## Alternative: Explicit Enum

For simple workflows, a `switch` over a `State` enum may be clearer. Choose State when the conditionals are sprawling or behavior grows independently per state, not just to eliminate every switch.

## Interview Check

Draw the state-transition graph, list valid operations in each state, and cover invalid transitions. Contrast State with Strategy: states typically transition as part of the object's lifecycle; strategies are usually selected externally to vary an algorithm.