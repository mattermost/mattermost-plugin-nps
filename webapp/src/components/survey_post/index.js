// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {doPostActionWithCookie} from 'mattermost-redux/actions/posts';

import {isSurveyPostSmall} from '../../selectors';

import SurveyPost from './survey_post';

function mapStateToProps(state, ownProps) {
    return {
        isSmall: isSurveyPostSmall(state, ownProps.isRHS),
    };
}

const mapDispatchToProps = {
    doPostActionWithCookie,
};

export default connect(mapStateToProps, mapDispatchToProps)(SurveyPost);
