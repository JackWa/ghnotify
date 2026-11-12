package poller

import "testing"

func TestDropRemoved(t *testing.T) {
	in := []Change{{Kind: ChangeAdded}, {Kind: ChangeRemoved}, {Kind: ChangeCIStatus}, {Kind: ChangeRemoved}}
	got := dropRemoved(in)
	if len(got) != 2 || got[0].Kind != ChangeAdded || got[1].Kind != ChangeCIStatus {
		t.Fatalf("dropRemoved = %+v", got)
	}
	if len(in) != 4 || in[1].Kind != ChangeRemoved {
		t.Fatalf("input was modified: %+v", in)
	}
}

func TestSignalQuietReplacesPendingRefresh(t *testing.T) {
	ch := make(chan bool, 1)
	m := &Manager{refresh: map[string]chan bool{"h": ch}}
	m.Refresh()
	m.Resync()
	if got := <-ch; !got {
		t.Fatal("pending refresh was not upgraded to quiet")
	}
	m.Resync()
	m.Refresh()
	if got := <-ch; !got {
		t.Fatal("quiet refresh was downgraded by a plain one")
	}
}
