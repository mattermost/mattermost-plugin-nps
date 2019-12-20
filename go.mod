module github.com/mattermost/mattermost-plugin-nps/server

go 1.12

require (
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/mattermost/mattermost-server/v5 v5.18.0
	github.com/pkg/errors v0.8.1
	github.com/segmentio/analytics-go/v2 v2.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.4.0
)

replace github.com/segmentio/analytics-go/v2 => github.com/segmentio/analytics-go v2.0.1-0.20160426181448-2d840d861c32+incompatible
