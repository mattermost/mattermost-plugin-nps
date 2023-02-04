# Mattermost User Satisfaction Survey Plugin

[![CI](https://github.com/mattermost/mattermost-plugin-nps/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/mattermost/mattermost-plugin-nps/actions/workflows/ci.yml)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-nps/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-nps)

A plugin for Mattermost to gather user feedback about Mattermost itself using user satisfaction surveys.

## Installation

__Requires Mattermost 5.12 or higher.__

The NPS plugin is installed and enabled by default on Mattermost 5.12 or higher. If you'd like to install a custom version of the plugin, you can disable the built-in one and install your custom version alongside it.

## Developing

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.

Use `make check-style` to check the style.

Use `make deploy` to deploy the plugin to your local server. 
Before running `make deploy`, make sure you have set the following values in your mattermost-server configuration (`config/config.json`):

```
.ServiceSettings.EnableLocalMode: true
.PluginSettings.EnableUploads: true
.LogSettings.EnableDiagnostics: true
```

### How it works - overview

The plugin sends a survey after 45 days when a new (as in never seen before - including downgrade) version of Mattermost is detected. It also sends a message to user 7 days after their registration to get early feedback, and allow users to give feedback at any time they desire.

Survey and feedbacks are sent to rudder as `Track` events.

#### Configuration

The only configuration option is to enable or disable the surveys automated.

#### The "Logs in" rule

The plugin only send DM to a user when this user logs in. "Logs in"  mean that the server has received a request by a user to retrieve their own info. It happens when a user logs in, but also if they refresh their browser as the webapp will do this request to the server. 

### Welcome feedback DM

7 days after a user created their account, they will be sent a message by the bot asking for a feedback. This event is triggered by the "logs in" rule described in the previous paragraph, so the message is not technically sent 7 days after the account creation, but as soon as they get online, at least 7 days after the account creation. 

While this message is not a survey per say, it will not be send if the configuration disables surveys.

### New version survey

When the plugin is enabled (first time install, after being disabled and reenabled, or when the server starts/restarts), we are checking if the server is running on a new version. **important** note that in this case, new does not mean more recent, it just means a version for which we never sent a survey before. Including downgrade.
If the server is running on a new version, we are scheduling a survey to be sent to all users 45 days after the new version is detected.
A notice letting sysadmin know know that a survey is scheduled is sent  by email, and a DM is schedulded to be sent once them next time they login.
When a user logs in, we do check if they are due for a survey. If they are, we are sending them a DM with a survey.
The survey consist in rating the app between 1 and 10, and giving a comment. The user also have the option to opt out of future surveys.

### Feedback

At any point, a user can engage in a DM with the bot and send a feedback. When the user is done typing, a modal will appear asking the user to confirm the feedback and optionnaly asks for email address.

### Rudder

Here are all the `Track` events sent to rudder:

- `nps_survey`, with the property `score` containing the score given by the user
- `nps_feedback`, with the property `feedback` containing the feedback given by the user and `email` containing the email address given by the user (can be empty)
- `nps_disable` with no extra property

All of those events also contains the following property (when available):

- `PluginID`, containing the plugin ID
- `PluginVersion`, containing the plugin version
- `ServerVersion`, containing the server version
- `UserActualID`. containing the curremt user ID
- `timestamp`, containing the timestamp of the event
- `server_install_date`, containing the server install date
- `user_role`, containing the user role
- `user_create_at`, containing the user creation date
- `license_id`, containing the license ID
- `license_sku`, containing the license SKU
