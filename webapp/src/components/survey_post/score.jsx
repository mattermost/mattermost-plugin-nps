import PropTypes from 'prop-types';
import React from 'react';

import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

export default class Score extends React.PureComponent {
    static propTypes = {
        score: PropTypes.number.isRequired,
        selected: PropTypes.bool.isRequired,
        selectScore: PropTypes.func.isRequired,
        theme: PropTypes.object.isRequired,
    }

    constructor(props) {
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
        const containerStyle = {...style.container};
        if (this.props.score !== 10) {
            containerStyle.marginRight = 8;
        }

        const bubbleStyle = {...style.bubble};
        if (this.props.selected) {
            bubbleStyle.backgroundColor = this.props.theme.sidebarBg;
            bubbleStyle.color = this.props.theme.sidebarText;
        } else if (this.state.hovered) {
            bubbleStyle.backgroundColor = changeOpacity(this.props.theme.sidebarBg, 0.1);
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
        borderRadius: 16,
    },
    container: {
        cursor: 'pointer',
        display: 'inline-block',
        height: 32,
        width: 32,
    },
};
