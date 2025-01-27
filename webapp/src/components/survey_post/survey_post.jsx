// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import PropTypes from 'prop-types';
import React from 'react';

import {changeOpacity, makeStyleFromTheme} from 'mattermost-redux/utils/theme_utils';

import Score from './score';

export default class SurveyPost extends React.PureComponent {
    static propTypes = {
        doPostActionWithCookie: PropTypes.func.isRequired,
        isSmall: PropTypes.bool.isRequired,
        post: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
    }

    state = {
        disabled: false,
    }

    selectScore = (score) => {
        const action = this.getAction();

        this.props.doPostActionWithCookie(this.props.post.id, action.id, action.cookie, score.toString());
    }

    getAction = (index = 0) => {
        const {post} = this.props;
        if (!post || !post.props || !post.props.attachments) {
            return null;
        }

        const attachment = post.props.attachments[0];
        if (!attachment || !attachment.actions) {
            return null;
        }

        return attachment.actions[index];
    }

    getSelectedScore = () => {
        const action = this.getAction();
        if (!action || !action.default_option) {
            return -1;
        }

        try {
            return parseInt(action.default_option, 10);
        } catch (e) {
            return -1;
        }
    }

    renderScores = (style) => {
        const selectedScore = this.getSelectedScore();

        const scores = [];
        for (let i = 0; i <= 10; i++) {
            scores.push(
                <Score
                    key={i}
                    isSmall={this.props.isSmall}
                    selectScore={this.selectScore}
                    score={i}
                    selected={i === selectedScore}
                    theme={this.props.theme}
                />,
            );
        }

        return (
            <div style={this.props.isSmall ? style.scoreContainerSmall : style.scoreContainer}>
                <div style={style.scoreLabels}>
                    <span>{'Not Likely'}</span>
                    <span style={style.scoreLabelRight}>{'Very Likely'}</span>
                </div>
                <div style={this.props.isSmall ? style.scoresSmall : style.scores}>
                    {scores}
                </div>
            </div>
        );
    }

    render() {
        const style = getStyle(this.props.theme);
        const disable = () => {
            const action = this.getAction(1);
            this.props.doPostActionWithCookie(this.props.post.id, action.id, action.cookie).then(() => this.setState({disabled: true}));
        };
        const disabledMessage = (
            <div>
                <span>{'You won\'t receive any more surveys but you can submit feedback about Mattermost by typing here at anytime.'}</span>
            </div>
        );
        const disableAction = (
            <div>
                <span>{'Don\'t want to see this survey? '}</span>
                <a
                    href='#'
                    onClick={disable}
                >
                    {'Click here'}
                </a><span>{' to disable it.'}</span>
            </div>
        );
        const footer = this.state.disabled ? disabledMessage : disableAction;
        return (
            <React.Fragment>
                {window.PostUtils.messageHtmlToComponent(window.PostUtils.formatText(this.props.post.message, {atMentions: true}))}
                <div style={style.container}>
                    <h1 style={style.title}>{'How likely are you to recommend Mattermost?'}</h1>
                    {this.renderScores(style)}
                </div>
                {footer}
            </React.Fragment>
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
            backgroundColor: theme.centerChannelBg,
            borderBottomRightRadius: 4,
            borderColor: changeOpacity(theme.centerChannelColor, 0.3),
            borderWidth: 1, // This needs to appear above the line that sets borderLeftWidth since that should take precedence
            borderLeftColor: changeOpacity(theme.linkColor, 0.5),
            borderLeftWidth: 4,
            borderStyle: 'solid',
            borderTopRightRadius: 4,
            marginBottom: 5,
            marginTop: 5,
            padding: 12,
        },
        scoreContainer: {
            display: 'flex',
            flexDirection: 'column',
            lineHeight: 32,
            marginBottom: 4,
            width: (32 * 11) + (8 * 10), // The width of all 11 score buttons plus the margins between them
        },
        scoreContainerSmall: {
            display: 'flex',
            flexDirection: 'column',
            marginBottom: 4,
            width: 24 * 11, // The width of all 11 score buttons
        },
        scoreLabels: {
            display: 'flex',
            flexGrow: 1,
            fontSize: 12,
            fontWeight: 600,
            justifyContent: 'space-between',
            lineHeight: '17px',
            opacity: 0.5,
        },
        scores: {
            backgroundColor: changeOpacity(theme.linkColor, 0.08),
            borderRadius: '16px',
            height: 32,
            lineHeight: '32px',
            textAlign: 'center',
        },
        scoresSmall: {
            backgroundColor: changeOpacity(theme.linkColor, 0.08),
            borderRadius: '16px',
            height: 32,
            lineHeight: '24px',
            textAlign: 'center',
        },
        title: {
            fontSize: 16,
            fontWeight: 600,
            lineHeight: '22px',
            marginBottom: 5,
            marginTop: 5,
        },
    };
});
