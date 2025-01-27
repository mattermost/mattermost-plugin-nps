// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

interface Props {
    isSmall: boolean;
    score: number;
    selected: boolean;
    selectScore: (score: number) => void;
    theme: {
        buttonBg: string;
        buttonColor: string;
    }
}

interface State {
    hovered: boolean;
}

export default class Score extends React.PureComponent<Props, State> {
    constructor(props: Props) {
        super(props);

        this.state = {
            hovered: false,
        };
    }

    handleClick = () => {
        this.props.selectScore(this.props.score);
    }

    handleMouseEnter = () => {
        this.setState({hovered: true});
    }

    handleMouseLeave = () => {
        this.setState({hovered: false});
    }

    render() {
        const containerStyle: Record<string, string|number> = {...style.container};
        if (this.props.score !== 10 && !this.props.isSmall) {
            containerStyle.marginRight = 8;
        }

        const bubbleStyle: Record<string, string|number> = this.props.isSmall ? {...style.bubbleSmall} : {...style.bubble};
        if (this.props.selected) {
            bubbleStyle.backgroundColor = this.props.theme.buttonBg;
            bubbleStyle.color = this.props.theme.buttonColor;
        } else if (this.state.hovered) {
            bubbleStyle.backgroundColor = changeOpacity(this.props.theme.buttonBg, 0.3);
        }

        return (
            <div
                style={containerStyle}
                onMouseEnter={this.handleMouseEnter}
                onMouseLeave={this.handleMouseLeave}
                onClick={this.handleClick}
                role='button'
            >
                <div style={bubbleStyle}>
                    {this.props.score}
                </div>
            </div>
        );
    }
}

const style = {
    bubble: {
        borderRadius: '100%',
        height: 32,
        width: 32,
    },
    bubbleSmall: {
        borderRadius: '100%',
        height: 24,
        marginTop: 4,
        width: 24,
    },
    container: {
        cursor: 'pointer',
        display: 'inline-block',
        height: '100%',
    },
};
