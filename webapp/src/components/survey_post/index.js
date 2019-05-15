import {connect} from 'react-redux';

import {doPostActionWithCookie} from 'mattermost-redux/actions/posts';

import {isScorePostSmall} from '../../selectors';

import SurveyPost from './survey_post';

function mapStateToProps(state, ownProps) {
    return {
        isSmall: isScorePostSmall(state, ownProps.isRHS),
    };
}

const mapDispatchToProps = {
    doPostActionWithCookie,
};

export default connect(mapStateToProps, mapDispatchToProps)(SurveyPost);
