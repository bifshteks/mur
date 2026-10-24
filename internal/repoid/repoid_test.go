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

func TestValidate(t *testing.T) {
	t.Parallel()

	valid := []string{
		"git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git",
		"git.nimbus.example:nimbus/collector",
		"https://git.nimbus.example/nimbus/services-team/svc-analytics-collector",
		"ssh://git@git.nimbus.example:2222/nimbus/collector.git",
		"git+ssh://git@git.nimbus.example/nimbus/collector.git",
		"HTTPS://oauth2:secret@git.nimbus.example:443/nimbus/collector/",
		"git://git.nimbus.example/nimbus/collector",
		"file:///srv/git/collector.git",
		"/srv/git/collector.git",
	}

	for _, url := range valid {
		t.Run(url, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, repoid.Validate(url))
		})
	}

	invalid := []string{
		"",
		"collector",
		"git.nimbus.example/nimbus/collector",
		"./collector",
		"git@git.nimbus.example:",
		"git@:nimbus/collector",
		"https://git.nimbus.example",
		"https://git.nimbus.example/",
		"https:///nimbus/collector",
		"https://git.nimbus.example:port/nimbus/collector",
		"ftp://git.nimbus.example/nimbus/collector",
		"file://srv/git/collector",
		"file:///",
		"/",
		"git@git.nimbus.example:nimbus/my collector",
		" git@git.nimbus.example:nimbus/collector",
	}

	for _, url := range invalid {
		t.Run("invalid "+url, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, repoid.Validate(url), repoid.ErrInvalid)
		})
	}
}
