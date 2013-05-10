package raft

import (
	"math/rand"
	"sync"
	"time"
)

//------------------------------------------------------------------------------
//
// Typedefs
//
//------------------------------------------------------------------------------

// The timer wraps the internal Go timer and provides the ability to pause,
// reset and stop. It also allows for the duration of the timer to be a random
// number between a min and max duration.
type Timer struct {
	c             chan time.Time
	rand          *rand.Rand
	minDuration   time.Duration
	maxDuration   time.Duration
	internalTimer *time.Timer
	mutex         sync.Mutex
}

//------------------------------------------------------------------------------
//
// Constructors
//
//------------------------------------------------------------------------------

// Creates a new timer. Panics if a non-positive duration is used.
func NewTimer(minDuration time.Duration, maxDuration time.Duration) *Timer {
	if minDuration <= 0 {
		panic("raft.Timer: Non-positive minimum duration not allowed")
	}
	if maxDuration <= 0 {
		panic("raft.Timer: Non-positive maximum duration not allowed")
	}
	if minDuration > maxDuration {
		panic("raft.Timer: Minimum duration cannot be greater than maximum duration")
	}
	return &Timer{
		c:           make(chan time.Time, 1),
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		minDuration: minDuration,
		maxDuration: maxDuration,
	}
}

//------------------------------------------------------------------------------
//
// Accessors
//
//------------------------------------------------------------------------------

// Retrieves the timer's channel.
func (t *Timer) C() chan time.Time {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.c
}

// Retrieves the minimum duration of the timer.
func (t *Timer) MinDuration() time.Duration {
	return t.minDuration
}

// Sets the minimum duration of the timer.
func (t *Timer) SetMinDuration(duration time.Duration) {
	t.minDuration = duration
	t.Reset()
}

// Retrieves the maximum duration of the timer.
func (t *Timer) MaxDuration() time.Duration {
	return t.maxDuration
}

// Sets the maximum duration of the timer.
func (t *Timer) SetMaxDuration(duration time.Duration) {
	t.maxDuration = duration
	t.Reset()
}

// Sets the minimum and maximum duration of the timer.
func (t *Timer) SetDuration(duration time.Duration) {
	t.minDuration = duration
	t.maxDuration = duration
	t.Reset()
}

//------------------------------------------------------------------------------
//
// Methods
//
//------------------------------------------------------------------------------

// Checks if the timer is currently running.
func (t *Timer) Running() bool {
	return t.internalTimer != nil
}

// Stops the timer and closes the channel.
func (t *Timer) Stop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.internalTimer != nil {
		t.internalTimer.Stop()
		t.internalTimer = nil
	}

	if t.c != nil {
		close(t.c)
		t.c = nil
	}
}

// Stops the timer.
func (t *Timer) Pause() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.internalTimer != nil {
		t.internalTimer.Stop()
		t.internalTimer = nil
	}
}

// Stops the timer if it is running and restarts it.
func (t *Timer) Reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Stop the timer if it's already running.
	if t.internalTimer != nil {
		t.internalTimer.Stop()
	}

	// Start a timer that will go off between the min and max duration.
	d := t.minDuration
	if t.maxDuration > t.minDuration {
		d += time.Duration(t.rand.Int63n(int64(t.maxDuration - t.minDuration)))
	}
	t.internalTimer = time.NewTimer(d)
	go func() {
		defer func() {
			recover()
		}()

		// Retrieve the current internal timer.
		t.mutex.Lock()
		internalTimer := t.internalTimer
		t.mutex.Unlock()

		// If the timer exists then grab the value from the channel and pass
		// it through to the timer's external channel.
		if internalTimer != nil {
			if v, ok := <-internalTimer.C; ok {
				t.mutex.Lock()
				t.c <- v
				t.mutex.Unlock()
			}
		}
	}()
}
