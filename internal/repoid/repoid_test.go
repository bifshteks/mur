package repoid_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/repoid"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	const collector = "git.mws-team.ru/mws/security-services/sec-audit-collector"

	cases := []struct {
		url  string
		want string
	}{
		{"git@git.mws-team.ru:mws/security-services/sec-audit-collector.git", collector},
		{"https://git.mws-team.ru/mws/security-services/sec-audit-collector.git", collector},
		{"https://git.mws-team.ru/mws/security-services/sec-audit-collector", collector},
		{"ssh://git@git.mws-team.ru:2222/mws/security-services/sec-audit-collector.git", collector},
		{"https://oauth2:secret@GIT.MWS-TEAM.RU:443/mws/security-services/sec-audit-collector/", collector},
		{"git.mws-team.ru/mws/security-services/sec-audit-collector", collector},
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
