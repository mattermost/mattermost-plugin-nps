import React from 'react';

import ConfirmFeedbackModal from './confirm_feedback_modal';

export default class Root extends React.PureComponent {
    render() {
        return (
            <React.Fragment>
                <ConfirmFeedbackModal/>
                <style dangerouslySetInnerHTML={{__html: injectedCSS}}/>
            </React.Fragment>
        );
    }
}

const injectedCSS = `
.sidebar-item[href$="@surveybot"] .icon__bot::before {
    content: url('/plugins/com.mattermost.nps/assets/icon-happy-bot.svg');
    height: 14px;
    width: 16px;
}

.sidebar-item[href$="@surveybot"] .icon__bot > svg {
    display: none;
}
`;
