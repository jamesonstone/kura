package worktree

type sweepSelection map[string]struct{}

func sweepSelectionKey(candidate SweepCandidate) string {
	return candidate.ID + "\x00" + candidate.CommonDir + "\x00" + candidate.Path + "\x00" + candidate.Branch
}

func (selection sweepSelection) contains(candidate SweepCandidate) bool {
	_, selected := selection[sweepSelectionKey(candidate)]
	return selected
}

func (selection sweepSelection) toggle(candidate SweepCandidate) {
	key := sweepSelectionKey(candidate)
	if _, selected := selection[key]; selected {
		delete(selection, key)
		return
	}
	selection[key] = struct{}{}
}

func (selection sweepSelection) add(candidate SweepCandidate) {
	selection[sweepSelectionKey(candidate)] = struct{}{}
}
