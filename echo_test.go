package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// A write the operator made comes back on its own watch with the
// resourceVersion the write answered. That event carries the state the
// pass itself wrote, so it wakes nothing. An event with any other version
// is another writer's change and wakes the loop. Each case writes a status
// that the server answers with version 60, then streams one event.
func TestTheOperatorsOwnWriteWakesNothingOnItsWatch(t *testing.T) {
	cases := []struct {
		name   string
		method string
		event  string
		wakes  bool
	}{
		{name: "the echo of a status write", method: http.MethodPut, event: "60", wakes: false},
		{name: "the echo of a metadata patch", method: http.MethodPatch, event: "60", wakes: false},
		{name: "a later change by another writer", method: http.MethodPut, event: "61", wakes: true},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			useWatchRetryPause(t)
			api := newWatchAPI()
			server := httptest.NewServer(api.handler())
			client := NewClient(server.URL, server.Client(), "")
			api.answersLists(listTurn{version: "60"})
			if err := client.RequestJSON(t.Context(), one.method, libraryPath("house", "movies")+"/status",
				[]byte(`{}`), nil); err != nil {
				t.Fatal(err)
			}
			api.answersWatches(watchTurn{events: []string{watchEvent("MODIFIED", one.event)}, hold: api.parked})
			wake := make(chan struct{}, 1)

			go watchLibraries(client, "42", wake, nil)

			if one.wakes {
				waitForWatchWake(t, wake)
			} else {
				expectNoWatchWake(t, wake)
			}
		})
	}
}
