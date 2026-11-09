package cmd

import (
	"io"
	"testing"

	"github.com/github/gh-stack/internal/config"
	"github.com/github/gh-stack/internal/git"
	"github.com/github/gh-stack/internal/github"
	"github.com/github/gh-stack/internal/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGuardedPush pushes a one-branch stack whose last recorded push is
// "pushed1", with origin/b1 at remoteSHA and b1 at "local1". unmatched is what
// the remote has that neither the last push nor local has.
func runGuardedPush(t *testing.T, remoteSHA string, unmatched []string, args ...string) ([]pushCall, string, *stack.StackFile, error) {
	t.Helper()
	tmpDir := t.TempDir()
	writeStackFile(t, tmpDir, stack.Stack{
		Trunk:    stack.BranchRef{Branch: "main"},
		Branches: []stack.BranchRef{{Branch: "b1", Pushed: "pushed1"}},
	})

	var pushCalls []pushCall
	mock := newPushMock(tmpDir, "b1")
	mock.PushFn = func(remote string, branches []string, force, atomic bool) error {
		pushCalls = append(pushCalls, pushCall{remote, branches, force, atomic})
		return nil
	}
	mock.RevParseFn = func(ref string) (string, error) {
		return map[string]string{"origin/main": "main1", "origin/b1": remoteSHA, "b1": "local1"}[ref], nil
	}
	mock.UnmatchedCommitsFn = func(head string, known []string, exclude string) ([]string, error) {
		assert.Equal(t, remoteSHA, head)
		assert.Equal(t, []string{"pushed1", "b1"}, known)
		assert.Equal(t, "origin/main", exclude)
		return unmatched, nil
	}
	mock.LogFn = func(ref string, n int) ([]git.CommitInfo, error) {
		return []git.CommitInfo{{SHA: ref, Subject: "reviewer fix"}}, nil
	}
	restore := git.SetOps(mock)
	defer restore()

	cfg, _, errR := config.NewTestConfig()
	cfg.GitHubClientOverride = &github.MockClient{}
	cmd := PushCmd(cfg)
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	cfg.Err.Close()
	out, _ := io.ReadAll(errR)

	sf, loadErr := stack.Load(tmpDir)
	require.NoError(t, loadErr)
	return pushCalls, string(out), sf, err
}

func TestPush_RefusesToDiscardCommitsOthersPushed(t *testing.T) {
	pushCalls, out, sf, err := runGuardedPush(t, "theirs1", []string{"fix0000000"})

	assert.ErrorIs(t, err, ErrSilent)
	assert.Empty(t, pushCalls, "nothing may be pushed")
	assert.Contains(t, out, "origin/b1 has 1 commit since your last push")
	assert.Contains(t, out, "reviewer fix")
	assert.Contains(t, out, "--overwrite-remote")
	assert.Equal(t, "pushed1", sf.Stacks[0].Branches[0].Pushed, "the baseline only moves on a push")
}

func TestPush_OverwriteRemoteDiscardsThem(t *testing.T) {
	pushCalls, out, sf, err := runGuardedPush(t, "theirs1", []string{"fix0000000"}, "--overwrite-remote")

	assert.NoError(t, err)
	require.Len(t, pushCalls, 1)
	assert.Contains(t, out, "Overwriting them")
	assert.Equal(t, "local1", sf.Stacks[0].Branches[0].Pushed)
}

func TestPush_RewrittenCopiesOfOurCommitsArePushedOver(t *testing.T) {
	// A rebase elsewhere with the same diffs leaves nothing unmatched.
	pushCalls, _, sf, err := runGuardedPush(t, "theirs1", nil)

	assert.NoError(t, err)
	require.Len(t, pushCalls, 1)
	assert.Equal(t, "local1", sf.Stacks[0].Branches[0].Pushed)
}

func TestPush_RemoteUnchangedSinceLastPushIsNotCompared(t *testing.T) {
	// Were the remote compared, "never" would be unmatched and block the push.
	pushCalls, _, _, err := runGuardedPush(t, "pushed1", []string{"never"})

	assert.NoError(t, err)
	require.Len(t, pushCalls, 1)
}
