package main

// The operator watches the collections it also writes: a Library's and a
// Catalog's status, and the finalizers on a Library, a Play, and a Person.
// Every write comes back on the watch as an event, and that event carries
// the state the pass itself wrote. A wake for it runs a pass that finds
// nothing to do. So the client remembers the resourceVersion each update
// answered with, and the watch skips the one event that carries it. Any
// other writer's change carries another version and still wakes the loop.

import (
	"encoding/json"
	"sync"
	"time"
)

// How long a remembered version waits for its event. An event arrives
// within a second of the write. A version whose event never comes, because
// the watch was down or the kind is not watched, is dropped after this.
const echoWindow = time.Minute

// The versions this client's own updates answered with, and when.
type echoes struct {
	mutex    sync.Mutex
	versions map[string]time.Time
}

// remember records the version an update answered with. The client calls
// it for an update alone: a create and a delete are followed by events the
// pass must act on, such as a pod that becomes ready or an object that is
// gone. An update that answers with a deleting object is followed by the
// delete, so it is not remembered either.
func (e *echoes) remember(answer []byte) {
	var written struct {
		Metadata ObjectMeta `json:"metadata"`
	}
	if json.Unmarshal(answer, &written) != nil || written.Metadata.ResourceVersion == "" ||
		written.Metadata.deleting() {
		return
	}
	now := time.Now()
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.versions == nil {
		e.versions = map[string]time.Time{}
	}
	for version, at := range e.versions {
		if now.Sub(at) > echoWindow {
			delete(e.versions, version)
		}
	}
	e.versions[written.Metadata.ResourceVersion] = now
}

// own answers whether one event's version is an update of this client's,
// and forgets it, because a version comes back once.
func (e *echoes) own(version string) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	_, held := e.versions[version]
	delete(e.versions, version)
	return held
}
