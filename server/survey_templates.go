package main

import (
	"html/template"
)

const adminEmailSubject = "[%s] User Satisfaction Survey scheduled in %d days"

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
                                    <img src="{{.SiteURL}}/static/images/logo-email.png" width="130px" alt="">
                                </td>
                            </tr>
                            <tr>
                                <td>
                                    <table border="0" cellpadding="0" cellspacing="0" style="padding: 20px 50px 0; text-align: center; margin: 0 auto">
                                        <tr>
                                            <td style="padding: 0 0 20px;">
                                                <h2 style="font-weight: normal; margin-top: 10px;"><a href="https://mattermost.com/pl/default-nps">User Satisfaction Survey</a> Scheduled</h2>
                                                <p>Mattermost sends quarterly in-product user satisfaction surveys to gather feedback from users and improve product quality. Surveys will be received by users in <strong>{{.DaysUntilSurvey}} days</strong>.</p>
                                                <p>Click <a href="{{.SiteURL}}/admin_console/plugins/plugin_{{.PluginID}}">here</a> to disable or learn more about user satisfaction surveys. Please refer to our <a href="https://about.mattermost.com/default-privacy-policy">privacy policy</a> for more information on the collection and use of information received through our services.</p>
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

[Click here](/admin_console/plugins/plugin_%s) to disable or learn more about user satisfaction surveys.

*This message is only visible to System Admins.*`

const surveyBody = ":wave: Hey @%s! Please take a few moments to help us improve your experience with Mattermost."
const surveyDropdownTitle = "How likely are you to recommend Mattermost?"
const surveyAnsweredBody = "You selected %d out of 10."

const welcomeFeedbackRequestBody = ":wave: Hey @%s! Can you spare a minute or two to tell me how do you like Mattermost so far? What do you like so far? Is there anything confusing or that you wish was better or different? This feedback will go to the Product team to help make improvements so any feedback is welcome!"
const feedbackRequestBody = "How can we make your experience better?"
const thanksFeedbackRequestBody = "Thanks! " + feedbackRequestBody
const feedbackResponseBody = ":tada: Thanks for helping us make Mattermost better!"
