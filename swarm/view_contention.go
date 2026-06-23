package swarm

import "slices"

/*
MergeContention records a lease contention event.
*/
func (view *View) MergeContention(contention Contention) error {
	if err := contention.Validate(); err != nil {
		return err
	}

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)
		updated.contentions = append(updated.contentions, contention)

		return updated
	})

	return nil
}

/*
RecentContentions returns a stable snapshot of non-expired contention events.
*/
func (view *View) RecentContentions() []Contention {
	contentions := append([]Contention(nil), view.state.Load().contentions...)

	slices.SortFunc(contentions, compareContentions)

	return contentions
}

func compareContentions(firstContention, secondContention Contention) int {
	if firstContention.At < secondContention.At {
		return -1
	}

	if firstContention.At > secondContention.At {
		return 1
	}

	if firstContention.ActorID < secondContention.ActorID {
		return -1
	}

	if firstContention.ActorID > secondContention.ActorID {
		return 1
	}

	return 0
}
