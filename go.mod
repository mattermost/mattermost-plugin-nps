module github.com/mattermost/mattermost-plugin-nps/server

go 1.12

require (
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/mattermost/mattermost-server v0.0.0-20190822022538-11b0a20d7df9
	github.com/nicksnyder/go-i18n v1.10.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/segmentio/analytics-go v3.0.1+incompatible
	github.com/segmentio/analytics-go/v2 v2.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.3.0
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1

replace github.com/segmentio/analytics-go/v2 => github.com/segmentio/analytics-go v2.0.1-0.20160426181448-2d840d861c32+incompatible
