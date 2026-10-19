package repoid_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/repoid"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	const collector = "git.nimbus.example/nimbus/services-team/svc-analytics-collector"

	cases := []struct {
		url  string
		want string
	}{
		{"git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git", collector},
		{"https://git.nimbus.example/nimbus/services-team/svc-analytics-collector.git", collector},
		{"https://git.nimbus.example/nimbus/services-team/svc-analytics-collector", collector},
		{"ssh://git@git.nimbus.example:2222/nimbus/services-team/svc-analytics-collector.git", collector},
		{"https://oauth2:secret@GIT.NIMBUS.EXAMPLE:443/nimbus/services-team/svc-analytics-collector/", collector},
		{"git.nimbus.example/nimbus/services-team/svc-analytics-collector", collector},
		{"git@github.com:bifshteks/mur", "github.com/bifshteks/mur"},
		{"https://GitHub.com/Bifshteks/Mur.git", "github.com/Bifshteks/Mur"},
		{"file:///srv/git/collector.git", "/srv/git/collector"},
		{"/srv/git/collector.git/", "/srv/git/collector"},
	}

	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, repoid.Normalize(tc.url))
		})
	}
}
