module github.com/mattermost/mattermost-plugin-nps/server

go 1.12

require (
	github.com/mattermost/ldap v3.0.4+incompatible // indirect
	github.com/mattermost/mattermost-plugin-api v0.0.11-0.20200625065737-cc668766649c
	github.com/mattermost/mattermost-server/v5 v5.23.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.5.1
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
)

replace github.com/segmentio/analytics-go/v2 => github.com/segmentio/analytics-go v2.0.1-0.20160426181448-2d840d861c32+incompatible
