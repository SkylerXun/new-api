package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLogFilterOptionsReturnsSortedActiveUsersAndChannels(t *testing.T) {
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM channels")
	})

	emptyUsernames, emptyChannels, err := GetLogFilterOptions()
	require.NoError(t, err)
	require.NotNil(t, emptyUsernames)
	require.NotNil(t, emptyChannels)
	require.Empty(t, emptyUsernames)
	require.Empty(t, emptyChannels)

	users := []User{
		{Username: "zeta", Password: "password", AffCode: "zeta-code"},
		{Username: "alpha", Password: "password", AffCode: "alpha-code"},
		{Username: "deleted", Password: "password", AffCode: "deleted-code"},
	}
	require.NoError(t, DB.Create(&users).Error)
	require.NoError(t, DB.Delete(&users[2]).Error)

	channels := []Channel{
		{Id: 20, Name: "Secondary", Key: "key-2"},
		{Id: 10, Name: "Primary", Key: "key-1"},
	}
	require.NoError(t, DB.Create(&channels).Error)

	usernames, channelOptions, err := GetLogFilterOptions()

	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "zeta"}, usernames)
	require.Equal(t, []LogFilterChannelOption{
		{Id: 10, Name: "Primary"},
		{Id: 20, Name: "Secondary"},
	}, channelOptions)
}
