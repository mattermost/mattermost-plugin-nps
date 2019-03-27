package main

import (
	"html/template"
)

const adminEmailSubject = "[%s] Net Promoter Score survey scheduled in %d days"

var adminEmailBodyTemplate = template.Must(template.New("emailBody").Parse(`
<table align="center" border="0" cellpadding="0" cellspacing="0" width="100%" style="margin-top: 20px; line-height: 1.7; color: #555;">
    <tr>
        <td>
            <table align="center" border="0" cellpadding="0" cellspacing="0" width="100%" style="max-width: 660px; font-family: Helvetica, Arial, sans-serif; font-size: 14px; background: #FFF;">
                <tr>
                    <td style="border: 1px solid #ddd;">
                        <table align="center" border="0" cellpadding="0" cellspacing="0" width="100%" style="border-collapse: collapse;">
                            <tr>
                                <td style="padding: 20px 20px 10px; text-align:left;">
                                    <img src="{{.SiteURL}}/static/images/logo-email.png" width="130px" style="opacity: 0.5" alt="">
                                </td>
                            </tr>
                            <tr>
                                <td>
                                    <table border="0" cellpadding="0" cellspacing="0" style="padding: 20px 50px 0; text-align: center; margin: 0 auto">
                                        <tr>
                                            <td style="padding: 0 0 20px;">
                                                <h2 style="font-weight: normal; margin-top: 10px;">Net Promoter Survey Scheduled</h2>
                                                <p>Mattermost is introducing feedback surveys to measure user satisfaction and improve product quality. Surveys will start to be sent to users in <strong>{{.DaysUntilSurvey}} days</strong>.</p>
                                                <p><a href="{{.SiteURL}}/admin_console/plugins/custom/{{.PluginID}}">Click here</a> to disable or learn more about Net Promoter surveys.</p>
                                            </td>
                                        </tr>
                                        <tr>
                                            <td style="padding: 0 0 20px;">
                                                <a href="{{.SiteURL}}">{{.SiteURL}}</a>
                                            </td>
                                        </tr>
                                    </table>
                                </td>
                            </tr>
                            <tr>
								<td style="text-align: center;color: #AAA; font-size: 11px; padding-bottom: 10px;">
								    <p style="padding: 0 50px;">
								        {{.Organization}}
								    </p>
								</td>
                            </tr>
                        </table>
                    </td>
                </tr>
            </table>
        </td>
    </tr>
</table>
`))

const adminDMBody = `Mattermost uses feedback surveys to measure user satisfaction and improve product quality. User surveys will start to be sent on %s.

[Click here](/admin_console/plugins/custom/%s) to disable or learn more about Net Promoter Score Surveys.

*This message is only visible to System Admins.*`

const surveyBody = ":wave: Hey @%s! Please take a few moments to help us improve your experience with Mattermost."
const surveyDropdownTitle = "How likely are you to recommend Mattermost?"
const surveyAnsweredBody = "You selected %d out of 10."

const feedbackRequestBody = "Thanks! How can we make your experience better?"
const feedbackResponseBody = ":tada: Thanks for helping us make Mattermost better!"
