package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// The marketplace.json on master before the first release: the plugin installs
// from the marketplace checkout itself.
const unpinned = `{
  "name": "mur",
  "owner": {
    "name": "bifshteks"
  },
  "metadata": {
    "description": "Agent skill for the mur CLI."
  },
  "plugins": [
    {
      "name": "mur",
      "source": "./",
      "description": "Teaches agents to resolve repos."
    }
  ]
}
`

const pinnedV020 = `{
  "name": "mur",
  "owner": {
    "name": "bifshteks"
  },
  "metadata": {
    "description": "Agent skill for the mur CLI."
  },
  "plugins": [
    {
      "name": "mur",
      "version": "0.2.0",
      "source": {
        "source": "url",
        "url": "https://github.com/bifshteks/mur.git",
        "ref": "v0.2.0",
        "sha": "0123456789abcdef0123456789abcdef01234567"
      },
      "description": "Teaches agents to resolve repos."
    }
  ]
}
`

func TestPinUnpinned(t *testing.T) {
	t.Parallel()

	got, err := Pin([]byte(unpinned), "v0.2.0", "0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)
	require.Equal(t, pinnedV020, string(got))
}

func TestPinAlreadyPinned(t *testing.T) {
	t.Parallel()

	got, err := Pin([]byte(pinnedV020), "v0.10.3", "fedcba9876543210fedcba9876543210fedcba98")
	require.NoError(t, err)
	require.Contains(t, string(got), `"version": "0.10.3",`)
	require.Contains(t, string(got), `"ref": "v0.10.3",`)
	require.Contains(t, string(got), `"sha": "fedcba9876543210fedcba9876543210fedcba98"`)
}

func TestPinRejectsBadInput(t *testing.T) {
	t.Parallel()

	const sha = "0123456789abcdef0123456789abcdef01234567"

	cases := map[string]struct{ tag, sha string }{
		"tag without v":     {"0.2.0", sha},
		"prerelease tag":    {"v0.2.0-rc.1", sha},
		"branch name":       {"master", sha},
		"short sha":         {"v0.2.0", "0123456"},
		"empty sha":         {"v0.2.0", ""},
		"uppercase hex sha": {"v0.2.0", "0123456789ABCDEF0123456789ABCDEF01234567"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Pin([]byte(unpinned), tc.tag, tc.sha)
			require.Error(t, err)
		})
	}
}

func TestPinRequiresMurPlugin(t *testing.T) {
	t.Parallel()

	const other = `{"name": "mur", "owner": {"name": "bifshteks"}, "plugins": [{"name": "other", "source": "./"}]}`

	_, err := Pin([]byte(other), "v0.2.0", "0123456789abcdef0123456789abcdef01234567")
	require.ErrorContains(t, err, `no plugin "mur"`)
}

func TestPinRefusesFieldsItWouldDrop(t *testing.T) {
	t.Parallel()

	const withKeywords = `{"name": "mur", "owner": {"name": "bifshteks"},
		"plugins": [{"name": "mur", "source": "./", "keywords": ["git"]}]}`

	_, err := Pin([]byte(withKeywords), "v0.2.0", "0123456789abcdef0123456789abcdef01234567")
	require.ErrorContains(t, err, "keywords")
}

func TestPinKeepsNewerPin(t *testing.T) {
	t.Parallel()

	// A hotfix on an older line, or an old release's job re-run after a newer one.
	_, err := Pin([]byte(pinnedV020), "v0.1.9", "fedcba9876543210fedcba9876543210fedcba98")
	require.ErrorIs(t, err, ErrNotNewer)

	// Compared as numbers, not strings: 0.10.0 is newer than 0.9.0.
	pinnedV090, err := Pin([]byte(unpinned), "v0.9.0", "0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)
	_, err = Pin(pinnedV090, "v0.10.0", "fedcba9876543210fedcba9876543210fedcba98")
	require.NoError(t, err)
}

func TestPinSameVersionAgain(t *testing.T) {
	t.Parallel()

	// A re-run of the same release must not fail.
	got, err := Pin([]byte(pinnedV020), "v0.2.0", "0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)
	require.Equal(t, pinnedV020, string(got))
}

// The manifest on this branch is the one the next release pins. A field Pin does
// not model must fail here, under go test, before goreleaser publishes anything.
func TestPinRepoMarketplace(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../.claude-plugin/marketplace.json")
	require.NoError(t, err)

	_, err = Pin(data, "v999.0.0", "0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)
}
