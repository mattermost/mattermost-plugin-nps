// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin()
	require.NotNil(t, plugin)
	require.NotNil(t, plugin.now)
	require.NotNil(t, plugin.readFile)
}
