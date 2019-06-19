# Mattermost Net Promoter Score Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-nps/master.svg)

A plugin for Mattermost to gather user feedback about Mattermost itself using NPS (Net Promoter Score) surveys.

## Installation

__Requires Mattermost 5.12 or higher.__

The NPS plugin is installed and enabled by default on Mattermost 5.12 or higher. If you'd like to install a custom version of the plugin, you can disable the built-in one and install your custom version alongside it.

## Developing 

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.

Use `make check-style` to check the style.

Use `make deploy` to deploy the plugin to your local server. Before running `make deploy` you need to set a few environment variables:

```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
```
