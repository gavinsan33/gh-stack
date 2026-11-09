package cmd

import (
	"github.com/github/gh-stack/internal/config"
	"github.com/github/gh-stack/internal/git"
	"github.com/github/gh-stack/internal/stack"
)

// checkRemoteRewrites guards a force push against discarding commits someone
// else pushed to a stack branch (a reviewer's fix, or a rebase done outside
// gh stack). --force-with-lease alone can't: the push fetches first, so its
// lease is whatever is on the remote at that moment.
//
// Each branch is compared with the SHA this clone last pushed for it. If the
// remote has moved since, every new commit must be a merge, already on trunk,
// or a rewritten copy (same patch-id) of a commit the last push or the local
// branch has. Anything else is listed, and the push is refused unless
// overwrite is set. Branches with no recorded push (stack files written
// before this check, or by another clone) aren't checked.
func checkRemoteRewrites(cfg *config.Config, s *stack.Stack, remote string, branches []string, overwrite bool) bool {
	pushing := make(map[string]bool, len(branches))
	for _, b := range branches {
		pushing[b] = true
	}
	trunk := remote + "/" + s.Trunk.Branch
	if _, err := git.RevParse(trunk); err != nil {
		trunk = s.Trunk.Branch
	}

	found := false
	for _, br := range s.Branches {
		if !pushing[br.Branch] || br.Pushed == "" {
			continue
		}
		remoteSHA, err := git.RevParse(remote + "/" + br.Branch)
		if err != nil || remoteSHA == br.Pushed {
			continue
		}
		if exists, err := git.CommitExists(br.Pushed); err != nil || !exists {
			continue
		}
		unmatched, err := git.UnmatchedCommits(remoteSHA, []string{br.Pushed, br.Branch}, trunk)
		if err != nil {
			found = true
			cfg.Warningf("Could not compare %s with %s/%s: %v", br.Branch, remote, br.Branch, err)
			continue
		}
		if len(unmatched) == 0 {
			continue
		}
		found = true
		cfg.Warningf("%s/%s has %d %s since your last push that %s doesn't have:",
			remote, br.Branch, len(unmatched), plural(len(unmatched), "commit", "commits"), br.Branch)
		for _, sha := range unmatched {
			subject := ""
			if commits, err := git.Log(sha, 1); err == nil && len(commits) == 1 {
				subject = commits[0].Subject
			}
			cfg.Printf("  %s %s", short(sha), subject)
		}
	}
	if !found {
		return true
	}
	if overwrite {
		cfg.Warningf("Overwriting them (--overwrite-remote)")
		return true
	}
	cfg.Printf("Add them to the local branch (e.g. `git cherry-pick <sha>`), or rerun with `%s` to discard them.",
		cfg.ColorAccent("--overwrite-remote"))
	return false
}

// recordPushed stores each pushed branch's SHA as the baseline for the next
// checkRemoteRewrites.
func recordPushed(s *stack.Stack, branches []string) {
	shas, err := git.RevParseMulti(branches)
	if err != nil || len(shas) != len(branches) {
		return
	}
	pushed := make(map[string]string, len(branches))
	for i, b := range branches {
		if shas[i] != "" {
			pushed[b] = shas[i]
		}
	}
	for i := range s.Branches {
		if sha, ok := pushed[s.Branches[i].Branch]; ok {
			s.Branches[i].Pushed = sha
		}
	}
}
