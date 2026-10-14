// Copyright (c) 2026-present Patika. All Rights Reserved.
// See LICENSE for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mattermost rejects preference categories over 32 characters, and the web app cuts the key to
// fit; reading the full "pp_<id>" would never find what Settings saved.
func TestPreferenceCategoryMatchesTheWebApp(t *testing.T) {
	assert.Equal(t, "pp_dev.patika.reaction-notificat", preferenceCategory("dev.patika.reaction-notifications"))
	assert.Equal(t, "pp_short.id", preferenceCategory("short.id"))
}
