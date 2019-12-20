package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/assert"
)

func TestGetEventProperties(t *testing.T) {
	userID := model.NewId()
	userCreateAt := int64(1546304461000)
	timestamp := int64(1552331717000)

	serverVersion := "5.10.0"
	systemInstallDate := int64(1497898133094)
	diagnosticID := model.NewId()

	licenseID := model.NewId()
	skuShortName := model.NewId()

	for _, test := range []struct {
		Name            string
		SetupAPI        func() *plugintest.API
		OtherProperties map[string]interface{}
		Expected        map[string]interface{}
	}{
		{
			Name: "everything found",
			SetupAPI: func() *plugintest.API {
				api := &plugintest.API{}

				api.On("GetSystemInstallDate").Return(systemInstallDate, nil)

				api.On("GetUser", userID).Return(&model.User{
					Id:       userID,
					CreateAt: userCreateAt,
					Roles:    "system_user",
				}, nil)

				api.On("GetLicense").Return(&model.License{
					Id:           licenseID,
					SkuShortName: skuShortName,
				})

				return api
			},
			Expected: map[string]interface{}{
				"user_actual_id":      userID,
				"timestamp":           timestamp,
				"server_version":      serverVersion,
				"server_install_date": systemInstallDate,
				"server_id":           diagnosticID,
				"user_role":           "user",
				"user_create_at":      userCreateAt,
				"license_id":          licenseID,
				"license_sku":         skuShortName,
			},
		},
		{
			Name: "system install date not found",
			SetupAPI: func() *plugintest.API {
				api := &plugintest.API{}

				api.On("GetSystemInstallDate").Return(int64(0), &model.AppError{})

				api.On("GetUser", userID).Return(&model.User{
					Id:       userID,
					CreateAt: userCreateAt,
					Roles:    "system_user",
				}, nil)

				api.On("GetLicense").Return(&model.License{
					Id:           licenseID,
					SkuShortName: skuShortName,
				})

				return api
			},
			Expected: map[string]interface{}{
				"user_actual_id":      userID,
				"timestamp":           timestamp,
				"server_version":      serverVersion,
				"server_install_date": int64(0),
				"server_id":           diagnosticID,
				"user_role":           "user",
				"user_create_at":      userCreateAt,
				"license_id":          licenseID,
				"license_sku":         skuShortName,
			},
		},
		{
			Name: "user not found",
			SetupAPI: func() *plugintest.API {
				api := &plugintest.API{}

				api.On("GetUser", userID).Return(nil, &model.AppError{})

				api.On("GetSystemInstallDate").Return(systemInstallDate, nil)

				api.On("GetLicense").Return(&model.License{
					Id:           licenseID,
					SkuShortName: skuShortName,
				})

				return api
			},
			Expected: map[string]interface{}{
				"user_actual_id":      userID,
				"timestamp":           timestamp,
				"server_version":      serverVersion,
				"server_install_date": systemInstallDate,
				"server_id":           diagnosticID,
				"user_role":           "",
				"user_create_at":      int64(0),
				"license_id":          licenseID,
				"license_sku":         skuShortName,
			},
		},
		{
			Name: "license not found",
			SetupAPI: func() *plugintest.API {
				api := &plugintest.API{}

				api.On("GetUser", userID).Return(&model.User{
					Id:       userID,
					CreateAt: userCreateAt,
					Roles:    "system_user",
				}, nil)

				api.On("GetSystemInstallDate").Return(systemInstallDate, nil)

				api.On("GetLicense").Return(nil)

				return api
			},
			Expected: map[string]interface{}{
				"user_actual_id":      userID,
				"timestamp":           timestamp,
				"server_version":      serverVersion,
				"server_install_date": systemInstallDate,
				"server_id":           diagnosticID,
				"user_role":           "user",
				"user_create_at":      userCreateAt,
				"license_id":          "",
				"license_sku":         "",
			},
		},
		{
			Name: "with other properties",
			SetupAPI: func() *plugintest.API {
				api := &plugintest.API{}

				api.On("GetSystemInstallDate").Return(systemInstallDate, nil)

				api.On("GetUser", userID).Return(&model.User{
					Id:       userID,
					CreateAt: userCreateAt,
					Roles:    "system_user",
				}, nil)

				api.On("GetLicense").Return(&model.License{
					Id:           licenseID,
					SkuShortName: skuShortName,
				})

				return api
			},
			OtherProperties: map[string]interface{}{
				"other_1": 1234,
				"other_2": "abcd",
			},
			Expected: map[string]interface{}{
				"user_actual_id":      userID,
				"timestamp":           timestamp,
				"server_version":      serverVersion,
				"server_install_date": systemInstallDate,
				"server_id":           diagnosticID,
				"user_role":           "user",
				"user_create_at":      userCreateAt,
				"license_id":          licenseID,
				"license_sku":         skuShortName,
				"other_1":             1234,
				"other_2":             "abcd",
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			api := test.SetupAPI()
			defer api.AssertExpectations(t)

			api.On("GetServerVersion").Return("5.10.0")
			api.On("GetDiagnosticId").Return(diagnosticID)

			api.On("GetTeamMembersForUser", userID, 0, 50).Return([]*model.TeamMember{}, nil).Maybe()

			p := Plugin{}
			p.SetAPI(api)

			assert.Equal(t, test.Expected, p.getEventProperties(userID, timestamp, test.OtherProperties))
		})
	}
}

func TestGetUserRole(t *testing.T) {
	userId := model.NewId()

	for _, test := range []struct {
		Name        string
		User        *model.User
		TeamMembers []*model.TeamMember
		Expected    string
	}{
		{
			Name: "system admin",
			User: &model.User{
				Id:    userId,
				Roles: model.SYSTEM_ADMIN_ROLE_ID + " " + model.SYSTEM_USER_ROLE_ID,
			},
			Expected: "system_admin",
		},
		{
			Name: "system and team admin",
			User: &model.User{
				Id:    userId,
				Roles: model.SYSTEM_ADMIN_ROLE_ID + " " + model.SYSTEM_USER_ROLE_ID,
			},
			TeamMembers: []*model.TeamMember{
				{
					Roles: model.TEAM_ADMIN_ROLE_ID + " " + model.TEAM_USER_ROLE_ID,
				},
			},
			Expected: "system_admin",
		},
		{
			Name: "team admin",
			User: &model.User{
				Id:    userId,
				Roles: model.SYSTEM_USER_ROLE_ID,
			},
			TeamMembers: []*model.TeamMember{
				{
					Roles: model.TEAM_USER_ROLE_ID,
				},
				{
					Roles: model.TEAM_ADMIN_ROLE_ID + " " + model.TEAM_USER_ROLE_ID,
				},
			},
			Expected: "team_admin",
		},
		{
			Name: "regular user",
			User: &model.User{
				Id:    userId,
				Roles: model.SYSTEM_USER_ROLE_ID,
			},
			TeamMembers: []*model.TeamMember{
				{
					Roles: model.TEAM_USER_ROLE_ID,
				},
			},
			Expected: "user",
		},
		{
			Name: "regular user without teams",
			User: &model.User{
				Id:    userId,
				Roles: model.SYSTEM_USER_ROLE_ID,
			},
			TeamMembers: []*model.TeamMember{},
			Expected:    "user",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			api := &plugintest.API{}
			defer api.AssertExpectations(t)

			api.On("GetTeamMembersForUser", test.User.Id, 0, 50).Return(test.TeamMembers, nil).Maybe()

			p := Plugin{}
			p.SetAPI(api)

			assert.Equal(t, test.Expected, p.getUserRole(test.User))
		})
	}
}
