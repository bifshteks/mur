package main

import (
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
